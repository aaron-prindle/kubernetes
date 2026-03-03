# [PUBLIC] Proposal: Reducing managedFields High Memory Usage via String Interning
**Status**: Draft
**Author**: Aaron Prindle
**Reviewers**: [ ] Jordan Liggitt, [ ] Joe Betz, [ ] Marek Siarkowicz, [ ] deads@redhat.com
**Last revised**: 3/2/2026

## TLDR;
*   **The Problem:** `managedFields` string duplication causes memory bloat in the API server proportional to the number of fields in `managedFields` as part of a k8s objects (eg: large obj config -> more dupe memory) AS WELL AS the # of k8s objects (scales proportionally). In highly replicated workloads w/ large configs (DaemonSets, etc.), `managedFields` can account for ~50% of the serialized pod size. 
*   **The Solution:** Transition `metav1.FieldsV1` from a `[]byte` to an immutable string and use Go 1.23 `unique.Handle[string]` for interning. This removes the duplicated `managedFields` entries from api-server memory. 
*   **Rollout Plan:** 
    *   **v1.36**: Encapsulate `FieldsV1` with accessors and use build tags to safely opt-in to `string` (not the default). 
    *   **v1.37**: Flip the default to the string implementation
    *   **v1.38 (v1.39+?)**: remove the legacy `[]byte` implementation in v1.38
*   **Memory Savings:** Memory savings are directly proportional to the size (number of fields, labels, annotations, etc.) of the k8s object that is duplicated. The figure on the left below shows if we have a 20k code cluster, as we scale the # of config fields per Pod, the memory reduction interning yields is greater. In the benchmarking done we see **15-20% reduction** in total API Server memory usage for representative “complex” Pod configuration cases (600-1200 additional fields). The graph on the right shows that as we increase the # of pods, the `managedFields` reduction stays constant proportionally to the # of fields (eg: X% savings maintained at 10k pods, 100k pods, etc. where X% is proportional to pod complexity as prev. noted)

<p align="center">
  <img src="./complexity_scaling_plot.png" width="45%" />
  <img src="./memory_scaling_plot.png" width="45%" />
</p>

*   **unique.Make Contention:** Profiling shows the standard library interning (`unique.Make`) lock implementation causes 0 mutex contention in the benchmark tests which attempted a burst of CREATEs with unique values (where the lock is held per write). the write-path lock operates too fast in cluster testing (nanosecond range) to where it is not the bottleneck in my testing behind the API server's networking and security layers. 

## Problem Statement
From large-scale cluster profiling, `managedFields` is a dominant factor in `kube-apiserver` memory exhaustion. In environments with highly replicated resources (ex: DaemonSets, ReplicaSets, StatefulSets, and Jobs) thousands of k8s objects (mainly Pods) are created from identical templates.

When these Pods are processed by the API server and held in the WatchCache, their `managedFields` payloads are duplicated as distinct `[]byte` slices in memory. At scales of 50,000+ pods (or large Pod configs), this results in large amounts of redundant data trapped in the heap, causing O(N) memory bloat for identical metadata.

