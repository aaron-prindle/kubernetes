# 05 — Deep Copy Safety Analysis

## The Problem

Interning means multiple objects share the same `[]byte` backing array for
`FieldsV1.Raw`. If any code path mutates those bytes in-place, it would corrupt
all objects sharing that slice. We need to verify that no code path mutates
`FieldsV1.Raw` after interning.

## Deep Copy Chain

When an object is deep-copied, the generated code at
`zz_generated.deepcopy.go:693-738` does:

```go
// ObjectMeta.DeepCopyInto (line 731-737):
if in.ManagedFields != nil {
    in, out := &in.ManagedFields, &out.ManagedFields
    *out = make([]ManagedFieldsEntry, len(*in))
    for i := range *in {
        (*in)[i].DeepCopyInto(&(*out)[i])
    }
}

// ManagedFieldsEntry.DeepCopyInto (line 658-670):
if in.FieldsV1 != nil {
    in, out := &in.FieldsV1, &out.FieldsV1
    *out = new(FieldsV1)
    (*in).DeepCopyInto(*out)
}

// FieldsV1.DeepCopyInto (line 357-365):
*out = *in
if in.Raw != nil {
    in, out := &in.Raw, &out.Raw
    *out = make([]byte, len(*in))     // NEW allocation
    copy(*out, *in)                    // COPIES bytes
}
```

**Key finding**: `FieldsV1.DeepCopyInto` always allocates a new `[]byte` and copies.
This means:
- Deep-copied objects get their own independent `FieldsV1.Raw`
- The interned copy in the cache is never shared with deep-copied consumers
- Mutations on deep-copied objects cannot corrupt the interned bytes

## Where Deep Copies Happen

### 1. cachingObject.GetObject() (caching_object.go:160-166)

```go
func (o *cachingObject) GetObject() runtime.Object {
    return o.object.DeepCopyObject().(metaRuntimeInterface)
}
```

Called during serialization (`CacheEncode`, line 144). The deep copy creates
independent bytes, so serialization operates on its own copy. Safe.

### 2. cache_watcher.go getMutableObject() (line 346-354)

```go
func getMutableObject(object runtime.Object) runtime.Object {
    if _, ok := object.(*cachingObject); ok {
        return object  // No deep copy — but cachingObject does lazy deep copy
    }
    return object.DeepCopyObject()  // Deep copy for non-cached objects
}
```

For objects wrapped in `cachingObject`: returns the wrapper, which deep-copies
on field access. For non-cached objects (like PrevObject in some cases): explicit
deep copy. Both paths produce independent `FieldsV1.Raw`. Safe.

### 3. Apply/Update path (fieldmanager.go:120-140)

The Apply path reads `FieldsV1.Raw` via `FieldsToSet()` (fields.go:38-41):
```go
func FieldsToSet(f metav1.FieldsV1) (s fieldpath.Set, err error) {
    err = s.FromJSON(bytes.NewReader(f.Raw))  // READ-ONLY access to Raw
    return s, err
}
```

`bytes.NewReader` only reads the bytes, never mutates them. After building the
fieldpath.Set, the Apply path works on the Set structure, not the Raw bytes.

When writing back, `SetToFields` (fields.go:44-47) creates NEW bytes:
```go
func SetToFields(s fieldpath.Set) (f metav1.FieldsV1, err error) {
    f.Raw, err = s.ToJSON()  // Creates brand-new []byte
    return f, err
}
```

The Apply path never mutates existing `FieldsV1.Raw` in-place. Safe.

### 4. Protobuf serialization (generated.pb.go:719)

```go
func (m *FieldsV1) MarshalToSizedBuffer(dAtA []byte) (int, error) {
    if m.Raw != nil {
        i -= len(m.Raw)
        copy(dAtA[i:], m.Raw)  // COPIES Raw into output buffer, does not mutate Raw
    }
}
```

Read-only access. Safe.

## Conclusion: Interning Is Safe With Current Code

All paths that consume `FieldsV1.Raw` either:
1. Deep-copy the object first (creating independent bytes), or
2. Read the bytes without mutation (`bytes.NewReader`, `copy` to output buffer)

No code path mutates `FieldsV1.Raw` in-place after the object is stored in
the watch cache.

## Defensive Measure

Despite the above analysis, we can add a safety check: make the interned bytes
read-only by using a wrapper that panics on write attempts. This is only useful
during testing/development:

```go
// Only for testing — verifies nobody mutates interned bytes
func (p *fieldsV1InternPool) InternReadOnly(raw []byte) []byte {
    interned := p.Intern(raw)
    if testing.Testing() {
        // Could snapshot a hash and verify periodically
    }
    return interned
}
```

In practice, Go doesn't support read-only byte slices, so the real defense is the
analysis above plus test coverage that verifies bytes remain unchanged.

## What If We Add Compression Later

If we later add FieldsV1 compression on top of interning:
- The interned bytes would be compressed bytes
- `FieldsToSet()` would need to decompress before parsing
- Deep copies would copy compressed bytes (smaller = faster copies)
- Serialization would decompress before encoding to wire format

This is compatible — compressed interned bytes are still just `[]byte` from the
pool's perspective. The pool doesn't care about the content, only the hash.
