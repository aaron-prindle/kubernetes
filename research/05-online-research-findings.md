# Online Research Findings - SSA Performance and Memory Issues

## Key Resources Found

### 1. kube.rs Memory Optimization Guide
**URL**: https://kube.rs/controllers/optimization/

**Key findings**:
- "managed-fields often accounts for close to half of the metadata yaml"
- Stripping managed fields from reflector caches achieved ~30% memory improvement
- Combined with metadata watchers, total memory reduction was ~50-80%
- Recommendation: `pod.managed_fields_mut().clear()` before storing in cache
- Labels and annotations are safe to drop if not needed, but NOT `.metadata.name`, `.resourceVersion`, `.namespace`, or `.ownerReferences`

**Benchmark data**:
- ~2000 object Pod reflector cache
- Metadata watcher alone: significant reduction vs standard watcher
- Managed field pruning: additional ~30% on top of metadata watcher
- Full optimization stack (metadata watcher + managed field pruning): 60-80% total reduction

### 2. kube.rs Watcher Memory Improvements Blog Post
**URL**: https://kube.rs/blog/2024/06/11/watcher-memory-improvements/

**Key findings**:
- ~50% drop in memory usage for certain workloads
- Optimization included metadata_watcher, page_size 50, and pruning of managed fields
- Default memory stored for each object includes managed fields which kubectl hides but API always includes
- Synthetic benchmarks: 60% reduction using stores, 80% without stores

### 3. KEP-1152: Less Object Serializations
**URL**: https://github.com/kubernetes/enhancements/issues/1152

**Key findings**:
- "creation of a single large Endpoints object (almost 1MB of size, due to 5k pods backing it) in 5k-node cluster can completely overload kube-apiserver for 5 seconds"
- Introduced `CacheableObject` interface and `cachingObject` wrapper
- Results: ~5% lower CPU usage, ~15% fewer memory allocations
- Implemented in Kubernetes 1.17
- Serialization cache is temporary (only during watch dispatch)

### 4. API Streaming / WatchList (Kubernetes 1.32)
**URL**: https://kubernetes.io/blog/2024/12/17/kube-apiserver-api-streaming/

**Key findings**:
- 10x reduction in memory usage (20 GB -> 2 GB in synthetic tests)
- Before: Server must fetch, deserialize, construct complete response in memory
- After: Stream items individually from watch cache, constant memory overhead
- Server previously crashed at 16 concurrent informers; stable with streaming
- Feature addresses unbounded memory allocation for LIST requests
- Beta in 1.32, controller-manager enabled by default

### 5. KEP-4222: CBOR Serialization
**Key context**: CBOR (Concise Binary Object Representation) is being added as a serialization format. While more compact than JSON, it does NOT address the fundamental issue of managedFields being included in all serializations.

### 6. GitHub Issues on Apiserver Memory

#### Issue #97262: Optimize memory usage
**URL**: https://github.com/kubernetes/kubernetes/issues/97262
- Discussion of watch cache memory consumption
- Focus on reducing object copies in cache

#### Issue #90179: More memory efficient watch cache
**URL**: https://github.com/kubernetes/kubernetes/issues/90179
- Proposal for more efficient caching strategies
- Discussion of object deduplication

#### Issue #114276: Apiserver builds up high memory after large LIST requests
**URL**: https://github.com/kubernetes/kubernetes/issues/114276
- Demonstrates how LIST requests cause memory spikes
- Each LIST materializes all objects in memory simultaneously

#### Issue #98423: kube-apiserver high memory on pending pods storm
**URL**: https://github.com/kubernetes/kubernetes/issues/98423
- High memory during pod creation storms
- Each pod carries managedFields overhead

#### Issue #111699: Higher kube-apiserver memory in 1.21 vs 1.20
**URL**: https://github.com/kubernetes/kubernetes/issues/111699
- Memory increase correlated with SSA becoming default
- Additional managedFields data on all objects

