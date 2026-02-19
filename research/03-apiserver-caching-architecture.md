# API Server Caching Architecture

## Overview

The Kubernetes API server maintains an in-memory cache layer between etcd and API clients. This cache serves two purposes:
1. **Watch cache**: Delivers watch events to clients without re-querying etcd
2. **List cache**: Serves LIST requests from memory (especially with WatchList/streaming)

Understanding this caching layer is critical because **every cached object includes its full managedFields data**, and the cache is where the memory bottleneck manifests.

## Architecture Diagram

```
                                    ┌─────────────────┐
                                    │   Watch Client 1 │
                                    │  (kube-proxy)    │
                                    └────────┬────────┘
                                             │
                                    ┌────────┴────────┐
                                    │  cacheWatcher    │
                                    │  (input chan)    │
                                    └────────┬────────┘
                                             │
┌──────┐    ┌──────────┐    ┌────────────────┴─────────────────┐
│ etcd │───>│  Cacher   │───>│         watchCache               │
│      │    │ (storage/ │    │  ┌──────────────────────┐        │
│      │    │  cacher)  │    │  │  Cyclic Event Buffer │        │──> cacheWatcher 2
│      │    └──────────┘    │  │  []*watchCacheEvent   │        │──> cacheWatcher 3
│      │                    │  │  (100 - 102,400 items)│        │──> cacheWatcher N
│      │                    │  └──────────────────────┘        │
│      │                    │  ┌──────────────────────┐        │
│      │                    │  │  Store (Indexer)      │        │
│      │                    │  │  BTree/HashMap of     │        │
│      │                    │  │  store.Element        │        │
│      │                    │  │  (current state)      │        │
│      │                    │  └──────────────────────┘        │
│      │                    │  ┌──────────────────────┐        │
│      │                    │  │  Snapshots (optional) │        │
│      │                    │  │  B-tree of store      │        │
│      │                    │  │  clones for LIST      │        │
│      │                    │  └──────────────────────┘        │
│      │                    └──────────────────────────────────┘
└──────┘
```

## Key Data Structures

### watchCacheEvent (watch_cache.go:71-82)
```go
type watchCacheEvent struct {
    Type            watch.EventType
    Object          runtime.Object      // Full object including managedFields
    ObjLabels       labels.Set
    ObjFields       fields.Set
    PrevObject      runtime.Object      // Previous version (for Modified events)
    PrevObjLabels   labels.Set
    PrevObjFields   fields.Set
    Key             string
    ResourceVersion uint64
    RecordTime      time.Time
}
```

**Memory per event**: ~250 bytes struct overhead + object size + previous object size

### store.Element (store/store.go:88-98)
```go
type Element struct {
    Key    string           // e.g., "/pods/default/my-pod"
    Object runtime.Object   // Full object including managedFields
    Labels labels.Set
    Fields fields.Set
}
```

### cachingObject (caching_object.go:64-84)
```go
type cachingObject struct {
    lock           sync.RWMutex
    deepCopied     bool
    object         metaRuntimeInterface     // The actual object
    serializations atomic.Value             // map[Identifier]*serializationResult
}

type serializationResult struct {
    once sync.Once
    raw  []byte     // Cached serialized bytes (JSON, Protobuf, or CBOR)
    err  error
}
```

## Object Lifecycle in Cache

### 1. Object enters from etcd
```
etcd bytes -> codec.Decode() -> runtime.Object (with managedFields)
                                     |
                                     v
                              watchCacheEvent.Object
```

### 2. Object stored in cache
The watchCache stores the object in TWO places:
- **Cyclic event buffer** (`cache []*watchCacheEvent`): Historical events
- **Store** (BTree or HashMap): Current state of all objects

### 3. Object dispatched to watchers
```go
// cacher.go:922-951
func setCachingObjects(event *watchCacheEvent, versioner storage.Versioner) {
    switch event.Type {
    case watch.Added, watch.Modified:
        if object, err := newCachingObject(event.Object); err == nil {
            event.Object = object  // Wrap in cachingObject
        }
    case watch.Deleted:
        if object, err := newCachingObject(event.PrevObject); err == nil {
            event.PrevObject = object
        }
    }
}
```

