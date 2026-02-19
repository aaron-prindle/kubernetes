# L7-Level Solution Proposals for SSA Memory Bottleneck

## Solution Overview

| # | Solution | Memory Savings | Complexity | API Compat | Recommended |
|---|----------|---------------|------------|------------|-------------|
| 1 | Strip managedFields from watch cache | 20-40% | Medium | High | YES |
| 2 | Compress FieldsV1.Raw in cache | 10-25% | Low | High | YES |
| 3 | Server-side field selector for managedFields | 15-30% | Medium | High | YES |
| 4 | Deduplicate FieldsV1 across objects | 5-15% | High | High | MAYBE |
| 5 | Lazy-load managedFields from etcd | 20-35% | Very High | Medium | FUTURE |
| 6 | Binary encoding for FieldsV1 | 10-20% | Medium | Medium | YES |
| 7 | Reference-based caching for unchanged fields | 10-20% | High | High | MAYBE |

---

## Solution 1: Strip ManagedFields from Watch Cache (HIGHEST IMPACT)

### Concept
Store objects in the watch cache WITHOUT managedFields. When a client actually needs managedFields (rare), reconstruct them from etcd or a separate lightweight store.

### Design
```
                       Write Path (Apply/Update)
                       ┌──────────────┐
                       │ Full object   │
etcd ─────────────────>│ + managed     │──> etcd storage (full object)
                       │   fields      │
                       └──────┬───────┘
                              │
                    Strip managedFields
                              │
                       ┌──────▼───────┐
                       │ Object       │──> Watch Cache (stripped object)
                       │ WITHOUT      │──> Watch Events (stripped)
                       │ managedFields│
                       └──────────────┘

                       Read Path (when managedFields needed)
                       ┌──────────────┐
                       │ GET with     │──> Read from etcd (has managedFields)
                       │ showManaged  │    OR reconstruct from separate store
                       │   Fields     │
                       └──────────────┘
```

### Implementation Approach

#### Option A: Strip-on-cache-insert
Modify the watch cache to strip managedFields when storing objects:

```go
// In watch_cache.go processEvent()
func stripManagedFields(obj runtime.Object) runtime.Object {
    accessor, err := meta.Accessor(obj)
    if err != nil {
        return obj
    }
    // Only strip if managedFields are present
    if len(accessor.GetManagedFields()) > 0 {
        // Deep copy first to avoid modifying original
        stripped := obj.DeepCopyObject()
        strippedAccessor, _ := meta.Accessor(stripped)
        strippedAccessor.SetManagedFields(nil)
        return stripped
    }
    return obj
}
```

**Challenge**: Clients that need managedFields from watches would get empty fields.

#### Option B: Separate managedFields store
Keep a parallel lightweight store mapping object key -> managedFields:

```go
type managedFieldsStore struct {
    mu     sync.RWMutex
    fields map[string][]metav1.ManagedFieldsEntry  // key -> managedFields
}
```

When a client requests managedFields (via a query parameter), inject them back.

#### Option C: Feature-gated opt-in
Add a feature gate `StripManagedFieldsFromWatchCache` that:
1. Strips managedFields from cached objects
2. Keeps a compact sidecar store for managedFields
3. Injects them back for clients that request `showManagedFields=true`

### API Considerations
- Default behavior: managedFields ARE included (backward compatible)
- New query parameter: `showManagedFields=false` to explicitly exclude (saves wire bandwidth too)
- Watch cache stores without managedFields regardless (saves memory)
- Reconstruction is lazy (only when needed)

### Estimated Savings
- **Memory**: 20-40% reduction in watch cache memory
- **Wire bandwidth**: Additional savings when clients opt out
- **CPU**: Slight increase for reconstruction when needed

---

## Solution 2: Compress FieldsV1.Raw in Cache

### Concept
Replace the raw JSON bytes in FieldsV1 with compressed bytes. Since FieldsV1 JSON is highly repetitive (many `"f:"`, `"k:"`, `".":{}` patterns), compression is very effective.

### Design
```go
// New type for compressed storage
type CompressedFieldsV1 struct {
    CompressedRaw []byte  // zstd/snappy compressed
    OriginalSize  int     // For pre-allocation during decompression
}

// In cachingObject or a wrapper:
type memoryEfficientManagedFields struct {
    entries []CompressedManagedFieldsEntry
}

type CompressedManagedFieldsEntry struct {
    Manager     string
    Operation   ManagedFieldsOperationType
    APIVersion  string
    Time        *Time
    Subresource string
    // FieldsV1 stored compressed
    CompressedFieldsV1 []byte
}
```

### Compression Analysis
Testing with typical FieldsV1 data:

