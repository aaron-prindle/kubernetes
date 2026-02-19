# Key Code Paths in the SSA System

## Overview
This document annotates the critical code paths through the Server Side Apply system, focusing on where managedFields data is created, stored, transformed, and transmitted. Each section identifies the file, line numbers, and the memory implications.

---

## 1. Apply Request Handler

### Entry Point: PatchHandler
**File**: `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go`

```
Client PATCH request (Content-Type: application/apply-patch+yaml)
    |
    v
PatchResource() (patch.go:~100)
    |-- Determines patch type: ApplyYAMLPatchType or ApplyCBORPatchType
    |-- Creates applyPatcher (line 474-486)
    |
    v
patcher.patchResource() (patch.go:~580)
    |-- Reads current object from storage
    |-- Calls p.mechanism.applyPatchToCurrentObject()
    |
    v
applyPatcher.applyPatchToCurrentObject() (patch.go:500-530)
    |-- Unmarshals patch YAML into unstructured.Unstructured
    |-- Calls p.fieldManager.Apply(obj, patchObj, fieldManager, force)
    |
    v
FieldManager.Apply() (internal/fieldmanager.go:183-209)
    |-- Gets accessor for live object
    |-- DECODES managedFields from live object:
    |   managed, err := DecodeManagedFields(accessor.GetManagedFields())
    |   ^^ MEMORY: Parses all FieldsV1 JSON -> fieldpath.Set trees
    |
    |-- Calls internal Apply chain
    |-- ENCODES managedFields back:
    |   EncodeObjectManagedFields(object, managed)
    |   ^^ MEMORY: Serializes all Sets back to JSON
    |
    v
Internal Manager Chain (Apply path)
```

### Memory Hotspots in Apply Path
1. `DecodeManagedFields()` - Parses N manager entries' FieldsV1 JSON
2. `configObject.ToFieldSet()` - Walks entire applied config
3. Set operations (Union, Intersection, Difference) for conflict detection
4. `SetToFields()` - Serializes updated Sets back to JSON
5. `EncodeObjectManagedFields()` - Attaches to object

---

## 2. ManagedFields Decode Path

### DecodeManagedFields
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go:98-131`

```go
func DecodeManagedFields(encodedManagedFields []metav1.ManagedFieldsEntry) (ManagedInterface, error) {
    managed := managedStruct{}
    managed.fields = make(fieldpath.ManagedFields, len(encodedManagedFields))
    // ^^ ALLOC: map[string]VersionedSet - one entry per manager

    managed.times = make(map[string]*metav1.Time, len(encodedManagedFields))

    for i, encodedVersionedSet := range encodedManagedFields {
        // Build manager identifier (JSON marshal of metadata fields)
        manager, err := BuildManagerIdentifier(&encodedVersionedSet)
        // ^^ ALLOC: JSON marshaling creates string (~100-200 bytes)

        // Decode FieldsV1 -> fieldpath.Set
        managed.fields[manager], err = decodeVersionedSet(&encodedVersionedSet)
        // ^^ ALLOC: Calls FieldsToSet -> Set.FromJSON
        //    Creates entire tree of PathElements and SetNodes

        managed.times[manager] = encodedVersionedSet.Time
    }
    return &managed, nil
}
```

### FieldsToSet
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fields.go:38-41`

```go
func FieldsToSet(f metav1.FieldsV1) (s fieldpath.Set, err error) {
    err = s.FromJSON(bytes.NewReader(f.Raw))
    // ^^ ALLOC: Creates new bytes.Reader wrapping Raw
    //    Parses JSON trie into fieldpath.Set tree
    //    Each node: PathElement (string allocs) + SetNode
    return s, err
}
```

### Set.FromJSON
**File**: `vendor/sigs.k8s.io/structured-merge-diff/v6/fieldpath/serialize.go`

This recursively parses the JSON trie:
```
For each key in JSON object:
    1. Parse key prefix ("f:", "k:", "v:", "i:")
    2. Create PathElement (string allocation for field name)
    3. If value is non-empty object: recursively parse children
    4. Add to Set.Members or Set.Children
```

**Memory cost**: ~100-200 bytes per tracked field (PathElement + string + SetNode pointer)

---

## 3. ManagedFields Encode Path

