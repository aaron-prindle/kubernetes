# API Server Cache Memory Evidence Relevant to managedFields

## watch cache stores full runtime objects in events
File:
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go`

Evidence:
- `watchCacheEvent` contains `Object runtime.Object` and often `PrevObject runtime.Object`.
- Event ring buffer can dynamically resize up to large upper bound based on churn/freshness.

Implication:
- Larger per-object metadata (including managedFields) inflates each cached event footprint.
- Under high churn and expanded cache capacity, memory multiplies quickly.

## cache process retains previous object for updates
File:
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go`

Evidence:
- `processEvent()` fetches previous object from store and attaches it to event (`PrevObject`) when available.

Implication:
- Modified events can transiently hold both current and previous object references.
- Large metadata increases temporary and retained memory pressure.

## serialization caching tradeoff explicitly mentions memory cost
File:
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go`

Evidence:
- Comments explain that retaining serializations in memory could help performance but is avoided due to increased memory usage.

Implication:
- Upstream already recognizes memory-vs-speed tradeoff in watch fanout.
- Any managedFields compression/caching strategy must account for this tradeoff explicitly.

## practical synthesis
- Even if managedFields is only one metadata field, watch/list/object-caching paths replicate object payloads across many in-memory structures.
- Therefore, per-object managedFields bytes can become cluster-scale RAM multipliers.
