# Memory Bottleneck Analysis: Why SSA Causes Apiserver Memory Issues

## Executive Summary

Server Side Apply's `managedFields` metadata is a significant contributor to apiserver memory usage in many high-cardinality workloads. In local reproduction and public reports, managedFields frequently appears as a large fraction of object metadata and can become a material fraction of total object size. At scale, this can add hundreds of megabytes to multiple gigabytes across cache and fanout paths.

Important caveat:
- Exact percentages are workload-dependent.
- managedFields is often one of the top contributors, but not always the sole dominant one.

## The Core Problem

### Memory Scaling Formula
```
Total managedFields Memory ≈
    NumObjects × NumManagers × AvgFieldsV1Size × CacheMultiplier

Where:
  NumObjects:       10,000 - 200,000 (varies by cluster size)
  NumManagers:      3-11 per object (Update capped at 10, Apply uncapped)
  AvgFieldsV1Size:  500 bytes - 20 KB (depends on object complexity)
  CacheMultiplier:  2-4x (store + event buffer + serialization + copies)
```

### Scale Example: 5,000 Node Cluster
| Resource | Count | Avg managedFields | Watch Cache Total |
|----------|-------|-------------------|-------------------|
| Pods | 150,000 | 8 KB | 1.2 GB |
| ConfigMaps | 20,000 | 3 KB | 60 MB |
| Secrets | 30,000 | 2 KB | 60 MB |
| Services | 5,000 | 3 KB | 15 MB |
| Endpoints | 5,000 | 15 KB | 75 MB |
| EndpointSlices | 10,000 | 5 KB | 50 MB |
| Nodes | 5,000 | 10 KB | 50 MB |
| Deployments | 10,000 | 5 KB | 50 MB |
| ReplicaSets | 30,000 | 4 KB | 120 MB |
| **Total** | **265,000** | | **~1.68 GB** |

This is an estimate for managedFields bytes before runtime effects (GC behavior, object sharing/copying patterns, watch topology, and feature-gated cache behavior) are fully accounted for.

## Bottleneck Analysis

### Bottleneck #1: FieldsV1.Raw Size (CRITICAL)

**The Problem**: FieldsV1 stores a JSON-encoded trie that mirrors the structure of the object. For objects with many fields (especially lists with associative keys), this trie is verbose.

**Why it's expensive**:
- List items with key fields generate paths like `k:{"name":"container-1","protocol":"TCP"}` - the key values are embedded in the path, creating long strings
- Deeply nested objects create many intermediate nodes (each `"f:fieldname": {...}"`)
- The JSON encoding adds quotes, colons, braces for every node

**Example**: A Pod managed by 3 managers
```
Pod actual data:          ~4 KB
managedFields (3 entries): ~6 KB (150% of actual data!)
  - Entry 1 (kubectl):    ~2 KB (FieldsV1 for spec)
  - Entry 2 (controller): ~2 KB (FieldsV1 for status)
  - Entry 3 (scheduler):  ~2 KB (FieldsV1 for nodeName, status conditions)
```

**Impact at scale**: With 150,000 pods, managedFields alone consumes **900 MB** in the watch cache.

### Bottleneck #2: Cache Stores Full Objects (CRITICAL)

**The Problem**: The watch cache stores complete `runtime.Object` instances including all managedFields. There is no mechanism to store objects with managedFields stripped or compressed.

**Where objects are stored with full managedFields**:
1. `watchCache.store` (Indexer) - current state of all objects
2. `watchCache.cache` (cyclic buffer) - recent events, up to 102,400 per resource type
3. `cachingObject.object` - wrapped objects during dispatch
4. `cachingObject.serializations` - serialized bytes including managedFields

**Code path** (watch_cache.go:processEvent):
```go
func (w *watchCache) processEvent(event watch.Event) {
    // Object from etcd includes FULL managedFields
    wcEvent := &watchCacheEvent{
        Object:     event.Object,  // Full object with managedFields
        PrevObject: prevObject,    // Previous version with managedFields
    }
    // Stored in cyclic buffer AND store - both with full managedFields
    w.cache[w.endIndex%w.capacity] = wcEvent
    w.store.Update(elem)
}
```

### Bottleneck #3: Serialization Multiplier (HIGH)

**The Problem**: Each cached object may be serialized in multiple formats (JSON, Protobuf, CBOR). Each serialization includes the full managedFields data.

**In cachingObject** (caching_object.go:136):
```go
func (o *cachingObject) CacheEncode(id runtime.Identifier, encode func(runtime.Object, io.Writer) error, w io.Writer) error {
    result.once.Do(func() {
        // Encodes FULL object including managedFields
        result.err = encode(o.GetObject(), buffer)
        result.raw = buffer.Bytes()
    })
}
```

**Multiplier effect**:
- If watchers use both JSON and Protobuf: 2x serialized size
- With CBOR added: 3x serialized size
- Each serialization contains the full managedFields

### Bottleneck #4: Deep Copy Cost (MEDIUM)

