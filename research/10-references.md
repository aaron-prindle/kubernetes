# Complete Reference List

## Official Kubernetes Documentation
- [Server-Side Apply Documentation](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
- [Large Cluster Best Practices](https://kubernetes.io/docs/setup/best-practices/cluster-large/)

## Kubernetes Blog Posts
- [Server Side Apply Is Great And You Should Be Using It (2022)](https://kubernetes.io/blog/2022/10/20/advanced-server-side-apply/)
- [Server Side Apply GA (2021)](https://kubernetes.io/blog/2021/08/06/server-side-apply-ga/)
- [Server-Side Apply Beta 2 in 1.18 (2020)](https://kubernetes.io/blog/2020/04/01/kubernetes-1.18-feature-server-side-apply-beta-2/)
- [API Streaming in Kubernetes 1.32 (2024)](https://kubernetes.io/blog/2024/12/17/kube-apiserver-api-streaming/)
- [Snapshottable API Server Cache v1.34 (2025)](https://kubernetes.io/blog/2025/09/09/kubernetes-v1-34-snapshottable-api-server-cache/)
- [Consistent Read from Cache Beta (2024)](https://kubernetes.io/blog/2024/08/15/consistent-read-from-cache-beta/)
- [Streaming List Responses v1.33 (2025)](https://kubernetes.io/blog/2025/05/09/kubernetes-v1-33-streaming-list-responses/)

## KEPs (Kubernetes Enhancement Proposals)
- [KEP-555: Server-Side Apply](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/555-server-side-apply/README.md)
- [KEP-1152: Less Object Serializations](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1152-less-object-serializations)
- [KEP-3157: Watch List / Stream-Based Data Priming](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/3157-watch-list/README.md)
- [KEP-4222: CBOR Serializer](https://github.com/kubernetes/enhancements/issues/4222)
- [KEP-4988: Snapshottable API Server Cache](https://github.com/kubernetes/enhancements/issues/4988)
- [KEP-2340: Consistent Reads from Cache](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/2340-Consistent-reads-from-cache/README.md)

## GitHub Issues - Memory and Performance
- [#76219: SSA Protobuf Serialization Performance](https://github.com/kubernetes/kubernetes/issues/76219) - 2.5x response size with SSA
- [#102259: API Server Memory Spikes (watches)](https://github.com/kubernetes/kubernetes/issues/102259) - 59.72% of memory from ObjectMetaFieldsSet
- [#90066: managedFields Verbosity](https://github.com/kubernetes/kubernetes/issues/90066) - 55% of YAML output
- [#89080: Massive managedFields After 1.18 Upgrade](https://github.com/kubernetes/kubernetes/issues/89080)
- [#90179: More Memory Efficient Watch Cache](https://github.com/kubernetes/kubernetes/issues/90179)
- [#97262: Optimize Memory Usage (2000 nodes)](https://github.com/kubernetes/kubernetes/issues/97262)
- [#114276: Memory Buildup from LIST Requests](https://github.com/kubernetes/kubernetes/issues/114276)
- [#98423: High Memory on Pending Pods Storm](https://github.com/kubernetes/kubernetes/issues/98423) - 90GB spike
- [#111699: Higher Memory in 1.21 vs 1.20](https://github.com/kubernetes/kubernetes/issues/111699)
- [#65732: Unbounded Memory Growth](https://github.com/kubernetes/kubernetes/issues/65732)
- [#115699: KCM High Memory from Full Object Caching](https://github.com/kubernetes/kubernetes/issues/115699)
- [#82292: Large CRDs Go Over Size Limits](https://github.com/kubernetes/kubernetes/issues/82292)
- [#124680: Watch for CRs Costs 10-15x More Memory](https://github.com/kubernetes/kubernetes/issues/124680)

## GitHub PRs - Optimizations
- [#91946: Strip managedFields from kubectl edit](https://github.com/kubernetes/kubernetes/pull/91946)
- [#96878: Strip managedFields from kubectl get](https://github.com/kubernetes/kubernetes/pull/96878)
- [#94986: Drop managedFields from Audit Entries](https://github.com/kubernetes/kubernetes/pull/94986)
- [#77355: Fix SSA Protobuf Serialization (memoization)](https://github.com/kubernetes/kubernetes/pull/77355)
- [#78742: GC/Quota Using Metadata Client](https://github.com/kubernetes/kubernetes/pull/78742)
- [#115700: Metadata Informers for KCM](https://github.com/kubernetes/kubernetes/pull/115700)
- [#84043: Don't Use CachingObject if Few Watchers](https://github.com/kubernetes/kubernetes/pull/84043)

## Third-Party Research and Blog Posts
- [kube.rs: Controller Optimization Guide](https://kube.rs/controllers/optimization/)
- [kube.rs: Watcher Memory Improvements](https://kube.rs/blog/2024/06/11/watcher-memory-improvements/)
- [Kubernetes Controllers at Scale (Medium)](https://medium.com/@timebertt/kubernetes-controllers-at-scale-clients-caches-conflicts-patches-explained-aa0f7a8b4332)
- [Diving into Kubernetes Watch Cache (Pierre Zemb)](https://pierrezemb.fr/posts/diving-into-kubernetes-watch-cache/)
- [Google: Server-Side Apply in Kubernetes (2021)](https://opensource.googleblog.com/2021/10/server-side-apply-in-kubernetes.html)
- [Google: 130,000-Node GKE Cluster](https://cloud.google.com/blog/products/containers-kubernetes/how-we-built-a-130000-node-gke-cluster/)
- [Kubernetes Apply: Client-Side vs. Server-Side](https://support.tools/kubernetes-apply-client-side-vs-server-side/)
- [Datadog: Managing etcd Storage](https://www.datadoghq.com/blog/managing-etcd-storage/)

## Libraries and Packages
- [controller-runtime cache package (TransformStripManagedFields)](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache)
- [structured-merge-diff repository](https://github.com/kubernetes-sigs/structured-merge-diff)

## Red Hat Bug Reports
- [#1323733: etcd Cache in kube-apiserver Fixed Size High Memory](https://bugzilla.redhat.com/show_bug.cgi?id=1323733)
- [#1953305: kube-apiserver Memory Utilization Analysis](https://bugzilla.redhat.com/show_bug.cgi?id=1953305)

## Key Codebase Files
| File | Purpose |
|------|---------|
| `staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go` | ManagedFieldsEntry, FieldsV1 types |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fieldmanager.go` | FieldManager core |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go` | Encode/Decode |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fields.go` | FieldsToSet/SetToFields |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go` | Manager count limit |
| `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/stripmeta.go` | Strip system fields |
| `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go` | Apply handler |
| `staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go` | Watch cache |
| `staging/src/k8s.io/apiserver/pkg/storage/cacher/caching_object.go` | Serialization cache |
| `staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go` | Cache dispatcher |
| `staging/src/k8s.io/apiserver/pkg/storage/cacher/store/store_btree.go` | BTree store |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/fieldpath/set.go` | Set operations |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/fieldpath/serialize.go` | JSON serialization |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/merge/update.go` | Merge algorithm |
| `vendor/sigs.k8s.io/structured-merge-diff/v6/typed/tofieldset.go` | ToFieldSet |