### EncodeObjectManagedFields
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go:81-94`

```go
func EncodeObjectManagedFields(obj runtime.Object, managed ManagedInterface) error {
    encodedManagedFields, err := encodeManagedFields(managed)
    // ^^ ALLOC: Creates []ManagedFieldsEntry slice
    //    For each manager: JSON-encodes Set -> FieldsV1.Raw bytes

    accessor.SetManagedFields(encodedManagedFields)
    // ^^ Sets on the object - this data will be stored in etcd and cache
    return nil
}
```

### encodeManagedFields
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go:175-192`

```go
func encodeManagedFields(managed ManagedInterface) ([]metav1.ManagedFieldsEntry, error) {
    for manager := range managed.Fields() {
        v, err := encodeManagerVersionedSet(manager, versionedSet)
        // ^^ ALLOC: Per manager:
        //    1. JSON unmarshal manager string -> ManagedFieldsEntry
        //    2. SetToFields(*versionedSet.Set()) -> FieldsV1.Raw bytes
        //    3. Each FieldsV1.Raw: complete JSON trie serialization
    }
    return sortEncodedManagedFields(encodedManagedFields)
}
```

### SetToFields
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fields.go:44-47`

```go
func SetToFields(s fieldpath.Set) (f metav1.FieldsV1, err error) {
    f.Raw, err = s.ToJSON()
    // ^^ ALLOC: Serializes entire Set tree to JSON bytes
    //    Uses jsoniter streaming writer
    //    Allocates buffer for JSON output
    return f, err
}
```

---

## 4. Watch Cache Storage Path

### Object Enters Watch Cache
**File**: `staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go`

```go
func (w *watchCache) processEvent(event watch.Event) {
    // Object from etcd includes FULL managedFields
    key, _ := w.keyFunc(event.Object)
    // ^^ event.Object has ManagedFields set

    elem := &storeElement{
        Key:    key,
        Object: event.Object,  // STORES full object with managedFields
        Labels: labels,
        Fields: fields,
    }

    wcEvent := &watchCacheEvent{
        Type:            event.Type,
        Object:          event.Object,      // FULL object with managedFields
        PrevObject:      prevObject,         // FULL previous object with managedFields
        Key:             key,
        ResourceVersion: rv,
    }

    // Store in BOTH locations:
    w.store.Update(elem)                     // Current state store
    w.cache[w.endIndex%w.capacity] = wcEvent // Cyclic event buffer

    // Optional: snapshot for LIST
    if w.snapshots != nil {
        w.snapshots.Add(w.resourceVersion, orderedLister)
    }
}
```

**Memory**: Object stored 2x (store + event buffer), both with full managedFields.

### Object Wrapped for Dispatch
**File**: `staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go:922-951`

```go
func setCachingObjects(event *watchCacheEvent, versioner storage.Versioner) {
    switch event.Type {
    case watch.Added, watch.Modified:
        if object, err := newCachingObject(event.Object); err == nil {
            event.Object = object
            // ^^ Wraps full object (with managedFields) in cachingObject
            // NOTE: does NOT deep copy yet (lazy copy)
        }
    }
}
```

### Object Serialized for Watchers
**File**: `staging/src/k8s.io/apiserver/pkg/storage/cacher/caching_object.go:136-157`

```go
func (o *cachingObject) CacheEncode(id runtime.Identifier, encode func(runtime.Object, io.Writer) error, w io.Writer) error {
    result := o.getSerializationResult(id)
    result.once.Do(func() {
        buffer := bytes.NewBuffer(nil)
        result.err = encode(o.GetObject(), buffer)
        // ^^ GetObject() deep-copies the FULL object including managedFields
        // ^^ encode() serializes FULL object including managedFields
        result.raw = buffer.Bytes()
        // ^^ ALLOC: Full serialized bytes stored in cache
        //    JSON: includes full managedFields JSON
        //    Protobuf: includes full managedFields protobuf
    })
    // Write cached bytes to all watchers
    _, err := w.Write(result.raw)
    return err
}
```

**Memory**: Serialized bytes cached per format. Each format includes full managedFields.

---

## 5. ToFieldSet Path (Per Apply Operation)

### configObject.ToFieldSet()
**File**: `vendor/sigs.k8s.io/structured-merge-diff/v6/typed/tofieldset.go:31-149`

```go
func (tv TypedValue) ToFieldSet() (*fieldpath.Set, error) {
    walker := tPool.Get().(*toFieldSetWalker)
    // Uses sync.Pool to reduce allocations

    v.set = &fieldpath.Set{}
    v.allocator = value.NewFreelistAllocator()
    // ^^ ALLOC: Creates new Set and allocator

    return v.set, v.toFieldSet()
    // ^^ Recursively walks ENTIRE object tree
}
```

### Recursive Walk
```go
func (v *toFieldSetWalker) visitListItems(t *schema.List, list value.List) {
    for i := 0; i < list.Length(); i++ {
        // For EACH list item:
        pe, _ := listItemToPathElement(v.allocator, v.schema, t, child)
        // ^^ ALLOC: Creates PathElement
        //    For associative lists: includes full key-value JSON
        //    Example: PathElement{Key: map[string]interface{}{"name": "container-1"}}

        v2 := v.prepareDescent(pe, t.ElementType)
        errs = append(errs, v2.toFieldSet()...)
        // ^^ Recursive descent into list item
    }
}
```

**Memory per Apply**: O(total_fields_in_config) PathElement allocations
- Simple config (10 fields): ~1-2 KB
- Complex config (100 fields): ~10-20 KB
- Very complex config (1000 fields): ~100-200 KB

---

## 6. CapManagers Path

### Manager Count Limiting
**File**: `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go:65-133`

```go
func (f *capManagersManager) capUpdateManagers(managed Managed) (newManaged Managed, err error) {
    // Count non-Apply managers
    updaters := []string{}
    for manager, fields := range managed.Fields() {
        if !fields.Applied() {
            updaters = append(updaters, manager)
        }
    }

    if len(updaters) <= f.maxUpdateManagers {
        return managed, nil  // Under cap, no action
    }

    // Sort oldest first
    sort.Slice(updaters, ...)

    // Merge oldest into versioned buckets
    for i := range updaters {
        // ALLOC: Set Union operation
        managed.Fields()[bucket] = fieldpath.NewVersionedSet(
            vs.Set().Union(managed.Fields()[bucket].Set()),
            // ^^ Creates new Set from union of two Sets
            vs.APIVersion(), vs.Applied())
    }
}
```

**Note**: Apply managers are NOT capped. This means objects managed by many different Apply managers can accumulate unbounded managedFields entries.

---

## 7. etcd Storage Path

### Object Written to etcd
**File**: `staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go`

```go
func (s *store) GuaranteedUpdate(...) {
    // Object is serialized with FULL managedFields
    data, err := runtime.Encode(s.codec, obj)
    // ^^ ALLOC: Full serialization including managedFields

    // Stored in etcd
    txnResp, err := s.client.KV.Txn(ctx).If(...).Then(
        clientv3.OpPut(key, string(data)),
    ).Commit()
}
```

### Size Overflow Handling
**File**: `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go:704-728`

```go
// When object exceeds etcd size limit (~1.5MB):
if isTooLargeError(err) && p.patchType != types.ApplyYAMLPatchType {
    // Strip managedFields and retry
    func(_ context.Context, obj, _ runtime.Object) (runtime.Object, error) {
        accessor, _ := meta.Accessor(obj)
        accessor.SetManagedFields(nil)  // EMERGENCY strip
        return obj, nil
    }
}
```

**Critical**: This fallback ONLY works for non-Apply patches. Apply patches that exceed the size limit will FAIL, meaning SSA is more vulnerable to size limits.

---

## Summary: Memory Allocation Points

| Location | What's Allocated | Size | Frequency |
|----------|-----------------|------|-----------|
| `DecodeManagedFields` | fieldpath.Set per manager | 1-50 KB each | Every Apply/Update |
| `FieldsToSet` (FromJSON) | PathElement tree | 1-50 KB | Per manager per operation |
| `SetToFields` (ToJSON) | FieldsV1.Raw bytes | 0.5-20 KB | Per manager per operation |
| `ToFieldSet` | PathElement tree from config | 1-200 KB | Every Apply |
| `processEvent` (store) | Full object in store | object size | Every change |
| `processEvent` (buffer) | Full object in event buffer | object size | Every change |
| `newCachingObject` | Wrapper (lazy copy) | ~100 bytes | Every dispatch |
| `CacheEncode` | Serialized bytes per format | object size | Per format per dispatch |
| `GetObject` | Deep copy of full object | object size | Per access |
| `Set.Union/Intersection/Difference` | New Set from merge | 1-50 KB | Per Apply conflict check |

**Total per Apply operation**: ~100-500 KB transient allocations (freed after request)
**Total in watch cache per object**: object_size * 2-4 (including managedFields portion)
