/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fairqueuing

import (
	"math"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/klog"
)

// QueueSetFactoryImpl implements the QueueSetFactory interface
// QueueSetFactoryImpl makes QueueSetSystem objects.
// This filter makes a QueueSetSystem for each priority level.
type QueueSetFactoryImpl struct {
	// TODO(aaron-prindle) is waitgroup implicit like mutex?
	wg  *sync.WaitGroup
	clk clock.Clock
}

// NewQueueSetFactory creates a new NewQueueSetFactory object
func NewQueueSetFactory(clk clock.Clock, wg *sync.WaitGroup) QueueSetFactory {
	return &QueueSetFactoryImpl{
		wg:  wg,
		clk: clk,
	}
}

// NewQueueSet creates a new QueueSetSystem object
// There is a new QueueSet created for each priority level.
func (qsf QueueSetFactoryImpl) NewQueueSet(concurrencyLimit, desiredNumQueues, queueLengthLimit int, requestWaitLimit time.Duration) QueueSet {
	return NewQueueSetImpl(concurrencyLimit, desiredNumQueues,
		queueLengthLimit, requestWaitLimit, qsf.clk, qsf.wg)
}

// QueueSetImpl is a fair queuing implementation designed for the kube-apiserver.
// FQ is designed for
// 1) dispatching requests to be served rather than packets to be transmitted
// 2) serving multiple requests at once
// 3) accounting for unknown and varying service times
// implementation of:
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md
type QueueSetImpl struct {
	lock                 sync.Mutex
	wg                   *sync.WaitGroup
	queues               []FQQueue
	clk                  clock.Clock
	vt                   float64
	estimatedServiceTime float64
	lastRealTime         time.Time
	robinIdx             int
	// numPacketsEnqueued is the number of packets currently enqueued
	// (eg: incremeneted on Enqueue, decremented on Dequue)
	numPacketsEnqueued int
	concurrencyLimit   int
	desiredNumQueues   int
	queueLengthLimit   int
	requestWaitLimit   time.Duration
	quiescent          bool // emptyHandler?
}

// initQueues is a helper method for initializing an array of n queues
func initQueues(numQueues int) []FQQueue {
	fqqueues := make([]FQQueue, numQueues, numQueues)
	for i := 0; i < numQueues; i++ {
		fqqueues[i] = &Queue{Index: i, Packets: make([]FQPacket, 0)}

	}

	return fqqueues
}

// NewQueueSetImpl creates a new QueueSetImpl from passed in parameters and
func NewQueueSetImpl(concurrencyLimit, desiredNumQueues, queueLengthLimit int,
	requestWaitLimit time.Duration, clk clock.Clock, wg *sync.WaitGroup) QueueSetImpl {
	fq := QueueSetImpl{
		wg:               wg,
		queues:           initQueues(desiredNumQueues),
		clk:              clk,
		vt:               0,
		lastRealTime:     clk.Now(),
		desiredNumQueues: desiredNumQueues,
		concurrencyLimit: concurrencyLimit,
		queueLengthLimit: queueLengthLimit,
		requestWaitLimit: requestWaitLimit,
	}
	return fq
}

// LockAndSyncTime is used to ensure that the virtual time of a QueueSetImpl
// is synced everytime its fields are accessed
func (qs *QueueSetImpl) LockAndSyncTime() {
	qs.lock.Lock()
	qs.synctime()
}

// SetConfiguration is used to set the configuration for a QueueSetImpl
// update handling for when fields are updated is handled here as well -
// eg: if desiredNumQueues is increased, SetConfiguration reconciles by
// adding more queues.
func (qs QueueSetImpl) SetConfiguration(concurrencyLimit, desiredNumQueues, queueLengthLimit int, requestWaitLimit time.Duration) {
	// TODO(aaron-prindle) verify updating queues makes sense here vs elsewhere

	// lock required as method can change Queues which has its indexes and length used
	// concurrently
	qs.lock.Lock()
	defer qs.lock.Unlock()

	// Adding queues is the only thing that requires immediate action
	// Removing queues is handled by omitting indexes >desiredNumQueues from
	// chooseQueueIdx

	numQueues := len(qs.queues)
	if desiredNumQueues > numQueues {
		qs.addQueues(desiredNumQueues - numQueues)
	}

	qs.concurrencyLimit = concurrencyLimit
	qs.desiredNumQueues = desiredNumQueues
	qs.queueLengthLimit = queueLengthLimit
	qs.requestWaitLimit = requestWaitLimit
}

