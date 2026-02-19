# Memory Flow Diagram: Where ManagedFields Consume Memory

## Object Lifecycle and Memory Copies

```
                    ┌──────────────────────────────────┐
                    │         etcd Storage              │
                    │  ┌────────────────────────────┐   │
                    │  │ Object + managedFields     │   │  COPY 1: Persistent storage
                    │  │ (protobuf serialized)      │   │  ~5-50 KB per object
                    │  └────────────────────────────┘   │
                    └──────────────┬───────────────────┘
                                  │
                         Watch/List from etcd
                                  │
                    ┌─────────────▼────────────────────┐
                    │      API Server Process           │
                    │                                    │
                    │  ┌─────────────────────────────┐  │
                    │  │    Watch Cache (per resource)│  │
                    │  │                              │  │
                    │  │  ┌────────────────────────┐  │  │
                    │  │  │ Store (BTree/HashMap)  │  │  │  COPY 2: Current state
                    │  │  │ 10,000+ Elements       │  │  │  Each Element has full object
                    │  │  │ Object + managedFields │  │  │  + managedFields
                    │  │  └────────────────────────┘  │  │
                    │  │                              │  │
                    │  │  ┌────────────────────────┐  │  │
                    │  │  │ Cyclic Event Buffer    │  │  │  COPY 3: Historical events
                    │  │  │ Up to 102,400 events   │  │  │  Each event has current +
                    │  │  │ Object + managedFields │  │  │  previous object with mf
                    │  │  │ PrevObj + managedFields │  │  │  (COPY 3a: PrevObject)
                    │  │  └────────────────────────┘  │  │
                    │  │                              │  │
                    │  │  ┌────────────────────────┐  │  │
                    │  │  │ Snapshots (optional)   │  │  │  COPY 4: Lazy-copy snapshots
                    │  │  │ B-tree clones          │  │  │  (pointers, minimal overhead)
                    │  │  └────────────────────────┘  │  │
                    │  └─────────────────────────────┘  │
                    │                                    │
                    │  During Watch Event Dispatch:       │
                    │  ┌─────────────────────────────┐  │
                    │  │ cachingObject wrapper       │  │  COPY 5: Wrapped object
                    │  │  ┌───────────────────────┐  │  │
                    │  │  │ object (lazy copy)     │  │  │
                    │  │  │ + managedFields        │  │  │
                    │  │  └───────────────────────┘  │  │
                    │  │  ┌───────────────────────┐  │  │
                    │  │  │ serializations cache   │  │  │
                    │  │  │ JSON bytes (+ mf)     │  │  │  COPY 6: JSON serialized
                    │  │  │ Protobuf bytes (+ mf) │  │  │  COPY 7: Protobuf serialized
                    │  │  │ CBOR bytes (+ mf)     │  │  │  COPY 8: CBOR serialized
                    │  │  └───────────────────────┘  │  │
                    │  └─────────────────────────────┘  │
                    │                                    │
                    │  During Apply/Update Processing:    │
                    │  ┌─────────────────────────────┐  │
                    │  │ fieldpath.ManagedFields map  │  │  COPY 9: Decoded managed fields
                    │  │  ┌───────────────────────┐  │  │  N Sets (one per manager)
                    │  │  │ Set 1 (PathElements)  │  │  │  Each Set: 1-50 KB
                    │  │  │ Set 2 (PathElements)  │  │  │
                    │  │  │ ...                   │  │  │
                    │  │  │ Set N (PathElements)  │  │  │
                    │  │  └───────────────────────┘  │  │
                    │  │ configObject.ToFieldSet()    │  │  COPY 10: Applied config set
                    │  │  ┌───────────────────────┐  │  │  1-200 KB transient
                    │  │  │ Applied Set (temp)    │  │  │
                    │  │  └───────────────────────┘  │  │
                    │  └─────────────────────────────┘  │
                    └──────────────┬───────────────────┘
                                  │
                         Watch Events to clients
                                  │
                    ┌─────────────▼────────────────────┐
                    │    Client (kubelet, controller,    │
                    │    kube-proxy, etc.)               │
                    │  ┌────────────────────────────┐   │
                    │  │ Informer Cache             │   │  COPY 11: Client-side cache
                    │  │ Object + managedFields     │   │  (unless using TransformStrip)
                    │  └────────────────────────────┘   │
                    └──────────────────────────────────┘
```

## Memory Per Object at Each Stage

For a typical Deployment managed by 3 managers:
```
Object actual data (spec, status):       ~3 KB
managedFields (3 entries):               ~4 KB  (57% of metadata!)
Total object size:                       ~7 KB

Memory copies in apiserver:
  etcd:                                  ~7 KB  (protobuf, slightly smaller)
  Store (current state):                 ~7 KB
  Event buffer (current):               ~7 KB
  Event buffer (previous):              ~7 KB
  Serialization (JSON):                 ~8 KB  (JSON overhead)
  Serialization (Protobuf):             ~6 KB  (protobuf compression)
  ─────────────────────────────────────────────
  Total per object in apiserver:        ~42 KB

  Without managedFields:
  Object only:                          ~3 KB per copy
  Total per object:                     ~18 KB (6 copies * 3 KB)
  ─────────────────────────────────────────────
  managedFields overhead:               ~24 KB per object (57%)
```

## Cluster-Scale Impact

### 100,000 Objects (Medium Cluster)
```
With managedFields:     100,000 * 42 KB  = 4.2 GB
Without managedFields:  100,000 * 18 KB  = 1.8 GB
Savings:                                  = 2.4 GB (57%)
```

### 500,000 Objects (Large Cluster)
```
With managedFields:     500,000 * 42 KB  = 21 GB
Without managedFields:  500,000 * 18 KB  = 9 GB
Savings:                                  = 12 GB (57%)
```

## Where Solutions Apply

```
┌─────────────────────────────────────────────────────────────┐
│                                                              │
│  Solution 1: Compress FieldsV1.Raw                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Affects: All copies containing FieldsV1.Raw          │   │
│  │ Savings: 60-80% of FieldsV1 portion                  │   │
│  │ Impact:  10-25% total memory reduction               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Solution 2: Server-side exclusion parameter                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Affects: Serialization cache (copies 6-8)            │   │
│  │         Wire bandwidth                                │   │
│  │         Client caches (copy 11)                       │   │
│  │ Savings: 20-40% of serialized data                    │   │
│  │ Impact:  15-30% total memory + bandwidth reduction    │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Solution 3: Strip from watch cache                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Affects: Store (copy 2), Event buffer (copies 3/3a)  │   │
│  │         Serialization cache (copies 6-8)              │   │
│  │         cachingObject (copy 5)                        │   │
│  │ Savings: ALL managedFields from cache                 │   │
│  │ Impact:  20-40% total memory reduction                │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Solution 4: Deduplication                                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Affects: All FieldsV1.Raw copies                      │   │
│  │ Savings: Depends on workload homogeneity              │   │
│  │ Impact:  5-15% (best for homogeneous workloads)       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Profiling Evidence Summary

| Source | managedFields Memory % | Total Memory |
|--------|----------------------|--------------|
| KEP-555 (official statement) | "up to 60% of total object size" | N/A |
| Issue #102259 (profiling) | 59.72% (ObjectMetaFieldsSet) | 45.65 GB |
| Issue #76219 (benchmarks) | 2.5x response size increase | N/A |
| kube.rs benchmarks | ~50% of metadata YAML | N/A |
| Issue #90066 (output analysis) | 55.9% of YAML lines | N/A |
