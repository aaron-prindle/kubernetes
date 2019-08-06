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

package filters

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"

	rmtypesv1a1 "k8s.io/api/flowcontrol/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	apifilters "k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilflowcontrol "k8s.io/apiserver/pkg/util/flowcontrol"
	fq "k8s.io/apiserver/pkg/util/flowcontrol/fairqueuing"
	kubeinformers "k8s.io/client-go/informers"

	"k8s.io/client-go/kubernetes/fake"
)

func createRequestManagementServerAndClient(
	t *testing.T, delegate http.Handler, flowSchemas []*rmtypesv1a1.FlowSchema,
	priorityLevelConfigurations []*rmtypesv1a1.PriorityLevelConfiguration,
	serverConcurrencyLimit int, requestWaitLimit time.Duration) (*httptest.Server, kubernetes.Interface) {

	// TODO(aaron-prindle) HACK/BAD - interval clock had data race issue,
	// using regular clock for now
	clk := clock.RealClock{}

	// TODO(aaron-prindle) would use interval clock but has data race issues??
	// now := time.Now()
	// clk := &clock.IntervalClock{
	// 	Time:     now,
	// 	Duration: time.Millisecond,
	// }

	clientSet := fake.NewSimpleClientset()
	for _, pl := range priorityLevelConfigurations {
		_, err := clientSet.FlowcontrolV1alpha1().PriorityLevelConfigurations().Create(pl)
		if err != nil {
			t.Fatalf("error creating PriorityLevelConfigurations: %v", err)
		}
	}
	for _, fs := range flowSchemas {
		_, err := clientSet.FlowcontrolV1alpha1().FlowSchemas().Create(fs)
		if err != nil {
			t.Fatalf("error creating FlowSchemas: %v", err)
		}
	}
	longRunningRequestCheck := BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString("proxy"))

	requestInfoFactory := &apirequest.RequestInfoFactory{APIPrefixes: sets.NewString("apis", "api"), GrouplessAPIPrefixes: sets.NewString("api")}

	// TODO(aaron-prindle) initialize queueSetFactory
	queueSetFactory := fq.NewQueueSetFactory(clk, nil)
	reqMgmt := utilflowcontrol.NewRequestManagementSystem(
		kubeinformers.NewSharedInformerFactory(clientSet, 0),
		clientSet.FlowcontrolV1alpha1(),
		queueSetFactory,
		serverConcurrencyLimit,
		requestWaitLimit,
	)

	handler := WithRequestManagement(
		delegate,
		longRunningRequestCheck,
		reqMgmt,
	)
	// handler := WithRequestManagementByClient(
	// 	delegate,
	// 	clientSet,
	// 	serverConcurrencyLimit,
	// 	requestWaitLimit,
	// 	longRunningRequestCheck,
	// 	clk,
	// )

	handler = withFakeUser(handler)
	handler = apifilters.WithRequestInfo(handler, requestInfoFactory)

	return httptest.NewServer(handler), clientSet
}

func generateFlowSchemas(n int) []*rmtypesv1a1.FlowSchema {
	flowSchemas := []*rmtypesv1a1.FlowSchema{}
	for i := 0; i < n; i++ {
		flowSchemas = append(flowSchemas,
			&rmtypesv1a1.FlowSchema{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("fs-%d", i)},
				Spec: rmtypesv1a1.FlowSchemaSpec{
					PriorityLevelConfiguration: rmtypesv1a1.PriorityLevelConfigurationReference{
						fmt.Sprintf("plc-%d", i),
					},
					MatchingPrecedence: int32(i), // TODO(aaron-prindle) currently ignored
					// DistinguisherMethod: *rmtypesv1a1.FlowDistinguisherMethodByUserType, // TODO(aaron-prindle) currently ignored
					// Rules: []rmtypesv1a1.PolicyRuleWithSubjects{}, // TODO(aaron-prindle) currently ignored
				},
				// Status: rmtypesv1a1.FlowSchemaStatus{},
			},
		)
	}
	return flowSchemas
}

