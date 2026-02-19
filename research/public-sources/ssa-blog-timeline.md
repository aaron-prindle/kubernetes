# SSA Blog Timeline and Implications

## 2020-04-01: SSA Beta 2
Source: https://kubernetes.io/blog/2020/04/01/kubernetes-1.18-feature-server-side-apply-beta-2/

Notes:
- SSA maturity progression; ownership/conflict model became practical for wider use.
- Established behavior that field ownership is tracked server-side.

Implication:
- Earlier adoption period already exposed managedFields verbosity and size concerns in ecosystem feedback.

## 2021-08-06: SSA GA (Kubernetes 1.22)
Source: https://kubernetes.io/blog/2021/08/06/server-side-apply-ga/

Notes:
- SSA marked GA and recommended for controllers.
- Broad production usage increased the total cardinality of objects carrying managedFields.

Implication:
- Memory and payload costs move from edge-cases to cluster-wide baseline overhead at scale.

## 2022-10-20: Advanced SSA
Source: https://kubernetes.io/blog/2022/10/20/advanced-server-side-apply/

Notes:
- Deep usage guidance and ownership best practices.
- Encourages robust apply-centric workflows.

Implication:
- Better SSA correctness and collaboration, but potentially increased managedFields churn if many controllers repeatedly apply.

## 2022-11-04: Kluctl + SSA experience
Source: https://kubernetes.io/blog/2022/11/04/live-and-let-live-with-kluctl-and-ssa/

Notes:
- Real-world multi-manager workflows highlighted.

Implication:
- Multi-manager ownership is valuable, but expands metadata complexity and potential per-object overhead.

## 2024-12-17: API Streaming efficiency
Source: https://kubernetes.io/blog/2024/12/17/kube-apiserver-api-streaming/

Why this matters for SSA research:
- Not SSA-specific, but directly related to API server memory behavior with large responses and large objects.
- Demonstrates active upstream work to avoid expensive full-response materialization patterns.

Implication:
- Any managedFields memory strategy should align with broader API efficiency work: stream earlier, allocate less, fan out lighter payloads where possible.
