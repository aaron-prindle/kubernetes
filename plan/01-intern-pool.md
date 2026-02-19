# 01 — The FieldsV1 Interning Pool

## New File

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/fieldsv1_intern_pool.go
```

## What It Does

A thread-safe pool that maps `FieldsV1.Raw` content to a single shared `[]byte`.
When an object enters the watch cache, each `FieldsV1.Raw` is looked up in the pool.
If an identical byte sequence already exists, the object's slice header is pointed at the
shared copy. If not, the bytes are stored as a new entry.

## Data Structure

```go
package cacher

import (
    "sync"

    "github.com/cespare/xxhash/v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/meta"
    "k8s.io/apimachinery/pkg/runtime"
)

// fieldsV1InternPool deduplicates FieldsV1.Raw byte slices across cached objects.
// Objects with identical managed field structures (common for replicas from the
// same controller) share a single backing byte slice instead of independent copies.
type fieldsV1InternPool struct {
    mu      sync.RWMutex
    entries map[uint64]*internEntry
}

type internEntry struct {
    raw      []byte // the canonical shared copy
    refCount int64  // number of objects referencing this entry
}

func newFieldsV1InternPool() *fieldsV1InternPool {
    return &fieldsV1InternPool{
        entries: make(map[uint64]*internEntry),
    }
}
```

## Core Operations

### Intern (called when object enters cache)

```go
// Intern returns a shared copy of raw if an identical byte sequence is already
// pooled, or stores raw as a new entry. The caller must call Release with the
// same content when the object leaves the cache.
func (p *fieldsV1InternPool) Intern(raw []byte) []byte {
    if len(raw) == 0 {
        return raw
    }
    hash := xxhash.Sum64(raw)

    // Fast path: read lock
    p.mu.RLock()
    if e, ok := p.entries[hash]; ok && bytesEqual(e.raw, raw) {
        e.refCount++ // atomic would be ideal but we're under lock anyway
        p.mu.RUnlock()
        return e.raw
    }
    p.mu.RUnlock()

    // Slow path: write lock
    p.mu.Lock()
    defer p.mu.Unlock()

    // Double-check after acquiring write lock
    if e, ok := p.entries[hash]; ok && bytesEqual(e.raw, raw) {
        e.refCount++
        return e.raw
    }

    // Store new entry — make our own copy so the caller's buffer can be reused
    owned := make([]byte, len(raw))
    copy(owned, raw)
    p.entries[hash] = &internEntry{raw: owned, refCount: 1}
    return owned
}
```

### Release (called when object leaves cache)

```go
// Release decrements the reference count for the given raw bytes.
// When the count reaches zero, the entry is removed from the pool.
func (p *fieldsV1InternPool) Release(raw []byte) {
    if len(raw) == 0 {
        return
    }
    hash := xxhash.Sum64(raw)

    p.mu.Lock()
    defer p.mu.Unlock()

    if e, ok := p.entries[hash]; ok {
        e.refCount--
        if e.refCount <= 0 {
            delete(p.entries, hash)
        }
    }
}
```

### InternObject (convenience: walks an entire object's managedFields)

```go
// InternObject interns all FieldsV1.Raw slices in the object's managedFields.
func (p *fieldsV1InternPool) InternObject(obj runtime.Object) {
    accessor, err := meta.Accessor(obj)
    if err != nil {
        return
    }
    mf := accessor.GetManagedFields()
    if len(mf) == 0 {
        return
    }
    changed := false
    for i := range mf {
        if mf[i].FieldsV1 != nil && len(mf[i].FieldsV1.Raw) > 0 {
            interned := p.Intern(mf[i].FieldsV1.Raw)
            if &interned[0] != &mf[i].FieldsV1.Raw[0] {
                mf[i].FieldsV1.Raw = interned
                changed = true
            }
        }
    }
    if changed {
        accessor.SetManagedFields(mf)
    }
}

// ReleaseObject releases all interned FieldsV1.Raw slices for the object.
func (p *fieldsV1InternPool) ReleaseObject(obj runtime.Object) {
    accessor, err := meta.Accessor(obj)
    if err != nil {
        return
    }
    for _, entry := range accessor.GetManagedFields() {
        if entry.FieldsV1 != nil && len(entry.FieldsV1.Raw) > 0 {
            p.Release(entry.FieldsV1.Raw)
        }
    }
}
```

### Stats (for metrics)

```go
// Stats returns pool statistics for metrics.
func (p *fieldsV1InternPool) Stats() (uniqueEntries int, totalRefs int64, totalBytes int64) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    for _, e := range p.entries {
        uniqueEntries++
        totalRefs += e.refCount
        totalBytes += int64(len(e.raw))
    }
    return
}
```

## Hash Collision Handling

xxhash has a collision probability of ~1 in 2^64. We add `bytesEqual` comparison
after hash match to guarantee correctness:

```go
func bytesEqual(a, b []byte) bool {
    return string(a) == string(b) // no alloc in modern Go
}
```

If a collision is detected (same hash, different bytes), the simplest approach is
to not intern the new entry — it falls back to its own allocation. This is safe
because collisions are astronomically rare and the cost is just one un-deduplicated
entry.

For a more robust approach, we could use a `map[uint64][]*internEntry` (slice of
entries per hash bucket), but this adds complexity for a scenario that essentially
never happens. Start simple.

## Dependencies

- `github.com/cespare/xxhash/v2` — already vendored in kubernetes (used by prometheus client)

Verify:
```
vendor/github.com/cespare/xxhash/v2/
```

If not available, `hash/fnv` from stdlib is an alternative (slightly slower, same
collision properties for our purposes).

## Why Not sync.Pool

`sync.Pool` is for temporary object reuse (GC can reclaim entries at any time).
We need long-lived deduplication where entries persist as long as at least one
object references them. A custom pool with explicit reference counting is required.

## Why Not a Sharded Pool

For the first implementation, a single `sync.RWMutex` is sufficient. The critical
section is small (hash lookup + pointer swap). If profiling shows lock contention
under extreme churn, we can shard by hash prefix:

```go
const shardCount = 16
type shardedPool struct {
    shards [shardCount]fieldsV1InternPool
}
func (p *shardedPool) shard(hash uint64) *fieldsV1InternPool {
    return &p.shards[hash%shardCount]
}
```

This is a straightforward follow-up optimization, not needed for v1.
