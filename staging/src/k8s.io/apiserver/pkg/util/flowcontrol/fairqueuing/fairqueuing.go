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
)

// FQScheduler is a fair queuing implementation designed for the kube-apiserver.
// FQ is designed for
// 1) dispatching requests to be served rather than packets to be transmitted
// 2) serving multiple requests at once
// 3) accounting for unknown and varying service times
// implementation of:
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md
type FQScheduler struct {
	Lock                 sync.Mutex
	Queues               []FQQueue
	clock                clock.Clock
	vt                   float64
	estimatedServiceTime float64
	lastRealTime         time.Time
	robinIdx             int
	// NumPacketsEnqueued is the number of packets currently enqueued
	// (eg: incremeneted on Enqueue, decremented on Dequue)
	NumPacketsEnqueued int
	concurrencyLimit   int
	desiredNumQueues   int
	queueLengthLimit   int
	requestWaitLimit   time.Duration
	Quiescent          bool // emptyHandler
}

// initQueues is a helper method for initializing an array of n queues
func initQueues(numQueues int) []FQQueue {
	fqqueues := make([]FQQueue, numQueues, numQueues)
	for i := 0; i < numQueues; i++ {
		fqqueues[i] = &Queue{Index: i, Packets: make([]FQPacket, 0)}

	}

	return fqqueues
}

// NewFQScheduler creates a new FQScheduler from passed in parameters and
func NewFQScheduler(concurrencyLimit, desiredNumQueues, queueLengthLimit int,
	requestWaitLimit time.Duration, clock clock.Clock) *FQScheduler {
	fq := &FQScheduler{
		Queues:           initQueues(desiredNumQueues),
		clock:            clock,
		vt:               0,
		lastRealTime:     clock.Now(),
		desiredNumQueues: desiredNumQueues,
		concurrencyLimit: concurrencyLimit,
		queueLengthLimit: queueLengthLimit,
		requestWaitLimit: requestWaitLimit,
	}
	return fq
}

// LockAndSyncTime is used to ensure that the virtual time of a FQScheduler
// is synced everytime its fields are accessed
func (fqs *FQScheduler) LockAndSyncTime() {
	fqs.Lock.Lock()
	fqs.synctime()
}

// SetConfiguration is used to set the configuration for a FQScheduler
// update handling for when fields are updated is handled here as well -
// eg: if desiredNumQueues is increased, SetConfiguration reconciles by
// adding more queues.
func (fqs *FQScheduler) SetConfiguration(concurrencyLimit, desiredNumQueues, queueLengthLimit int, requestWaitLimit time.Duration) {
	// TODO(aaron-prindle) verify updating queues makes sense here vs elsewhere

	// lock required as method can change Queues which has its indexes and length used
	// concurrently
	fqs.Lock.Lock()
	defer fqs.Lock.Unlock()

	// Adding queues is the only thing that requires immediate action
	// Removing queues is handled by omitting indexes >desiredNumQueues from
	// chooseQueueIdx

	numQueues := len(fqs.Queues)
	if desiredNumQueues > numQueues {
		fqs.addQueues(desiredNumQueues - numQueues)
	}

	fqs.concurrencyLimit = concurrencyLimit
	fqs.desiredNumQueues = desiredNumQueues
	fqs.queueLengthLimit = queueLengthLimit
	fqs.requestWaitLimit = requestWaitLimit
}

// TimeoutOldRequestsAndRejectOrEnqueue encapsulates the lock sharing logic required
// to validated and enqueue a request for the FQScheduler/FairQueueingSystem:
// 1) Start with shuffle sharding, to pick a queue.
// 2) Reject old requests that have been waiting too long
// 3) Reject current request if there is not enough concurrency shares and
// we are at max queue length
// 4) If not rejected, create a packet and enqueue
// returns true on a successful enqueue
// returns false in the case that there is no available concurrency or
// the queuelengthlimit has been reached
func (fqs *FQScheduler) TimeoutOldRequestsAndRejectOrEnqueue(hashValue uint64, handSize int32) FQPacket {
	// TODO(aaron-prindle) removing locking now and doing it all in Wait()
	// fqs.Lock.Lock()
	// defer fqs.Lock.Unlock()

	//	Start with the shuffle sharding, to pick a queue.
	queueIdx := fqs.ChooseQueueIdx(hashValue, int(handSize))
	queue := fqs.Queues[queueIdx]
	// The next step is the logic to reject requests that have been waiting too long
	fqs.removeTimedOutPacketsFromQueue(queue)
	// NOTE: currently timeout is only checked for each new request.  This means that there can be
	// requests that are in the queue longer than the timeout if there are no new requests
	// We think this is a fine tradeoff

	// Create a packet and enqueue
	pkt := &Packet{
		DequeueChannel: make(chan bool, 1),
		EnqueueTime:    fqs.clock.Now(),
		Queue:          queue,
	}
	if ok := fqs.rejectOrEnqueue(pkt); !ok {
		return nil
	}
	return pkt

}

