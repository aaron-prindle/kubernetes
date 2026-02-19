# ManagedFields Deep Dive - Data Structures and Memory Impact

## Data Structure Hierarchy

```
ObjectMeta
  └── ManagedFields []ManagedFieldsEntry    (protobuf field 17, repeated)
        ├── Manager string                   (~20-50 bytes)
        ├── Operation ManagedFieldsOperationType ("Apply" or "Update")
        ├── APIVersion string                (~10-20 bytes, e.g. "apps/v1")
        ├── Time *metav1.Time                (24 bytes - 8 ptr + 16 struct)
        ├── FieldsType string                ("FieldsV1")
        ├── FieldsV1 *FieldsV1              (pointer to raw bytes)
        │     └── Raw []byte                 (**VARIABLE: 100 bytes to 200+ KB**)
        └── Subresource string               (0-10 bytes)
```

## Source Files

| File | Purpose |
|------|---------|
| `staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go:1340-1402` | Type definitions |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go` | Encode/decode logic |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fields.go` | FieldsV1 <-> Set conversion |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fieldmanager.go` | Core field manager |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go` | Manager count limiter |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/fieldpath/set.go` | Set data structure |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/fieldpath/serialize.go` | JSON serialization |

## FieldsV1 Internal Representation

### Storage Format
`FieldsV1.Raw` is a `[]byte` containing a JSON-encoded trie. The format uses a hierarchical structure where:

```go
// types.go:1399
type FieldsV1 struct {
    Raw []byte `json:"-" protobuf:"bytes,1,opt,name=Raw"`
}
```

### Trie Node Types
- `"f:<name>"` -> named struct field
- `"k:{\"key\":\"value\"}"` -> list item by key (associative list)
- `"v:<value>"` -> list item by value (set list)
- `"i:<index>"` -> list item by index (atomic list)
- `"."` -> the field itself (leaf marker)

### Example: A Simple Deployment's FieldsV1

For a manager that controls `spec.replicas` and `spec.template.spec.containers[name=web].image`:

```json
{
  "f:spec": {
    ".": {},
    "f:replicas": {},
    "f:template": {
      ".": {},
      "f:spec": {
        ".": {},
        "f:containers": {
          "k:{\"name\":\"web\"}": {
            ".": {},
            "f:image": {}
          }
        }
      }
    }
  }
}
```

This is **203 bytes** of raw JSON for just 2 actual fields being tracked.

### Example: A Complex Deployment's FieldsV1

For a manager controlling a full Deployment spec (replicas, selector, template with 3 containers,
volumes, affinity, tolerations, env vars, etc.), the FieldsV1 can easily reach **5-20 KB**.

## In-Memory Representation During Processing

When the FieldManager processes an Apply request, `FieldsV1.Raw` gets deserialized into a `fieldpath.Set`:

```go
// fields.go:38
func FieldsToSet(f metav1.FieldsV1) (s fieldpath.Set, err error) {
    err = s.FromJSON(bytes.NewReader(f.Raw))
    return s, err
}
```

The `fieldpath.Set` is a tree structure:
```go
// fieldpath/set.go:30
type Set struct {
    Members  PathElementSet  // Direct field members
    Children SetNodeMap      // Child sets (nested fields)
}
```

**Memory cost of fieldpath.Set:**
- Each PathElement: ~100-200 bytes (includes string allocation for field name)
- Each SetNode: ~48 bytes (PathElement + pointer to child Set)
- For a Pod with 100 tracked fields: ~20-40 KB in-memory Set representation
- This is **allocated and discarded on every Apply/Update operation**

## Encode/Decode Cycle

### Decoding (API request -> internal)
```go
// managedfields.go:98
func DecodeManagedFields(encodedManagedFields []metav1.ManagedFieldsEntry) (ManagedInterface, error) {
    managed := managedStruct{}
    managed.fields = make(fieldpath.ManagedFields, len(encodedManagedFields))
    managed.times = make(map[string]*metav1.Time, len(encodedManagedFields))

    for i, entry := range encodedManagedFields {
        // 1. Build manager identifier string (JSON marshal of metadata)
        manager, err := BuildManagerIdentifier(&entry)

        // 2. Decode FieldsV1 -> fieldpath.Set (JSON parse + tree construction)
        managed.fields[manager], err = decodeVersionedSet(&entry)

        // 3. Store timestamp
        managed.times[manager] = entry.Time
    }
    return &managed, nil
}
```

