# Parallel Contention Benchmarking Results: `unique.Make` vs `[]byte` Allocation

**Date:** <Current Date>
**Objective:** Evaluate the lock contention and performance overhead of `unique.Make` across high-concurrency decoding scenarios in the API server.

## 1. Methodology
We ran parallel Go benchmarks on an AMD EPYC machine using `b.RunParallel` with `-cpu=1,10,50,100,500,1000`.

- **Baseline (`[]byte`):** Simulates the current behavior where `metav1.FieldsV1.Unmarshal` performs a `make([]byte)` and `copy`.
- **Proposed (`unique.Make`):** Simulates interning via Go 1.23 `unique.Make(string(data))`.

## 2. Benchmark Results

| Benchmark | Goroutines | ns/op | B/op | allocs/op |
| :--- | :--- | :--- | :--- | :--- |
| `BenchmarkBaseline` | 1 | 76.00 ns/op | 288 B/op | 1 allocs/op |
| `BenchmarkBaseline` | 10 | 66.95 ns/op | 288 B/op | 1 allocs/op |
| `BenchmarkBaseline` | 50 | 74.37 ns/op | 288 B/op | 1 allocs/op |
| `BenchmarkBaseline` | 100 | 73.60 ns/op | 288 B/op | 1 allocs/op |
| `BenchmarkBaseline` | 500 | 89.92 ns/op | 288 B/op | 1 allocs/op |
| `BenchmarkBaseline` | 1000 | 103.9 ns/op | 288 B/op | 1 allocs/op |
| **`BenchmarkInterning`** | **1** | **49.29 ns/op** | **0 B/op** | **0 allocs/op** |
| **`BenchmarkInterning`** | **10** | **5.069 ns/op** | **0 B/op** | **0 allocs/op** |
| **`BenchmarkInterning`** | **50** | **1.328 ns/op** | **0 B/op** | **0 allocs/op** |
| **`BenchmarkInterning`** | **100** | **1.413 ns/op** | **0 B/op** | **0 allocs/op** |
| **`BenchmarkInterning`** | **500** | **1.661 ns/op** | **0 B/op** | **0 allocs/op** |
| **`BenchmarkInterning`** | **1000** | **1.339 ns/op** | **0 B/op** | **0 allocs/op** |

## 3. Analysis

**Throughput & Memory:**
The benchmark demonstrates that `unique.Make` behaves incredibly well under load. Not only does it drop memory allocations to `0 B/op`, but its execution time collapses from ~49 ns (single goroutine) down to ~1.3 ns at high concurrency (due to parallel throughput scaling smoothly over CPUs). In contrast, the baseline byte copying gets progressively slower per operation (from 76ns up to 103ns) at high concurrency because it triggers substantial memory allocator and Garbage Collector overhead.

**Contention Profile (`-mutexprofile`):**
Running `go tool pprof -top mutex.out` confirmed that the vast majority of lock contention (60.82% inside `runtime.unlock` and 39.13% in `runtime._LostContendedRuntimeLock`) originated entirely from the `runtime.mallocgc` functions used by the `Baseline` allocations. There is zero significant contention reported from the `unique` package map locks themselves. The background GC and memory allocator spinlocks are vastly more expensive and contended than `unique.Make`.

**Conclusion:**
There are no hidden problems with high-parallelization contention for `unique.Make` in this workflow. It is safely capable of executing concurrently without degrading API server throughput; in fact, bypassing `mallocgc` dramatically *improves* parallel API server throughput.