### 7. Kubernetes Controllers at Scale (Medium article)
**URL**: https://medium.com/@timebertt/kubernetes-controllers-at-scale-clients-caches-conflicts-patches-explained-aa0f7a8b4332

**Key findings**:
- Detailed explanation of how controllers interact with SSA
- Discusses caching behavior and informer memory patterns
- Notes that managedFields add overhead to every cached object

### 8. Red Hat Bug Report: etcd cache causing high memory
**URL**: https://bugzilla.redhat.com/show_bug.cgi?id=1323733
- Fixed-size etcd cache in apiserver causing high memory
- When objects are large (including managedFields), cache memory grows significantly

### 9. Red Hat Analysis: kube-apiserver memory utilization
**URL**: https://bugzilla.redhat.com/show_bug.cgi?id=1953305
- Detailed memory profiling of kube-apiserver
- Watch cache identified as dominant memory consumer

## Key Themes from Research

### Theme 1: managedFields is a known problem
Multiple sources confirm that managedFields is a significant portion of object size and memory. The kube.rs project explicitly recommends stripping them for memory optimization.

### Theme 2: No merged general server-side solution (yet)
While client-side solutions exist (strip before caching in informers), there is not yet a broadly merged, GA server-side mechanism that solves managedFields overhead end-to-end for GET/LIST/WATCH and cache internals.

### Theme 3: The problem scales with cluster size
Every additional object adds managedFields overhead. At scale (5,000+ nodes):
- The apiserver stores hundreds of thousands of objects
- Each object carries 2-20+ KB of managedFields
- Total memory impact is measured in gigabytes

### Theme 4: Existing optimizations are insufficient
- KEP-1152 (serialization caching): Only helps during watch dispatch, not storage
- API Streaming (WatchList): Helps with LIST memory spikes, not steady-state cache
- CBOR: More compact serialization, but managedFields still included
- CapManagers (max 10): Helps limit growth but doesn't reduce per-entry size

### Theme 5: There's a clear need for new solutions
There is active discussion and implementation work, but no single merged architecture change that fully addresses:
- Removing managedFields from the watch cache
- Compressing FieldsV1 data in cache/storage paths
- Broad server-side filtering for managedFields across read APIs
- Lazy loading of managedFields from etcd when needed

### 10. KEP-555: Server-Side Apply (Primary KEP)
**URL**: https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/555-server-side-apply/README.md

**Critical data point**: The KEP explicitly warns that **"managedFields metadata fields can represent up to 60% of the total size of an object"**, directly increasing cache memory requirements, network bandwidth usage, etcd storage consumption, and controller memory overhead.

### 11. Issue #76219: SSA Protobuf Serialization Performance
**URL**: https://github.com/kubernetes/kubernetes/issues/76219

**Benchmark data with 1,000 pod replicas**:
- Without SSA: Response size 980,436-980,470 bytes, duration 22-47ms (avg 31ms)
- With SSA: Response size 2,465,211 bytes (**2.5x larger**), duration 97-146ms (avg 110ms, **3.5x slower**)
- Root cause: Quadratic complexity in protobuf marshaling for nested FieldsV1 maps

### 12. Issue #102259: API Server Memory Spikes (THE SMOKING GUN)
**URL**: https://github.com/kubernetes/kubernetes/issues/102259

**The most concrete profiling evidence that managedFields are the dominant memory consumer**:
- Test: ~85,000 secrets, 5,000-30,000 concurrent watchers
- Before test: `ObjectMetaFieldsSet` used 28.01 MB (1.58% of 1,771 MB total)
- During test: `ObjectMetaFieldsSet` used **16,869.57 MB (58.54%** of 28,818 MB total)
- After test killed: `ObjectMetaFieldsSet` used **27,266.59 MB (59.72%** of 45,654 MB total)
- Peak spike: `ObjectMetaFieldsSet` used **26.37 GB (34.74%** of 75.90 GB total)