// TimeoutOldRequestsAndRejectOrEnqueue encapsulates the lock sharing logic required
// to validated and enqueue a request for the QueueSetImpl/QueueSetSystem:
// 1) Start with shuffle sharding, to pick a queue.
// 2) Reject old requests that have been waiting too long
// 3) Reject current request if there is not enough concurrency shares and
// we are at max queue length
// 4) If not rejected, create a packet and enqueue
// returns true on a successful enqueue
// returns false in the case that there is no available concurrency or
// the queuelengthlimit has been reached
func (qs *QueueSetImpl) TimeoutOldRequestsAndRejectOrEnqueue(hashValue uint64, handSize int32) FQPacket {
	// TODO(aaron-prindle) removing locking now and doing it all in Wait()
	// qs.lock.Lock()
	// defer qs.lock.Unlock()

	//	Start with the shuffle sharding, to pick a queue.
	queueIdx := qs.ChooseQueueIdx(hashValue, int(handSize))
	queue := qs.queues[queueIdx]
	// The next step is the logic to reject requests that have been waiting too long
	qs.removeTimedOutPacketsFromQueue(queue)
	// NOTE: currently timeout is only checked for each new request.  This means that there can be
	// requests that are in the queue longer than the timeout if there are no new requests
	// We think this is a fine tradeoff

	// Create a packet and enqueue
	pkt := &Packet{
		DequeueChannel: make(chan bool, 1),
		EnqueueTime:    qs.clk.Now(),
		Queue:          queue,
	}
	if ok := qs.rejectOrEnqueue(pkt); !ok {
		return nil
	}
	return pkt

}

// removeTimedOutPacketsFromQueue rejects old requests that have been enqueued
// past the requestWaitLimit
func (qs *QueueSetImpl) removeTimedOutPacketsFromQueue(queue FQQueue) {
	timeoutIdx := -1
	now := qs.clk.Now()
	pkts := queue.GetPackets()
	// pkts are sorted oldest -> newest
	// can short circuit loop (break) if oldest packets are not timing out
	// as newer packets also will not have timed out

	// now - requestWaitLimit = waitLimit
	waitLimit := now.Add(-qs.requestWaitLimit)
	for i, pkt := range pkts {
		channelPkt := pkt.(*Packet)
		if waitLimit.After(channelPkt.EnqueueTime) {
			if qs.wg != nil {
				qs.wg.Add(1)
			}
			channelPkt.DequeueChannel <- false
			close(channelPkt.DequeueChannel)
			// // TODO(aaron-prindle) verify this makes sense here
			// get idx for timed out packets
			timeoutIdx = i

		} else {
			break
		}
	}
	// remove timed out packets from queue
	if timeoutIdx != -1 {
		// timeoutIdx + 1 to remove the last timeout pkt
		removeIdx := timeoutIdx + 1

		// remove all the timeout packets
		queue.SetPackets(pkts[removeIdx:])
		qs.DecrementPackets(removeIdx)
	}
}

// DecrementPackets decreases the # of packets for the QueueSetImpl w/ lock
func (qs *QueueSetImpl) DecrementPackets(i int) {
	qs.numPacketsEnqueued -= i
}

// GetRequestsExecuting gets the # of requests which are "executing":
// this is the# of requests/packets which have been dequeued but have not had
// finished (via the FinishPacket method invoked after service)
func (qs *QueueSetImpl) GetRequestsExecuting() int {
	total := 0
	for _, queue := range qs.queues {
		total += queue.GetRequestsExecuting()
	}
	return total
}

