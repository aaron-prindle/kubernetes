# Future Plan: Fixing the SSA ManagedFields Memory Bottleneck

## Problem Statement

The Kubernetes API server's memory usage scales poorly with the number of resources, primarily due to `managedFields` metadata introduced by Server Side Apply (SSA). At scale (5,000+ nodes, 100,000+ objects), managedFields can consume **1-3+ GB of apiserver memory** - representing 20-40% of total memory usage. This memory is almost entirely wasted because <5% of API clients ever use managedFields data.

## Root Cause

Every Kubernetes object carries `managedFields` in its `ObjectMeta`:
- Each manager (controller, user, tool) adds a `ManagedFieldsEntry` with a `FieldsV1` trie
- `FieldsV1` is a JSON-encoded tree of all field paths the manager owns
- A typical object has 3-11 entries, each 500 bytes to 20+ KB
- The watch cache stores the FULL object (including managedFields) in memory
- No compression, lazy-loading, or filtering exists

## Proposed Fix: Multi-Phase Approach

### Phase 1: FieldsV1 Compression in Watch Cache (3-4 weeks)

**Goal**: Reduce the memory footprint of FieldsV1 data by 60-80% through in-memory compression.

**Approach**:
1. Introduce a `CompressedFieldsV1` wrapper that stores zstd-compressed `FieldsV1.Raw` bytes
2. Apply compression when objects enter the watch cache
3. Decompress lazily when managedFields are accessed (serialization, Apply operations)

**Key Changes**:
```
staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/
  - Add CompressedFieldsV1 type (internal only, not on wire format)

staging/src/k8s.io/apiserver/pkg/storage/cacher/
  - watch_cache.go: Compress managedFields when storing in cache
  - caching_object.go: Decompress when creating serialization

staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/
  - Add compress/decompress helpers
```

**Implementation Details**:
```go
// New internal type - not part of API, only used in cache
type compressedManagedFieldsEntry struct {
    Manager         string
    Operation       ManagedFieldsOperationType
    APIVersion      string
    Time            *metav1.Time
    Subresource     string
    CompressedRaw   []byte  // zstd compressed FieldsV1.Raw
    UncompressedLen int     // Original size for pre-allocation
}

// Compress on cache insert
func compressManagedFields(mf []metav1.ManagedFieldsEntry) []compressedManagedFieldsEntry {
    result := make([]compressedManagedFieldsEntry, len(mf))
    for i, entry := range mf {
        result[i] = compressedManagedFieldsEntry{
            Manager:     entry.Manager,
            Operation:   entry.Operation,
            APIVersion:  entry.APIVersion,
            Time:        entry.Time,
            Subresource: entry.Subresource,
        }
        if entry.FieldsV1 != nil && len(entry.FieldsV1.Raw) > 0 {
            result[i].CompressedRaw = zstdEncode(entry.FieldsV1.Raw)
            result[i].UncompressedLen = len(entry.FieldsV1.Raw)
        }
    }
    return result
}
```

**Feature Gate**: `CompressManagedFieldsInCache` (Alpha -> Beta -> GA)

**Expected Impact**:
- 10-25% reduction in total apiserver memory
- 60-80% reduction in managedFields memory specifically
- <1% CPU overhead (zstd is very fast, especially for small payloads)
- Zero API compatibility risk (internal-only change)

**Risks**: Low - purely internal optimization, no API changes.

---

### Phase 2: Server-Side ManagedFields Exclusion (4-6 weeks)

**Goal**: Allow API clients to exclude managedFields from responses, saving both cache serialization memory and wire bandwidth.

**Approach**:
1. Add `showManagedFields=false` query parameter to GET/LIST/WATCH
2. When excluded, strip managedFields from serialized response
3. Cache the stripped serialization separately (most watchers will use it)

**API Change**:
```
# Existing behavior (backward compatible)
GET /api/v1/pods?watch=true
  -> Returns full objects including managedFields

# New behavior
GET /api/v1/pods?watch=true&showManagedFields=false
  -> Returns objects WITHOUT managedFields
  -> Default for watch events could eventually change
```

**Key Changes**:
```
staging/src/k8s.io/apiserver/pkg/endpoints/handlers/
  - get.go: Parse showManagedFields parameter
  - responsewriters/writers.go: Strip managedFields before serialization

staging/src/k8s.io/apiserver/pkg/storage/cacher/
  - cacher.go: Support filtering managedFields per watcher
  - caching_object.go: Cache both with/without managedFields serializations
```

