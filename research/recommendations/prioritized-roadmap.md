# Prioritized Roadmap (Actionable)

## Near-term (0-1 month)
- Build reproducible benchmark harness from `research/experiments/local-cluster-repro-plan.md`.
- Quantify managedFields contribution to object size for top resource kinds.
- Roll out/validate managedFields trimming in internal consumers that do not require ownership.

## Short-term (1-3 months)
- Prototype API read-path omission option (`omitManagedFields`) behind feature gate.
- Evaluate no-op apply mutation suppression logic for metadata timestamps.
- Validate compatibility against common clients/controllers.

## Mid-term (3-6 months)
- Upstream proposal: opt-in omit behavior for GET/LIST/WATCH and clear API docs.
- Add per-resource/per-verb observability that tracks managedFields bytes and omission usage.

## Longer-term (6+ months)
- Explore compact in-memory ownership representation and dedup strategies.
- Evaluate ownership-lite response format if demand exists.
- Consider KEP for broader representation-level changes.

## Decision Gates
- Gate 1: >=15% memory reduction without p99 latency regressions.
- Gate 2: no API correctness regressions in conflict/ownership semantics.
- Gate 3: client compatibility and operational debuggability remain acceptable.