## Proposed Solution - Convert FieldsV1 to string + string Interning 
We propose transitioning `metav1.FieldsV1` from a mutable `[]byte` to an immutable string. We would then intern this string (specifically, Go 1.23's `unique.Handle[string]`) to deduplicate `managedField` values across the kube-api-server. By enforcing immutability at the type level (via string), we unlock the ability to safely cache and natively intern payloads when they are deserialized.

The solution relies on three key parts (based on initial PoC by @liggitt (fieldsv1-string)):

### 1. Accessor Encapsulation (internal representation is still []byte)
To do the conversion without breaking downstream `client-go` consumers, we can encapsulate how the codebase interacts with `FieldsV1` via standard accessor methods and eliminate all direct, in-tree use of the `.Raw` field.

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

*(See Jordan's initial accessors commit: Add GetRaw/SetRaw methods to FieldsV1)*

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
*(See the full decoding implementation on the experimental branch: ssa-fieldsv1-string-interning-poc)*

If a DaemonSet spawns 50,000 pods with identical `managedFields`, the first payload allocates the string. The subsequent 49,999 identical payloads hit the `unique.Make` fast-path, discarding the incoming bytes and pointing their `FieldsV1.handle` directly at the original string in memory.

## Performance Validation

### Proving the Bottleneck Testing (Baseline Scaling @ k/k master)
**Objective:** Prove that `managedFields` is a true scaling bottleneck for general Kubernetes users by empirically mapping its memory footprint scaling against replicated workloads on the standard master branch.

**Script:** `run-kind-baseline-scaling-benchmark.sh`

**Steps:**
1.  Build a custom kind node image directly from the Kubernetes source tree (master branch).
2.  Provision a local cluster and install Kwok to simulate fake nodes.
3.  Deploy a StatefulSet and incrementally scale it to 1,000, 10,000, and 50,000 duplicated Pods.
4.  Pause at each milestone to allow the WatchCache to stabilize.

**Data Collection:** At each scale milestone, we captured the live heap profile from the API Server's `/debug/pprof/heap` endpoint. We then isolated the `inuse_space` metric specifically tied to allocations originating from `k8s.io/apimachinery/pkg/apis/meta/v1.(*FieldsV1).Unmarshal`.

The baseline memory usage scales with the number of replicas in this example (is the case for all objects as `managedFields` is across all k8s API objects). At scales of hundreds of thousands of identical objects across a large cluster, `metav1.FieldsV1.Unmarshal` operations consume gigabytes of raw API server RAM just holding duplicate bytes.

### Memory Footprint Reduction Testing - Comparing master vs. experimental
**Objective:** Prove that string interning collapses this live API server memory footprint from O(N) to O(1).

**Scripts:** `run-kind-benchmark.sh`

**Steps:**
1.  Build a custom kind node image using the experimental string `FieldsV1` + `unique.Handle` branch.
2.  Provision a local cluster and install Kwok to simulate fake nodes.
3.  Deploy a StatefulSet configured to create X duplicated Pods to recreate large `managedFields` duplication.
4.  Wait for the WatchCache to completely stabilize with the X pods.

**Data Collection:** We captured the live heap profiles from the API Server's `/debug/pprof/heap` endpoint for both the baseline and experimental clusters. The "Total Apiserver Heap" metric represents the complete memory footprint of the kube-apiserver process.

**Results:**

| Branch | Total Heap (+0 Fields) | Total Heap (+300 Fields) | Total Heap (+1200 Fields) |
| :--- | :--- | :--- | :--- |
| master (Baseline `[]byte`) | 733 MB | 3,036 MB | 7,196 MB |
| experimental (`stringhandle`) | 720 MB | 2,475 MB | 5,823 MB |

| Branch | Total Heap (100k Pods) | Total Heap (200k Pods) | WatchCache Footprint |
| :--- | :--- | :--- | :--- |
| master (Baseline `[]byte`) | 1,780 MB | 3,052 MB | O(N) |
| experimental (`stringhandle`) | 1,613 MB | 2,908 MB | O(1) |


### unique.Make Parallel Contention Testing
**Objective:** Address concern that `unique.Make()` does not become a global lock bottleneck during parallel API Server operations. For this there are two distinct contention benchmarks against the tuned cluster to test both the write and read paths independently.

#### Write-Path Contention unique.Make Testing
**Objective:** Address the concern that the global `unique.Make` lock could create a write-path bottleneck by directly targeting the deserialization boundary. We aim to prove that even when forcing 100% lock acquisition under massive parallel load, the API server's end-to-end latency is unaffected.

**Script:** `run-brutal-write-contention.sh`

**Steps:**
1.  Launch a high-performance **Go-native load generator** with a persistent HTTP/2 connection pool.
2.  Spawn **100 concurrent workers** blasting Server-Side Apply (SSA) PATCH requests against a tuned cluster for 30 seconds.
3.  Inject a **randomized field value and manager name** into every single request. This guarantees a novel `managedFields` JSON blob, completely bypassing the interning fast-path cache and forcing the API server to acquire the global `unique.Make()` lock for every single call.

**Data Collection:** We authored a custom Go load generator (`brutal_write_client.go`) using `net/http` to send the payload. For every request, it recorded a `time.Now()` timestamp immediately before calling `client.Do(req)`, and computed `time.Since(start)` the moment the response returned HTTP 200 OK. These durations were collected into a thread-safe slice, and at the end of the 30-second window, they were sorted to calculate the Average, Median (P50), and P99 round-trip latencies. We simultaneously captured the `/debug/pprof/mutex` profile to definitively check for internal lock queues.

**Results:**

| Metric (100 parallel workers, 30s) | master (Baseline `[]byte`) | experimental (`stringhandle`) |
| :--- | :--- | :--- |
| **Interning Path Taken** | N/A | 100% Novel Strings (Forced Lock) |
| **Average Request Latency** | 10.57 seconds | 10.50 seconds |
| **P50 Latency (Median)** | 6.31 seconds | 6.30 seconds |
| **P99 Request Latency** | 33.57 seconds | 33.34 seconds |
| **Write-Path Mutex Contention** | 0 significant delays | 0 significant delays |

*(Note: Latencies are high globally because 100 concurrent workers are intentionally attempting to DDoS a single local API Server).*

**Conclusion:** 
Even under this synthetic worst-case stress, the end-to-end request latencies for the interning branch were identical (and marginally faster) than the baseline branch. The `-mutexprofile` returned 0 wait samples. 

While `unique.Make()` does take a lock for novel strings, the lock's critical section itself executes in **~1-5 nanoseconds**. Because the API Server's networking, authentication, and validation layers take milliseconds to process each request, they act as a natural rate-limiter. Requests arrive at the deserialization layer staggered, meaning they almost never collide to fight over the nanosecond-scale lock. The theoretical "millisecond-scale" lock contention only appears in artificial microbenchmarks (like the Protobuf test below) where thousands of threads are hard-coded to bypass networking and hit the lock simultaneously.

#### Protobuf Decode Contention Testing
**Objective:** Address concern regarding Protobuf deserialization contention (e.g., thousands of parallel decoders making 100s of `unique.Make` calls each inside a large Protobuf message).

**Script:** `bench_contention_protobuf_test.go`

**Steps:**
1.  Simulate 6,400 parallel goroutines (64 cores * 100 parallelism multiplier) each executing 500 `unique.Make` calls.
2.  Run the benchmark with duplicated strings to test the caching fast-path.
3.  Run the benchmark by injecting 500 novel, random strings into all 6,400 parallel decoders simultaneously to force maximum lock contention.

**Data Collection:** Captured total completion time for the parallel execution via standard `testing.B` results on a 64-core machine.

**Results:**

| Load Scenario | Completion Time (500 fields) | Per-Field Latency |
| :--- | :--- | :--- |
| Duplicated Data (Fast-Path) | ~710 nanoseconds | ~1.4 nanoseconds |
| Novel Data (Global Lock) | ~1.02 milliseconds | ~2.0 microseconds |

Even when directly testing lock contention with 6,400 concurrent workers, the operation still completed in ~1 millisecond. In a real Kubernetes API Server, requests are staggered by other api-server layers (auth, etc.), and the number of parallel decoders is capped by max-requests-inflight (usually < 2000). This means in practice the `unique.Make` lock contention would not be a bottleneck.

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