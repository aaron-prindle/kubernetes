# Proposal: Mitigating managedFields Memory Bloat via String Interning

**Status**: Draft
**Authors**: Aaron Prindle
**Last revised**: <Current Date>

## 1. Problem Statement
Based on recent analysis (e.g., large-scale cluster profiling, KCP memory reduction efforts), `managedFields` has been identified as a dominant factor in memory exhaustion at scale. In environments with highly replicated resources (ex: `DaemonSet`s, `ReplicaSet`s, `StatefulSet`s, and `Job`s) thousands of Pods are created from identical templates.

When these Pods are processed by the API server and held in the watch cache (e.g., 5-minute history), their `managedFields` payloads (often large JSON/CBOR structures) are duplicated as distinct `[]byte` slices in memory. At 50,000+ pods, this results in a large amount of redundant data trapped in the heap (large memory usage in the api-server).

## 2. Proposed Solution - String Interning
We propose a phased transition of `metav1.FieldsV1`'s underlying data representation from a mutable `[]byte` to an immutable `string` (or `unique.Handle[string]`). By enforcing immutability at the type level, we unlock the ability to safely and natively intern the payload. This drastically reduces the memory footprint of `managedFields` in the `kube-apiserver`.

Based on the experimental [fieldsv1-string](https://github.com/liggitt/kubernetes/commits/fieldsv1-string/) PoC branch by @liggit, the solution architecture involves the following technical specifics:

*   **Accessor Encapsulation:** The raw data of `FieldsV1` is encapsulated behind standard accessor methods (`GetRaw() string`, `SetRaw(string)`, `GetRawReader()`, and `Equal()`). This completely abstracts the underlying memory type away from the rest of the Kubernetes codebase.
*   **Build-Tagged Implementations:** By extracting `FieldsV1` from code generators into manually maintained files, we can provide multiple implementations swapped safely at compile-time via `//go:build` tags:
    *   `fieldsv1_byte.go`: The legacy `[]byte` implementation for safe default OSS rollout.
    *   `fieldsv1_string.go`: A pure `string` implementation.
    *   `fieldsv1_stringhandle.go`: An optimized implementation utilizing Go 1.23's `unique.Handle[string]`.
*   **Safe Caching (Immutability):** Because the target implementation uses a string (which is inherently thread-safe and immutable), deep copies are no longer required to protect the `WatchCache` from accidental mutations by downstream consumers or informers.
*   **Native Interning at the Boundary:** The `fieldsv1_stringhandle.go` variant leverages the standard library `unique` package. At the exact moment of deserialization (JSON, CBOR, or Protobuf), the byte stream is intercepted and passed through `unique.Make()`. Identical metadata payloads returned from the network immediately resolve to a single shared memory address, eliminating duplicate heap allocations.
*   **Semantic Equality:** A custom `Semantic.Equal` helper is registered for `FieldsV1` in the API machinery, ensuring deep equality checks succeed uniformly regardless of which build tag variant is compiled into the binary.

## 3. Rollout Plan
Transitioning a core API metadata field is highly disruptive to the ecosystem. To manage the blast radius for OSS and client-go developers, we propose a multi-release transition plan. 

A significant technical hurdle is that `FieldsV1.Raw` relies heavily on auto-generated code (Protobuf and DeepCopy). To allow swapping the underlying type without breaking or confusing the code generators, we must extract the `FieldsV1` declaration and its generated methods into isolated, manually maintained files (e.g., `fieldsv1_byte.go`, `fieldsv1_proto.go`).

Once isolated, we can provide multiple implementations swapped safely at compile-time via build tags:

*   **v1.3x (Initial Step): (v1.36?)**
    *   Extract `FieldsV1` from code generators and introduce string accessor methods (e.g., `GetRaw() string`, `SetRaw(string)`, `GetRawReader()`, and `Equal()`).
    *   Eliminate all direct in-tree use of the `Raw` `[]byte` field across the Kubernetes codebase in favor of the new accessors.
    *   Mark `FieldsV1.Raw` as deprecated to warn external consumers.
    *   Introduce opt-in build tags (e.g., `fieldsv1_string` or a `unique.Handle[string]` variant) while keeping the legacy `[]byte` implementation as the default. This allows early adopters operating large-scale environments to opt into the memory savings immediately via a custom build while the broader ecosystem absorbs the deprecation.
*   **v1.3y (Default Flip): (v1.37?)**
    *   Flip the default behavior so `FieldsV1` is internally backed by an unexported `unique.Handle[string]` (or `string`).
    *   Provide a reverse opt-out build tag (e.g., `fieldsv1_byte`) for clients who haven't migrated.
*   **v1.3z (Cleanup): (v1.38?)**
    *   Remove the legacy exported `[]byte` version and the opt-out build tags entirely.

## 4. Performance & Contention Analysis
To build consensus in the OSS community and address concerns regarding `unique.Make` global lock contention, we must empirically prove both the memory benefits across various workloads and the runtime safety under high parallelism.

### 4.1 Memory Reduction Profiles
**Objective:** The primary goal of this initiative is to eliminate redundant heap allocations caused by identical `managedFields` structures. While `DaemonSet`s are the traditional culprit for massive Pod duplication, modern Kubernetes scale testing (e.g., KCP large-scale environments) has revealed that `Job`s (especially `JobSet` waiting for execution) and `StatefulSet` overcommit scenarios generate similar, devastating duplication within the `WatchCache`.

We must empirically prove that transitioning to `unique.Make()` collapses the memory footprint of these high-replica workloads from O(N) to O(1) relative to Pod count.

**Methodology:**
We authored a custom memory profiling benchmark (available on the [PoC branch](https://github.com/aaron-prindle/kubernetes/tree/ssa-fieldsv1-string-interning-poc) at `hack/benchmark/bench_memory_footprint.go`). 

To simulate the `WatchCache` receiving massive `LIST` responses, we generated representative `managedFields` JSON payloads for `DaemonSet`, `Job`/`JobSet`, and `StatefulSet` Pods. The benchmark allocates 50,000 duplicated instances of these structures in a tight loop and measures the retained heap memory (post-GC) for two variants:
1.  **Baseline (`[]byte`):** Simulates the legacy behavior where each decoding operation allocates a distinct byte slice.
2.  **Proposed (`unique.Make`):** Simulates interning via the Go 1.23 standard library `unique.Make()`.

*Hypothesis:* Retained heap for `FieldsV1` using `unique.Make` scales at O(1) relative to pod count rather than O(N).

**Results & Analysis:**
The results conclusively validate the O(1) hypothesis. The `unique.Make()` approach nearly eradicates the duplicated data footprint.

| Controller | Pod Count | Baseline `[]byte` (MB) | Proposed `string` (MB) | Memory Reduction |
| :--- | :--- | :--- | :--- | :--- |
| **DaemonSet** | 50,000 | ~15.84 MB | ~1.68 MB | **~89.4%** |
| **Job/JobSet** | 50,000 | ~14.25 MB | ~1.67 MB | **~88.3%** |
| **StatefulSet** | 50,000 | ~17.31 MB | ~1.67 MB | **~90.3%** |

*   **Baseline Scaling:** As expected, the baseline memory usage scales linearly with the number of replicas, bloating the heap to ~14-17 MB for just 50k Pods (representing a single controller's footprint). At scales of hundreds of thousands of identical Pods across a cluster, this quickly consumes gigabytes of RAM.
*   **Interning Efficiency:** Regardless of the specific controller's payload structure, interning caps the retained memory at a static ~1.6 MB overhead for the struct pointers themselves, while the actual underlying `managedFields` data occupies virtually zero additional bytes beyond the first instance.

**Conclusion:** 
Across all tested high-replica controllers, transitioning to `unique.Make` ensures the `WatchCache` memory footprint drops by ~90%, and effectively scales as O(1) regardless of total Pod replication.

**Future Improvements:**
While this isolates and proves the core memory optimization, further empirical validation could include:
*   **Live Cluster Profiling:** Provisioning a massive `kwok` cluster with 100,000+ simulated nodes and deploying large `DaemonSet`s. We could capture actual `pprof` heap profiles from a live `kube-apiserver` comparing the `fieldsv1_byte` and `fieldsv1_stringhandle` build-tag variants.
*   **WatchCache History Scope:** Simulating the impact of the 5-minute watch cache history window across varying churn rates to prove memory savings aren't just for static duplication, but also for repeated metadata updates over time.

### 4.2 Parallel Decoding Contention Analysis
**Objective:** A primary concern raised by SIG API Machinery regarding `unique.Make()` is the potential for global lock contention. The standard library `unique` package relies on internal synchronization (maps and locks). If highly parallel API Server operations (e.g., massive `LIST` requests or parallel watch event decodes) experience severe lock contention on `unique.Make()`, the resulting latency could overwhelm any memory-saving benefits. 

We must empirically evaluate whether `unique.Make()` becomes a bottleneck across high-concurrency decoding scenarios compared to standard garbage collection overhead.

**Methodology:**
To simulate the `WatchCache` concurrently deserializing numerous `managedFields` objects, we authored a custom parallel benchmark (available on the [PoC branch](https://github.com/aaron-prindle/kubernetes/tree/ssa-fieldsv1-string-interning-poc) at `hack/benchmark/bench_parallel_test.go`). 

We ran `b.RunParallel` against a realistic, nested `managedFields` JSON payload across a matrix of concurrent goroutines (`-cpu=1,10,50,100,500,1000`) on an AMD EPYC machine. We analyzed two variants:
1.  **Baseline (`[]byte`):** Simulates the current legacy behavior. `metav1.FieldsV1.Unmarshal` performs a `make([]byte)` and a `copy` operation for every payload, allocating independent byte slices.
2.  **Proposed (`unique.Make`):** Simulates interning via Go 1.23 standard library `unique.Make(string(data))`.

To capture locking behavior, the tests were run with `-mutexprofile=mutex.out` and analyzed via `go tool pprof`.

**Results & Analysis:**
The benchmark demonstrates that `unique.Make` handles extreme load exceptionally well, drastically outperforming the legacy baseline.

| Benchmark | Concurrency | Time/op | Bytes/op | Allocs/op |
| :--- | :--- | :--- | :--- | :--- |
| **Baseline** | 1 | ~76 ns | 288 B | 1 |
| **Baseline** | 1000 | ~103 ns | 288 B | 1 |
| **Proposed** | 1 | ~49 ns | 0 B | 0 |
| **Proposed** | 1000 | ~1.3 ns | 0 B | 0 |

*   **Execution Time:** The baseline copying slowed down as concurrency increased (from 76ns to 103ns) because high allocation rates trigger heavy memory allocator and Garbage Collector (GC) pressure. Conversely, `unique.Make` execution time collapsed from ~49 ns/op down to ~1.3 ns/op at 1000 goroutines. By eliminating allocations (`0 B/op`), the operations scaled perfectly in parallel across CPUs.
*   **Contention Profile:** The `-mutexprofile` confirmed the most critical finding: **there is zero significant contention from the `unique` package map locks themselves.** The profile revealed that >99% of all lock contention during the `Baseline` tests originated entirely from `runtime.mallocgc` functions. 

**Conclusion:** 
Bypassing `mallocgc` dramatically *improves* parallel API server throughput. The background GC and memory allocator spinlocks are vastly more expensive and highly contended than the internal synchronization used by `unique.Make`. There are no hidden performance regressions with high-parallelization contention for `unique.Make` in this workflow.

**Future Improvements:**
While this test isolates the raw deserialization path and proves `unique.Make` is non-blocking, it is synthetic. To make the argument airtight for the OSS community, we should consider extending the benchmarks in the future:
*   **Realistic API Server Test:** Run an end-to-end `kube-apiserver` load test utilizing tools like `kwok` or `clusterloader2` to measure actual `LIST` latency at the HTTP boundary under heavy parallel request loads.
*   **WatchCache Simulation:** Benchmark the exact `WatchCache` deep-copy logic to show holistic CPU savings when readers no longer clone byte slices.

---

## Appendix: Next Steps
To finalize the path forward, the following research and implementation tasks will be executed:

### Step 1: Finalize Accessors and Build-Tag Architecture
*   **Action:** Build upon the experimental PoC work (`liggitt/fieldsv1-string` branch) that successfully proved the feasibility of isolating the `FieldsV1` generated code.
*   **Action:** Open an initial PR that introduces the extracted `fieldsv1_*.go` files, the `GetRaw()`, `SetRaw()`, and `Equal()` accessors, and the build-tagged architecture (`fieldsv1_byte`, `fieldsv1_string`).
*   **Action:** Deprecate `FieldsV1.Raw` and sweep the in-tree codebase (like `fieldmanager`, API machinery, and integration tests) to exclusively use the accessors. This safely lays the groundwork without changing the default struct type for OSS yet.