# Future Plan: Reducing SSA/managedFields Memory Bottlenecks

## Desired Outcome
Reduce kube-apiserver memory growth and SSA-related write/read amplification while preserving ownership/conflict semantics.

## Phase 0: Baseline and Guardrails (1-2 weeks)
- Capture baseline profiles in representative environments:
  - `heap`, `alloc_space`, `inuse_space`, `cpu` pprof for kube-apiserver.
  - API metrics split by verb/resource (`APPLY`, `PATCH`, `LIST`, `WATCH`).
  - Object-size histograms for high-cardinality resources.
- Add dashboard slices that isolate:
  - managedFields-heavy object types.
  - watch cache memory vs object count/churn.
- Define regression SLOs:
  - Max RSS growth per +10k objects.
  - P95/P99 apply latency under write churn.

## Phase 1: Low-Risk Controls (2-4 weeks)
- Prefer omitting `managedFields` from read-heavy client paths when ownership data is not consumed.
  - Align with existing upstream direction (`omitManagedFields` concepts in ecosystem and active WIP for get/list omission).
- Ensure consumers/informers that do not require ownership semantics trim metadata eagerly.
  - Scheduler already does this for Pods.
- Apply API read-volume reductions:
  - watch-list streaming, chunked list, response compression, and tighter selectors.

## Phase 2: API Serving Optimizations (4-8 weeks)
- Introduce/advance server-side option to omit managed fields in GET/LIST/WATCH responses.
  - Must keep default behavior backward compatible.
  - Include audit and authn/authz considerations.
- Evaluate internal fast-path transformations that drop or compact `managedFields` before fanout to watch clients that opt in.
- Benchmark memory tradeoff of short-lived compressed object representations in cache fanout path.

## Phase 3: managedFields Representation Improvements (8-16+ weeks)
- Prototype compact storage format for field ownership sets.
  - Goals: lower in-memory overhead, fast merge conflict checks, safe conversion.
- Evaluate bounded-history/TTL-like strategies for old update-manager entries beyond current cap behavior.
- Explore schema-aware canonicalization/dedup of repeated fieldset shapes.

## Phase 4: Standardization and Rollout
- KEP-level proposal for wire + in-memory behavior changes if API-visible.
- Compatibility matrix across:
  - SSA clients/controllers.
  - CRDs and version migration paths.
  - kubectl and ecosystem tools.
- Canary rollout gates:
  - Opt-in flag -> beta default-on -> GA.

## Risks and Constraints
- Ownership semantics are correctness-critical; dropping detail can break conflict behavior.
- Compression can shift pressure from memory to CPU and latency.
- Cache-path changes can affect watch behavior and consistency expectations.
- CRD/version conversion edge cases in managedFields must remain safe.

## Concrete Next Actions
1. Run the local repro plan and capture baseline profiles.
2. Implement and test read-path omission/trim experiments.
3. Quantify memory saved per resource under each experiment.
4. Select 1-2 upstreamable proposals with strongest cost/benefit.