### 13. Issue #90066: managedFields Verbosity
**URL**: https://github.com/kubernetes/kubernetes/issues/90066

A standard Deployment YAML output: 640 lines total, managedFields comprised **358 lines (>55%)**

### 14. Issue #97262: 2000-Node Cluster Memory
**URL**: https://github.com/kubernetes/kubernetes/issues/97262

Production cluster: 2,000 nodes, 6 API servers, 200 GB memory per physical machine (1.2 TB total). "Most of the memory is consumed by serializations triggered by all kinds of requests. The client fetches all the information about the object from apiserver, but in most situations, the client only needs a few fields."

### 15. KEP-4988: Snapshottable API Server Cache
**URL**: https://kubernetes.io/blog/2025/09/09/kubernetes-v1-34-snapshottable-api-server-cache/

Beta in v1.34. Creates lightweight "lazy copy" snapshots that store pointers instead of duplicating objects. Combined with streaming encoder (v1.33) that sends list items one by one.

### 16. Google's 130,000-Node GKE Cluster
**URL**: https://cloud.google.com/blog/products/containers-kubernetes/how-we-built-a-130000-node-gke-cluster/

Demonstrated watch cache optimizations at extreme scale:
- Consistent Reads from Cache: 3x database load reduction, 30% CPU savings
- Trade-off: 50-80% more API server memory (8 GB baseline -> ~12 GB with cache)

### 17. Kubernetes Controllers at Scale (Medium)
**URL**: https://medium.com/@timebertt/kubernetes-controllers-at-scale-clients-caches-conflicts-patches-explained-aa0f7a8b4332

Detailed explanation of controller-runtime `TransformStripManagedFields` for cache optimization.

### 18. KEP-2982: Drop managedFields from Audit Entries
**URL**: https://github.com/kubernetes/kubernetes/pull/94986

GA in v1.23. Added `omitManagedFields` to audit policies. Motivation: "ManagedFields of an object in the audit entries are not very useful and it consumes storage space pretty quickly especially in a big cluster under load."

### 19. Open PR #136760: get/list option to omit managedFields
**URL**: https://github.com/kubernetes/kubernetes/pull/136760

Open work-in-progress proposal for an API option to omit managed fields in read paths. This is a significant upstream signal that server-side omission is now being actively explored.

### 20. Issue #131175: no-op SSA metadata churn
**URL**: https://github.com/kubernetes/kubernetes/issues/131175

Open issue suggesting that no-op SSA can still update `resourceVersion` and managedFields timestamps. This matters because metadata churn can create avoidable watch/cache pressure.

### 21. PR #131016: scheduler managedFields trim fix
**URL**: https://github.com/kubernetes/kubernetes/pull/131016

In-tree evidence that managedFields trimming in memory-sensitive consumers is practical and useful.

## Current Upstream Status (as of 2026-02-13)

1. There are merged targeted mitigations:
- `kubectl` output stripping (`#96878`)
- audit omission policy (`#94986`)
- component-level trimming patterns (e.g., scheduler path + fix in `#131016`)

2. There is active but unmerged server-side read-path work:
- open PR `#136760` (omit managedFields in get/list)

3. There is no merged, universal, low-risk server-side fix that simultaneously:
- minimizes managedFields memory in apiserver cache internals,
- preserves SSA ownership/conflict semantics,
- and provides broad client compatibility by default.

## Absent Research

The following items were searched for and no clearly merged upstream end-state was found:
1. KEP for stripping managedFields from watch cache
2. KEP for compressing managedFields in apiserver cache
3. KEP for server-side managedFields filtering on watch/list
4. Any proposal for lazy-loading managedFields from etcd
5. Any proposal for deduplicating common FieldsV1 patterns

This confirms the managedFields memory problem is known and documented, with partial mitigations and active exploration, but without a merged comprehensive server-side solution today.
