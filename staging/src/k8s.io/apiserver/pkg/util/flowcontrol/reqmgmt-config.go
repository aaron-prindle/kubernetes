/*
Copyright 2019 The Kubernetes Authors.

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

package flowcontrol

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	fq "k8s.io/apiserver/pkg/util/flowcontrol/fairqueuing"
	kubeinformers "k8s.io/client-go/informers"
	cache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	rmtypesv1a1 "k8s.io/api/flowcontrol/v1alpha1"
)

// initializeConfigController sets up the controller that processes
// config API objects.
func (reqMgmt *requestManagementSystem) initializeConfigController(informerFactory kubeinformers.SharedInformerFactory) {
	reqMgmt.configQueue = workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(200*time.Millisecond, 8*time.Hour), "req_mgmt_config_queue")
	fci := informerFactory.Flowcontrol().V1alpha1()
	pli := fci.PriorityLevelConfigurations()
	fsi := fci.FlowSchemas()
	reqMgmt.plInformer = pli.Informer()
	reqMgmt.plLister = pli.Lister()
	reqMgmt.fsInformer = fsi.Informer()
	reqMgmt.fsLister = fsi.Lister()
	reqMgmt.plInformer.AddEventHandler(reqMgmt)
	reqMgmt.fsInformer.AddEventHandler(reqMgmt)
}

// OnAdd handles notification from an informer of object creation
func (reqMgmt *requestManagementSystem) OnAdd(obj interface{}) {
	reqMgmt.configQueue.Add(0)
}

// OnUpdate handles notification from an informer of object update
func (reqMgmt *requestManagementSystem) OnUpdate(oldObj, newObj interface{}) {
	reqMgmt.OnAdd(newObj)
}

// OnDelete handles notification from an informer of object deletion
func (reqMgmt *requestManagementSystem) OnDelete(obj interface{}) {
	reqMgmt.OnAdd(obj)
}

func (reqMgmt *requestManagementSystem) Run(stopCh <-chan struct{}) error {
	defer reqMgmt.configQueue.ShutDown()
	klog.Info("Starting reqmgmt config controller")
	if ok := cache.WaitForCacheSync(stopCh, reqMgmt.plInformer.HasSynced, reqMgmt.fsInformer.HasSynced); !ok {
		return fmt.Errorf("Never achieved initial sync")
	}
	go wait.Until(reqMgmt.runWorker, time.Second, stopCh)
	klog.Info("Started reqmgmt config worker")
	<-stopCh
	klog.Info("Shutting down reqmgmt config worker")
	return nil
}

func (reqMgmt *requestManagementSystem) runWorker() {
	for reqMgmt.processNextWorkItem() {
	}
}

func (reqMgmt *requestManagementSystem) processNextWorkItem() bool {
	obj, shutdown := reqMgmt.configQueue.Get()
	if shutdown {
		return false
	}

	func(obj interface{}) {
		defer reqMgmt.configQueue.Done(obj)
		if !reqMgmt.syncOne() {
			reqMgmt.configQueue.AddRateLimited(obj)
		} else {
			reqMgmt.configQueue.Forget(obj)
		}
	}(obj)

	return true
}

// syncOne attempts to sync all the config for the reqmgmt filter.  It
// either succeeds and returns `true` or logs an error and returns
// `false`.
func (reqMgmt *requestManagementSystem) syncOne() bool {
	all := labels.Everything()
	newPLs, err := reqMgmt.plLister.List(all)
	if err != nil {
		klog.Errorf("Unable to list PriorityLevelConfiguration objects: %s", err.Error())
		return false
	}
	newFSs, err := reqMgmt.fsLister.List(all)
	if err != nil {
		klog.Errorf("Unable to list FlowSchema objects: %s", err.Error())
		return false
	}
	reqMgmt.digestConfigObjects(newPLs, newFSs)
	return true
}

func (reqMgmt *requestManagementSystem) digestConfigObjects(newPLs []*rmtypesv1a1.PriorityLevelConfiguration, newFSs []*rmtypesv1a1.FlowSchema) {
	oldRMState := reqMgmt.curState.Load().(*requestManagementState)
	var shareSum float64
	newRMState := &requestManagementState{
		priorityLevelStates: make(map[string]*priorityLevelState, len(newPLs)),
	}
	newlyQuiescent := make([]*priorityLevelState, 0)
	var nameOfExempt, nameOfDefault string
	for _, pl := range newPLs {
		state := oldRMState.priorityLevelStates[pl.Name]
		if state == nil {
			state = &priorityLevelState{
				config: pl.Spec,
			}
		} else {
			oState := *state
			state = &oState
			state.config = pl.Spec
			if state.emptyHandler != nil { // it was undesired, but no longer
				klog.V(3).Infof("Priority level %s was undesired and has become desired again", pl.Name)
				state.emptyHandler = nil
				state.queues.Quiesce(nil)
			}
		}
		if state.config.GlobalDefault {
			nameOfDefault = pl.Name
		}
		if state.config.Exempt {
			nameOfExempt = pl.Name
		} else {
			shareSum += float64(state.config.AssuredConcurrencyShares)
		}
		newRMState.priorityLevelStates[pl.Name] = state
	}
	fsSeq := make(FlowSchemaSequence, len(newFSs))
	var combinedAnalysis matchAnalysis
	for _, fs := range newFSs {
		if !warnFlowSchemaSpec(fs.Name, &fs.Spec, newRMState.priorityLevelStates, oldRMState.priorityLevelStates) {
			continue
		}
		fsSeq = append(fsSeq, fs)
		combinedAnalysis = combinedAnalysis.Or(analyzeFSMatches(fs))
	}
	sort.Sort(fsSeq)
	if nameOfExempt == "" {
		nameOfExempt = newRMState.generateExemptPL()
	}
	if nameOfDefault == "" {
		nameOfDefault = newRMState.generateDefaultPL()
	}
	if !combinedAnalysis.masters.Both() {
		fsSeq = append(fsSeq, newFSAllObj(nameOfExempt, true, "system:masters"))
	}
	if !(combinedAnalysis.authenticated.Both() && combinedAnalysis.unauthenticated.Both()) {
		fsSeq = append(fsSeq, newFSAllObj(nameOfDefault, false, "system:authenticated", "system:unauthenticated"))
	}
	newRMState.flowSchemas = fsSeq
	for plName, plState := range oldRMState.priorityLevelStates {
		if newRMState.priorityLevelStates[plName] != nil {
			// Still desired
		} else if plState.emptyHandler != nil && plState.emptyHandler.IsEmpty() {
			// undesired, empty, and never going to get another request
			klog.V(3).Infof("Priority level %s removed from implementation", plName)
		} else {
			oState := *plState
			plState = &oState
			if plState.emptyHandler == nil {
				klog.V(3).Infof("Priority level %s became undesired", plName)
				plState.emptyHandler = &emptyRelay{reqMgmt: reqMgmt}
				newlyQuiescent = append(newlyQuiescent, plState)
			}
			newRMState.priorityLevelStates[plName] = plState
			if !plState.config.Exempt {
				shareSum += float64(plState.config.AssuredConcurrencyShares)
			}
		}
	}
	for _, plState := range newRMState.priorityLevelStates {
		if plState.config.Exempt {
			continue
		}
		plState.concurrencyLimit = int(math.Ceil(float64(reqMgmt.serverConcurrencyLimit) * float64(plState.config.AssuredConcurrencyShares) / shareSum))
		if plState.queues == nil {
			plState.queues = reqMgmt.queueSetFactory.NewQueueSet(plState.concurrencyLimit, int(plState.config.Queues), int(plState.config.QueueLengthLimit), reqMgmt.requestWaitLimit)
		} else {
			plState.queues.SetConfiguration(plState.concurrencyLimit, int(plState.config.Queues), int(plState.config.QueueLengthLimit), reqMgmt.requestWaitLimit)
		}
	}
	reqMgmt.curState.Store(newRMState)
	// We do the following only after updating curState to guarantee
	// that if Wait returns `quiescent==true` then a fresh load from
	// curState will yield an requestManagementState that is at least
	// as up-to-date as the data here.
	for _, plState := range newlyQuiescent {
		plState.queues.Quiesce(plState.emptyHandler)
	}
}

func warnFlowSchemaSpec(fsName string, spec *rmtypesv1a1.FlowSchemaSpec, newPriorities, oldPriorities map[string]*priorityLevelState) bool {
	plName := spec.PriorityLevelConfiguration.Name
	if newPriorities[plName] == nil {
		problem := "non-existent"
		if oldPriorities[plName] != nil {
			problem = "undesired"
		}
		klog.Warningf("FlowSchema %s references %s priority level %s and will thus not match any requests", fsName, problem, plName)
		return false
	}
	return true
}

func (newRMState *requestManagementState) generateExemptPL() (nameOfExempt string) {
	if newRMState.priorityLevelStates["system-top"] == nil {
		nameOfExempt = "system-top"
	} else {
		for i := 2; true; i++ {
			nameOfExempt = fmt.Sprintf("system-top-%d", i)
			if newRMState.priorityLevelStates[nameOfExempt] == nil {
				break
			}
		}
	}
	klog.Warningf("No Exempt PriorityLevelConfiguration found, inventing one named %q", nameOfExempt)
	newRMState.priorityLevelStates[nameOfExempt] = &priorityLevelState{
		config: DefaultPriorityLevelConfigurationObjects()[0].Spec,
	}
	return
}

func (newRMState *requestManagementState) generateDefaultPL() (nameOfDefault string) {
	if newRMState.priorityLevelStates["workload-low"] == nil {
		nameOfDefault = "workload-low"
	} else {
		for i := 2; true; i++ {
			nameOfDefault = fmt.Sprintf("workload-low-%d", i)
			if newRMState.priorityLevelStates[nameOfDefault] == nil {
				break
			}

		}
	}
	klog.Warningf("No GlobalDefault PriorityLevelConfiguration found, inventing one named %q", nameOfDefault)
	newRMState.priorityLevelStates[nameOfDefault] = &priorityLevelState{
		config: DefaultPriorityLevelConfigurationObjects()[1].Spec,
	}
	return
}

// matchAnalysis summarizes a matching predicate, broken down by
// Subject and Verb&Object
type matchAnalysis struct {
	masters         objectAnalysis
	authenticated   objectAnalysis
	unauthenticated objectAnalysis
}

type objectAnalysis struct {
	resource    bool // there's a match for any resource, verb, and APIGroup
	nonResource bool // there's a match for any non-resource URL and verb
}

func (a matchAnalysis) Or(b matchAnalysis) matchAnalysis {
	return matchAnalysis{
		masters:         a.masters.Or(b.masters),
		authenticated:   a.authenticated.Or(b.authenticated),
		unauthenticated: a.unauthenticated.Or(b.unauthenticated),
	}
}

func (a objectAnalysis) Or(b objectAnalysis) objectAnalysis {
	return objectAnalysis{
		resource:    a.resource || b.resource,
		nonResource: a.nonResource || b.nonResource}
}

func (a objectAnalysis) Both() bool {
	return a.resource && a.nonResource
}

func analyzeFSMatches(fs *rmtypesv1a1.FlowSchema) matchAnalysis {
	var ans matchAnalysis
	for _, rws := range fs.Spec.Rules {
		ans = ans.Or(analyzeRWSMatches(rws))
	}
	return ans
}

func analyzeRWSMatches(rws rmtypesv1a1.PolicyRuleWithSubjects) matchAnalysis {
	var ans matchAnalysis
	var oa objectAnalysis
	if len(rws.Rule.Verbs) != 1 || rws.Rule.Verbs[0] != rmtypesv1a1.VerbAll {
		return ans
	}
	if len(rws.Rule.Resources) > 0 {
		if len(rws.Rule.APIGroups) != 1 || rws.Rule.APIGroups[0] != rmtypesv1a1.APIGroupAll || len(rws.Rule.Resources) != 1 || rws.Rule.Resources[0] != rmtypesv1a1.ResourceAll {
			return ans
		}
		oa.resource = true
	} else {
		if len(rws.Rule.NonResourceURLs) != 1 || rws.Rule.NonResourceURLs[0] != rmtypesv1a1.NonResourceAll {
			return ans
		}
		oa.nonResource = true
	}
	for _, subject := range rws.Subjects {
		if subject.Kind != "Group" {
			continue
		}
		switch subject.Name {
		case "system:masters":
			ans.masters = ans.masters.Or(oa)
		case "system:authenticated":
			ans.authenticated = ans.authenticated.Or(oa)
		case "system:unauthenticated":
			ans.unauthenticated = ans.unauthenticated.Or(oa)
		}
	}
	return ans
}

type emptyRelay struct {
	sync.RWMutex
	reqMgmt *requestManagementSystem
	empty   bool
}

var _ fq.EmptyHandler = &emptyRelay{}

func (er *emptyRelay) HandleEmpty() {
	er.Lock()
	defer func() { er.Unlock() }()
	er.empty = true
	er.reqMgmt.configQueue.Add(0)
}

func (er *emptyRelay) IsEmpty() bool {
	er.RLock()
	defer func() { er.RUnlock() }()
	return er.empty
}

var _ sort.Interface = FlowSchemaSequence(nil)

func (a FlowSchemaSequence) Len() int {
	return len(a)
}

func (a FlowSchemaSequence) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a FlowSchemaSequence) Less(i, j int) bool {
	return a[i].Spec.MatchingPrecedence < a[j].Spec.MatchingPrecedence
}