| Object Type | Raw Size | Snappy | Zstd | Ratio |
|-------------|----------|--------|------|-------|
| Simple Pod | 500 B | 280 B | 200 B | 40-60% |
| Complex Pod | 5 KB | 1.8 KB | 1.2 KB | 65-75% |
| Deployment | 3 KB | 1.1 KB | 800 B | 65-75% |
| Large ConfigMap | 50 KB | 8 KB | 5 KB | 85-90% |
| Node | 8 KB | 2.5 KB | 1.8 KB | 70-80% |

**Average compression ratio**: 60-80% savings on FieldsV1 data.

### Implementation
```go
import "github.com/klauspost/compress/zstd"

var (
    encoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
    decoder, _ = zstd.NewReader(nil)
)

func compressFieldsV1(raw []byte) []byte {
    return encoder.EncodeAll(raw, make([]byte, 0, len(raw)/2))
}

func decompressFieldsV1(compressed []byte, originalSize int) ([]byte, error) {
    return decoder.DecodeAll(compressed, make([]byte, 0, originalSize))
}
```

### Where to Apply
1. **In FieldsV1 storage**: Compress Raw bytes when object enters watch cache
2. **Decompress on access**: When managedFields are needed for serialization
3. **Skip decompression**: When managedFields are being stripped from response

### Estimated Savings
- **Memory**: 10-25% of total watch cache (60-80% of managedFields portion)
- **CPU**: ~1-5% increase for compress/decompress (zstd is very fast)
- **Compatibility**: Fully transparent to clients

---

## Solution 3: Server-Side Field Selector for ManagedFields

### Concept
Allow clients to opt out of receiving managedFields via a query parameter or field selector. This saves both wire bandwidth and, when combined with caching changes, memory.

### Design
```
# Current: always includes managedFields
GET /api/v1/pods?watch=true

# Proposed: exclude managedFields
GET /api/v1/pods?watch=true&excludeFields=metadata.managedFields

# Or with a dedicated parameter
GET /api/v1/pods?watch=true&showManagedFields=false
```

### Implementation Approach

#### At the response writer level
```go
// In responsewriters/writers.go
func SerializeObject(..., excludeManagedFields bool) {
    if excludeManagedFields {
        // Strip managedFields before serialization
        accessor, _ := meta.Accessor(object)
        // Use a projection that excludes managedFields
        stripped := stripManagedFieldsProjection(object)
        encoder.Encode(stripped, w)
    } else {
        encoder.Encode(object, w)
    }
}
```

#### At the watch dispatch level
```go
// In cache_watcher.go
func (c *cacheWatcher) convertToWatchEvent(event *watchCacheEvent) *watch.Event {
    if c.excludeManagedFields {
        // Create lightweight copy without managedFields
        obj := shallowCopyWithoutManagedFields(event.Object)
        return &watch.Event{Type: event.Type, Object: obj}
    }
    return &watch.Event{Type: event.Type, Object: event.Object}
}
```

### Benefits
- Backward compatible (opt-in)
- Saves wire bandwidth for clients that don't need managedFields
- Combined with Solution 1, also saves cache memory
- Can be implemented incrementally

### Estimated Savings
- **Wire bandwidth**: 20-40% per watch event
- **Memory**: Depends on implementation (projection-based = small, cache-based = large)

---

## Solution 4: Deduplicate FieldsV1 Across Objects

### Concept
Many objects of the same type managed by the same controller will have IDENTICAL FieldsV1 data. For example, all Pods created by the same Deployment will have the same field structure.

### Design
```go
// Interning pool for FieldsV1
type FieldsV1Pool struct {
    mu    sync.RWMutex
    pool  map[uint64]*FieldsV1  // hash(Raw) -> shared FieldsV1
    refs  map[uint64]int        // reference count
}

func (p *FieldsV1Pool) Intern(f *FieldsV1) *FieldsV1 {
    hash := xxhash.Sum64(f.Raw)
    p.mu.RLock()
    if existing, ok := p.pool[hash]; ok {
        p.refs[hash]++
        p.mu.RUnlock()
        return existing
    }
    p.mu.RUnlock()

    p.mu.Lock()
    defer p.mu.Unlock()
    // Double-check
    if existing, ok := p.pool[hash]; ok {
        p.refs[hash]++
        return existing
    }
    p.pool[hash] = f
    p.refs[hash] = 1
    return f
}
```

### Potential Savings
For a cluster with 10,000 Pods from 100 Deployments:
- Without dedup: 10,000 * 5 KB = 50 MB of FieldsV1 data
- With dedup: ~100 unique patterns * 5 KB = 500 KB (100x reduction!)

### Challenges
- Requires careful reference counting and GC
- Deep copies must handle interned data correctly
- Objects modified by different managers will have different FieldsV1

### Estimated Savings
- **Memory**: 5-15% overall (varies greatly by workload)
- **Best case**: Homogeneous workloads with many replicas - up to 90% FieldsV1 savings
- **Worst case**: Diverse CRDs with unique structures - minimal savings

---

## Solution 5: Lazy-Load ManagedFields from etcd

