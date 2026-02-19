# SSA Docs and KEP Notes

## 1) Kubernetes Server-Side Apply docs
Source: https://kubernetes.io/docs/reference/using-api/server-side-apply/

Key points:
- SSA manages fields per manager and records ownership in `.metadata.managedFields`.
- Apply conflicts are semantic protections when another manager owns a field with differing value.
- Ownership transfers can happen via apply force or non-apply update/patch workflows.
- `managedFields` is API-visible metadata and can be large/verbose.

Relevance to memory bottlenecks:
- Every object can carry ownership metadata; in aggregate, this increases memory footprint across:
  - etcd payloads,
  - apiserver in-memory objects,
  - cache/list/watch fanout,
  - downstream controllers/informers.

## 2) KEP-555 (Server-Side Apply)
Source: https://raw.githubusercontent.com/kubernetes/enhancements/master/keps/sig-api-machinery/555-server-side-apply/README.md

Important KEP statements:
- KEP risk notes explicitly call out object-size growth from managed fields.
- KEP documents that managedFields can be a major fraction of object size (up to ~60% in cited discussion).
- KEP acknowledges resource usage increase (CPU/RAM/disk/IO) from larger objects and larger caches.

Relevance:
- Confirms that memory scaling impact is not accidental; it was a known tradeoff of SSA design.
- Validates that mitigation can target metadata volume without discarding SSA correctness semantics.

## 3) Kubernetes API Concepts (supporting context)
Source: https://kubernetes.io/docs/reference/using-api/api-concepts/

Useful sections:
- Response compression (`Accept-Encoding: gzip`) reduces wire size.
- Streaming lists and watch/list consistency options reduce large list materialization pressure.
- Watch cache behavior and list-from-cache/snapshots are central to apiserver scalability.

Relevance:
- Wire compression helps network bytes, but does not directly solve in-memory object bloat.
- Streaming/list features can reduce temporary allocation pressure and improve request-path efficiency.
