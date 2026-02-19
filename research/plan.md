# SSA ManagedFields Memory Bottleneck - Investigation Plan

## Objective
Investigate why Server Side Apply (SSA) causes apiserver memory to scale poorly with the number of resources, specifically how `managedFields` contributes to excessive memory consumption in the apiserver's watch cache, and propose solutions.

## Phase 1: Research & Understanding (COMPLETED)

### 1.1 Online Research
- [x] Collect blog posts, KEPs, and GitHub issues about SSA performance
- [x] Study kube.rs memory optimization findings (managed fields ~50% of metadata)
- [x] Review KEP-1152 (Less Object Serializations) for caching architecture
- [x] Study API Streaming (KEP-3157/WatchList) for memory improvements
- [x] Review CBOR serialization KEP-4222 for encoding efficiency

### 1.2 Codebase Exploration
- [x] Understand ManagedFieldsEntry type and FieldsV1 storage format
- [x] Trace the Apply handler flow through fieldmanager chain
- [x] Understand structured-merge-diff Set operations and memory cost
- [x] Study watch cache architecture (watchCache, cachingObject, store)
- [x] Understand how objects are serialized and cached for watchers
- [x] Review existing optimizations (capmanagers, stripMeta, timestamp equality)

### 1.3 Key Findings Documentation
- [x] Document the full SSA data flow from request to storage
- [x] Document memory multiplication factors in the cache
- [x] Quantify managedFields as percentage of object size
- [x] Identify all places managedFields data is duplicated

## Phase 2: Bottleneck Analysis (COMPLETED)

### 2.1 Identify Memory Hotspots
- [x] FieldsV1.Raw byte slices in each ManagedFieldsEntry
- [x] Multiple serialization formats cached per object (JSON, Protobuf, CBOR)
- [x] Watch cache event buffer storing full objects with managedFields
- [x] Deep copy costs during watch event dispatch
- [x] ToFieldSet() allocation cost during every Apply operation

### 2.2 Quantify Impact at Scale
- [x] Estimate managedFields overhead per typical object (2-50KB)
- [x] Project memory impact at 5,000 node / 100,000+ object clusters
- [x] Identify the O(managers * fields * objects) scaling problem

## Phase 3: Solution Design (COMPLETED)

### 3.1 Identify Solution Categories
- [x] Lazy loading of managedFields (don't deserialize until needed)
- [x] Compression of managedFields in cache
- [x] Stripping managedFields from watch cache / serving them on-demand
- [x] More efficient encoding of FieldsV1 data
- [x] Deduplication of common field set patterns
- [x] Server-side filtering to exclude managedFields from responses

### 3.2 Evaluate Solutions
- [x] Assess API compatibility constraints
- [x] Evaluate implementation complexity
- [x] Estimate memory savings potential
- [x] Consider CPU tradeoffs

## Phase 4: Reproduction Environment (IN PROGRESS)

### 4.1 Local Cluster Setup
- [ ] Create kind cluster configuration for testing
- [ ] Write scripts to generate large numbers of objects with SSA
- [ ] Instrument apiserver memory profiling
- [ ] Establish baseline measurements

### 4.2 Benchmarking
- [ ] Measure apiserver RSS with varying object counts
- [ ] Measure managedFields contribution via heap profiling
- [ ] Compare with/without managedFields stripping
- [ ] Test compression approaches

## Phase 5: Implementation (FUTURE)

### 5.1 Prototype Solutions
- [ ] Implement most promising solution from Phase 3
- [ ] Run against reproduction environment
- [ ] Measure memory savings
- [ ] Validate API compatibility

### 5.2 Upstream Preparation
- [ ] Write KEP draft
- [ ] Create proof-of-concept PR
- [ ] Gather benchmark data
- [ ] Present to sig-api-machinery

## Artifacts Produced
| File | Description |
|------|-------------|
| `plan.md` | This file - overall investigation plan |
| `future-plan.md` | Detailed future plan for fixing the issue |
| `01-ssa-overview.md` | How Server Side Apply works |
| `02-managed-fields-deep-dive.md` | Deep dive into managedFields data structures |
| `03-apiserver-caching-architecture.md` | How the apiserver caches objects |
| `04-memory-bottleneck-analysis.md` | Analysis of why SSA causes memory issues |
| `05-online-research-findings.md` | Compiled findings from public resources |
| `06-solution-proposals.md` | L7-level solution proposals |
| `07-local-repro-plan.md` | Plan for reproducing the issue locally |
| `08-key-code-paths.md` | Annotated code paths through the SSA system |
