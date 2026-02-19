# Benchmark Matrix for SSA Memory Investigation

## Dimensions
1. Object count: `1k`, `10k`, `25k`, `50k`
2. Managers per object: `1`, `3`, `5`, `10+`
3. Field breadth per object:
- narrow (few keys)
- medium
- wide (many keys/nested paths)
4. Operation mix:
- apply-heavy
- update-heavy
- mixed with list/watch traffic
5. Resource kind:
- built-in (`ConfigMap`, `Deployment`)
- CRD with nested schema

## Test Matrix
| ID | Count | Managers | Breadth | Mix | Kind | Expected stress |
|---|---:|---:|---|---|---|---|
| A1 | 10k | 1 | narrow | apply | ConfigMap | low-moderate |
| A2 | 10k | 5 | medium | apply | ConfigMap | metadata growth |
| A3 | 25k | 5 | wide | apply | ConfigMap | high managedFields size |
| B1 | 25k | 3 | medium | mixed | Deployment | cache/watch pressure |
| B2 | 50k | 3 | medium | mixed | ConfigMap | high cardinality |
| C1 | 10k | 5 | wide | apply | CRD | conversion+fieldset stress |
| C2 | 25k | 5 | wide | mixed | CRD | max stress scenario |

## Output Template
For each run record:
- peak apiserver RSS
- steady RSS 10m after delete
- apply p95/p99 latency
- list/watch p95 latency
- heap top retainers
- notable regression notes

## Comparison Table (to fill)
| Run | Mitigation | Peak RSS | Steady RSS | Apply p99 | Watch/List p95 | Notes |
|---|---|---:|---:|---:|---:|---|
| baseline | none |  |  |  |  |  |
| m1 | omit managedFields reads |  |  |  |  |  |
| m2 | churn reduction |  |  |  |  |  |
| m3 | cache compaction prototype |  |  |  |  |  |
