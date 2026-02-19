# managedFields Lifecycle and Existing Size Controls

## A) Managed fields can be too large for object storage
In endpoint handlers, create/update/patch include retry logic:
- If storage returns object-too-large, the apiserver may clear `managedFields` and retry (non-apply fallback paths).

Code references:
- `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create.go`
- `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/update.go`
- `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go`

Interpretation:
- Upstream code explicitly acknowledges managedFields can push objects over serialized size limits.

## B) Integration tests validate size-limit behavior
File:
- `test/integration/apiserver/apply/apply_test.go`

Relevant tests include scenarios where:
- large objects are created near size limit,
- subsequent apply pushes object over limit and fails,
- many update managers are grouped/capped.

Interpretation:
- Size pressure from managedFields is test-covered, not hypothetical.

## C) Existing partial mitigation: cap update manager entries
File:
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go`

What it does:
- Limits number of update-manager entries retained.

What it does not do:
- Does not fundamentally compress ownership fieldset payloads.
- Does not reduce apply-entry complexity where field ownership is broad.

## D) Existing partial mitigation: trim managedFields in memory-sensitive paths
File:
- `pkg/scheduler/scheduler.go`

Behavior:
- Scheduler Pod informer transform removes `.metadata.managedFields` to reduce memory usage.

Interpretation:
- At least one core control-plane component intentionally discards managedFields in cache-heavy read path to improve memory profile.

## E) Timestamp churn can amplify updates
File/source:
- public issue `#131175`

Risk:
- No-op apply causing managedFields timestamp/resourceVersion changes can still trigger write churn and downstream cache updates.

Interpretation:
- Reducing unnecessary managedFields mutations may provide indirect memory and CPU gains by reducing event churn.
