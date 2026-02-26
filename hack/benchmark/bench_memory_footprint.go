package main

import (
	"fmt"
	"runtime"
	"time"
	"unique"
)

var (
	// Representative managedFields payloads for different pod types
	daemonSetPayload   = []byte(`{"f:metadata":{"f:labels":{"f:app":{},"f:tier":{},"f:node-role.kubernetes.io/worker":{}}},"f:spec":{"f:affinity":{},"f:containers":{"k:{\"name\":\"ds-app\"}":{".":{},"f:image":{},"f:name":{},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:memory":{}}}}}}}`)
	jobPayload         = []byte(`{"f:metadata":{"f:labels":{"f:job-name":{},"f:controller-uid":{}}},"f:spec":{"f:restartPolicy":{},"f:containers":{"k:{\"name\":\"job-app\"}":{".":{},"f:image":{},"f:name":{},"f:command":{}},"k:{\"name\":\"sidecar\"}":{".":{},"f:image":{},"f:name":{}}}}}`)
	statefulSetPayload = []byte(`{"f:metadata":{"f:labels":{"f:app":{},"f:statefulset.kubernetes.io/pod-name":{}}},"f:spec":{"f:hostname":{},"f:subdomain":{},"f:containers":{"k:{\"name\":\"sts-app\"}":{".":{},"f:image":{},"f:name":{},"f:volumeMounts":{".":{},"k:{\"mountPath\":\"/data\"}":{".":{},"f:mountPath":{},"f:name":{}}}}}}}`)
)

type FieldsV1Baseline struct {
	Raw []byte
}

func (f *FieldsV1Baseline) Unmarshal(data []byte) error {
	dst := make([]byte, len(data))
	copy(dst, data)
	f.Raw = dst
	return nil
}

type FieldsV1Proposed struct {
	Raw string
}

func (f *FieldsV1Proposed) Unmarshal(data []byte) error {
	f.Raw = unique.Make(string(data)).Value()
	return nil
}

// measureMemory simulates WatchCache returning N items and measures retained heap
func measureMemory(payload []byte, count int, useInterning bool) float64 {
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var list []any
	for i := 0; i < count; i++ {
		if useInterning {
			f := &FieldsV1Proposed{}
			f.Unmarshal(payload)
			list = append(list, f)
		} else {
			f := &FieldsV1Baseline{}
			f.Unmarshal(payload)
			list = append(list, f)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	retainedMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024

	// keep list alive to prevent premature GC
	runtime.KeepAlive(list)

	return retainedMB
}

func main() {
	count := 50000 // Simulate 50k duplicated Pods returned in a WatchCache LIST

	fmt.Printf("| Controller | Pod Count | Baseline `[]byte` (MB) | Proposed `string` (MB) | Memory Reduction |\n")
	fmt.Printf("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"DaemonSet", daemonSetPayload},
		{"Job/JobSet", jobPayload},
		{"StatefulSet", statefulSetPayload},
	} {
		baselineMB := measureMemory(tc.data, count, false)
		internMB := measureMemory(tc.data, count, true)
		reduction := ((baselineMB - internMB) / baselineMB) * 100
		
		fmt.Printf("| %s | %d | %.2f MB | %.2f MB | **%.1f%%** |\n", tc.name, count, baselineMB, internMB, reduction)
	}
}
