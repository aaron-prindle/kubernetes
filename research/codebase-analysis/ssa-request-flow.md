# SSA Request Flow in `kubernetes/kubernetes`

## Entry Point: apply patch handler
Primary file:
- `staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go`

Flow:
1. Request enters `PatchResource`.
2. For `application/apply-patch+yaml` (or CBOR apply), handler builds `applyPatcher`.
3. `applyPatcher.applyPatchToCurrentObject()` unmarshals patch into unstructured object.
4. Calls `fieldManager.Apply(obj, patchObj, fieldManagerName, force)`.

Key detail:
- Apply path requires field manager; panic if not installed.
- Non-apply patch/update paths call `UpdateNoErrors` and also maintain managed fields.

## Field manager stack composition
Primary file:
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/fieldmanager.go`

`NewDefaultFieldManager(...)` wraps managers in layers:
- version checks,
- last-applied updater/interop logic,
- probabilistic tracking behavior,
- manager info builder,
- update manager capping (`DefaultMaxUpdateManagers = 10`),
- timestamp updater,
- structured merge core.

Interpretation:
- SSA is not a single function call; it is a pipeline with multiple allocations/transforms and metadata bookkeeping.

## Structured merge core
Primary file:
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/structuredmerge.go`

Key operations:
- Convert objects to versioned forms.
- Convert runtime objects into typed structured-merge-diff objects.
- Run `merge.Updater.Apply` / `merge.Updater.Update`.
- Convert result back, default, then return unversioned object.

Performance implication:
- CPU and allocations include conversion + typed representation + merge operation + back conversion.
- Object size (including metadata) increases conversion and copy costs.

## Where managedFields are encoded/decoded
Primary file:
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/managedfields.go`

Lifecycle:
- Decode API `[]ManagedFieldsEntry` into internal map/set structure.
- Merge/update via structured merge.
- Encode back into `[]ManagedFieldsEntry` and attach to object metadata.

Performance implication:
- Every update/apply can incur decode + merge + encode cost.
- Larger `managedFields` entries increase this cost and object serialized size.

## Update manager capping behavior
Primary file:
- `staging/src/k8s.io/apimachinery/pkg/util/managedfields/internal/capmanagers.go`

Behavior:
- Update managers beyond cap are merged into version buckets (e.g. `ancient-changes`).
- Prevents unbounded count growth of update-manager entries.

Limitation:
- Capping number of entries does not cap fieldset size itself.
- Large ownership sets can still produce large `managedFields` payloads.
