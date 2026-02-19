# SSA ManagedFields Memory Bottleneck in kube-apiserver

**Authors:** Aaron Prindle
**Date:** 2026-02-18
**Status:** Proposal
**Target:** sig-api-machinery

## Problem

Server-Side Apply (SSA) stores field ownership metadata (`managedFields`) on every
Kubernetes object. Each object carries 3-11 `ManagedFieldsEntry` records, each
containing a `FieldsV1.Raw` JSON trie encoding all field paths a manager owns.
This metadata is **stored at full size in the apiserver watch cache** — in the
object store, the event history buffer, and serialized responses — with no
compression, deduplication, or filtering.

In large clusters, managedFields becomes a dominant fraction of apiserver memory:

| Cluster size | Estimated managedFields in cache | % of total cache |
|---|---|---|
| 1,000 nodes (~50K objects) | ~400 MB | 40-50% |
| 5,000 nodes (~265K objects) | ~1.8 GB | 40-50% |
| 10,000 nodes (~500K objects) | ~3.5 GB | 40-50% |

The watch cache has **zero awareness of managedFields** — no references to it exist
in `watch_cache.go`, `cacher.go`, or `caching_object.go`. Every byte is treated
identically regardless of whether consumers need it.

## Evidence: Local Reproduction

We reproduced the issue on a local kind cluster (Kubernetes v1.35.0):

- **Setup:** kind cluster with apiserver profiling enabled
- **Load:** 2,000 ConfigMaps with 5 SSA field managers each (10,000 apply calls)
- **Measurement:** pprof heap profiles + object size analysis

| Metric | Value |
|---|---|
| managedFields as % of JSON object size | **49.1%** |
| managedFields as % of YAML object size | **62.1%** |
| Heap growth from load | 83 MB → 147 MB (+76%) |
| RSS growth from load | 481 MB → 678 MB (+41%) |
| Top allocator: FieldsV1.Unmarshal | 5 MB flat / 8 MB cumulative |

Reproduction scripts and full results are in `research/repro/`.

## Why This Happens

The watch cache stores full objects in three places, each including managedFields:

1. **Store** — indexed map of every current object (one copy each)
2. **Event buffer** — cyclic buffer of up to 102,400 recent events per resource type, each storing both the new and previous object version
3. **Serialization cache** — encoded bytes per wire format (JSON, Protobuf, CBOR)

Most watch consumers (kubelet, kube-proxy, scheduler, custom controllers) do not
use managedFields. It is only needed for Apply operations (conflict detection and
field ownership). Despite this, it is stored and transmitted at full size for all
operations.

## Existing Mitigations

| Mitigation | What it reduces | Apiserver memory impact |
|---|---|---|
| CapManagersManager (max 10 update entries) | Entry count | Slight |
| StripMetaManager (excludes system fields) | Entry size | Slight |
| Audit log omission (`OmitManagedFields`) | Audit log size | None |
| kubectl `--show-managed-fields=false` | CLI output | None |
| Client-side `TransformStripManagedFields` | Controller memory | None |

**No existing mitigation reduces apiserver watch cache memory.**

## Upstream Status

| Reference | Status | Scope |
|---|---|---|
| PR #136760 (read-path omission) | Open, WIP, experimental | GET/LIST only, no WATCH |
| Issue #131175 (no-op SSA churn) | Open, triage/accepted | No implementation |
| PR #131016 (scheduler trim) | Closed, not merged | Component-specific |

No merged solution addresses the core watch cache memory problem.

## Proposed Solution

Three complementary approaches, in order of implementation:

### 1. Read-Path Omission (aligns with upstream PR #136760)

Add `omitManagedFields` option to GET/LIST/WATCH. Clients that don't need
managedFields opt out, saving wire bandwidth and serialization cost. Extend
PR #136760 to cover WATCH (currently missing) and fix its object mutation bug.

- **Apiserver memory savings:** ~5% (serialization cache only)
- **Client/wire savings:** ~50% per opted-out response
- **Risk:** Low — opt-in, backward compatible

