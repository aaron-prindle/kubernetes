# FieldsV1 Interning Implementation Plan

## Goal

Deduplicate identical `FieldsV1.Raw` byte slices across objects in the apiserver
watch cache. Many objects of the same type, managed by the same controllers, carry
identical FieldsV1 data. Today each object holds its own independent `[]byte` copy.
Interning lets them share a single allocation.

## How It Works

```
Before interning:
  Pod-1.ManagedFields[0].FieldsV1.Raw → [2 KB bytes]   ← independent copy
  Pod-2.ManagedFields[0].FieldsV1.Raw → [2 KB bytes]   ← independent copy
  Pod-3.ManagedFields[0].FieldsV1.Raw → [2 KB bytes]   ← independent copy
  ...
  Pod-N.ManagedFields[0].FieldsV1.Raw → [2 KB bytes]   ← independent copy
  Memory: N × 2 KB

After interning:
  Pod-1.ManagedFields[0].FieldsV1.Raw ──┐
  Pod-2.ManagedFields[0].FieldsV1.Raw ──┼→ [2 KB bytes] ← ONE shared copy
  Pod-3.ManagedFields[0].FieldsV1.Raw ──┤
  ...                                    │
  Pod-N.ManagedFields[0].FieldsV1.Raw ──┘
  Memory: 2 KB + N × 8 bytes (slice headers pointing to shared backing array)
```

## Insertion Point

Objects enter the watch cache through two paths:

1. **Initial list** → `watchCache.Replace()` (`watch_cache.go:736`)
   - Reflector does a paginated LIST from etcd and bulk-loads all objects.

2. **Ongoing watch** → `watchCache.Add/Update/Delete()` → `processEvent()` (`watch_cache.go:283`)
   - Reflector watches etcd for changes and processes them one-by-one.

At both points, the object is a fully-decoded `runtime.Object` with populated
`ObjectMeta.ManagedFields[].FieldsV1.Raw` byte slices. We intern these byte
slices before the object is stored in the cache.

## Scope of Changes

| Area | Files | Risk |
|------|-------|------|
| New interning pool | 1 new file | Low — self-contained |
| Watch cache integration | 1 file modified | Medium — hot path |
| Feature gate | 1 file modified | Low — standard pattern |
| Metrics | 1 file modified | Low — observability |
| Tests | 2-3 new files | Low |

## File-by-File Plan

See the following documents:
- [01-intern-pool.md](01-intern-pool.md) — The interning pool implementation
- [02-watch-cache-integration.md](02-watch-cache-integration.md) — Hooking into the watch cache
- [03-feature-gate.md](03-feature-gate.md) — Feature gate registration
- [04-metrics.md](04-metrics.md) — Observability
- [05-deep-copy-safety.md](05-deep-copy-safety.md) — Deep copy correctness analysis
- [06-tests.md](06-tests.md) — Test plan
- [07-risks-and-mitigations.md](07-risks-and-mitigations.md) — Risk analysis

## Work Order

1. Add feature gate (03)
2. Implement intern pool (01)
3. Add metrics (04)
4. Integrate into watch cache (02)
5. Write tests (06)
6. Benchmark and validate