func shuffleDealAndPick(v, nq uint64,
	lengthOfQueue func(int) int,
	mr func(int /*in [0, nq-1]*/) int, /*in [0, numQueues-1] and excluding previously determined members of I*/
	nRem, minLen, bestIdx int) int {
	if nRem < 1 {
		return bestIdx
	}
	vNext := v / nq
	ai := int(v - nq*vNext)
	ii := mr(ai)
	mrNext := func(a int /*in [0, nq-2]*/) int /*in [0, numQueues-1] and excluding I[0], I[1], ... ii*/ {
		if a < ai {
			return mr(a)
		}
		return mr(a + 1)
	}
	lenI := lengthOfQueue(ii)
	if lenI < minLen {
		minLen = lenI
		bestIdx = ii
	}
	return shuffleDealAndPick(vNext, nq-1, lengthOfQueue, mrNext, nRem-1, minLen, bestIdx)
}

// ChooseQueueIdx uses shuffle sharding to select an queue index
// using a 'hashValue'.  The 'hashValue' derives a hand from a set range of
// indexes (range 'desiredNumQueues') and returns the queue with the least queued packets
// from a dealt hand (of size 'handSize')
func (qs *QueueSetImpl) ChooseQueueIdx(hashValue uint64, handSize int) int {
	// TODO(aaron-prindle) currently a lock is held for this in a larger anonymous function
	// verify that makes sense...

	// desiredNumQueues is used here instead of numQueues to omit quiesce queues
	return shuffleDealAndPick(hashValue, uint64(qs.desiredNumQueues),
		func(idx int) int { return len(qs.queues[idx].GetPackets()) },
		func(i int) int { return i }, handSize, math.MaxInt32, -1)
}

// rejectOrEnqueue rejects or enqueues the newly arrived request if
// resource criteria isn't met
func (qs *QueueSetImpl) rejectOrEnqueue(packet FQPacket) bool {
	queue := packet.GetQueue()
	curQueueLength := len(queue.GetPackets())
	// rejects the newly arrived request if resource criteria not met
	if qs.GetRequestsExecuting() >= qs.concurrencyLimit &&
		curQueueLength >= qs.queueLengthLimit {
		return false
	}

	qs.enqueue(packet)
	return true
}

// enqueues a packet into an QueueSetImpl
func (qs *QueueSetImpl) enqueue(packet FQPacket) {

	queue := packet.GetQueue()
	queue.Enqueue(packet)
	qs.updateQueueVirStartTime(packet, queue)
	qs.numPacketsEnqueued++
}

// Enqueue enqueues a packet directly into an QueueSetImpl w/ no restriction
func (qs *QueueSetImpl) Enqueue(packet FQPacket) bool {
	qs.LockAndSyncTime()
	defer qs.lock.Unlock()

	qs.enqueue(packet)
	return true
}

// synctime is used to sync the time of the QueueSetImpl by looking at the elapsed
// time since the last sync and this value based on the 'virtualtime ratio'
// which scales inversely to the # of active flows
func (qs *QueueSetImpl) synctime() {
	realNow := qs.clk.Now()
	timesincelast := realNow.Sub(qs.lastRealTime).Seconds()
	qs.lastRealTime = realNow
	qs.vt += timesincelast * qs.getvirtualtimeratio()
}

func (qs *QueueSetImpl) getvirtualtimeratio() float64 {
	NEQ := 0
	reqs := 0
	for _, queue := range qs.queues {
		reqs += queue.GetRequestsExecuting()
		// It might be best to delete this line. If everything is working
		//  correctly, there will be no waiting packets if reqs < C on current
		//  line 85; if something is going wrong, it is more accurate to say
		// that virtual time advanced due to the requests actually executing.

		// reqs += len(queue.GetPackets())
		if len(queue.GetPackets()) > 0 || queue.GetRequestsExecuting() > 0 {
			NEQ++
		}
	}
	// no active flows, virtual time does not advance (also avoid div by 0)
	if NEQ == 0 {
		return 0
	}
	return math.Min(float64(reqs), float64(qs.concurrencyLimit)) / float64(NEQ)
}