### 2. FieldsV1 Interning (new, not attempted upstream)

Deduplicate identical `FieldsV1.Raw` byte slices across objects in the watch
cache. Objects from the same controller template (e.g., Pods from one Deployment)
carry identical FieldsV1 data. An intern pool lets them share a single allocation.

- **Apiserver memory savings:** 33-40% typical, up to 40%+ for homogeneous workloads
- **Risk:** Medium — internal-only, feature-gated, no API changes
- **Key insight:** FieldsV1 encodes schema structure, not instance data. Replicas
  have identical structures, so deduplication ratios are very high in practice.

### 3. FieldsV1 Compression (can layer on top of interning)

Compress `FieldsV1.Raw` with zstd before storing in cache. The JSON trie format
is highly repetitive (`"f:"`, `"k:"`, `".":{}` patterns) and compresses 70-75%.

- **Apiserver memory savings:** ~26% standalone, backstops interning for heterogeneous workloads
- **Risk:** Low-Medium — internal-only, must decompress for Apply path

### Combined Impact

| Approach | Apiserver savings | Predictable? |
|---|---|---|
| Read-path omission alone | ~5% | Yes |
| Interning alone | 0-40% | No (workload-dependent) |
| Compression alone | ~26% | Yes |
| Interning + Compression | ~40% | Mostly |
| All three combined | ~46% | Yes |

For a 5,000-node cluster, the combined approach reduces apiserver cache memory
from ~4 GB to ~2.2 GB, with managedFields dropping from ~1.85 GB to ~40 MB.

## Implementation Plan (Interning)

We have a detailed plan in `plan/` covering 6 files:

| Change | File | Lines |
|---|---|---|
| Intern pool (new) | `storage/cacher/fieldsv1_intern_pool.go` | ~200 |
| Watch cache integration | `storage/cacher/watch_cache.go` | ~50 modified |
| Feature gate | `apiserver/pkg/features/kube_features.go` | ~10 |
| Metrics | `storage/cacher/metrics/metrics.go` | ~40 |
| Unit tests (new) | `storage/cacher/fieldsv1_intern_pool_test.go` | ~200 |
| Integration tests | `storage/cacher/watch_cache_test.go` | ~150 |

**Insertion point:** Objects are interned when they enter the watch cache
(`processEvent` and `Replace`). The intern pool maps `xxhash(FieldsV1.Raw)` →
shared `[]byte`. Multiple objects with identical field structures point to the
same backing array. Deep copies (for serialization and watcher dispatch) allocate
independent bytes, so interned data is never mutated.

**Feature gate:** `InternManagedFieldsInWatchCache` — Alpha (default off).

## Key Risks

| Risk | Mitigation |
|---|---|
| Reference counting bugs | Use periodic pool rebuild (scan store) instead of ref counting |
| Lock contention on hot path | Pool critical section is ~450ns; shard if needed |
| Byte mutation after interning | Analysis confirms all consumers deep-copy or read-only; tests verify |
| Hash collision | Full byte comparison after hash match eliminates possibility |
| No savings for heterogeneous workloads | Compression backstops interning; metrics detect and can disable |

## Open Questions

1. Should `omitManagedFields` become the default for WATCH in a future version?
2. Should interning and compression be separate feature gates or combined?
3. Should we propose a KEP for the combined approach or separate KEPs per phase?

## References

- [Local reproduction results](research/repro/measurements/results.md)
- [Implementation plan](plan/00-overview.md)
- [Memory bottleneck analysis](research/04-memory-bottleneck-analysis.md)
- [Solution proposals](research/06-solution-proposals.md)
- [KEP-555: Server Side Apply](https://github.com/kubernetes/enhancements/issues/555)
- [PR #136760: WIP read-path omission](https://github.com/kubernetes/kubernetes/pull/136760)
- [Issue #131175: No-op SSA metadata churn](https://github.com/kubernetes/kubernetes/issues/131175)
