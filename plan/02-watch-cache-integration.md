# 02 — Watch Cache Integration

## Modified File

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go
```

## What Changes

The `watchCache` struct gets a `fieldsV1InternPool` field. Every object that
enters the cache has its `FieldsV1.Raw` slices interned. Every object that
leaves the cache has its references released.

### 1. Add pool to watchCache struct (~line 89)

```go
type watchCache struct {
    sync.RWMutex

    // ... existing fields ...

    // fieldsV1Pool deduplicates FieldsV1.Raw across cached objects.
    // Only active when InternManagedFieldsInWatchCache feature gate is enabled.
    fieldsV1Pool *fieldsV1InternPool
}
```

### 2. Initialize pool in newWatchCache() (~line 178)

```go
func newWatchCache(...) *watchCache {
    wc := &watchCache{
        // ... existing init ...
    }
    if utilfeature.DefaultFeatureGate.Enabled(features.InternManagedFieldsInWatchCache) {
        wc.fieldsV1Pool = newFieldsV1InternPool()
    }
    // ... rest of init ...
    return wc
}
```

### 3. Intern on cache insert — processEvent() (~line 283)

The `processEvent` function is where every Add/Update/Delete flows through.
We intern after the object is decoded but before it's stored.

```go
func (w *watchCache) processEvent(event watch.Event, resourceVersion uint64, updateFunc func(*store.Element) error) error {
    // ... existing key computation (lines 286-293) ...

    elem := &store.Element{Key: key, Object: event.Object}
    elem.Labels, elem.Fields, err = w.getAttrsFunc(event.Object)

    // NEW: intern FieldsV1.Raw before storing
    if w.fieldsV1Pool != nil {
        w.fieldsV1Pool.InternObject(elem.Object)
    }

    // ... rest of existing processEvent (wcEvent creation, store update, etc.) ...
```

**Important**: We intern the object on `elem.Object` before it goes into both the
store AND the event buffer. Since `wcEvent.Object` is set to `elem.Object` (same
reference, line 298), both paths benefit from a single intern call.

### 4. Release on cache eviction — processEvent() (~line 315)

When an object is **replaced** (Update) or **removed** (Delete), the previous
version leaves the cache. We need to release its interned references.

```go
    previous, exists, err := w.store.Get(elem)
    if err != nil {
        return err
    }
    if exists {
        previousElem := previous.(*store.Element)
        wcEvent.PrevObject = previousElem.Object
        wcEvent.PrevObjLabels = previousElem.Labels
        wcEvent.PrevObjFields = previousElem.Fields

        // NEW: release interned refs from the object being replaced
        if w.fieldsV1Pool != nil {
            w.fieldsV1Pool.ReleaseObject(previousElem.Object)
        }
    }
```

### 5. Intern on bulk load — Replace() (~line 736)

`Replace()` is called during initial list and relists. It bulk-loads all objects.

```go
func (w *watchCache) Replace(objs []interface{}, resourceVersion string) error {
    // ... existing validation ...

    // NEW: release all existing objects before replacing
    if w.fieldsV1Pool != nil {
        allItems := w.store.List()
        for _, item := range allItems {
            if elem, ok := item.(*store.Element); ok {
                w.fieldsV1Pool.ReleaseObject(elem.Object)
            }
        }
    }

    toReplace := make([]interface{}, 0, len(objs))
    for _, obj := range objs {
        object, ok := obj.(runtime.Object)
        if !ok {
            return fmt.Errorf("didn't get runtime.Object for replace: %#v", obj)
        }

        // NEW: intern each incoming object
        if w.fieldsV1Pool != nil {
            w.fieldsV1Pool.InternObject(object)
        }

        key, err := w.keyFunc(object)
        // ... rest of existing Replace logic ...
    }
    // ...
}
```

### 6. Release on event buffer eviction — updateCache() (~line 360)

When the cyclic event buffer is full and the oldest event is evicted:

```go
func (w *watchCache) updateCache(event *watchCacheEvent) {
    w.resizeCacheLocked(event.RecordTime)
    if w.isCacheFullLocked() {
        // Cache is full - remove the oldest element.
        // NEW: release interned refs from evicted event's PrevObject.
        // Note: the Object in evicted events is the SAME object that was later
        // replaced in the store (and released there), OR the same pointer that
        // was interned into a newer event's PrevObject. We only need to release
        // PrevObject here because it's the only copy unique to the event buffer.
        if w.fieldsV1Pool != nil {
            evicted := w.cache[w.startIndex%w.capacity]
            if evicted != nil && evicted.PrevObject != nil {
                w.fieldsV1Pool.ReleaseObject(evicted.PrevObject)
            }
        }
        w.startIndex++
        w.removedEventSinceRelist = true
    }
    w.cache[w.endIndex%w.capacity] = event
    w.endIndex++
}
```

**Note on reference counting correctness**: This is the trickiest part. The
reference counting must match exactly:
- **+1** when an object enters the store (processEvent → store.Add/Update)
- **+1** when a PrevObject is captured in an event (if it has different interned refs than the store copy — but since PrevObject IS the old store copy, the refs were already counted)
- **-1** when an object leaves the store (replaced or deleted)
- **-1** when a PrevObject's event is evicted from the buffer

The key insight is that when `processEvent` runs:
1. The old object is in the store
2. It gets captured as `PrevObject` in the new event
3. The new object replaces it in the store

At step 2, `PrevObject` now holds the only reference to the old interned bytes
(the store reference was released at the replacement). So we release PrevObject's
refs when the event is evicted from the buffer.

Actually, this is subtle. Let me simplify: **intern once per unique object instance
entering the system, release once when that instance is no longer reachable from
any cache data structure.** See [05-deep-copy-safety.md](05-deep-copy-safety.md)
for detailed analysis.

### Alternative: Skip Reference Counting Entirely

A simpler approach: **don't reference count at all.** Instead, periodically rebuild
the pool from scratch by scanning all objects in the store.

```go
// Called periodically (e.g., every 5 minutes) or when pool size exceeds threshold
func (w *watchCache) rebuildInternPool() {
    newPool := newFieldsV1InternPool()
    items := w.store.List()
    for _, item := range items {
        if elem, ok := item.(*store.Element); ok {
            newPool.InternObject(elem.Object)
        }
    }
    w.fieldsV1Pool = newPool
    // Old pool becomes garbage when no more references exist
}
```

This is simpler and avoids all reference counting bugs. The cost is:
- Periodic O(N) scan of all objects (fast — just walking slice headers)
- Stale entries linger until next rebuild (bounded by rebuild interval)
- Event buffer objects may hold non-interned copies between rebuilds

For a first implementation, this may be the better approach. Optimize to
reference counting later if the periodic scan is too expensive.

## Modified File (secondary)

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go
```

No changes needed in cacher.go. The interning is fully encapsulated within the
watchCache. The `Cacher` calls `watchCache.processEvent` and `watchCache.Replace`
which handle interning internally.
