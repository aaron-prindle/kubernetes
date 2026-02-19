# Local Cluster Repro Plan: SSA Memory Bottleneck

## Goal
Reproduce memory pressure attributable to SSA/managedFields and measure impact of mitigation ideas.

## Setup Options
## Option 1: Fast iteration with kind
- Create 1 control-plane kind cluster.
- Enable metrics-server and pprof access for kube-apiserver.
- Use high-cardinality synthetic resources (ConfigMaps + CRDs optional).

## Option 2: Upstream realism with `local-up-cluster.sh`
- Use this repository's local cluster tooling.
- Easier to test apiserver code changes directly.

## Workload Shape
1. Baseline object population
- Create N objects (e.g., 10k/25k/50k) with moderate field breadth.

2. SSA manager fanout
- Apply from multiple field managers (`manager-a` ... `manager-k`) on overlapping/non-overlapping fields.

3. Churn loop
- Repeat no-op-ish apply and partial updates to generate metadata churn.

4. Read pressure
- Concurrent LIST/WATCH clients for hot resources.

## Suggested Test Resource
- `ConfigMap` for fast high-count creation.
- Optional CRD with nested maps/lists to stress fieldset complexity.

## Measurements
- kube-apiserver RSS over time.
- Heap profiles:
  - before load,
  - peak load,
  - post-delete steady state.
- Request metrics split by verbs (`APPLY`, `PATCH`, `LIST`, `WATCH`).
- Object serialized size sample (with and without managedFields).

## Validation Steps
1. Run baseline without mitigation.
2. Enable one mitigation at a time:
- trim/omit managedFields in selected clients,
- reduce no-op apply loops,
- test read-path omission prototype if available.
3. Compare:
- peak RSS,
- steady-state RSS after delete,
- apply latency p95/p99,
- GC pause/CPU impact.

## Pass/Fail Criteria
- Pass if mitigation reduces peak/steady RSS materially (target: >=15-30% on managedFields-heavy scenarios) without unacceptable latency regressions.
- Fail if memory gains are negligible or latency/CPU costs exceed operational budgets.

## Minimal Command Skeleton (example)
```bash
# 1) create objects (non-SSA)
kubectl create namespace ssa-lab
for i in $(seq 1 20000); do
  kubectl -n ssa-lab create configmap cm-$i --from-literal=k=v --dry-run=client -o yaml | kubectl apply -f - >/dev/null
done

# 2) multi-manager SSA churn
for m in manager-a manager-b manager-c manager-d manager-e; do
  for i in $(seq 1 20000); do
    cat <<YAML | kubectl -n ssa-lab apply --server-side --field-manager=$m -f - >/dev/null
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-$i
  labels:
    owner-$m: "true"
YAML
  done
done
```

Note:
- The shell loops above are intentionally simple and can be parallelized for faster stress generation.
