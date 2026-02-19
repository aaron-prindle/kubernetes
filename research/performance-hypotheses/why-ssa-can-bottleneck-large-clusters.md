# Why SSA Can Bottleneck Large Clusters

## Hypothesis 1: Per-object metadata inflation multiplies in caches
Mechanism:
- SSA introduces/maintains ownership metadata in `managedFields`.
- Larger object payloads are retained in watch cache and component informers.

Expected effect:
- kube-apiserver RSS grows faster than object count when object metadata complexity grows.

## Hypothesis 2: Write-path amplification from decode/merge/encode
Mechanism:
- For apply/update, field manager pipeline decodes managedFields, computes merged ownership sets, and re-encodes.
- Structured merge conversion pipeline introduces additional allocations and typed object transformations.

Expected effect:
- Elevated CPU and allocation rates on high-frequency apply/update resources.
- Increased GC pressure that can manifest as latency spikes.

## Hypothesis 3: Event churn can dominate even with low business-state changes
Mechanism:
- Metadata-only mutations (including managedFields timestamps/ownership shifts) can change object RV and generate watch events.

Expected effect:
- Higher watch fanout load and cache churn than expected from business-level spec/status changes.

## Hypothesis 4: Current cap controls are necessary but insufficient
Mechanism:
- Capping update manager count limits cardinality of entries, but does not compact large field ownership sets.

Expected effect:
- Objects with broad schemas and many managed paths remain large despite capped manager entry count.

## Hypothesis 5: Wire compression alone does not solve in-memory footprint
Mechanism:
- Gzip/response compression targets network payload.
- Apiserver and consumers still deserialize and retain object structures in memory.

Expected effect:
- Network and egress improve, but RSS reduction is limited unless in-memory representations or read-path omission also change.

## Observable Symptoms to Validate
- RSS tracks object cardinality and managedFields-heavy resources.
- Heap profiles show substantial retention in runtime objects and cache structures for hot resources.
- Apply-heavy workloads show higher alloc/op than update patterns without SSA.
- No-op/near-no-op apply loops still produce watch activity and memory churn.