**Critical**: The cachingObject wraps the FULL object including all managedFields.

### 4. Serialization for each watcher
```go
// caching_object.go:136-157
func (o *cachingObject) CacheEncode(id runtime.Identifier, encode func(runtime.Object, io.Writer) error, w io.Writer) error {
    result := o.getSerializationResult(id)
    result.once.Do(func() {
        buffer := bytes.NewBuffer(nil)
        result.err = encode(o.GetObject(), buffer)  // Serialize FULL object
        result.raw = buffer.Bytes()
    })
    // ... write cached bytes to writer
}
```

The serialization includes the ENTIRE object including managedFields. This means:
- JSON serialization: full JSON with managedFields
- Protobuf serialization: full protobuf with managedFields
- CBOR serialization: full CBOR with managedFields

## Memory Multiplication Factors

### Factor 1: Cyclic Buffer + Store (2x)
Each object exists at minimum twice:
- Once in the cyclic event buffer (as watchCacheEvent)
- Once in the store (as store.Element)

### Factor 2: Modified Events (2x for recent objects)
Modified events store both current and previous object:
```go
watchCacheEvent.Object     = current version
watchCacheEvent.PrevObject = previous version
```

### Factor 3: Serialization Cache (1-3x during dispatch)
During event dispatch, cachingObject can cache serializations in multiple formats:
- JSON (for clients using JSON)
- Protobuf (for clients using Protobuf)
- CBOR (for clients using CBOR)

**Important**: As of recent changes, serializations are only cached during dispatch, not permanently. However, during burst event processing, this can still be significant.

### Factor 4: Deep Copy for GetObject() (1x per read)
```go
func (o *cachingObject) GetObject() runtime.Object {
    o.lock.RLock()
    defer o.lock.RUnlock()
    return o.object.DeepCopyObject().(metaRuntimeInterface)
}
```
Each call to GetObject() creates a full deep copy including managedFields.

## Watch Cache Sizing

### Capacity Configuration (watch_cache.go:48-65)
```go
const (
    defaultLowerBoundCapacity = 100
    defaultUpperBoundCapacity = 100 * 1024  // 102,400
)
```

### Dynamic Sizing
The cache capacity is dynamic:
- Starts at lower bound (100 events)
- Doubles when full AND oldest event is within `eventFreshDuration` (75s default)
- Halves when quarter of events are older than `eventFreshDuration`
- Maximum capacity: 102,400 events per resource type

### Memory Impact
For a resource type (e.g., Pods) with 10,000 objects:
- **Store**: 10,000 * (object size + managedFields size)
- **Event buffer**: Up to 102,400 * (event size including objects)
- **Snapshot (if enabled)**: Additional B-tree clones for consistent LIST

## Compression - Current State

### Over the wire
- `APIResponseCompression` feature gate enables gzip for GET/LIST responses
- Only applies to HTTP response bodies, NOT to watch events
- Does NOT reduce in-memory size

### In cache
- **NO compression of cached objects**
- Objects are stored as Go structs (runtime.Object)
- Serialization cache stores raw bytes (uncompressed)
- FieldsV1.Raw is stored as-is (uncompressed JSON bytes)

### In etcd
- etcd itself uses bbolt with optional compression
- Kubernetes does NOT apply additional compression before storing
- Encryption transformer operates on raw bytes (no compression)

## Feature Gates Affecting Memory

| Feature Gate | Effect | Status |
|-------------|--------|--------|
| `APIResponseCompression` | Gzip for GET/LIST responses | Beta |
| `BtreeWatchCache` | B-tree store instead of HashMap | Beta (1.34) |
| `ListFromCacheSnapshot` | Serve LIST from cache snapshots | Beta (1.34) |
| `WatchList` | Stream LIST via watch protocol | Beta (1.32) |

## Key Observation

The watch cache is the primary memory consumer in the apiserver for large clusters. Every object in cache carries its FULL managedFields data, which:
1. Most watchers never need (e.g., kube-proxy watching Endpoints doesn't use managedFields)
2. Gets serialized into every response format
3. Gets deep-copied on every access
4. Gets stored in the cyclic buffer with both current AND previous versions

**ManagedFields are the single largest optimization target for apiserver memory reduction.**