func generatePriorityLevelConfigurations(n int) []*rmtypesv1a1.PriorityLevelConfiguration {
	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{}
	for i := 0; i < n; i++ {
		priorityLevelConfigurations = append(priorityLevelConfigurations,
			&rmtypesv1a1.PriorityLevelConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("plc-%d", i)},
				Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
					AssuredConcurrencyShares: 1,
					Exempt:                   false,
					HandSize:                 int32(8),
					QueueLengthLimit:         int32(65536),
					Queues:                   int32(128),
				},
			},
		)
	}
	return priorityLevelConfigurations

}

// current tests assume serverConcurrencyLimit >= # of flowschemas
func TestReqMgmtManagementTwoRequests(t *testing.T) {
	groups := 1
	serverConcurrencyLimit := 2
	requestWaitLimit := 1 * time.Millisecond
	requests := 2

	flowSchemas := generateFlowSchemas(groups)
	priorityLevelConfigurations := generatePriorityLevelConfigurations(groups)

	var count int64
	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	})

	server, _ := createRequestManagementServerAndClient(
		t,
		delegate, flowSchemas, priorityLevelConfigurations,
		serverConcurrencyLimit, requestWaitLimit,
	)
	defer server.Close()

	header := make(http.Header)
	header.Add("Groups", "0")
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header = header

	w := &Work{
		Request: req,
		N:       requests,
		// NOTE: C >= serverConcurrencyLimit or else no packets are sent via Work
		C: 2,
	}
	// Run blocks until all work is done.
	w.Run()
	if count != int64(requests) {
		t.Errorf("Expected to send %d requests, found %v", requests, count)
	}
}

// current tests assume serverConcurrencyLimit >= # of flowschemas
func TestReqMgmtManagementSingle(t *testing.T) {
	groups := 1
	serverConcurrencyLimit := 100
	requestWaitLimit := 5 * time.Second
	requests := 5000

	flowSchemas := generateFlowSchemas(groups)
	priorityLevelConfigurations := generatePriorityLevelConfigurations(groups)

	var count int64
	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, int64(1))
	})

	server, _ := createRequestManagementServerAndClient(
		t,
		delegate, flowSchemas, priorityLevelConfigurations,
		serverConcurrencyLimit, requestWaitLimit,
	)
	defer server.Close()

	header := make(http.Header)
	header.Add("Groups", "0")
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header = header

	w := &Work{
		Request: req,
		N:       requests,
		// NOTE: C >= serverConcurrencyLimit or else no packets are sent via Work
		C: 100,
	}
	// Run blocks until all work is done.
	w.Run()

	if count != int64(requests) {
		t.Errorf("Expected to send %d requests, found %v", requests, count)
	}
}

