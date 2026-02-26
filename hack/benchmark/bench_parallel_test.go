package main

import (
	"testing"
	"unique"
)

// To run: go test -bench=. -benchmem -cpu=1,10,50,100,500,1000

var payload = []byte(`{"f:metadata":{"f:labels":{"f:app":{},"f:tier":{},"f:env":{}}},"f:spec":{"f:replicas":{},"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"app\"}":{".":{},"f:image":{},"f:name":{},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:memory":{}}}}}}}}}`)

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

// BenchmarkBaseline simulates the current []byte behavior (copying bytes).
func BenchmarkBaseline(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f := &FieldsV1Baseline{}
			_ = f.Unmarshal(payload)
		}
	})
}

// BenchmarkInterning simulates the proposed behavior using unique.Make.
func BenchmarkInterning(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f := &FieldsV1Proposed{}
			_ = f.Unmarshal(payload)
		}
	})
}
