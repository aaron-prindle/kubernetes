# Server Side Apply (SSA) - Comprehensive Overview

## What is Server Side Apply?

Server Side Apply (SSA) is a Kubernetes feature that moves the logic for `kubectl apply` from the client to the server. It enables declarative object management with proper field ownership tracking through a mechanism called **managed fields**.

Unlike client-side apply which uses the `kubectl.kubernetes.io/last-applied-configuration` annotation, SSA uses the `managedFields` metadata to track which fields each manager (user, controller, or tool) owns.

## How SSA Works - Request Flow

```
Client                    API Server                     etcd
  |                          |                              |
  |-- PATCH (apply) -------->|                              |
  |  Content-Type:           |                              |
  |  application/apply-patch |                              |
  |  ?fieldManager=foo       |                              |
  |                          |-- Read current object ------>|
  |                          |<-- Current object + mf ------|
  |                          |                              |
  |                          | 1. Decode managedFields      |
  |                          | 2. Convert patch to typed    |
  |                          | 3. Extract fieldset from     |
  |                          |    patch (ToFieldSet())      |
  |                          | 4. Detect conflicts with     |
  |                          |    other managers            |
  |                          | 5. Merge patch into object   |
  |                          | 6. Update managedFields      |
  |                          | 7. Encode back to JSON       |
  |                          |                              |
  |                          |-- Write updated object ----->|
  |                          |<-- Confirmation -------------|
  |                          |                              |
  |                          |-- Notify watch cache         |
  |                          |   (full object + mf)         |
  |<-- Response -------------|                              |
```

## Key Components

### 1. ManagedFieldsEntry (types.go:1340)
Every Kubernetes object's metadata contains a `managedFields` array. Each entry tracks:
- **Manager**: Who owns these fields (e.g., "kubectl-client-side-apply", "kube-controller-manager")
- **Operation**: "Apply" or "Update"
- **APIVersion**: The API version the fields apply to
- **Time**: When the entry was last modified
- **FieldsV1**: A trie-based representation of all fields this manager owns
- **Subresource**: Which subresource (if any) this applies to

### 2. FieldsV1 Format
FieldsV1 stores field ownership as a JSON-serialized trie. For example:
```json
{
  "f:metadata": {
    "f:labels": {
      "f:app": {}
    }
  },
  "f:spec": {
    "f:replicas": {},
    "f:selector": {
      "f:matchLabels": {
        "f:app": {}
      }
    },
    "f:template": {
      "f:spec": {
        "f:containers": {
          "k:{\"name\":\"nginx\"}": {
            "f:image": {},
            "f:ports": {
              "k:{\"containerPort\":80,\"protocol\":\"TCP\"}": {
                ".": {},
                "f:containerPort": {}
              }
            }
          }
        }
      }
    }
  }
}
```

Key notation:
- `f:<name>` - a named field in a struct
- `k:<keys>` - a list item identified by its key fields
- `v:<value>` - a list item identified by its value
- `.` - the field itself

### 3. FieldManager Chain (fieldmanager.go:57-74)
The Apply/Update operations pass through a chain of manager wrappers:

```
VersionCheckManager
  -> LastAppliedUpdater
    -> LastAppliedManager
      -> ProbabilisticSkipNonAppliedManager
        -> CapManagersManager (max 10 update managers)
          -> BuildManagerInfoManager
            -> ManagedFieldsUpdater
              -> StripMetaManager
                -> StructuredMergeManager (core merge logic)
```

Each wrapper adds specific behavior:
- **StripMetaManager**: Removes system fields (apiVersion, kind, metadata.name, etc.) from tracked fields
- **ManagedFieldsUpdater**: Updates timestamps and merges same-manager entries
- **CapManagersManager**: Limits update managers to 10, merges oldest into "ancient-changes"
- **ProbabilisticSkipNonAppliedManager**: Tracks fields from object creation
- **LastAppliedManager**: Manages last-applied-configuration compatibility
- **VersionCheckManager**: Validates API version compatibility

### 4. Structured Merge Diff
The core merge algorithm (from `sigs.k8s.io/structured-merge-diff`):

```go
func (s *Updater) Apply(liveObject, configObject *typed.TypedValue, ...) {
    // 1. Reconcile managed fields with schema changes
    managers = s.reconcileManagedFieldsWithSchemaChanges(liveObject, managers)

    // 2. Merge config into live object
    newObject = liveObject.Merge(configObject)

    // 3. Extract field set from applied config (EXPENSIVE)
    set = configObject.ToFieldSet()

    // 4. Update manager's field set
    managers[manager] = NewVersionedSet(set, version, true)

    // 5. Prune fields removed by this manager
    newObject = s.prune(newObject, managers, manager, lastSet)

    // 6. Update conflict tracking
    managers = s.update(liveObject, newObject, version, managers, manager, force)
}
```

## Why SSA Exists

Before SSA, Kubernetes used three approaches for declarative management:
1. **kubectl apply (client-side)**: Uses `last-applied-configuration` annotation to compute 3-way merge
2. **Strategic Merge Patch**: Merges based on schema-defined patch strategies
3. **JSON Merge Patch**: Simple RFC 7386 merge

Problems with client-side apply:
- Annotation bloat (`last-applied-configuration` can be very large)
- No conflict detection between different managers
- Client must have full schema knowledge
- Race conditions between concurrent updaters

SSA solves these by:
- Moving merge logic to the server (authoritative schema knowledge)
- Tracking per-field ownership to detect conflicts
- Enabling multiple managers to safely co-own an object
- Providing force-override semantics for conflict resolution

## The Cost of SSA

The trade-off is that every object now carries additional metadata:
- Each manager has a `ManagedFieldsEntry` with a complete `FieldsV1` trie
- A typical object might have 3-11 managers (Update capped at 10, Apply uncapped)
- The `FieldsV1` data is essentially a copy of the object's structure (keys without values)
- This metadata is stored in etcd, cached in the watch cache, and sent over the wire

**This is the root cause of the memory bottleneck we're investigating.**
