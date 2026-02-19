# 06 — Test Plan

## New Test Files

### 1. Unit Tests for Intern Pool

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/fieldsv1_intern_pool_test.go
```

**Tests to write:**

```go
func TestInternBasic(t *testing.T)
// Intern same bytes twice → same pointer returned
// Intern different bytes → different pointers

func TestInternRelease(t *testing.T)
// Intern, release → entry removed from pool
// Intern twice, release once → entry still present
// Intern twice, release twice → entry removed

func TestInternEmpty(t *testing.T)
// Empty/nil bytes → returned as-is, no pool entry

func TestInternConcurrent(t *testing.T)
// Many goroutines interning/releasing simultaneously
// Verify no races (run with -race)

func TestInternObject(t *testing.T)
// Create a Pod with managedFields, call InternObject
// Verify FieldsV1.Raw points to interned bytes
// Create second Pod with same managedFields → same pointer

func TestInternObjectDifferentManagers(t *testing.T)
// Two Pods with different managedFields
// Verify they get different interned entries

func TestInternStats(t *testing.T)
// Intern several entries, verify Stats() returns correct counts

func TestInternHashCollision(t *testing.T)
// Manually craft two different byte slices with same hash (hard to do with xxhash)
// Alternative: test the bytesEqual guard by mocking
// Verify different content is not incorrectly shared

func BenchmarkIntern(b *testing.B)
// Benchmark intern lookup for typical FieldsV1 sizes (500B, 2KB, 10KB)

func BenchmarkInternConcurrent(b *testing.B)
// Benchmark with GOMAXPROCS goroutines interning concurrently
```

### 2. Integration Tests for Watch Cache

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache_test.go  (existing file)
```

**Tests to add to existing file:**

```go
func TestWatchCacheInterning(t *testing.T)
// Enable feature gate
// Add 100 objects with identical managedFields structures
// Verify they share FieldsV1.Raw pointers in the store
// Verify pool Stats() shows deduplication

func TestWatchCacheInterningOnReplace(t *testing.T)
// Enable feature gate
// Call Replace() with bulk objects
// Verify interning is applied

func TestWatchCacheInterningOnUpdate(t *testing.T)
// Add object, update it (managedFields unchanged)
// Verify new version still uses interned bytes
// Verify old version's refs are released

func TestWatchCacheInterningOnDelete(t *testing.T)
// Add object, delete it
// Verify pool entry ref count decremented
// If no other refs, verify entry removed

func TestWatchCacheInterningDisabled(t *testing.T)
// Disable feature gate
// Verify objects stored with independent FieldsV1.Raw (no sharing)
// Verify no pool allocated

func TestWatchCacheInterningEventBufferEviction(t *testing.T)
// Fill event buffer to capacity
// Trigger eviction of oldest event
// Verify PrevObject refs released correctly

func TestWatchCacheInterningDoesNotCorruptData(t *testing.T)
// Add objects with interned fields
// Deep-copy an object, modify its FieldsV1.Raw
// Verify original cached object is unchanged
// Verify other objects sharing the same interned bytes are unchanged
```

### 3. Benchmark Tests

```
staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache_benchmark_test.go  (existing or new)
```

**Benchmarks:**

```go
func BenchmarkWatchCacheAddWithInterning(b *testing.B)
// Compare Add() performance with and without interning enabled
// Use realistic objects (Pods with 3-5 managedFields entries)

func BenchmarkWatchCacheMemoryWithInterning(b *testing.B)
// Add 10,000 objects (from 100 templates) with interning on/off
// Report memory via runtime.ReadMemStats
// This is the key benchmark proving the feature's value

func BenchmarkWatchCacheReplaceWithInterning(b *testing.B)
// Benchmark Replace() with 10,000 objects
```

## Test Helpers

```go
// createPodWithManagedFields creates a test Pod with the specified number of
// managedFields entries, each with realistic FieldsV1.Raw data.
func createPodWithManagedFields(name string, numManagers int) *v1.Pod

// createHomogeneousPods creates N Pods that all have identical managedFields
// (simulating replicas from the same Deployment).
func createHomogeneousPods(n int, managers int) []*v1.Pod

// createHeterogeneousPods creates N Pods that each have different managedFields
// (simulating diverse workloads).
func createHeterogeneousPods(n int, managers int) []*v1.Pod

// assertSameBackingArray verifies two byte slices share the same backing array.
func assertSameBackingArray(t *testing.T, a, b []byte)

// assertDifferentBackingArray verifies two byte slices have different backing arrays.
func assertDifferentBackingArray(t *testing.T, a, b []byte)
```

The backing array check:
```go
func assertSameBackingArray(t *testing.T, a, b []byte) {
    t.Helper()
    if len(a) > 0 && len(b) > 0 {
        if &a[0] != &b[0] {
            t.Error("expected same backing array, got different")
        }
    }
}
```

## Feature Gate Testing

Use `featuregatetesting.SetFeatureGateDuringTest` (standard pattern in k8s tests):

```go
func TestWatchCacheInterning(t *testing.T) {
    featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate,
        features.InternManagedFieldsInWatchCache, true)
    // ... test with interning enabled ...
}
```

## What to Measure in Benchmarks

The key proof point is a benchmark showing:

```
BenchmarkWatchCacheMemory/without_interning-8    HeapInuse: 850 MB
BenchmarkWatchCacheMemory/with_interning-8       HeapInuse: 420 MB    (-50%)
```

This should use 10,000+ objects from ~100 templates (realistic replica-to-deployment
ratio) with 3-5 managedFields entries per object.
