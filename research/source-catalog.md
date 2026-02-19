# SSA Research Source Catalog

## Authoritative Docs
1. Server-Side Apply docs (Kubernetes)
- URL: https://kubernetes.io/docs/reference/using-api/server-side-apply/
- Why relevant: canonical semantics for field ownership, conflicts, and managed fields.

2. Kubernetes API Concepts
- URL: https://kubernetes.io/docs/reference/using-api/api-concepts/
- Why relevant: response compression, watch/list semantics, watch cache behavior implications for memory and payload size.

3. SSA KEP (KEP-555)
- URL: https://raw.githubusercontent.com/kubernetes/enhancements/master/keps/sig-api-machinery/555-server-side-apply/README.md
- Why relevant: explicit discussion of managedFields size overhead and resource usage impact.

4. structured-merge-diff repository README
- URL: https://raw.githubusercontent.com/kubernetes-sigs/structured-merge-diff/master/README.md
- Why relevant: underlying data model and apply algorithm used by SSA internals.

## Kubernetes Blog Posts
1. Kubernetes 1.18 feature: Server-Side Apply beta 2 (2020-04-01)
- URL: https://kubernetes.io/blog/2020/04/01/kubernetes-1.18-feature-server-side-apply-beta-2/

2. Server-Side Apply GA (2021-08-06)
- URL: https://kubernetes.io/blog/2021/08/06/server-side-apply-ga/

3. Advanced Server-Side Apply (2022-10-20)
- URL: https://kubernetes.io/blog/2022/10/20/advanced-server-side-apply/

4. Live and let live with Kluctl and SSA (2022-11-04)
- URL: https://kubernetes.io/blog/2022/11/04/live-and-let-live-with-kluctl-and-ssa/

5. API streaming efficiency (2024-12-17)
- URL: https://kubernetes.io/blog/2024/12/17/kube-apiserver-api-streaming/
- Why included: not SSA-specific, but directly relevant to API server memory and large-object response behavior.

## Public Issues / PRs
1. `managedFields` verbosity pain (#90066)
- URL: https://github.com/kubernetes/kubernetes/issues/90066

2. `kubectl get` strips managed fields (#96878)
- URL: https://github.com/kubernetes/kubernetes/pull/96878

3. Omit managed fields from audit entries (#94986)
- URL: https://github.com/kubernetes/kubernetes/pull/94986

4. No-op SSA updates managedFields timestamps (#131175)
- URL: https://github.com/kubernetes/kubernetes/issues/131175

5. Scheduler trim fix for managedFields (#131016)
- URL: https://github.com/kubernetes/kubernetes/pull/131016

6. Open WIP: omit managed fields in get/list (#136760)
- URL: https://github.com/kubernetes/kubernetes/pull/136760

## Local Code References (this repo)
1. SSA patch handler path
- `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go`

2. Field manager orchestration
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/fieldmanager.go`
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fieldmanager.go`

3. managedFields encoding/decoding
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go`

4. structured merge and updater
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/structuredmerge.go`
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfieldsupdater.go`
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go`

5. Cache + memory-relevant code
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go`
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go`

6. Existing in-tree explicit mitigation
- `pkg/scheduler/scheduler.go` (trim managedFields in informer transform)

7. Integration tests showing size pressure
- `test/integration/apiserver/apply/apply_test.go`