// removeTimedOutPacketsFromQueue rejects old requests that have been enqueued
// past the requestWaitLimit
func (fqs *FQScheduler) removeTimedOutPacketsFromQueue(queue FQQueue) {
	timeoutIdx := -1
	now := fqs.clock.Now()
	pkts := queue.GetPackets()
	// pkts are sorted oldest -> newest
	// can short circuit loop (break) if oldest packets are not timing out
	// as newer packets also will not have timed out

	// now - requestWaitLimit = waitLimit
	waitLimit := now.Add(-fqs.requestWaitLimit)
	for i, pkt := range pkts {
		channelPkt := pkt.(*Packet)
		if waitLimit.After(channelPkt.EnqueueTime) {
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
		fqs.DecrementPackets(removeIdx)
	}
}

// DecrementPackets decreases the # of packets for the FQScheduler w/ lock
func (fqs *FQScheduler) DecrementPackets(i int) {
	fqs.NumPacketsEnqueued -= i
}

// GetRequestsExecuting gets the # of requests which are "executing":
// this is the# of requests/packets which have been dequeued but have not had
// finished (via the FinishPacket method invoked after service)
func (fqs *FQScheduler) GetRequestsExecuting() int {
	total := 0
	for _, queue := range fqs.Queues {
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

// // shuffleDealAndPick uses shuffle sharding to select an index from a set of queues
// func (fqs *FQScheduler) shuffleDealAndPick(v, nq uint64,
// 	mr func(int /*in [0, nq-1]*/) int, /*in [0, numQueues-1] and excluding previously determined members of I*/
// 	nRem, minLen, bestIdx int) int {
// 	if nRem < 1 {
// 		return bestIdx
// 	}
// 	vNext := v / nq
// 	ai := int(v - nq*vNext)
// 	ii := mr(ai)
// 	mrNext := func(a int /*in [0, nq-2]*/) int /*in [0, numQueues-1] and excluding I[0], I[1], ... ii*/ {
// 		if a < ai {
// 			return mr(a)
// 		}
// 		return mr(a + 1)
// 	}
// 	lenI := fqs.lengthOfQueue(ii)
// 	if lenI < minLen {
// 		minLen = lenI
// 		bestIdx = ii
// 	}
// 	return fqs.shuffleDealAndPick(vNext, nq-1, mrNext, nRem-1, minLen, bestIdx)
// }

// ChooseQueueIdx uses shuffle sharding to select an queue index
// using a 'hashValue'.  The 'hashValue' derives a hand from a set range of
// indexes (range 'desiredNumQueues') and returns the queue with the least queued packets
// from a dealt hand (of size 'handSize')
func (fqs *FQScheduler) ChooseQueueIdx(hashValue uint64, handSize int) int {
	// TODO(aaron-prindle) currently a lock is held for this in a larger anonymous function
	// verify that makes sense...

	// desiredNumQueues is used here instead of numQueues to omit quiesce queues
	return shuffleDealAndPick(hashValue, uint64(fqs.desiredNumQueues),
		func(idx int) int { return len(fqs.Queues[idx].GetPackets()) },
		func(i int) int { return i }, handSize, math.MaxInt32, -1)
}

// rejectOrEnqueue rejects or enqueues the newly arrived request if
// resource criteria isn't met
func (fqs *FQScheduler) rejectOrEnqueue(packet FQPacket) bool {
	queue := packet.GetQueue()
	curQueueLength := len(queue.GetPackets())
	// rejects the newly arrived request if resource criteria not met
	if fqs.GetRequestsExecuting() >= fqs.concurrencyLimit &&
		curQueueLength >= fqs.queueLengthLimit {
		return false
	}

	fqs.enqueue(packet)
	return true
}

// enqueues a packet into an FQScheduler
func (fqs *FQScheduler) enqueue(packet FQPacket) {

	queue := packet.GetQueue()
	queue.Enqueue(packet)
	fqs.updateQueueVirStartTime(packet, queue)
	fqs.NumPacketsEnqueued++
}

// Enqueue enqueues a packet directly into an FQScheduler w/ no restriction
func (fqs *FQScheduler) Enqueue(packet FQPacket) bool {
	fqs.LockAndSyncTime()
	defer fqs.Lock.Unlock()

	fqs.enqueue(packet)
	return true
}

// synctime is used to sync the time of the FQScheduler by looking at the elapsed
// time since the last sync and this value based on the 'virtualtime ratio'
// which scales inversely to the # of active flows
func (fqs *FQScheduler) synctime() {
	realNow := fqs.clock.Now()
	timesincelast := realNow.Sub(fqs.lastRealTime).Seconds()
	fqs.lastRealTime = realNow
	fqs.vt += timesincelast * fqs.getvirtualtimeratio()
}

func (fqs *FQScheduler) getvirtualtimeratio() float64 {
	NEQ := 0
	reqs := 0
	for _, queue := range fqs.Queues {
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
	return math.Min(float64(reqs), float64(fqs.concurrencyLimit)) / float64(NEQ)
}

// updateQueueVirStartTime updates the virtual start time for a queue
// this is done when a new packet is enqueued.  For more info see:
// https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md#dispatching
func (fqs *FQScheduler) updateQueueVirStartTime(packet FQPacket, queue FQQueue) {
	// When a request arrives to an empty queue with no requests executing:
	// len(queue.GetPackets()) == 1 as enqueue has just happened prior (vs  == 0)
	if len(queue.GetPackets()) == 1 && queue.GetRequestsExecuting() == 0 {
		// the queue’s virtual start time is set to the virtual time.
		queue.SetVirStart(fqs.vt)
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

// FinishPacketAndDequeueNextPacket is a convenience method which calls finishPacket
// for a given packet and then dequeues a packet and updates that packet's channel
// signifying it is is dequeued
// this is a callback used for the FairQueuingSystem the FQScheduler supports
func (fqs *FQScheduler) FinishPacketAndDequeueNextPacket(pkt FQPacket) {
	fqs.LockAndSyncTime()
	defer fqs.Lock.Unlock()

	fqs.finishPacket(pkt)
	fqs.dequeueWithChannel()
}

// FinishPacket is a callback that should be used when a previously dequeud packet
// has completed it's service.  This callback updates imporatnt state in the
// FQScheduler
func (fqs *FQScheduler) finishPacket(p FQPacket) {

	S := fqs.clock.Since(p.GetStartTime()).Seconds()

	// When a request finishes being served, and the actual service time was S,
	// the queue’s virtual start time is decremented by G - fqs.
	virstart := p.GetQueue().GetVirStart()
	virstart -= fqs.estimatedServiceTime - S
	p.GetQueue().SetVirStart(virstart)

	// request has finished, remove from requests executing
	requestsExecuting := p.GetQueue().GetRequestsExecuting()
	requestsExecuting--
	p.GetQueue().SetRequestsExecuting(requestsExecuting)

	// Logic to remove quiesced queues
	// >= as QueueIdx=25 is out of bounds for desiredNumQueues=25 [0...24]
	if p.GetQueue().GetIndex() >= fqs.desiredNumQueues &&
		len(p.GetQueue().GetPackets()) == 0 &&
		p.GetQueue().GetRequestsExecuting() == 0 {
		fqs.Queues = removeQueueAndUpdateIndexes(fqs.Queues, p.GetQueue().GetIndex())
		// At this point, if the fqs is quiescing,
		// has zero requests executing, and has zero requests enqueued
		// then a call to the EmptyHandler should be forked.
		if fqs.Quiescent && fqs.NumPacketsEnqueued == 0 &&
			fqs.GetRequestsExecuting() == 0 {
			// then a call to the EmptyHandler should be forked.
			go func() {
				// TODO(aaron-prindle) store the emptyHandler to call it here?
			}()
		}
	}
}

// dequeue dequeues a packet from the FQScheduler
func (fqs *FQScheduler) dequeue() (FQPacket, bool) {

	queue := fqs.selectQueue()

	if queue == nil {
		return nil, false
	}
	packet, ok := queue.Dequeue()

	if ok {
		// When a request is dequeued for service -> fqs.VirStart += G
		virstart := queue.GetVirStart()
		virstart += fqs.estimatedServiceTime
		queue.SetVirStart(virstart)

		packet.SetStartTime(fqs.clock.Now())
		// request dequeued, service has started
		queue.SetRequestsExecuting(queue.GetRequestsExecuting() + 1)
	} else {
		// TODO(aaron-prindle) verify this statement is needed...
		return nil, false
	}
	fqs.NumPacketsEnqueued--
	return packet, ok
}

// Dequeue dequeues a packet from the FQScheduler
func (fqs *FQScheduler) Dequeue() (FQPacket, bool) {
	fqs.LockAndSyncTime()
	defer fqs.Lock.Unlock()
	return fqs.dequeue()
}

// isEmpty is a convenience method that returns 'true' when all of the queues
// in an FQScheduler have no packets (and is "empty")
func (fqs *FQScheduler) isEmpty() bool {
	return fqs.NumPacketsEnqueued == 0
}

// DequeueWithChannelAsMuchAsPossible runs a loop, as long as there
// are non-empty queues and the number currently executing is less than the
// assured concurrency value.  The body of the loop uses the fair queuing
// technique to pick a queue, dequeue the request at the head of that
// queue, increment the count of the number executing, and send `{true,
// handleCompletion(that dequeued request)}` to the request's channel.
func (fqs *FQScheduler) DequeueWithChannelAsMuchAsPossible() {
	for !fqs.isEmpty() && fqs.GetRequestsExecuting() < fqs.concurrencyLimit {
		_, ok := fqs.dequeueWithChannel()
		// TODO(aaron-prindle) verify checking ok makes senes
		if !ok {
			break
		}
	}
}

// dequeueWithChannel is convenience method for dequeueing packets that
// require a message to be sent through the packets channel
// this is a required pattern for the FairQueuingSystem the FQScheduler supports
func (fqs *FQScheduler) dequeueWithChannel() (FQPacket, bool) {
	packet, ok := fqs.dequeue()
	if !ok {
		return nil, false
	}
	reqMgmtPkt, conversionOK := packet.(*Packet)
	if !conversionOK {
		// TODO(aaron-prindle) log an error
		return nil, false
	}
	reqMgmtPkt.DequeueChannel <- true
	return packet, ok
}

func (fqs *FQScheduler) roundrobinqueue() int {
	fqs.robinIdx = (fqs.robinIdx + 1) % len(fqs.Queues)
	return fqs.robinIdx
}

// selectQueue selects the minimum virtualfinish time from the set of queues
// the starting queue is selected via roundrobin
// TODO(aaron-prindle) verify that the roundrobin usage is correct
// unsure if the code currently prioritizes the correct queues for ties
func (fqs *FQScheduler) selectQueue() FQQueue {
	minvirfinish := math.Inf(1)
	var minqueue FQQueue
	var minidx int
	for range fqs.Queues {
		// TODO(aaron-prindle) how should this work with queue deletion?
		idx := fqs.roundrobinqueue()
		queue := fqs.Queues[idx]
		if len(queue.GetPackets()) != 0 {
			curvirfinish := queue.GetVirtualFinish(0, fqs.estimatedServiceTime)
			if curvirfinish < minvirfinish {
				minvirfinish = curvirfinish
				minqueue = queue
				minidx = idx
			}
		}
	}
	fqs.robinIdx = minidx
	return minqueue
}

// AddQueues adds additional queues to the FQScheduler
// the complementary DeleteQueues is not an explicit fxn as queue deletion requires draining
// the queues first, queue deletion is done for the proper cases
// in the the FinishPacket function
func (fqs *FQScheduler) addQueues(n int) {
	for i := 0; i < n; i++ {
		fqs.Queues = append(fqs.Queues, &Queue{
			Packets: []FQPacket{},
		})
	}
}
