# Proposal: Reducing managedFields High Memory Usage via String Interning

**Status**: Draft
**Author**: Aaron Prindle
**Reviewers**: [ ] Jordan Liggitt, [ ] Joe Betz, [ ] Marek Siarkowicz
**Last revised**: 2/27/2026

## TLDR;
*   **The Problem:** `managedFields` string duplication causes O(N) memory bloat in the API server where N is the number of fields in `managedFields` as part of a k8s objects (notably Pods). In highly replicated workloads (DaemonSets, etc.), `managedFields` can account for ~50% of the serialized pod size. 
*   **The Solution:** Transition `metav1.FieldsV1` from a `[]byte` to an immutable string and use Go 1.23 `unique.Handle[string]` for interning. This removes the duplicated `managedFields` entries from api-server memory. 
*   **Rollout Plan:** 
    *   **v1.36**: Encapsulate `FieldsV1` with accessors and use build tags to safely opt-in to `string` (not the default). 
    *   **v1.37**: Flip the default to the string implementation.
    *   **v1.38 (v1.39+?)**: Remove the legacy `[]byte` implementation in v1.38.
*   **Memory Savings:** Memory savings are directly proportional to the size (number of fields, labels, annotations, etc.) of the k8s object that is duplicated. The figure on the left below shows if we have a 20k code cluster, as we scale the # of config fields per Pod, the memory reduction interning yields is greater. In the benchmarking done we see **15-20% reduction** in total API Server memory usage for the “complex” Pod configuration cases (600-1200 additional fields) representative of actual complex Pods from the clusters we analyzed. The graph on the right shows that as we increase the # of pods, the `managedFields` difference stays constant proportionally to the # of fields (eg: X% savings maintain at 10k pods and 100k pods where X% is proportional to pod complexity).

<p align="center">
  <img src="./complexity_scaling_plot.png" width="45%" />
  <img src="./memory_scaling_plot.png" width="45%" />
</p>

*   **unique.Make Contention:** Profiling shows the standard library interning (`unique.Make`) lock implementation causes 0 contention in the benchmark tests which attempted a burst of CREATEs with unique values (where the lock is held per write). The read-path bypasses interning entirely (notably reducing CPU load by ~50%), and the write-path lock operates so fast in testing (nanosecond range), it is not the bottleneck behind the ~millisecond-scale latency of the API server's networking and security layers.

## Problem Statement
From large-scale cluster profiling, `managedFields` has emerged as a dominant factor in `kube-apiserver` memory exhaustion. In environments with highly replicated resources (ex: DaemonSets, ReplicaSets, StatefulSets, and Jobs) thousands of k8s objects (mainly Pods) are created from identical templates.

When these Pods are processed by the API server and held in the WatchCache (eg: for the 5-minute history window), their `managedFields` payloads are duplicated as distinct `[]byte` slices in memory. At scales of 50,000+ pods, this results in large amounts of redundant data trapped in the heap, causing O(N) memory bloat for identical metadata.