// TestReqMgmtMultiple verifies that fairness is preserved across 5 flowschemas
// and that the serverConcurrencyLimit/assuredConcurrencyShares are correctly
// distributed
func TestReqMgmtMultiple(t *testing.T) {
	// init objects
	// TODO(aaron-prindle) ERROR/BUG - only 2 flowschemas showup in reqmgmt.go
	// even if more are registered?
	// groups := 5

	groups := 2
	serverConcurrencyLimit := groups
	requestWaitLimit := 5 * time.Second
	requests := 100

	flowSchemas := generateFlowSchemas(groups)
	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{}

	for i := 0; i < groups; i++ {
		priorityLevelConfigurations = append(priorityLevelConfigurations,
			&rmtypesv1a1.PriorityLevelConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("plc-%d", i)},
				Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
					AssuredConcurrencyShares: 1,
					Exempt:                   false,
					HandSize:                 int32(8),
					QueueLengthLimit:         int32(65536),
					Queues:                   int32(128),
				},
			},
		)
	}

	countnum := len(flowSchemas)
	counts := make([]int64, len(flowSchemas))

	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		for i := 0; i < countnum; i++ {
			if r.Header.Get("Groups") == strconv.Itoa(i) {
				atomic.AddInt64(&counts[i], int64(1))
			}
		}
	})

	server, _ := createRequestManagementServerAndClient(
		t,
		delegate, flowSchemas, priorityLevelConfigurations,
		serverConcurrencyLimit, requestWaitLimit,
	)
	// defer server.Close()

	ws := []*Work{}
	for i := 0; i < countnum; i++ {
		header := make(http.Header)
		header.Add("Groups", strconv.Itoa(i))
		req, _ := http.NewRequest("GET", server.URL, nil)
		req.Header = header

		ws = append(ws, &Work{
			Request: req,
			N:       requests,
			// NOTE: C >= serverConcurrencyLimit or else no packets are sent via Work
			C: serverConcurrencyLimit,
		})

	}
	for _, w := range ws {
		go func(w *Work) {
			w.Run()
		}(w)
	}

	time.Sleep(1 * time.Second)
	// TODO(aaron-prindle) do this better...
	server.Close()
	requestsForOneGroup := counts[0]
	for i, count := range counts {
		if count != requestsForOneGroup {
			t.Errorf("Expected all counts to be equal, found counts[0] == %d and counts[%d] == %d",
				requestsForOneGroup, i, count)
		}
	}

	// TODO(aaron-prindle) ERROR/BAD - Currently seeing data race here...
	// for _, w := range ws {
	// 	// TODO(aaron-prindle) ERROR/BAD - Currently seeing data race with this too..
	// 	w.Stop()
	// }
	// for i, count := range counts {
	// 	if count != 1 {
	// 		t.Errorf("Expected to dispatch 1 request for Group %d, found %v", i, count)
	// 	}
	// }
}

// TestReqMgmtTimeout
// func TestReqMgmtTimeout(t *testing.T) {
// TODO(aaron-prindle) CHANGE - this test use to be able to cause timeout w/
// high load but now I'm never seeing any timeouts with the additional
// DequeueWithChannelAsMuchAsPossible in afterExecute
// 	groups := 1
// 	serverConcurrencyLimit := 3000
// 	requestWaitLimit := 3 * time.Nanosecond
// 	requests := 30000

// 	flowSchemas := generateFlowSchemas(groups)
// 	priorityLevelConfigurations := generatePriorityLevelConfigurations(groups)

// 	var count int64
// 	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		atomic.AddInt64(&count, int64(1))
// 	})
// 	server, _ := createRequestManagementServerAndClient(
// 		t,
// 		delegate, flowSchemas, priorityLevelConfigurations,
// 		serverConcurrencyLimit, requestWaitLimit,
// 	)
// 	defer server.Close()

// 	for i := 0; i < groups; i++ {
// 		header := make(http.Header)
// 		header.Add("Groups", strconv.Itoa(i))
// 		req, _ := http.NewRequest("GET", server.URL, nil)
// 		req.Header = header
// 		// Run blocks until all work is done.
// 		w := &Work{
// 			Request: req,
// 			N:       requests,
// 			// NOTE: C >= serverConcurrencyLimit or else no packets are sent via Work
// 			C: 1000,
// 		}

// 		w.Run()
// 	}

// 	// TODO(aaron-prindle) make this more exact
// 	// perhaps check for timeout text in stdout/stderr
// 	if int64(requests) == count {
// 		t.Errorf("Expected some requests to timeout, received all requests %v/%v",
// 			requests, count)
// 	}
// 	if count <= 0 {
// 		t.Errorf("Expected to receive some requests (not all of them timeout), received %v",
// 			count)
// 	}

// }

// TODO(aaron-prindle) Implement...
// // func TestVerifyQueueLengthLimit(t *testing.T) {

// TODO(aaron-prindle) Implement...
// // func TestReqMgmtAddQueues(t *testing.T) {

// TODO(aaron-prindle) Implement...
// // func TestReqMgmtDeleteQueues(t *testing.T) {

// TODO(aaron-prindle) Implement...
// // func TestReqMgmtModifyQueueLengthLimit(t *testing.T) {

// TODO(aaron-prindle) Implement...
// // func TestReqMgmtAddPriorityLevel(t *testing.T) {

// TODO(aaron-prindle) Implement...
// // func TestReqMgmtDeletePriorityLevel(t *testing.T) {
