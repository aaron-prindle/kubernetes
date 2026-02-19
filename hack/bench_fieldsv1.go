package main

import (
	"fmt"
	"runtime"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var globalFields []*metav1.FieldsV1

func main() {
	orig := &metav1.FieldsV1{Raw: []byte(`{"f:metadata":{"f:labels":{"f:app":{},"f:tier":{},"f:env":{}}},"f:spec":{"f:replicas":{},"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"app\"}":{".":{},"f:image":{},"f:name":{},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:memory":{}}}}}}}}}`)}
	data, _ := orig.Marshal()

	count := 200000

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	globalFields = make([]*metav1.FieldsV1, 0, count)
	for i := 0; i < count; i++ {
		b := append([]byte(nil), data...)
		f := &metav1.FieldsV1{}
		if err := f.Unmarshal(b); err != nil {
			panic(err)
		}
		globalFields = append(globalFields, f)
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	retainedMB := float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
	fmt.Printf("%.2f\n", retainedMB)
}
