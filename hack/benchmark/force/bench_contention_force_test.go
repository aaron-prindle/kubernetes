package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"unique"
)

// BenchmarkStandardMutex forces contention on a standard sync.Mutex to prove
// our profiling tools can indeed capture non-zero contention.
func BenchmarkStandardMutex(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]int)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := fmt.Sprintf("val-%d", rand.Int())
			mu.Lock()
			m[s] = 1
			// A tiny bit of work inside the lock to ensure other goroutines park
			for i := 0; i < 100; i++ {
				_ = i * i
			}
			mu.Unlock()
		}
	})
}

// BenchmarkUniqueMakeExtreme forces extreme contention on unique.Make by
// simultaneously inserting completely novel strings across thousands of goroutines.
func BenchmarkUniqueMakeExtreme(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := fmt.Sprintf("val-%d-%d", rand.Int(), rand.Int())
			_ = unique.Make(s)
		}
	})
}