// updateQueueVirStartTime updates the virtual start time for a queue
// this is done when a new packet is enqueued.  For more info see:
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md#dispatching
func (qs *QueueSetImpl) updateQueueVirStartTime(packet FQPacket, queue FQQueue) {
	// When a request arrives to an empty queue with no requests executing:
	// len(queue.GetPackets()) == 1 as enqueue has just happened prior (vs  == 0)
	if len(queue.GetPackets()) == 1 && queue.GetRequestsExecuting() == 0 {
		// the queue’s virtual start time is set to the virtual time.
		queue.SetVirStart(qs.vt)
	}
}

// removeQueueAndUpdateIndexes uses reslicing to remove an index from a slice
//  and then updates the 'Index' field of the queues to be correct
func removeQueueAndUpdateIndexes(queues []FQQueue, index int) []FQQueue {
	removedQueues := removeIndex(queues, index)
	for i := index; i < len(removedQueues); i++ {
		removedQueues[i].SetIndex(removedQueues[i].GetIndex() - 1)
	}
	return removedQueues
}

// removeIndex uses reslicing to remove an index from a slice
func removeIndex(s []FQQueue, index int) []FQQueue {
	return append(s[:index], s[index+1:]...)
}

// FinishPacketAndDequeueWithChannelAsMuchAsPossible is a convenience method which calls finishPacket
// for a given packet and then dequeues as many packets as possible
// and updates that packet's channel signifying it is is dequeued
// this is a callback used for the filter that the QueueSetImpl supports
func (qs *QueueSetImpl) FinishPacketAndDequeueWithChannelAsMuchAsPossible(pkt FQPacket) {
	qs.LockAndSyncTime()
	defer qs.lock.Unlock()

	qs.finishPacket(pkt)
	qs.DequeueWithChannelAsMuchAsPossible()
}

// FinishPacket is a callback that should be used when a previously dequeud packet
// has completed it's service.  This callback updates imporatnt state in the
// QueueSetImpl
func (qs *QueueSetImpl) finishPacket(p FQPacket) {

	S := qs.clk.Since(p.GetStartTime()).Seconds()

	// When a request finishes being served, and the actual service time was S,
	// the queue’s virtual start time is decremented by G - S.
	virstart := p.GetQueue().GetVirStart()
	virstart -= qs.estimatedServiceTime - S
	p.GetQueue().SetVirStart(virstart)

	// request has finished, remove from requests executing
	requestsExecuting := p.GetQueue().GetRequestsExecuting()
	requestsExecuting--
	p.GetQueue().SetRequestsExecuting(requestsExecuting)

	// Logic to remove quiesced queues
	// >= as QueueIdx=25 is out of bounds for desiredNumQueues=25 [0...24]
	if p.GetQueue().GetIndex() >= qs.desiredNumQueues &&
		len(p.GetQueue().GetPackets()) == 0 &&
		p.GetQueue().GetRequestsExecuting() == 0 {
		qs.queues = removeQueueAndUpdateIndexes(qs.queues, p.GetQueue().GetIndex())
		// At this point, if the qs is quiescing,
		// has zero requests executing, and has zero requests enqueued
		// then a call to the EmptyHandler should be forked.
		if qs.quiescent && qs.numPacketsEnqueued == 0 &&
			qs.GetRequestsExecuting() == 0 {
			// then a call to the EmptyHandler should be forked.
			go func() {
				// TODO(aaron-prindle) store the emptyHandler to call it here?
			}()
		}
	}
}

// dequeue dequeues a packet from the QueueSetImpl
func (qs *QueueSetImpl) dequeue() (FQPacket, bool) {

	queue := qs.selectQueue()

	if queue == nil {
		return nil, false
	}
	packet, ok := queue.Dequeue()

	if ok {
		// When a request is dequeued for service -> qs.VirStart += G
		virstart := queue.GetVirStart()
		virstart += qs.estimatedServiceTime
		queue.SetVirStart(virstart)

		packet.SetStartTime(qs.clk.Now())
		// request dequeued, service has started
		queue.SetRequestsExecuting(queue.GetRequestsExecuting() + 1)
	} else {
		// TODO(aaron-prindle) verify this statement is needed...
		return nil, false
	}
	qs.numPacketsEnqueued--
	return packet, ok
}

