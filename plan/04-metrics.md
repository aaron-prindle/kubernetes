# 04 — Metrics

## Modified File

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/metrics/metrics.go
```

## What Changes

Add metrics to observe interning effectiveness. Follow existing pattern in the file.

### New Metrics

```go
var (
    // Gauge: number of unique FieldsV1 patterns in the intern pool
    fieldsV1InternPoolUniqueEntries = compbasemetrics.NewGaugeVec(
        &compbasemetrics.GaugeOpts{
            Namespace:      namespace,
            Subsystem:      subsystem,
            Name:           "fieldsv1_intern_pool_unique_entries",
            Help:           "Number of unique FieldsV1.Raw patterns in the intern pool",
            StabilityLevel: compbasemetrics.ALPHA,
        },
        []string{"group", "resource"},
    )

    // Gauge: total number of references to interned entries
    fieldsV1InternPoolTotalRefs = compbasemetrics.NewGaugeVec(
        &compbasemetrics.GaugeOpts{
            Namespace:      namespace,
            Subsystem:      subsystem,
            Name:           "fieldsv1_intern_pool_total_refs",
            Help:           "Total number of object references to interned FieldsV1 entries",
            StabilityLevel: compbasemetrics.ALPHA,
        },
        []string{"group", "resource"},
    )

    // Gauge: bytes saved by interning (totalRefs * avgSize - uniqueEntries * avgSize)
    fieldsV1InternPoolBytesDeduped = compbasemetrics.NewGaugeVec(
        &compbasemetrics.GaugeOpts{
            Namespace:      namespace,
            Subsystem:      subsystem,
            Name:           "fieldsv1_intern_pool_bytes_deduped",
            Help:           "Estimated bytes saved by FieldsV1 interning",
            StabilityLevel: compbasemetrics.ALPHA,
        },
        []string{"group", "resource"},
    )
)
```

### Registration

Add to the `Register()` function:

```go
legacyregistry.MustRegister(fieldsV1InternPoolUniqueEntries)
legacyregistry.MustRegister(fieldsV1InternPoolTotalRefs)
legacyregistry.MustRegister(fieldsV1InternPoolBytesDeduped)
```

### Recording Function

```go
func RecordFieldsV1InternPoolMetrics(groupResource schema.GroupResource, unique int, totalRefs int64, totalBytes int64) {
    gr := groupResource
    fieldsV1InternPoolUniqueEntries.WithLabelValues(gr.Group, gr.Resource).Set(float64(unique))
    fieldsV1InternPoolTotalRefs.WithLabelValues(gr.Group, gr.Resource).Set(float64(totalRefs))
    // Estimate: if we have R total refs to U unique entries using B total bytes,
    // then without interning we'd use (R/U)*B bytes. Savings = (R/U)*B - B = B*(R/U - 1)
    if unique > 0 {
        avgSize := float64(totalBytes) / float64(unique)
        withoutInterning := float64(totalRefs) * avgSize
        savings := withoutInterning - float64(totalBytes)
        fieldsV1InternPoolBytesDeduped.WithLabelValues(gr.Group, gr.Resource).Set(savings)
    }
}
```

### Where to Call

In `watch_cache.go`, periodically report metrics. The simplest approach is to
report after each `processEvent` or on a timer. Since `processEvent` is hot,
prefer a periodic approach — e.g., every N events or piggyback on an existing
periodic operation.

```go
// In processEvent, after updateFunc:
if w.fieldsV1Pool != nil {
    // Report every 1000 events to avoid metric overhead
    if resourceVersion%1000 == 0 {
        unique, refs, bytes := w.fieldsV1Pool.Stats()
        metrics.RecordFieldsV1InternPoolMetrics(w.groupResource, unique, refs, bytes)
    }
}
```

## Why These Metrics Matter

- **unique_entries**: Shows how many distinct FieldsV1 patterns exist per resource.
  Low values (hundreds) for homogeneous workloads, high values for heterogeneous.

- **total_refs**: Shows how many objects reference interned entries. The ratio
  `total_refs / unique_entries` is the deduplication factor.

- **bytes_deduped**: Direct measure of memory saved. This is the key metric for
  validating the feature's value.

Example dashboard query:
```
sum(apiserver_watch_cache_fieldsv1_intern_pool_bytes_deduped) by (resource)
```
This immediately shows how much memory the feature is saving per resource type.