### Encoding (internal -> API response)
```go
// managedfields.go:175
func encodeManagedFields(managed ManagedInterface) ([]metav1.ManagedFieldsEntry, error) {
    for manager := range managed.Fields() {
        versionedSet := managed.Fields()[manager]

        // 1. Unmarshal manager identifier -> ManagedFieldsEntry metadata
        json.Unmarshal([]byte(manager), encodedVersionedSet)

        // 2. Encode fieldpath.Set -> FieldsV1 (tree -> JSON)
        fields, err := SetToFields(*versionedSet.Set())
        encodedVersionedSet.FieldsV1 = &fields
    }
    return sortEncodedManagedFields(encodedManagedFields)
}
```

### Cost of Each Cycle
For an object with 5 managers, each managing ~50 fields:
- **Decode**: Parse ~5KB JSON -> build 5 Sets with ~250 PathElements -> ~50-100 KB allocations
- **Encode**: Serialize 5 Sets back to ~5KB JSON -> ~50-100 KB intermediate allocations
- **This happens on EVERY Apply/Update that touches the object**

## Manager Count Limits

### Update Managers (capmanagers.go)
```go
const DefaultMaxUpdateManagers int = 10
```
- Update managers (non-Apply) are capped at 10
- When exceeded, oldest entries are merged into versioned buckets named "ancient-changes"
- Merging uses set Union: `merged = vs.Set().Union(existing.Set())`

### Apply Managers
- **No limit on Apply managers!**
- Each distinct Apply manager adds a new ManagedFieldsEntry
- In theory, hundreds of different Apply managers could exist on a single object

### Typical Entry Counts
| Scenario | Update Entries | Apply Entries | Total |
|----------|---------------|---------------|-------|
| kubectl-only | 1 | 0 | 1 |
| GitOps (Flux/ArgoCD) | 2-3 | 1 | 3-4 |
| Controllers + operators | 5-8 | 1-3 | 6-11 |
| Complex multi-team | 10 (capped) | 2-5 | 12-15 |

## Memory Size Estimates

### Per-Entry Overhead
| Component | Size |
|-----------|------|
| ManagedFieldsEntry struct | ~48 bytes |
| Manager string | 20-50 bytes |
| Operation string | 5-6 bytes |
| APIVersion string | 10-20 bytes |
| Time pointer + value | 24 bytes |
| FieldsType string | 8 bytes |
| Subresource string | 0-10 bytes |
| **Entry overhead (excl. FieldsV1)** | **~120-170 bytes** |

### FieldsV1 Sizes by Object Type
| Object Type | Typical FieldsV1 Size | Notes |
|-------------|----------------------|-------|
| ConfigMap (small) | 200-500 bytes | Few keys |
| ConfigMap (large, 1000 keys) | 20-100 KB | Each key tracked |
| Pod (simple) | 500-2000 bytes | Few containers |
| Pod (complex) | 5-20 KB | Many containers, volumes, env vars |
| Deployment | 1-5 KB | Includes template spec |
| StatefulSet | 2-10 KB | VolumeClaimTemplates |
| CRD (complex) | 5-50 KB | Schema-dependent |
| Node | 2-10 KB | Status has many fields |

### Total managedFields Per Object
| Scenario | Entries | Total Size | % of 1.5MB limit |
|----------|---------|------------|-------------------|
| Simple Pod, 1 manager | 1 | 300-700 bytes | <0.05% |
| Deployment, 3 managers | 3 | 3-15 KB | 0.2-1% |
| Complex object, capped | 11 | 10-50 KB | 0.7-3% |
| Large ConfigMap, capped | 11 | 50-250 KB | 3-17% |
| Node object, many managers | 8 | 20-80 KB | 1-5% |

### At Cluster Scale
For a cluster with 100,000 objects (pods, configmaps, services, etc.):
- **Average managedFields per object**: ~5-10 KB
- **Total managedFields in watch cache**: 500 MB - 1 GB
- **With serialization caching** (3 formats): potentially 1.5 - 3 GB
- **This is JUST the managedFields data, not the actual object data**

## The Key Problem

ManagedFields essentially stores a **structural mirror of the object** for each manager. If an object has 100 fields and 5 managers each managing 50 fields, the managedFields stores 250 field paths - which is 2.5x the structural complexity of the object itself, but as raw bytes (FieldsV1.Raw).

**The managedFields data often exceeds the size of the actual object data it describes.**

At scale (>10,000 objects), this becomes the dominant factor in apiserver memory consumption because:
1. Every object in the watch cache carries full managedFields
2. Watch events include full managedFields
3. Serialization caches include managedFields in each format
4. Most clients (watchers, informers) never use managedFields but still receive them