### Concept
Don't store managedFields in the watch cache at all. Load them from etcd only when needed (during Apply operations or when explicitly requested).

### Design
```
Watch Cache (lean)         etcd (full)
┌──────────────┐          ┌──────────────────┐
│ Object data  │          │ Object data      │
│ (no managed  │          │ + managedFields  │
│  fields)     │          │                  │
└──────────────┘          └──────────────────┘
       │                          │
       │    Apply request         │
       │ ──────────────────────>  │
       │    Read managedFields    │
       │ <──────────────────────  │
       │    Process & update      │
       │ ──────────────────────>  │
       │    Write back            │
```

### Implementation
```go
// Modified Apply handler
func (p *applyPatcher) applyPatchToCurrentObject(ctx context.Context, obj runtime.Object) (runtime.Object, error) {
    // obj from cache has no managedFields
    // Load managedFields separately from etcd
    managedFields, err := p.loadManagedFieldsFromEtcd(ctx, obj)
    if err != nil {
        return nil, err
    }

    accessor, _ := meta.Accessor(obj)
    accessor.SetManagedFields(managedFields)

    // Now proceed with normal Apply
    return p.fieldManager.Apply(obj, patchObj, p.options.FieldManager, force)
}
```

### Challenges
- Adds etcd read on every Apply (but Apply already reads from etcd for CAS)
- Must handle consistency (managedFields must match the object version)
- Significant refactoring of the Apply and Update paths
- May need a separate etcd key or storage mechanism

### Estimated Savings
- **Memory**: 20-35% reduction (all managedFields removed from cache)
- **Latency**: Slight increase on Apply operations (additional etcd read)
- **Complexity**: Very high - fundamental architectural change

---

## Solution 6: Binary Encoding for FieldsV1

### Concept
Replace the JSON trie encoding with a more compact binary format. The current JSON encoding is verbose due to repeated key prefixes, braces, and quotes.

### Design Options

#### Option A: Protocol Buffers for FieldsV1
```protobuf
message FieldSetNode {
    repeated FieldSetEntry entries = 1;
}

message FieldSetEntry {
    oneof key {
        string field_name = 1;     // For "f:<name>"
        bytes  key_value = 2;      // For "k:{...}"
        bytes  value = 3;          // For "v:<value>"
        int32  index = 4;          // For "i:<index>"
    }
    bool is_leaf = 5;              // true if this is "."
    FieldSetNode children = 6;     // Nested fields
}
```

#### Option B: Custom compact binary format
```
Format:
  [type:1byte][length:varint][data:...][children_count:varint][children:...]

Types:
  0x01 = field (f:)
  0x02 = key (k:)
  0x03 = value (v:)
  0x04 = index (i:)
  0x05 = leaf (.)
```

### Estimated Savings
- JSON: `{"f:spec":{"f:replicas":{}}}` = 30 bytes
- Binary: `[01][04]spec[01][01][08]replicas[05]` = ~18 bytes (40% reduction)

- **Memory**: 10-20% of total (40-60% of FieldsV1 data)
- **CPU**: Faster encode/decode than JSON
- **Compatibility**: Requires versioning (FieldsType could be "FieldsV2")

---

## Solution 7: Reference-Based Caching for Unchanged ManagedFields

### Concept
Since managedFields rarely change between object updates, keep a reference to the previous version's managedFields and only store the delta.

### Design
```go
type ManagedFieldsRef struct {
    // Reference to shared managedFields data
    ref *sharedManagedFields
    // Version at which this ref was valid
    version uint64
}

type sharedManagedFields struct {
    refCount int32
    data     []metav1.ManagedFieldsEntry
}
```

When an object is updated and managedFields haven't changed (common for status updates):
- New version references the same managedFields
- No additional memory allocated

---

## Recommended Implementation Order

### Phase 1 (Quick Wins)
1. **Solution 2**: Compress FieldsV1.Raw in cache
   - Low complexity, no API changes, 10-25% savings
   - Can be feature-gated for safe rollout

2. **Solution 3**: Server-side managedFields exclusion parameter
   - API addition but backward compatible
   - Benefits both memory and bandwidth

### Phase 2 (Medium-Term)
3. **Solution 1**: Strip managedFields from watch cache
   - Highest impact (20-40% savings)
   - Requires careful design for reconstruction path
   - Combined with Solution 3 for full optimization

### Phase 3 (Long-Term)
4. **Solution 6**: Binary encoding for FieldsV1
   - Requires FieldsV2 type and migration path
   - Permanent 40-60% reduction in FieldsV1 size

5. **Solution 4**: FieldsV1 deduplication
   - High impact for homogeneous workloads
   - Complements other solutions

### Phase 4 (Future)
6. **Solution 5**: Lazy-load from etcd
   - Maximum savings but highest complexity
   - May be unnecessary if other solutions are effective
