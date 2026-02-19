# 07 — Risks and Mitigations

## Risk 1: Reference Count Bugs → Memory Leak or Use-After-Free

**Risk**: If Intern/Release calls don't balance, entries either leak (never freed)
or are freed while still referenced.

**Mitigation options**:

A) **Skip reference counting entirely.** Use periodic pool rebuild instead
   (described in 02-watch-cache-integration.md). The pool is rebuilt from scratch
   every N minutes by scanning the store. Old pool entries become garbage
   naturally. No Intern/Release bookkeeping needed.
   - Pro: Simple, impossible to get wrong
   - Con: Stale entries linger until rebuild; event buffer objects may not benefit

B) **Reference count but add a safety check.** Periodically verify that pool
   ref counts match actual object references (count FieldsV1.Raw pointers in
   the store that point to pool entries). Log warnings on mismatch. This is a
   debug/audit mechanism, not relied on for correctness.

C) **Use Go's garbage collector instead.** Don't track ref counts at all. Just
   store `[]byte` in the pool and let shared slice headers keep entries alive via
   GC. The pool holds `map[hash][]byte` and never deletes entries. Periodically,
   scan the pool for entries whose bytes are no longer referenced by any cached
   object (requires comparing pointers) and remove them.

**Recommendation**: Start with option A (periodic rebuild). It's the safest
approach for an initial implementation. Move to reference counting if the
periodic scan proves too expensive.

## Risk 2: Lock Contention on Hot Path

**Risk**: `processEvent()` is called for every object change. If the intern pool
lock becomes contended under high churn, it could slow down the watch cache.

**Analysis**: The critical section is small:
- Hash computation: ~50ns for 2 KB input (xxhash is very fast)
- RLock + map lookup + RUnlock: ~100ns
- Total: ~150ns per managedFields entry, ~450ns for 3 entries per object

For context, `processEvent` already does: store.Get, store.Update, lock/unlock
the watch cache mutex, broadcast on condition variable. Adding ~450ns is a
small fraction of the total ~5-10μs per processEvent call.

**Mitigation**: If profiling shows contention:
1. Shard the pool by hash prefix (16 shards = 16 independent locks)
2. Use `sync.Map` instead (good for read-heavy workloads, which this is)
3. Batch interning: collect all FieldsV1.Raw from an object, sort by hash,
   acquire lock once

## Risk 3: Hash Collision → Silent Data Corruption

**Risk**: Two different FieldsV1.Raw byte sequences with the same xxhash could
be incorrectly shared, assigning wrong field ownership to objects.

**Analysis**: xxhash64 collision probability: ~1 in 2^64 = 1 in 1.8×10^19.
Even with 1 million unique patterns, probability of any collision is ~2.7×10^-8
(basically zero).

**Mitigation**: Full byte comparison after hash match (`bytesEqual` guard in
the Intern function). This makes collisions impossible at the cost of one
extra comparison per cache hit. Since the comparison only happens when hashes
match (which means bytes are almost certainly equal), the branch predictor
handles this efficiently.

## Risk 4: Interned Bytes Mutated → Corruption Across Objects

**Risk**: Code somewhere mutates `FieldsV1.Raw` in-place after interning,
corrupting all objects sharing those bytes.

**Analysis**: See [05-deep-copy-safety.md](05-deep-copy-safety.md) for full
analysis. All consumer paths either deep-copy first or read-only access.

**Mitigation**: Tests that verify interned bytes are not modified after
object operations (deep copy, serialization, Apply).

## Risk 5: Increased Complexity for Future Changes

**Risk**: Future code changes to the managedFields path may not be aware of
interning and could introduce mutations.

**Mitigation**:
- Clear documentation in the pool code explaining the invariant
- Test in `TestWatchCacheInterningDoesNotCorruptData` that catches mutations
- The feature gate allows disabling if problems are discovered

## Risk 6: Memory Overhead of the Pool Itself

**Risk**: The pool's map and entries consume memory that could exceed the savings
for heterogeneous workloads.

**Analysis**: Per unique pattern:
- Map entry: ~80 bytes (hash + pointer + overhead)
- internEntry struct: ~40 bytes (slice header + refcount + padding)
- Total overhead: ~120 bytes per unique pattern

With 10,000 unique patterns: 1.2 MB of overhead.
With 100,000 unique patterns: 12 MB of overhead.

This is negligible compared to potential savings (hundreds of MB to GB).

For the worst case (every object has unique FieldsV1), the overhead is:
- 100,000 objects × 3 entries × 120 bytes = 36 MB overhead
- Savings: 0 (no deduplication)
- Net: -36 MB (slight regression)

**Mitigation**: Add a heuristic that disables interning for a resource type
if the dedup ratio falls below a threshold (e.g., if unique patterns > 50%
of total entries, stop interning for that resource). The metrics from
[04-metrics.md](04-metrics.md) provide the data for this decision.

## Summary

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Ref count bugs | High | Medium | Use periodic rebuild instead |
| Lock contention | Medium | Low | Shard pool if needed |
| Hash collision | Critical | Near-zero | Full byte comparison guard |
| Byte mutation | Critical | Near-zero | Analysis + tests confirm safety |
| Future changes unaware | Medium | Low | Documentation + tests |
| Pool overhead | Low | Low | Heuristic to disable when unhelpful |