**Implementation Details**:
```go
// In caching_object.go - add managed-fields-stripped serialization
func (o *cachingObject) CacheEncode(id runtime.Identifier, encode func(runtime.Object, io.Writer) error, w io.Writer) error {
    // Check if this is a "stripped" request
    if isStrippedID(id) {
        result := o.getSerializationResult(id)
        result.once.Do(func() {
            obj := o.GetObject()
            accessor, _ := meta.Accessor(obj)
            accessor.SetManagedFields(nil)  // Strip before serialize
            buffer := bytes.NewBuffer(nil)
            result.err = encode(obj, buffer)
            result.raw = buffer.Bytes()
        })
        // Return stripped serialization
    }
    // ... normal path
}
```

**Expected Impact**:
- 15-30% reduction in wire bandwidth for watches
- Up to 50% reduction in serialization cache when most watchers opt out
- Enables client-go informers to automatically opt out (most don't need managedFields)

**Risks**: Medium - API change requires careful design and backward compatibility testing.

---

### Phase 3: Watch Cache ManagedFields Separation (6-8 weeks)

**Goal**: Remove managedFields from the main watch cache objects entirely. Store them in a separate, lightweight sidecar structure.

**Approach**:
1. Strip managedFields before storing objects in the watch cache
2. Maintain a parallel `map[objectKey][]compressedManagedFieldsEntry` for reconstruction
3. Inject managedFields back only when needed (Apply operations, explicit requests)

**Architecture**:
```
┌─────────────────────────────────────────────────┐
│                Watch Cache                        │
│                                                   │
│  ┌──────────────────────┐  ┌──────────────────┐  │
│  │   Store (BTree)      │  │  ManagedFields   │  │
│  │   Objects WITHOUT    │  │  Sidecar Store   │  │
│  │   managedFields      │  │  (compressed)    │  │
│  │                      │  │  map[key][]cmfe  │  │
│  └──────────────────────┘  └──────────────────┘  │
│                                                   │
│  ┌──────────────────────┐                         │
│  │   Event Buffer       │                         │
│  │   Events WITHOUT     │                         │
│  │   managedFields      │                         │
│  └──────────────────────┘                         │
└─────────────────────────────────────────────────┘
```

**Key Changes**:
```
staging/src/k8s.io/apiserver/pkg/storage/cacher/
  - watch_cache.go: Strip managedFields on insert, store in sidecar
  - cacher.go: Reconstruct managedFields for Apply path
  - New file: managed_fields_store.go

staging/src/k8s.io/apiserver/pkg/endpoints/handlers/
  - patch.go: Read managedFields from sidecar for Apply
```

**Implementation Details**:
```go
// New sidecar store
type managedFieldsSidecar struct {
    mu     sync.RWMutex
    fields map[string][]compressedManagedFieldsEntry
}

func (s *managedFieldsSidecar) Store(key string, mf []metav1.ManagedFieldsEntry) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.fields[key] = compressManagedFields(mf)
}

func (s *managedFieldsSidecar) Load(key string) ([]metav1.ManagedFieldsEntry, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if cmf, ok := s.fields[key]; ok {
        return decompressManagedFields(cmf), true
    }
    return nil, false
}

// Modified watch cache processEvent
func (w *watchCache) processEvent(event watch.Event) {
    key, _ := w.keyFunc(event.Object)

    // Extract and store managedFields separately
    accessor, _ := meta.Accessor(event.Object)
    mf := accessor.GetManagedFields()
    if len(mf) > 0 {
        w.managedFieldsSidecar.Store(key, mf)
        accessor.SetManagedFields(nil)  // Strip from cached object
    }

    // Store stripped object in cache
    elem := &storeElement{Key: key, Object: event.Object, ...}
    w.store.Update(elem)
}
```

**Expected Impact**:
- 20-40% reduction in total apiserver memory
- ManagedFields stored compressed in sidecar (~20% of original size)
- Watch events sent without managedFields by default (massive bandwidth savings)
- Apply operations still work (reconstruct from sidecar)

**Risks**: High - significant architectural change, must ensure Apply path correctness.

---

### Phase 4: Binary FieldsV1 Encoding (4-6 weeks, can parallel Phase 3)

**Goal**: Replace the verbose JSON encoding of FieldsV1 with a compact binary format.

**Approach**:
1. Define a new `FieldsV2` binary encoding format
2. Add `FieldsType: "FieldsV2"` support
3. Migrate gradually with fallback to FieldsV1

**Encoding Design**:
```
FieldsV2 Binary Format:
  Header: [version:1byte]
  Nodes:  [type:1byte][name_len:varint][name:bytes][child_count:varint][children:...]

  Types:
    0x01 = Named field (f:)
    0x02 = Map key (k:)
    0x03 = Value (v:)
    0x04 = Index (i:)
    0x05 = Leaf (.)

  Example:
    JSON:  {"f:spec":{"f:replicas":{}}} = 31 bytes
    Binary: [01 04 spec 01 01 08 replicas 05] = 15 bytes
```

**Expected Impact**:
- 40-60% reduction in FieldsV1/V2 raw data size
- Faster encode/decode (no JSON parsing overhead)
- Compounds with compression for even greater savings

---

### Phase 5: FieldsV1 Deduplication Pool (2-3 weeks, can parallel Phase 4)

**Goal**: Deduplicate identical FieldsV1 data across objects.

**Approach**:
1. Create an interning pool for FieldsV1/compressed data
2. Hash FieldsV1 content and share references for identical entries
3. Reference-count for GC

**Expected Impact**:
- 5-15% additional memory savings
- Highest impact for homogeneous workloads (many Pods from same Deployment)
- Nearly free for heterogeneous workloads

---

## Timeline

```
Month 1-2:     Phase 1 (Compression) - Alpha
Month 2-3:     Phase 2 (Server-side exclusion) - Alpha
Month 3-5:     Phase 3 (Cache separation) - Alpha
Month 4-6:     Phase 4 (Binary encoding) - Alpha
Month 5-6:     Phase 5 (Deduplication) - Alpha
Month 6-8:     Phase 1-2 Beta, Phase 3 Alpha testing
Month 8-10:    Phase 1-2 GA, Phase 3 Beta
Month 10-12:   Phase 3 GA, Phase 4-5 Beta
```

## Success Metrics

| Metric | Current | After Phase 1 | After Phase 3 | Target |
|--------|---------|---------------|---------------|--------|
| Apiserver RSS (100K objects) | ~8 GB | ~6.5 GB | ~5 GB | <5 GB |
| managedFields memory | ~2 GB | ~600 MB | ~200 MB | <300 MB |
| Watch event size (avg) | 5 KB | 5 KB | 3 KB | <3 KB |
| Apply latency (p99) | 50ms | 52ms | 55ms | <60ms |

## KEP Requirements

Each phase should have a KEP:
1. **KEP: Compress ManagedFields in Watch Cache** (Phase 1)
   - sig-api-machinery
   - No API changes, feature-gated
   - Benchmark data from local reproduction

2. **KEP: Server-Side ManagedFields Filtering** (Phase 2)
   - sig-api-machinery
   - New query parameter
   - client-go changes for informers to opt out

3. **KEP: Watch Cache ManagedFields Separation** (Phase 3)
   - sig-api-machinery
   - Architectural change
   - Extensive testing required
   - Companion to Phase 2

4. **KEP: FieldsV2 Binary Encoding** (Phase 4)
   - sig-api-machinery
   - New FieldsType, backward compatible
   - Migration strategy required

## Open Questions

1. **Should Phase 2 change the default for watches?**
   - Pro: Immediate memory savings for all watchers
   - Con: Breaking change for clients that rely on managedFields in watches
   - Recommendation: Default to `showManagedFields=true` initially, change default in a future version

2. **Should Apply operations read managedFields from etcd directly?**
   - Currently, Apply reads the full object from the watch cache
   - If we strip managedFields from cache, Apply needs a way to get them
   - Options: sidecar store (Phase 3), direct etcd read, or reconstruct from events

3. **How to handle the FieldsV1 -> FieldsV2 migration?**
   - Need to support both formats during transition
   - etcd objects can be migrated via storage migration
   - Wire format must be negotiated

4. **What about CRDs with very large managedFields?**
   - CRDs with 100+ fields can have very large FieldsV1
   - Should there be a per-object managedFields size limit?
   - Currently, the only limit is the overall 1.5 MB etcd object size

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Compression CPU overhead | Benchmark with zstd/snappy; use fastest mode |
| Apply path correctness | Comprehensive integration tests |
| Backward compatibility | Feature gates, gradual rollout |
| Race conditions in sidecar | Careful locking, same consistency guarantees |
| etcd migration for FieldsV2 | Dual-format support, lazy migration |
| Client breakage (Phase 2) | Opt-in initially, clear deprecation timeline |

## Conclusion

The SSA managedFields memory bottleneck is a solvable problem with significant impact. The proposed multi-phase approach allows incremental deployment with increasing benefits:
- Phase 1 alone provides 10-25% memory savings with minimal risk
- Phases 1-3 combined provide 30-50% memory savings
- All phases together could reduce managedFields memory by 80-90%

The most critical insight is that **managedFields are needed only for write operations (Apply/Update) but are stored and transmitted for all operations**. By separating the storage and transmission paths, we can dramatically reduce memory usage without compromising SSA functionality.