// Dequeue dequeues a packet from the QueueSetImpl
func (qs *QueueSetImpl) Dequeue() (FQPacket, bool) {
	qs.LockAndSyncTime()
	defer qs.lock.Unlock()
	return qs.dequeue()
}

// isEmpty is a convenience method that returns 'true' when all of the queues
// in an QueueSetImpl have no packets (and is "empty")
func (qs *QueueSetImpl) isEmpty() bool {
	return qs.numPacketsEnqueued == 0
}

// DequeueWithChannelAsMuchAsPossible runs a loop, as long as there
// are non-empty queues and the number currently executing is less than the
// assured concurrency value.  The body of the loop uses the fair queuing
// technique to pick a queue, dequeue the request at the head of that
// queue, increment the count of the number executing, and send `{true,
// handleCompletion(that dequeued request)}` to the request's channel.
func (qs *QueueSetImpl) DequeueWithChannelAsMuchAsPossible() {
	for !qs.isEmpty() && qs.GetRequestsExecuting() < qs.concurrencyLimit {
		_, ok := qs.dequeueWithChannel()
		// TODO(aaron-prindle) verify checking ok makes senes
		if !ok {
			break
		}
	}
}

// dequeueWithChannel is convenience method for dequeueing packets that
// require a message to be sent through the packets channel
// this is a required pattern for the QueueSetSystem the QueueSetImpl supports
func (qs *QueueSetImpl) dequeueWithChannel() (FQPacket, bool) {
	packet, ok := qs.dequeue()
	if !ok {
		return nil, false
	}
	reqMgmtPkt, conversionOK := packet.(*Packet)
	if !conversionOK {
		// TODO(aaron-prindle) log an error
		return nil, false
	}
	if qs.wg != nil {
		qs.wg.Add(1)
	}
	reqMgmtPkt.DequeueChannel <- true
	return packet, ok
}

func (qs *QueueSetImpl) roundrobinqueue() int {
	qs.robinIdx = (qs.robinIdx + 1) % len(qs.queues)
	return qs.robinIdx
}

// selectQueue selects the minimum virtualfinish time from the set of queues
// the starting queue is selected via roundrobin
// TODO(aaron-prindle) verify that the roundrobin usage is correct
// unsure if the code currently prioritizes the correct queues for ties
func (qs *QueueSetImpl) selectQueue() FQQueue {
	minvirfinish := math.Inf(1)
	var minqueue FQQueue
	var minidx int
	for range qs.queues {
		// TODO(aaron-prindle) how should this work with queue deletion?
		idx := qs.roundrobinqueue()
		queue := qs.queues[idx]
		if len(queue.GetPackets()) != 0 {
			curvirfinish := queue.GetVirtualFinish(0, qs.estimatedServiceTime)
			if curvirfinish < minvirfinish {
				minvirfinish = curvirfinish
				minqueue = queue
				minidx = idx
			}
		}
	}
	qs.robinIdx = minidx
	return minqueue
}

// AddQueues adds additional queues to the QueueSetImpl
// the complementary DeleteQueues is not an explicit fxn as queue deletion requires draining
// the queues first, queue deletion is done for the proper cases
// in the the FinishPacket function
func (qs *QueueSetImpl) addQueues(n int) {
	for i := 0; i < n; i++ {
		qs.queues = append(qs.queues, &Queue{
			Packets: []FQPacket{},
		})
	}
}

// ===========================================================================
// ===========================================================================

