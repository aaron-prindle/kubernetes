# SSA ManagedFields Memory Bottleneck - Research Artifacts

This directory contains a comprehensive investigation of Server-Side Apply (SSA), managedFields overhead, and kube-apiserver memory bottlenecks in large Kubernetes clusters.

## Key Finding

`managedFields` is a high-confidence contributor to memory and payload overhead in large clusters. Public evidence and local repro both show it can be a substantial fraction of object/caching cost, though exact percentages are workload-dependent.

As of 2026-02-13:
- targeted mitigations are merged in specific surfaces (for example audit omission, client/component trimming),
- active upstream work exists on read-path omission (`#136760`),
- no merged comprehensive end-to-end server-side solution exists yet.

## Artifact Index

### Top-Level Plans
| File | Description |
|------|-------------|
| [plan.md](plan.md) | Overall investigation plan with phases and status |
| [future-plan.md](future-plan.md) | Detailed multi-phase plan for fixing the issue |

### Core Research Documents (Numbered Series)
| File | Description | Lines |
|------|-------------|-------|
| [01-ssa-overview.md](01-ssa-overview.md) | How Server Side Apply works end-to-end | 166 |
| [02-managed-fields-deep-dive.md](02-managed-fields-deep-dive.md) | Deep dive into managedFields data structures and memory | 231 |
| [03-apiserver-caching-architecture.md](03-apiserver-caching-architecture.md) | How the apiserver caches objects and where memory goes | 235 |
| [04-memory-bottleneck-analysis.md](04-memory-bottleneck-analysis.md) | Why SSA causes apiserver memory issues at scale | 206 |
| [05-online-research-findings.md](05-online-research-findings.md) | Compiled findings from 18+ public resources | 193 |
| [06-solution-proposals.md](06-solution-proposals.md) | 7 L7-level solution proposals with implementation details | 448 |
| [07-local-repro-plan.md](07-local-repro-plan.md) | Plan for reproducing the issue with a local kind cluster | 579 |
| [08-key-code-paths.md](08-key-code-paths.md) | Annotated code paths through the SSA system | 376 |
| [09-memory-flow-diagram.md](09-memory-flow-diagram.md) | Visual diagrams of memory copies and flow | 173 |
| [10-references.md](10-references.md) | Complete reference list with all URLs | 83 |

### Codebase Analysis (from exploration agents)
| File | Description |
|------|-------------|
| [codebase-analysis/ssa-request-flow.md](codebase-analysis/ssa-request-flow.md) | SSA request flow through the codebase |
| [codebase-analysis/managedfields-lifecycle-and-size-controls.md](codebase-analysis/managedfields-lifecycle-and-size-controls.md) | Lifecycle and size controls for managedFields |
| [codebase-analysis/apiserver-cache-memory-evidence.md](codebase-analysis/apiserver-cache-memory-evidence.md) | Evidence of cache memory patterns |

### Performance Hypotheses
| File | Description |
|------|-------------|
| [performance-hypotheses/why-ssa-can-bottleneck-large-clusters.md](performance-hypotheses/why-ssa-can-bottleneck-large-clusters.md) | Analysis of bottleneck mechanisms |
| [performance-hypotheses/managedfields-problem-analysis.md](performance-hypotheses/managedfields-problem-analysis.md) | Root cause analysis |
| [performance-hypotheses/compression-and-caching-options.md](performance-hypotheses/compression-and-caching-options.md) | Compression and caching possibilities |

### Public Sources
| File | Description |
|------|-------------|
| [public-sources/ssa-docs-and-kep-notes.md](public-sources/ssa-docs-and-kep-notes.md) | KEP notes and documentation |
| [public-sources/ssa-blog-timeline.md](public-sources/ssa-blog-timeline.md) | Timeline of SSA blog posts |
| [public-sources/issues-and-prs-managedfields.md](public-sources/issues-and-prs-managedfields.md) | GitHub issues and PRs |

### Experiments
| File | Description |
|------|-------------|
| [experiments/local-cluster-repro-plan.md](experiments/local-cluster-repro-plan.md) | Local cluster reproduction plan |
| [experiments/benchmark-matrix.md](experiments/benchmark-matrix.md) | Benchmark test matrix |
| [experiments/observability-playbook.md](experiments/observability-playbook.md) | Observability and profiling guide |

### Recommendations
| File | Description |
|------|-------------|
| [recommendations/l7-solutions.md](recommendations/l7-solutions.md) | L7-level solution details |
| [recommendations/prioritized-roadmap.md](recommendations/prioritized-roadmap.md) | Prioritized implementation roadmap |

## Quick Summary of Proposed Solutions

| Priority | Solution | Expected Savings | Complexity |
|----------|----------|-----------------|------------|
| 1 | Server-side managedFields exclusion parameter | scenario-dependent | Medium |
| 2 | Reduce no-op metadata churn | scenario-dependent | Medium |
| 3 | Compress FieldsV1.Raw in watch cache | scenario-dependent | Medium |
| 4 | Strip managedFields from watch cache entirely (prototype) | potentially high | High |
| 5 | Binary FieldsV1 encoding (FieldsV2) | scenario-dependent | High |
| 6 | FieldsV1 deduplication pool | scenario-dependent | High |
| 7 | Lazy-load managedFields from etcd | scenario-dependent | Very High |

## Total Artifacts: 28 files across 6 directories (~3,300 lines of research)
