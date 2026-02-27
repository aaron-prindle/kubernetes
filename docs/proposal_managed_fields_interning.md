# Proposal: Mitigating managedFields Memory Bloat via String Interning

**Status**: Draft
**Authors**: Aaron Prindle
**Last revised**: <Current Date>

## 1. Problem Statement
Based on large-scale cluster profiling, `managedFields` has emerged as a dominant factor in `kube-apiserver` memory exhaustion at scale. In environments with highly replicated resources (ex: `DaemonSet`s, `ReplicaSet`s, `StatefulSet`s, and `Job`s) thousands of Pods are created from identical templates.

When these Pods are processed by the API server and held in the `WatchCache` (e.g., for the 5-minute history window), their `managedFields` payloads are duplicated as distinct `[]byte` slices in memory. At scales of 50,000+ pods, this results in large amounts of redundant data trapped in the heap, causing O(N) memory bloat for identical metadata.

## 2. Proposed Solution
We propose transitioning `metav1.FieldsV1` from a mutable `[]byte` to an immutable `string` (specifically, Go 1.23's `unique.Handle[string]`). By enforcing immutability at the type level, we unlock the ability to safely cache and natively intern payloads at the exact moment they are deserialized.

This ensures that duplicate `managedFields` data across thousands of pods resolves to a single shared pointer in memory, collapsing the footprint from O(N) to O(1).

Based on the architectural proof-of-concept by [@liggitt (fieldsv1-string)](https://github.com/liggitt/kubernetes/commits/fieldsv1-string/), the solution relies on two key mechanisms:

### Accessor Encapsulation
To safely orchestrate this transition without breaking downstream `client-go` consumers, we must abstract how the codebase interacts with `FieldsV1`. We will introduce standard accessor methods (`GetRaw()`, `SetRaw()`) and eliminate all direct, in-tree use of the `.Raw` field.

### Build-Tagged Implementations
Because `FieldsV1` relies heavily on auto-generated Protobuf and DeepCopy code, changing its underlying type dynamically breaks the code generators. The solution extracts the `FieldsV1` declaration and its unmarshal/deepcopy methods into isolated, manually maintained files governed by `//go:build` tags:

*   [`fieldsv1_byte.go`](https://github.com/liggitt/kubernetes/blob/fieldsv1-string/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/fieldsv1_byte.go): The legacy `[]byte` implementation, remaining the default for standard OSS builds.
*   [`fieldsv1_stringhandle.go`](https://github.com/liggitt/kubernetes/blob/fieldsv1-string/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/fieldsv1_stringhandle.go): The optimized implementation utilizing `unique.Make()`.

When compiled with the `stringhandle` tag, the API server intercepts payloads during JSON, CBOR, or Protobuf deserialization. The first payload allocates the string, while subsequent identical payloads hit the `unique.Make` fast-path, pointing their handle directly at the original string in memory. Furthermore, because the target implementation is an immutable string, expensive defensive deep copies currently required by the `WatchCache` can be bypassed entirely.

## 3. Performance Validation
To build consensus and address concerns regarding `unique.Make` global lock contention, we designed rigorous, end-to-end live cluster benchmarks simulating extreme scaling conditions.

### 3.1 Proving the Bottleneck (Baseline Scaling)
To prove that `managedFields` is a true scaling bottleneck for general Kubernetes users, we tested the standard `master` branch using `Kwok` to simulate the growth of duplicated workloads (e.g., a massive `DaemonSet`). We captured the `inuse_space` metric from the API Server's `/debug/pprof/heap` endpoint at each scale milestone and isolated the allocations originating specifically from `k8s.io/apimachinery/pkg/apis/meta/v1.(*FieldsV1).Unmarshal`.

| Number of Pods | Baseline Heap Allocation for `FieldsV1` |
| :--- | :--- |
| **1,000** | ~16 MB |
| **10,000** | ~41.5 MB |
| **50,000** | ~134.6 MB |

![Baseline Scaling Plot](./baseline_scaling_plot.png)

The baseline memory usage scales with the number of replicas in this example (is the case for all objects as `managedFields` is across all k8s API objects). At scales of hundreds of thousands of identical objects across a massive cluster, `metav1.FieldsV1.Unmarshal` operations consume gigabytes of raw API server RAM just holding duplicate bytes.

### 3.2 Memory Footprint Reduction
**Objective:** Prove that string interning collapses this live API server memory footprint from O(N) to O(1).

**Methodology:** We ran the exact same 50,000 Pod `Kwok` simulation against a `kind` node compiled from our experimental `unique.Handle` branch. We extracted the live `pprof` heap profiles from `/debug/pprof/heap` for both clusters. The "Total Apiserver Heap" metric represents the complete memory footprint of the `kube-apiserver` process, while the "FieldsV1 Allocation Profile" specifically isolates the `inuse_space` tracked to `metav1.FieldsV1.Unmarshal` operations.

| Branch | Total Apiserver Heap | `FieldsV1` Allocation Profile (`inuse_space`) | WatchCache Footprint Scaling |
| :--- | :--- | :--- | :--- |
| **master** (Baseline `[]byte`) | ~1.45 GB | 130.59 MB | `O(N)` |
| **experimental** (`stringhandle`) | ~1.37 GB | 27.52 MB | `O(1)` |

![Memory Scaling Plot](./memory_scaling_plot.png)

With string interning enabled, `FieldsV1` allocations were reduced by ~80% down to 27.52 MB (representing only the mandatory baseline allocations for the struct pointers themselves).

### 3.3 Parallel Contention Safety
**Objective:** Address concern that the standard lib `unique` package relies on internal maps and locks. We must prove that `unique.Make()` does not become a global lock bottleneck during highly parallel API Server operations. We authored two distinct contention benchmarks against the tuned cluster to test both the read and write paths independently.

#### 3.3.1 Read-Path Isolation (Massive LISTs)
To test if serialization overhead from parallel reads causes contention, we designed the [`run-kind-contention-benchmark.sh`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/run-kind-contention-benchmark.sh) test. This script seeds the API Server with 10,000 duplicated Kwok Pods and bombards it with 50 highly concurrent `LIST` clients for 30 seconds.

During this 30-second sustained load window, we captured the active CPU profiles from the API Server's `/debug/pprof/profile?seconds=30` endpoint. The "Read-Path Total CPU Load" metric was calculated by extracting the cumulative CPU sample time reported by `go tool pprof` across all goroutines executing during the trace.

| Metric (30s window) | `master` (Baseline `[]byte`) | `experimental` (`stringhandle`) |
| :--- | :--- | :--- |
| **Read-Path Total CPU Load** | ~1336.79s | ~678.41s |

![CPU Scaling Plot](./contention_scaling_plot.png)

The CPU profiles revealed that `unique.Make` is not in the critical path for parallel `LIST` requests. Decoding (and thus interning) occurs only when objects are written to etcd or initially loaded into the `WatchCache`. By eliminating duplicate heap allocations, the experimental branch sliced total read-path CPU time in half (from ~1336s down to ~678s) by removing the need for background garbage collection (`mallocgc`) to thrash.

#### 3.3.2 Write-Path Safety (Architectural Rate Limiting)
To directly target the deserialization boundary and test lock contention, we designed the [`run-kind-write-contention-benchmark.sh`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/run-kind-write-contention-benchmark.sh) test. This script floods the API Server with 50 concurrent Server-Side Apply (SSA) `PATCH` requests using randomly generated, entirely novel strings. This forces the API server to heavily decode `managedFields` and repeatedly hit the `unique.Make()` locking path.

To capture block and mutex delays, we instrumented the API server with `runtime.SetMutexProfileFraction(1)` and exposed the `/debug/pprof/mutex?seconds=30` endpoint. We measured contention by tracking the sum of wait times (delays) reported across all mutex events during the sustained 30-second concurrent write window.

| Metric (30s window) | `master` (Baseline `[]byte`) | `experimental` (`stringhandle`) |
| :--- | :--- | :--- |
| **Write-Path Mutex Contention** | 0 significant delays | 0 significant delays |

![Mutex Contention Plot](./mutex_contention_plot.png)

Even when explicitly forcing parallel deserialization of novel strings via SSA, the `-mutexprofile` returned completely empty on the live cluster.

While `unique.Make()` does take a lock for novel strings, the critical section executes in 1-5 nanoseconds. Before a concurrent request can reach the deserialization layer, it must traverse TLS, Authentication, RBAC, and JSON parsing. These millisecond-scale network and security layers act as a natural rate-limiter. With the benchmark tests as they were setup up it was not possible to deliver parallel requests fast enough over an HTTP boundary to overwhelm the lock-free spin-phase of Go's mutex.

*   **Addressing the Protobuf Decode Concern:** During SIG discussions, a specific hypothetical was raised regarding Protobuf deserialization: *"What if 10 things are decoding in parallel, making 100s of unique.Make calls each inside a massive Protobuf message?"* We authored a dedicated microbenchmark ([`bench_contention_protobuf_test.go`](https://github.com/aaron-prindle/kubernetes/blob/ssa-fieldsv1-string-interning-poc/hack/benchmark/force/bench_contention_protobuf_test.go)) exactly mirroring these parameters (10 parallel goroutines each executing 500 contiguous `unique.Make` calls). When the fields represented duplicated data, the array completed in just **~2,053 nanoseconds** (4ns per string). Even when we maliciously injected 500 entirely novel, random strings into all 10 parallel decoders simultaneously to force maximum contention, the operation still completed in **<1 millisecond** (~900 microseconds).

## 4. Rollout Strategy
Transitioning a core API metadata field requires managing the blast radius for OSS and client-go developers. We propose a multi-release transition plan:

*   **Phase 1: Encapsulation (v1.36)**
    *   Extract `FieldsV1` from code generators and introduce string accessor methods (`GetRaw()`, `SetRaw()`).
    *   Deprecate `FieldsV1.Raw` and sweep the in-tree codebase to exclusively use the accessors. 
    *   Introduce the opt-in build tag (`fieldsv1_stringhandle`) while keeping the legacy `[]byte` implementation as the default. This safely lays the groundwork and allows early adopters to compile custom memory-saving binaries.
*   **Phase 2: Default Flip (v1.37)**
    *   Flip the default behavior so `FieldsV1` is internally backed by `unique.Handle[string]`.
    *   Provide a reverse opt-out build tag (`fieldsv1_byte`) for clients who haven't migrated.
*   **Phase 3: Cleanup (v1.38)**
    *   Remove the legacy exported `[]byte` version and the opt-out build tags entirely.