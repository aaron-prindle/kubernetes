# SSA Memory Bottleneck Investigation Plan

## Objective
Identify why Server-Side Apply (SSA) can become a bottleneck in large clusters (especially API server memory growth vs resource count), document evidence from upstream sources and code, and define actionable mitigation paths.

## Scope
- Public material: Kubernetes docs/blog posts/issues/PRs relevant to SSA and managedFields overhead.
- Code-level understanding from `kubernetes/kubernetes`:
  - SSA request path and field management pipeline.
  - `managedFields` encoding/decoding and update behavior.
  - API server cache/list/watch paths where object size amplifies memory use.
- Architecture-level recommendations (L7/API behavior and semantics, not only infra tuning).
- Repro plan for local cluster validation.

## Workstreams
1. Source collection
- Gather authoritative SSA docs and Kubernetes blog posts.
- Gather public issues/PRs discussing managedFields verbosity/size and omission strategies.

2. Code tracing
- Map request flow from apply patch endpoint to field manager internals.
- Identify where `managedFields` is produced, capped, transformed, serialized, stored, and read.

3. Bottleneck analysis
- Explain memory growth mechanisms and scaling amplifiers.
- Separate CPU hotspots from memory-resident footprint effects.
- Identify coupling between watch cache/list paths and object metadata size.

4. Mitigation options
- Immediate operational options.
- Mid-term API behavior improvements.
- Longer-term design options for managedFields representation and serving paths.

5. Reproduction and validation
- Local cluster scenario that stresses managedFields growth.
- Profiling and metrics plan to baseline and compare candidate mitigations.

## Deliverables
- `research/source-catalog.md`
- `research/public-sources/*`
- `research/codebase-analysis/*`
- `research/performance-hypotheses/*`
- `research/experiments/*`
- `research/recommendations/*`
- `future-plan.md`

## Success Criteria
- Clear explanation of why API server memory can scale poorly with resource count under SSA-heavy usage.
- Concrete evidence that `managedFields` is a major contributor in specific paths.
- Ranked mitigation plan with measurable outcomes and rollout strategy.
- Reproducible local benchmark workflow for regression testing.
