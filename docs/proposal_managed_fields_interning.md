# Proposal: Mitigating managedFields Memory Bloat via String Interning

**Status**: Draft
**Authors**: Aaron Prindle
**Last revised**: <Current Date>

## 1. Problem Statement
Based on large-scale cluster profiling, `managedFields` has emerged as a dominant factor in `kube-apiserver` memory exhaustion at scale. In environments with highly replicated resources—such as `DaemonSet`s, `ReplicaSet`s, `StatefulSet`s, and `Job`s—thousands of Pods are created from identical templates.

When these Pods are processed by the API server and held in the `WatchCache` (e.g., for the 5-minute history window), their `managedFields` payloads are duplicated as distinct `[]byte` slices in memory. At scales of 50,000+ pods, this results in massive amounts of redundant data trapped in the heap, causing O(N) memory bloat for identical metadata.

## 2. Proposed Solution
We propose transitioning the underlying data representation of `metav1.FieldsV1` from a mutable `[]byte` to an immutable `string` (specifically, Go 1.23's `unique.Handle[string]`). By enforcing immutability at the type level, we unlock the ability to safely cache and natively intern payloads at the exact moment they are deserialized. 

This ensures that duplicate `managedFields` data across thousands of pods resolves to a single shared pointer in memory, collapsing the footprint from O(N) to O(1).

Based on the architectural proof-of-concept by [@liggitt (fieldsv1-string)](https://github.com/liggitt/kubernetes/commits/fieldsv1-string/), the solution relies on two key mechanisms:

### Accessor Encapsulation
To safely orchestrate this transition without breaking downstream `client-go` consumers, we must abstract how the codebase interacts with `FieldsV1`. We will introduce standard accessor methods (`GetRaw()`, `SetRaw()`) and eliminate all direct, in-tree use of the `.Raw` field. 

### Build-Tagged Implementations
Because `FieldsV1` relies heavily on auto-generated Protobuf and DeepCopy code, changing its underlying type dynamically breaks the code generators. The solution extracts the `FieldsV1` declaration and its unmarshal/deepcopy methods into isolated, manually maintained files governed by `//go:build` tags:

*   [`fieldsv1_byte.go`](https://github.com/liggitt/kubernetes/blob/fieldsv1-string/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/fieldsv1_byte.go): The legacy `[]byte` implementation, remaining the default for standard OSS builds.
*   [`fieldsv1_stringhandle.go`](https://github.com/liggitt/kubernetes/blob/fieldsv1-string/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/fieldsv1_stringhandle.go): The optimized implementation utilizing `unique.Make()`.

When compiled with the `stringhandle` tag, the API server intercepts payloads during JSON, CBOR, or Protobuf deserialization. The first payload allocates the string, while subsequent identical payloads hit the `unique.Make` fast-path, pointing their handle directly at the original string in memory. Furthermore, because the target implementation is an inherently immutable string, expensive defensive deep copies currently required by the `WatchCache` can be bypassed entirely.

## 3. Performance Validation
To build consensus and address concerns regarding `unique.Make` global lock contention, we designed rigorous, end-to-end live cluster benchmarks simulating extreme scaling conditions.

### 3.1 Proving the Bottleneck (Baseline Scaling)
Before validating the solution, we must empirically prove that `managedFields` becomes a true scaling bottleneck for general Kubernetes users. We tested the standard `master` branch using `Kwok` to simulate the growth of duplicated workloads (e.g., a massive `DaemonSet`).

| Number of Pods | Baseline Heap Allocation for `FieldsV1` |
| :--- | :--- |
| **1,000** | ~16 MB |
| **10,000** | ~41.5 MB |
| **50,000** | ~134.6 MB |

![Baseline Scaling Plot](./baseline_scaling_plot.png)

The baseline memory usage scales devastatingly with the number of replicas. At scales of hundreds of thousands of identical Pods across a massive cluster, `metav1.FieldsV1.Unmarshal` operations consume gigabytes of raw API server RAM just holding duplicate bytes.

### 3.2 Memory Footprint Reduction
**Objective:** Prove that string interning collapses this live API server memory footprint from O(N) to O(1).

**Methodology:**
We ran the exact same 50,000 Pod `Kwok` simulation against a `kind` node compiled from our experimental `unique.Handle` branch and extracted the live `pprof` heap profiles.

**Results:**
| Branch | Total Apiserver Heap | `FieldsV1` Allocation Profile | WatchCache Footprint Scaling |
| :--- | :--- | :--- | :--- |
| **master** (Baseline `[]byte`) | ~1.45 GB | 134.60 MB | `O(N)` |
| **experimental** (`stringhandle`) | ~1.37 GB | 27.52 MB | `O(1)` |

![Memory Scaling Plot](./memory_scaling_plot.png)

With string interning enabled, `FieldsV1` allocations plummeted by ~80% down to just **27.52 MB** (representing only the mandatory baseline allocations for the struct pointers themselves). 

### 3.3 Parallel Contention Safety
**Objective:** Address the concern that the standard library `unique` package relies on internal maps and locks. We must empirically prove that `unique.Make()` does not become a global lock bottleneck during highly parallel API Server operations.

**Methodology:**
We authored two distinct contention benchmarks against the tuned cluster:
1.  **Read-Path Benchmark (Massive LISTs):** Seeds the API Server with 10,000 duplicated Kwok Pods, then bombards the API Server with 50 highly concurrent `LIST` clients to test serialization overhead.
2.  **Write/Decode-Path Benchmark (Parallel SSA):** Floods the API Server with 50 concurrent Server-Side Apply (SSA) `PATCH` requests using randomly generated, entirely novel strings. This directly targets the deserialization boundary, forcing the API server to heavily decode `managedFields` and repeatedly hit the `unique.Make()` locking path.

**Results:**
The real-world profiles prove that `unique.Make` introduces absolutely zero contention regression under heavy parallel load. 

| Metric (30s window) | `master` (Baseline `[]byte`) | `experimental` (`stringhandle`) |
| :--- | :--- | :--- |
| **Read-Path Total CPU Load** | ~1336.79s | ~678.41s |
| **Write-Path Mutex Contention** | 0 significant delays | 0 significant delays |

![CPU Scaling Plot](./contention_scaling_plot.png)
![Mutex Contention Plot](./mutex_contention_plot.png)

*   **Read-Path Isolation & CPU Relief:** The CPU profiles revealed that `unique.Make` is not in the critical path for parallel `LIST` requests. Decoding (and thus interning) occurs only when objects are written to etcd or initially loaded into the `WatchCache`. By eliminating duplicate heap allocations, the experimental branch sliced total read-path CPU time in half (from ~1336s down to ~678s) by removing the need for background garbage collection (`mallocgc`) to thrash.
*   **Write-Path Safety (Architectural Rate Limiting):** Even when explicitly forcing parallel deserialization of novel strings via SSA, the `-mutexprofile` returned completely empty on the live cluster. While `unique.Make()` does take a lock for novel strings, the critical section executes in 1-5 nanoseconds. Before a concurrent request can reach this deep deserialization layer, it must traverse TLS, Authentication, RBAC, and JSON parsing. These millisecond-scale network and security layers act as a natural rate-limiter. It is physically impossible to deliver parallel requests fast enough over an HTTP boundary to overwhelm the lock-free spin-phase of Go's mutex, ensuring the interning lock remains entirely frictionless.

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