// Quiesce controls whether this system is quiescing.  Passing a
// non-nil handler means the system should become quiescent, a nil
// handler means the system should become non-quiescent.  A call
// to Wait while the system is quiescent will be rebuffed by
// returning `quiescent=true`.  If all the queues have no requests
// waiting nor executing while the system is quiescent then the
// handler will eventually be called with no locks held (even if
// the system becomes non-quiescent between the triggering state
// and the required call).
//
// The filter uses this for a priority level that has become
// undesired, setting a handler that will cause the priority level
// to eventually be removed from the filter if the filter still
// wants that.  If the filter later changes its mind and wants to
// preserve the priority level then the filter can use this to
// cancel the handler registration.
func (qs QueueSetImpl) Quiesce(eh EmptyHandler) {
	qs.lock.Lock()
	defer qs.lock.Unlock()
	if eh == nil {
		qs.quiescent = false
		return
	}
	// Here we check whether there are any requests queued or executing and
	// if not then fork an invocation of the EmptyHandler.
	if qs.numPacketsEnqueued == 0 && qs.GetRequestsExecuting() == 0 {
		// fork an invocation of the EmptyHandler.
		go func() {
			eh.HandleEmpty()
		}()
	}
	qs.quiescent = true
}

// Wait in the happy case, shuffle shards the given request into
// a queue and eventually dispatches the request from that queue.
// Dispatching means to return with `quiescent==false` and
// `execute==true`.  In one unhappy case the request is
// immediately rebuffed with `quiescent==true` (which tells the
// filter that there has been a timing splinter and the filter
// re-calcuates the priority level to use); in all other cases
// `quiescent` will be returned `false` (even if the system is
// quiescent by then).  In the non-quiescent unhappy cases the
// request is eventually rejected, which means to return with
// `execute=false`.  In the happy case the caller is required to
// invoke the returned `afterExecution` after the request is done
// executing.  The hash value and hand size are used to do the
// shuffle sharding.
func (qs QueueSetImpl) Wait(hashValue uint64, handSize int32) (quiescent, execute bool, afterExecution func()) {
	// TODO(aaron-prindle) verify what should/shouldn't be locked!!!!
	// TODO(aaron-prindle) collapse all of FQ into one layer/lock (vs 3)
	//   currently able to collapse to 1 impl layer and 2 locks...

	qs.LockAndSyncTime()
	// TODO(aaron-prindle) verify and test quiescent
	// A call to Wait while the system is quiescent will be rebuffed by
	// returning `quiescent=true`.
	if qs.quiescent {
		return true, false, func() {}
	}

	// ========================================================================
	// Step 1:
	// 1) Start with shuffle sharding, to pick a queue.
	// 2) Reject old requests that have been waiting too long
	// 3) Reject current request if there is not enough concurrency shares and
	// we are at max queue length
	// 4) If not rejected, create a packet and enqueue
	pkt := qs.TimeoutOldRequestsAndRejectOrEnqueue(hashValue, handSize)
	// pkt == nil means that the request was rejected - no remaining
	// concurrency shares and at max queue length already
	if pkt == nil {
		return false, false, func() {}
	}
	// ========================================================================

	// ------------------------------------------------------------------------
	// Step 2:
	// 1) The next step is to invoke the method that dequeues as much as possible.

	// This method runs a loop, as long as there
	// are non-empty queues and the number currently executing is less than the
	// assured concurrency value.  The body of the loop uses the fair queuing
	// technique to pick a queue, dequeue the request at the head of that
	// queue, increment the count of the number executing, and send `{true,
	// handleCompletion(that dequeued request)}` to the request's channel.
	qs.DequeueWithChannelAsMuchAsPossible()
	// ------------------------------------------------------------------------
	qs.lock.Unlock()

	// ************************************************************************
	// Step 3:
	// After that method finishes its loop and returns, the final step in Wait
	// is to `select` on either request timeout or receipt of a record on the
	// newly arrived request's channel, and return appropriately.  If a record
	// has been sent to the request's channel then this `select` will
	// immediately complete
	if qs.wg != nil {
		qs.wg.Done()
	}
	channelPkt := pkt.(*Packet)
	select {
	case execute := <-channelPkt.DequeueChannel:
		if execute {
			// execute
			return false, true, func() {
				qs.FinishPacketAndDequeueWithChannelAsMuchAsPossible(pkt)
			}

		}
		// timed out
		klog.V(5).Infof("channelPkt.DequeueChannel timed out\n")
		return false, false, func() {}
	}
	// ************************************************************************
}