## Proposed Solution - Convert FieldsV1 to string + string Interning 
We propose transitioning `metav1.FieldsV1` from a mutable `[]byte` to an immutable string. We would then intern this string (specifically, Go 1.23's `unique.Handle[string]`) to deduplicate `managedField` values across the kube-api-server. By enforcing immutability at the type level (via string), we unlock the ability to safely cache and natively intern payloads when they are deserialized.

The solution relies on three key parts (based on initial PoC by @liggitt (fieldsv1-string)):

### 1. Accessor Encapsulation (internal representation is still []byte)
To do the conversion without breaking downstream `client-go` consumers, we can abstract how the codebase interacts with `FieldsV1`. We will introduce standard accessor methods and eliminate all direct, in-tree use of the `.Raw` field.

To avoid introducing a conversion penalty (when the internal representation is still a `[]byte` but we are preparing for the string transition), we will introduce type-specific accessors. This allows callers to fetch or set the data in exactly the format they need.

```go
// staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go
// Before:
type FieldsV1 struct {
    Raw []byte `json:"-" protobuf:"bytes,1,opt,name=Raw"`
}

// After: Direct access is deprecated.
// Phase 1 provides format-specific accessors to avoid []byte <-> string conversion penalties.
func (f *FieldsV1) GetRawBytes() []byte { ... }
func (f *FieldsV1) GetRawString() string { ... }
func (f *FieldsV1) SetRawBytes(b []byte) { ... }
func (f *FieldsV1) SetRawString(s string) { ... }
```
*(See Jordan's initial accessors commit: [Add GetRaw/SetRaw methods to FieldsV1](https://github.com/liggitt/kubernetes/commit/5a1b32d20b6016e7f8e874cc6d628d009b0b467e))*

### 2. Build-Tagged Implementations For []byte and string
Instead of directly changing `FieldsV1` type from `[]byte` to `string`, we want to make this opt-in for v1.36 and eventually make this the default over time. The solution here is to extract the `FieldsV1` declaration and its unmarshal/deepcopy methods into isolated, manually maintained files governed by `//go:build` tags to allow toggling.

This approach provides a safe swap mechanism at compile-time:
*   `fieldsv1_byte.go`: The legacy `[]byte` implementation. This remains the default for standard OSS builds to prevent immediate downstream breakages.
*   `fieldsv1_stringhandle.go`: The optimized implementation utilizing Go 1.23's `unique.Handle[string]`.

### 3. Native Interning at the Decoding Boundary
When compiled with the `stringhandle` tag, the API server intercepts payloads during JSON, CBOR, or Protobuf deserialization and passes them directly through the standard library interning pool (`unique`).

```go
// Inside fieldsv1_stringhandle.go Unmarshal logic
func (m *FieldsV1) Unmarshal(dAtA []byte) error {
    // ... protobuf boundary interception ...
    m.handle = unique.Make(string(dAtA[iNdEx:postIndex]))
    return nil
}
```
*(See the full decoding implementation on the experimental branch: [ssa-fieldsv1-string-interning-poc](https://github.com/aaron-prindle/kubernetes/tree/ssa-fieldsv1-string-interning-poc/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1))*

If a DaemonSet spawns 50,000 pods with identical `managedFields`, the first payload allocates the string. The subsequent 49,999 identical payloads hit the `unique.Make` fast-path, discarding the incoming bytes and pointing their `FieldsV1.handle` directly at the original string in memory.

## Performance Validation

### Proving the Bottleneck (Baseline Scaling @ k/k master)
**Objective:** Prove that `managedFields` is a true scaling bottleneck for general Kubernetes users by empirically mapping its memory footprint scaling against replicated workloads on the standard master branch.

**Script:** [`run-kind-baseline-scaling-benchmark.sh`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/run-kind-baseline-scaling-benchmark.sh)

**Steps:**
1.  Build a custom kind node image directly from the Kubernetes source tree (master branch).
2.  Provision a local cluster and install Kwok to simulate fake nodes.
3.  Deploy a StatefulSet and incrementally scale it to 1,000, 10,000, and 50,000 duplicated Pods.
4.  Pause at each milestone to allow the WatchCache to stabilize.

**Data Collection:** At each scale milestone, we captured the live heap profile from the API Server's `/debug/pprof/heap` endpoint. We then isolated the `inuse_space` metric specifically tied to allocations originating from `k8s.io/apimachinery/pkg/apis/meta/v1.(*FieldsV1).Unmarshal`.

The baseline memory usage scales with the number of replicas in this example (is the case for all objects as `managedFields` is across all k8s API objects). At scales of hundreds of thousands of identical objects across a large cluster, `metav1.FieldsV1.Unmarshal` operations consume gigabytes of raw API server RAM just holding duplicate bytes.

### Memory Footprint Reduction - Comparing master vs. experimental PoC
**Objective:** Prove that string interning collapses this live API server memory footprint from O(N) to O(1).

**Script:** [`run-kind-benchmark.sh`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/run-kind-benchmark.sh)

**Steps:**
1.  Build a custom kind node image using the experimental string `FieldsV1` + `unique.Handle` branch.
2.  Provision a local cluster and install Kwok to simulate fake nodes.
3.  Deploy a StatefulSet configured to create 50,000 duplicated Pods to recreate large `managedFields` duplication.
4.  Wait for the WatchCache to completely stabilize with the 50,000 pods.

**Data Collection:** We captured the live heap profiles from the API Server's `/debug/pprof/heap` endpoint for both the baseline and experimental clusters. The "Total Apiserver Heap" metric represents the complete memory footprint of the kube-apiserver process, while the "FieldsV1 Allocation Profile" specifically isolates the `inuse_space` tracked to `metav1.FieldsV1.Unmarshal` operations.

#### Results:
| Branch | Total Heap (+0 Fields) | Total Heap (+300 Fields) | Total Heap (+1200 Fields) |
| :--- | :--- | :--- | :--- |
| master (Baseline `[]byte`) | 733 MB | 3,036 MB | 7,196 MB |
| experimental (`stringhandle`) | 720 MB | 2,475 MB | 5,823 MB |

| Branch | Total Heap (100k Pods) | Total Heap (200k Pods) | WatchCache Footprint |
| :--- | :--- | :--- | :--- |
| master (Baseline `[]byte`) | 1,780 MB | 3,052 MB | O(N) |
| experimental (`stringhandle`) | 1,613 MB | 2,908 MB | O(1) |

### unique.Make Parallel Contention Safety
**Objective:** Address concern that `unique.Make()` does not become a global lock bottleneck during highly parallel API Server operations. For this there are two distinct contention benchmarks against the tuned cluster to test both the read and write paths independently.

#### Write-Path Safety (Guaranteed Lock Contention Test)
**Objective:** Proactively test the "breaking point" of the `unique` package by forcing the API Server to process 100% novel strings in parallel, bypassing the fast-path and forcing the global interning lock for every request.

**Script:** [`run-brutal-write-contention.sh`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/run-brutal-write-contention.sh)

**Steps:**
1.  Launch a high-performance **Go-native load generator** with a persistent HTTP/2 connection pool.
2.  Spawn **100 concurrent workers** blasting SSA PATCH requests against a tuned cluster.
3.  Inject a **randomized field value and manager name** into every single request to guarantee a novel `managedFields` JSON blob, forcing `unique.Make()` into its "slow-path" (global lock) for every single call.

Data Collection: Captured block and mutex delays from the `/debug/pprof/mutex` endpoint during the sustained 30-second bombardment. We also tracked end-to-end request latency for all PATCH operations.

#### Results:
| Metric (30s window) | master (Baseline `[]byte`) | experimental (`stringhandle`) |
| :--- | :--- | :--- |
| **Workers** | 100 Parallel Goroutines | 100 Parallel Goroutines |
| **Interning Path** | N/A | 100% Novel Strings (Forced Lock) |
| **Mutex Contention** | 0 significant delays | 0 significant delays |
| **Avg Request Latency** | **10.57s** | **10.50s** |
| **P99 Request Latency** | **33.57s** | **33.34s** |

Even under this synthetic worst-case stress, the `-mutexprofile` returned 0 samples. Furthermore, the end-to-end request latencies for the interning branch were nearly identical (and even slightly faster) than the baseline branch. While `unique.Make()` does take a lock for novel strings, the critical section is so optimized (nanosecond scale) that even 100 parallel workers cannot stack up fast enough to create measurable contention behind the millisecond-scale overhead of the API Server's networking, authentication, and garbage collection layers. 

`unique.Make()` is demonstrably not a bottleneck for the Kubernetes write-path.

#### Protobuf Decode Contention
Objective: Address specific concern regarding Protobuf deserialization contention (e.g., thousands of parallel decoders making 100s of unique.Make calls each inside a large Protobuf message).

Script: bench_contention_protobuf_test.go

Steps:
* Simulate parallel goroutines at both high-end production scale (1,024) and extreme stress scale (6,400) each executing 500 contiguous unique.Make calls.
* Run the benchmark with duplicated strings to test the caching fast-path.
* Run the benchmark by injecting 500 novel, random strings into all parallel decoders simultaneously to force maximum lock contention.

Data Collection: Captured total completion time for the parallel execution via standard testing.B results on a 64-core machine.

Results:

| Load Scenario | Parallel Workers | Completion Time (500 fields) | Per-Field Latency |
| :--- | :--- | :--- | :--- |
| Duplicated Data (Fast-Path) | 6,400 | ~727 nanoseconds | ~1.4 nanoseconds |
| Novel Data (Global Lock - Prod Scale) | 1,024 | ~1.14 milliseconds | ~2.2 microseconds |
| Novel Data (Global Lock - Stress Scale) | 6,400 | ~1.00 milliseconds | ~2.0 microseconds |

Even when forcing an extreme, artificial "lock-fight" with thousands of concurrent workers, the operation still completed in ~1.1 milliseconds. In a real Kubernetes API Server, requests are staggered by networking and authentication, and the number of parallel decoders is capped by max-requests-inflight (usually < 2000). These results prove that unique.Make contention is not a viable bottleneck.

## 4. Rollout Strategy
Transitioning a core API metadata field requires managing the blast radius for OSS and client-go developers who might directly use the `[]byte` from `FieldsV1` (which we will convert to string which could break such users if not rolled out in phases). We propose a multi-release transition plan:

*   **Phase 1: Encapsulation (v1.36)**
    *   Extract `FieldsV1` from code generators and introduce string accessor methods (`GetRaw()`, `SetRaw()`).
    *   Deprecate `FieldsV1.Raw` and sweep the in-tree codebase to exclusively use the accessors.
    *   Introduce the opt-in build tag (`fieldsv1_stringhandle`) while keeping the legacy `[]byte` implementation as the default. This safely lays the groundwork and allows early adopters to compile custom memory-saving binaries.
*   **Phase 2: Default Flip (v1.37)**
    *   Flip the default behavior so `FieldsV1` is internally backed by `unique.Handle[string]`.
    *   Provide a reverse opt-out build tag (`fieldsv1_byte`) for clients who haven't migrated.
*   **Phase 3: Cleanup (v1.38? v1.39+?)**
    *   Remove the legacy exported `[]byte` version and the opt-out build tags entirely.