**The Problem**: Accessing the cached object requires deep copy, which copies all managedFields data.

```go
func (o *cachingObject) GetObject() runtime.Object {
    return o.object.DeepCopyObject().(metaRuntimeInterface)
}
```

Deep copy of managedFields involves:
- Copying the `[]ManagedFieldsEntry` slice
- Copying each entry's string fields
- Copying each `FieldsV1.Raw` byte slice
- **For 10 KB of managedFields: ~10 KB allocation per deep copy**

### Bottleneck #5: ToFieldSet() Per-Operation Cost (MEDIUM)

**The Problem**: Every Apply operation calls `configObject.ToFieldSet()` which walks the entire applied configuration and builds a new Set in memory.

**Code path** (vendor/.../merge/update.go):
```go
func (s *Updater) Apply(...) {
    // EXPENSIVE: walks entire config object
    set, err := configObject.ToFieldSet()
    // Creates Set with PathElement for every field/list item
}
```

**Cost**: For a config with 100 fields, this creates ~100 PathElement allocations (~10-20 KB of transient allocations per Apply).

### Bottleneck #6: Event Buffer Historical Objects (MEDIUM)

**The Problem**: The cyclic event buffer stores up to 102,400 historical events per resource type. Each event includes the full object with managedFields.

**Worst case**: A high-churn resource type (e.g., Endpoints in a 5000-node cluster)
- 102,400 events * ~20 KB (Endpoints with managedFields) = **~2 GB** just for the event buffer of one resource type

### Bottleneck #7: No Deduplication (LOW-MEDIUM)

**The Problem**: When the same object appears in both the store and the event buffer, the managedFields data is not deduplicated. Even though the managedFields rarely change, they're stored as separate byte slices.

## Why This Matters

### Most Watchers Don't Need managedFields

Analysis of typical Kubernetes watchers:
| Watcher | Uses managedFields? |
|---------|-------------------|
| kube-proxy (Endpoints/Services) | No |
| kubelet (Pods, Secrets, ConfigMaps) | No |
| kube-scheduler (Pods, Nodes) | No |
| kube-controller-manager (various) | Sometimes (for SSA) |
| Custom controllers (informers) | Rarely |
| kubectl get/watch | No (hidden by default) |

Estimate:
- Many common watchers do not consume managedFields directly.
- The exact ratio should be measured per environment rather than assumed globally.

In default behavior today, watch/list responses and cached objects generally include managedFields unless a component/client explicitly trims or omits it.

### Memory vs. Information Value

```
┌──────────────────────────────────────────────┐
│           Apiserver Memory Usage              │
│                                               │
│  ┌───────────────────────────┐               │
│  │     Object Data           │  60-70%        │
│  │  (spec, status, etc.)     │               │
│  ├───────────────────────────┤               │
│  │     managedFields         │  20-30%  <--- Almost entirely wasted
│  │  (FieldsV1 tries)        │               │
│  ├───────────────────────────┤               │
│  │     Other Metadata        │  5-10%        │
│  │  (labels, annotations)    │               │
│  ├───────────────────────────┤               │
│  │     Cache Overhead        │  5-10%        │
│  │  (indexes, pointers)      │               │
│  └───────────────────────────┘               │
└──────────────────────────────────────────────┘
```

### The Asymmetry
- **Write path**: managedFields are essential (conflict detection, field ownership)
- **Read path**: many consumers do not require managedFields
- **Serving path**: managedFields is frequently present even for consumers that do not use it

## Do Not Conflate These Three Buckets

1. apiserver heap/RSS:
- In-memory object and cache retention in apiserver process.

2. controller/informer memory:
- Separate process memory that can be reduced with client-side transforms (for example `TransformStripManagedFields`).

3. wire-size and serialization cost:
- Response payload and CPU/alloc overhead for encoding/decoding.
- Wire compression helps this bucket but does not directly remove apiserver in-memory object cost.

## Comparison with Other Metadata Overhead

| Metadata Component | Typical Size | Optimization Exists? |
|-------------------|-------------|---------------------|
| labels | 100-500 bytes | Yes (selector-based filtering) |
| annotations | 200-2000 bytes | Partial (some can be pruned) |
| ownerReferences | 100-200 bytes | No, but small |
| managedFields | 2,000-20,000+ bytes | **No** |
| last-applied-config* | 0-50,000+ bytes | SSA removes this |

*Note: SSA was designed to replace last-applied-configuration annotation, which could be even larger. But the managedFields replacement, while more structured, is still substantial.

## Conclusion

The apiserver memory pressure from SSA/managedFields is a structural scaling concern: ownership metadata grows with object complexity and manager activity, and this overhead can be multiplied by cache and fanout behavior.

Updated conclusion:
- managedFields is a high-confidence contributor to memory growth.
- expected gains from targeted mitigations should be treated as scenario-dependent until validated with cluster-specific profiling.
- a practical working range for planning remains significant (often double-digit percentages), but should be presented as measured outcomes, not universal constants.
