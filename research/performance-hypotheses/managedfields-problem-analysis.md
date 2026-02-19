# ManagedFields Problem Analysis

## What managedFields buys us
- Correct multi-manager collaboration semantics.
- Conflict detection and ownership transfer.
- Better declarative convergence for controllers and users.

## Why it hurts at scale
1. High cardinality
- Nearly all persisted objects can carry managed fields.

2. Non-trivial payload
- Entries include manager identity, version, operation, timestamp, and encoded fieldset.
- Fieldset can be large for broad resources or wide ownership.

3. Multiplication across system layers
- Stored in etcd payload.
- Read into apiserver object graph.
- Repeated in watch/list/cache paths.
- Propagated to clients/controllers unless omitted/trimmed.

4. Churn sensitivity
- Frequent apply/update operations can repeatedly touch ownership metadata.
- Even small metadata changes can trigger downstream event flow.

## Existing mitigations and gaps
Existing:
- cap update manager entries (`DefaultMaxUpdateManagers`).
- selective omission in some surfaces (`kubectl`, audit policy, scheduler transform).
- fallback behavior for too-large serialized objects in non-apply request handling.

Gaps:
- No universal read-path omit/compact option widely available by default.
- No built-in compact in-memory managedFields representation for hot caches.
- No clear upstream general solution yet for managedFields RAM footprint at very large scale.

## Key framing
The problem is not that SSA is incorrect; the problem is that correctness metadata is expensive when replicated across high-cardinality and high-churn control-plane paths.
