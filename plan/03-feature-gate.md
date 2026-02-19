# 03 — Feature Gate

## Modified File

```
staging/src/k8s.io/apiserver/pkg/features/kube_features.go
```

## What Changes

Add a new feature gate following the existing pattern. The gate controls whether
FieldsV1 interning is active in the watch cache.

### 1. Add constant (~alphabetical position, around line 130)

```go
// owner: @aaronprindle
//
// Enables interning (deduplication) of FieldsV1.Raw byte slices across objects
// in the watch cache. Objects managed by the same controllers share identical
// FieldsV1 data; interning lets them share a single byte slice allocation
// instead of independent copies.
InternManagedFieldsInWatchCache featuregate.Feature = "InternManagedFieldsInWatchCache"
```

### 2. Add versioned spec (~line 298+, in defaultVersionedKubernetesFeatureGates)

```go
InternManagedFieldsInWatchCache: {
    {Version: version.MustParse("1.36"), Default: false, PreRelease: featuregate.Alpha},
},
```

Starting as Alpha (default: false) means:
- Must be explicitly enabled with `--feature-gates=InternManagedFieldsInWatchCache=true`
- No impact on existing clusters unless opted in
- Can be promoted to Beta (default: true) after validation

### 3. Usage

In `watch_cache.go`:

```go
import "k8s.io/apiserver/pkg/features"

if utilfeature.DefaultFeatureGate.Enabled(features.InternManagedFieldsInWatchCache) {
    wc.fieldsV1Pool = newFieldsV1InternPool()
}
```

When the gate is disabled, `fieldsV1Pool` is nil, and all interning code paths
are skipped via nil checks. Zero overhead when disabled.
