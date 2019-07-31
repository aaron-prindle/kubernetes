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

	"k8s.io/client-go/kubernetes"

	rmtypesv1a1 "k8s.io/api/flowcontrol/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/sets"
	apifilters "k8s.io/apiserver/pkg/endpoints/filters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	"k8s.io/client-go/kubernetes/fake"
)

func createRequestManagementServerAndClient(
	t *testing.T, delegate http.Handler, flowSchemas []*rmtypesv1a1.FlowSchema,
	priorityLevelConfigurations []*rmtypesv1a1.PriorityLevelConfiguration,
	serverConcurrencyLimit int, requestWaitLimit time.Duration) (*httptest.Server, kubernetes.Interface) {

	// TODO(aaron-prindle) verify interval fake clock setup is correct...
	now := time.Now()
	clk := &clock.IntervalClock{
		Time:     now,
		Duration: time.Millisecond,
	}

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

	handler := WithRequestManagementByClient(
		delegate,
		clientSet,
		serverConcurrencyLimit,
		requestWaitLimit,
		longRunningRequestCheck,
		clk,
	)

	handler = withFakeUser(handler)
	handler = apifilters.WithRequestInfo(handler, requestInfoFactory)

	return httptest.NewServer(handler), clientSet
}

// current tests assume serverConcurrencyLimit >= # of flowschemas
func TestReqMgmtManagementTwoRequests(t *testing.T) {
	flowSchemas := []*rmtypesv1a1.FlowSchema{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "fs-0"},
			Spec: rmtypesv1a1.FlowSchemaSpec{
				PriorityLevelConfiguration: rmtypesv1a1.PriorityLevelConfigurationReference{
					"plc-0",
				},
				// MatchingPrecedence: 1, // TODO(aaron-prindle) currently ignored
				// DistinguisherMethod: *rmtypesv1a1.FlowDistinguisherMethodByUserType, // TODO(aaron-prindle) currently ignored
				// Rules: []rmtypesv1a1.PolicyRuleWithSubjects{}, // TODO(aaron-prindle) currently ignored
			},
			// Status: rmtypesv1a1.FlowSchemaStatus{},
		},
	}

	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plc-0"},
			Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
				AssuredConcurrencyShares: 10,
				Exempt:                   false,
				GlobalDefault:            false,
				HandSize:                 int32(8),
				QueueLengthLimit:         int32(65536),
				Queues:                   int32(128),
			},
		},
	}

	serverConcurrencyLimit := 2
	requestWaitLimit := 1 * time.Millisecond
	requests := 2

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
	header.Add("PRIORITY", "0")
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
	flowSchemas := []*rmtypesv1a1.FlowSchema{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "fs-0"},
			Spec: rmtypesv1a1.FlowSchemaSpec{
				PriorityLevelConfiguration: rmtypesv1a1.PriorityLevelConfigurationReference{
					"plc-0",
				},
				// MatchingPrecedence: 1, // TODO(aaron-prindle) currently ignored
				// DistinguisherMethod: *rmtypesv1a1.FlowDistinguisherMethodByUserType, // TODO(aaron-prindle) currently ignored
				// Rules: []rmtypesv1a1.PolicyRuleWithSubjects{}, // TODO(aaron-prindle) currently ignored
			},
			// Status: rmtypesv1a1.FlowSchemaStatus{},
		},
	}

	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plc-0"},
			Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
				AssuredConcurrencyShares: 10,
				Exempt:                   false,
				GlobalDefault:            false,
				HandSize:                 int32(8),
				QueueLengthLimit:         int32(65536),
				Queues:                   int32(128),
			},
		},
	}

	serverConcurrencyLimit := 100
	requestWaitLimit := 1 * time.Millisecond
	requests := 5000

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
	header.Add("PRIORITY", "0")
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
	groups := 5
	flowSchemas := []*rmtypesv1a1.FlowSchema{}
	for i := 0; i < groups; i++ {
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
	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{}

	for i := 0; i < groups; i++ {
		priorityLevelConfigurations = append(priorityLevelConfigurations,
			&rmtypesv1a1.PriorityLevelConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("plc-%d", i)},
				Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
					AssuredConcurrencyShares: 1,
					Exempt:                   false,
					GlobalDefault:            false,
					HandSize:                 int32(8),
					QueueLengthLimit:         int32(65536),
					Queues:                   int32(128),
				},
			},
		)
	}

	serverConcurrencyLimit := groups
	requestWaitLimit := 5 * time.Second
	requests := 100

	countnum := len(flowSchemas)
	counts := make([]int64, len(flowSchemas))

	delegate := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		for i := 0; i < countnum; i++ {
			if r.Header.Get("PRIORITY") == strconv.Itoa(i) {
				atomic.AddInt64(&counts[i], int64(1))
			}
		}
	})

	server, _ := createRequestManagementServerAndClient(
		t,
		delegate, flowSchemas, priorityLevelConfigurations,
		serverConcurrencyLimit, requestWaitLimit,
	)
	defer server.Close()

	ws := []*Work{}
	for i := 0; i < countnum; i++ {
		header := make(http.Header)
		header.Add("PRIORITY", strconv.Itoa(i))
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
		w := w
		go func() { w.Run() }()
	}

	time.Sleep(1 * time.Second)
	for i, count := range counts {
		if count != 1 {
			t.Errorf("Expected to dispatch 1 request for Group %d, found %v", i, count)
		}
	}
}

// TestReqMgmtTimeout
func TestReqMgmtTimeout(t *testing.T) {
	flowSchemas := []*rmtypesv1a1.FlowSchema{
		{
			// metav1.TypeMeta
			ObjectMeta: metav1.ObjectMeta{Name: "fs-0"},
			Spec: rmtypesv1a1.FlowSchemaSpec{
				PriorityLevelConfiguration: rmtypesv1a1.PriorityLevelConfigurationReference{
					"plc-0",
				},
				// MatchingPrecedence: 1, // TODO(aaron-prindle) currently ignored
				// DistinguisherMethod: *rmtypesv1a1.FlowDistinguisherMethodByUserType, // TODO(aaron-prindle) currently ignored
				// Rules: []rmtypesv1a1.PolicyRuleWithSubjects{}, // TODO(aaron-prindle) currently ignored
			},
			// Status: rmtypesv1a1.FlowSchemaStatus{},
		},
	}

	priorityLevelConfigurations := []*rmtypesv1a1.PriorityLevelConfiguration{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plc-0"},
			Spec: rmtypesv1a1.PriorityLevelConfigurationSpec{
				AssuredConcurrencyShares: 10,
				Exempt:                   false,
				GlobalDefault:            false,
				HandSize:                 int32(1),
				QueueLengthLimit:         int32(65536),
				Queues:                   int32(1),
			},
		},
	}

	serverConcurrencyLimit := 100
	requestWaitLimit := 10 * time.Nanosecond
	requests := 1000

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
	header.Add("PRIORITY", "0")
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header = header

	w := &Work{
		Request: req,
		N:       requests,
		// NOTE: C >= serverConcurrencyLimit or else no packets are sent via Work
		C: 1000,
	}

	// Run blocks until all work is done.
	w.Run()

	// TODO(aaron-prindle) make this more exact
	// perhaps check for timeout text in stdout/stderr
	if int64(requests) == count && count > 0 {
		t.Errorf("Expected some requests to timeout, recieved all requests %v/%v",
			requests, count)
	}
}

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
