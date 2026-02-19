# Compression and Caching Options for managedFields

## Clarifying the compression question
- Wire compression (`gzip`) reduces response bytes, not necessarily apiserver heap.
- To reduce heap, compression/compaction must apply to in-memory or cached representations.

## Option A: Read-path omission (strong near-term candidate)
Idea:
- Add/extend API options to omit managedFields on GET/LIST/WATCH when callers do not need ownership metadata.

Pros:
- Directly reduces payload and downstream informer memory.
- Low semantic risk if opt-in and default-compatible.

Cons:
- Clients needing ownership data must opt out of omission.

## Option B: Cache-level compact representation
Idea:
- Store managedFields in compact binary/compressed form in cache and decode lazily only when requested.

Pros:
- Direct RSS savings for watch cache and other in-memory layers.

Cons:
- CPU overhead for decode paths.
- Complexity/risk in mutability, deep copy, and fanout code.

## Option C: Canonicalization + dedup of fieldset structures
Idea:
- Intern/deduplicate repeated ownership shapes across objects of same kind.

Pros:
- Good fit when many similar controllers manage similar paths.

Cons:
- Requires careful lifecycle management to avoid leaks.
- Complex invalidation rules across versions/schemas.

## Option D: Churn reduction
Idea:
- Avoid no-op metadata ownership/timestamp mutations where semantically possible.

Pros:
- Fewer write events and less cache churn.

Cons:
- Must preserve ownership correctness and observable API invariants.

## Option E: Bounded retention policies for old ownership details
Idea:
- Strengthen policies around old update-manager details beyond current caps.

Pros:
- Controls worst-case entry growth.

Cons:
- Potential diagnosability and conflict-resolution behavior changes if too aggressive.

## Recommendation
- Start with Option A + D for near-term risk/benefit.
- Evaluate B/C with focused prototypes and strict perf+correctness test gates.
