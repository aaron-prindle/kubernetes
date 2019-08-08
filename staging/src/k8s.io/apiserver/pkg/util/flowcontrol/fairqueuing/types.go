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
	"sync"
	"time"
)

// FQPacket is an interface that a packet stored in an FQScheduler must support
type FQPacket interface {
	GetServiceTime() float64
	GetQueue() FQQueue
	SetQueue(FQQueue)
	GetStartTime() time.Time
	SetStartTime(time.Time)
}

// Packet is a temporary container for "requests" with additional tracking fields
// required for the functionality FQScheduler
type Packet struct {
	servicetime    float64
	Queue          FQQueue
	startTime      time.Time
	DequeueChannel chan bool
	EnqueueTime    time.Time
}

// GetServiceTime returns the servicetime of a packet
// the servicetime represents the amount of time elapsed from
// request dequeue to request finish
func (p *Packet) GetServiceTime() float64 {
	return p.servicetime
}

// GetQueue returns the queue that a packet is designated-for/inside
func (p *Packet) GetQueue() FQQueue {
	return p.Queue
}

// SetQueue sets the queue that a packet is designated-for/inside
func (p *Packet) SetQueue(queue FQQueue) {
	p.Queue = queue
}

// GetStartTime gets the starttime for a packet
// the starttime represents the time start time of a request after dispatching
func (p *Packet) GetStartTime() time.Time {
	return p.startTime
}

// SetStartTime gets the starttime for a packet
// the starttime represents the time start time of a request after dispatching
// starttime is set as a packet is dequeud
func (p *Packet) SetStartTime(starttime time.Time) {
	p.startTime = starttime

}

// FQQueue is an interface that a queue used in an FQScheduler must support
type FQQueue interface {
	GetPackets() []FQPacket
	SetPackets([]FQPacket)
	GetVirtualFinish(J int, G float64) float64
	GetVirStart() float64
	SetVirStart(float64)
	GetRequestsExecuting() int
	SetRequestsExecuting(int)
	GetIndex() int
	SetIndex(int)
	Enqueue(packet FQPacket)
	Dequeue() (FQPacket, bool)
}

// Queue is an array of packets with additional metadata required for
// the FQScheduler
type Queue struct {
	lock              sync.Mutex
	Packets           []FQPacket
	VirStart          float64
	RequestsExecuting int
	Index             int
}

// GetPackets gets the  packets for a queue
func (q *Queue) GetPackets() []FQPacket {
	return q.Packets
}

// SetPackets sets the  packets for a queue
func (q *Queue) SetPackets(pkts []FQPacket) {
	q.Packets = pkts
}

// GetRequestsExecuting gets the RequestsExecuting for a Queue
// RequestsExecuting represents the total # of packets that are being serviced
// serviced = dequeud but not finished
func (q *Queue) GetRequestsExecuting() int {
	// TODO(aaron-prindle) Seeing DataRace issue here? Did not before refactor/changes
	// q.lock.Lock()
	// defer q.lock.Unlock()

	return q.RequestsExecuting
}

// SetRequestsExecuting gets the RequestsExecuting for a Queue
// RequestsExecuting represents the total # of packets that are being serviced
// serviced = dequeud but not finished
func (q *Queue) SetRequestsExecuting(requestsExecuting int) {
	// TODO(aaron-prindle) Seeing DataRace issue here? Did not before refactor/changes
	// q.lock.Lock()
	// defer q.lock.Unlock()

	q.RequestsExecuting = requestsExecuting
}

// GetIndex gets the index for a queue
// The index represents it's position in an FQScheduler Queues array
// storing the index is required for quiesce logic and deletion
func (q *Queue) GetIndex() int {
	return q.Index
}

// SetIndex sets the index for a queue
func (q *Queue) SetIndex(idx int) {
	q.Index = idx
}

// GetVirStart gets the virtual start time for a queue
// this value is used to calculate the virtual finish time of packets
// which is used for fair queuing
func (q *Queue) GetVirStart() float64 {
	return q.VirStart
}

// SetVirStart gets the virtual start time for a queue
func (q *Queue) SetVirStart(virstart float64) {
	q.VirStart = virstart
}

// Enqueue enqueues a packet into the queue
func (q *Queue) Enqueue(packet FQPacket) {
	q.Packets = append(q.GetPackets(), packet)
}

// Dequeue dequeues a packet from the queue
func (q *Queue) Dequeue() (FQPacket, bool) {
	if len(q.Packets) == 0 {
		return nil, false
	}
	packet := q.Packets[0]
	q.Packets = q.Packets[1:]

	return packet, true
}

// GetVirtualFinish returns the expected virtual finish time of the packet at
// index J in the queue with estimated finish time G
func (q *Queue) GetVirtualFinish(J int, G float64) float64 {
	// The virtual finish time of request number J in the queue
	// (counting from J=1 for the head) is J * G + (virtual start time).

	// counting from J=1 for the head (eg: queue.Packets[0] -> J=1) - J+1
	jg := float64(J+1) * float64(G)
	return jg + q.VirStart
}
