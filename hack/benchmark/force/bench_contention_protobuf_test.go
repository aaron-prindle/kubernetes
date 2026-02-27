package main

import (
	"fmt"
	"math/rand"
	"testing"
	"unique"
)

// BenchmarkProtobufDecodes simulates a specific decoding concern:
// 10 concurrent requests, each decoding a massive Protobuf message containing
// 100s of FieldsV1 objects, resulting in 100s of unique.Make() calls per request
// in parallel.
func BenchmarkProtobufDecodes(b *testing.B) {
	// Simulate 100s of unique.Make() calls per decode operation
	numFieldsPerRequest := 500 
	
	// Pre-generate the "duplicate" data that would exist in a real WatchCache
	// or highly replicated StatefulSet scenario
	duplicatePayloads := make([]string, numFieldsPerRequest)
	for i := 0; i < numFieldsPerRequest; i++ {
		duplicatePayloads[i] = fmt.Sprintf(`{"f:metadata":{"f:labels":{"f:app":{},"f:tier":{}}},"f:spec":{"f:replicas":{}}}`)
	}

	b.SetParallelism(10) // Specifically target the "10 things decoding in parallel"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate the decoding loop inside a massive List/Watch or large object
			for i := 0; i < numFieldsPerRequest; i++ {
				// We do the cast and Make exactly as it happens in generated.pb.go
				m := struct{ Raw string }{}
				// The PoC uses unique.Handle but for legacy testing we mimic the call boundary
				m.Raw = unique.Make(duplicatePayloads[i]).Value()
				_ = m
			}
		}
	})
}

// BenchmarkProtobufDecodesNovel tests the same scenario, but assumes every single
// one of the 100s of fields is a completely unique string, forcing the slow-path lock every time.
func BenchmarkProtobufDecodesNovel(b *testing.B) {
	numFieldsPerRequest := 500 

	b.SetParallelism(10)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for i := 0; i < numFieldsPerRequest; i++ {
				// Generate a purely novel string inside the loop
				novelString := fmt.Sprintf(`{"f:metadata":{"f:labels":{"f:rand%d-%d":{}}}}`, rand.Int(), rand.Int())
				m := struct{ Raw string }{}
				m.Raw = unique.Make(novelString).Value()
				_ = m
			}
		}
	})
}