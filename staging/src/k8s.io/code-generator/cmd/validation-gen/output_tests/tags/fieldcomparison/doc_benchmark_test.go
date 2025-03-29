// staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/tags/fieldcomparison/doc_benchmark_test.go
/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package fieldcomparison

import (
	"context"
	"testing"
	"time"

	// Correct import for operation string type
	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// BenchmarkValidateExampleStruct measures the performance of the generated validation function.
func BenchmarkValidateExampleStruct(b *testing.B) {
	// --- Setup test data outside the loop ---
	ctx := context.Background()

	// *** FIX: Use operation.Operation (string) type and corresponding value ***
	op := operation.Operation{Type: operation.Update}

	// Use nil path, matching how it's called in RegisterValidations
	path := (*field.Path)(nil)

	now := time.Now()
	later := now.Add(1 * time.Hour)
	earlier := now.Add(-1 * time.Hour)

	// Create a set of diverse inputs to average performance across different paths
	type benchmarkCase struct {
		name   string // For potential debugging
		obj    *ExampleStruct
		oldObj *ExampleStruct // Provide non-nil oldObj for Update operation
	}

	// Define old object once, can be reused or varied if needed
	baseOldObj := &ExampleStruct{
		Replicas: 1,
		NestedStruct: NestedStruct{
			Replicas: 1,
		},
	}

	benchmarkData := []benchmarkCase{
		// --- Cases for top-level comparison ---
		{
			name:   "TopLevel_CondTrue_Valid",
			obj:    &ExampleStruct{StartTime: &now, EndTime: &later, Replicas: 1},
			oldObj: baseOldObj,
		},
		{
			name:   "TopLevel_CondTrue_Invalid",
			obj:    &ExampleStruct{StartTime: &now, EndTime: &later, Replicas: 0}, // Validation fails
			oldObj: baseOldObj,
		},
		{
			name:   "TopLevel_CondFalse",
			obj:    &ExampleStruct{StartTime: &now, EndTime: &earlier, Replicas: 0}, // Validation skipped
			oldObj: baseOldObj,
		},
		{
			name:   "TopLevel_NilTimes",
			obj:    &ExampleStruct{StartTime: nil, EndTime: nil, Replicas: 0}, // Validation skipped
			oldObj: baseOldObj,
		},
		// --- Cases for nested comparison ---
		{
			name:   "Nested_CondTrue_Valid",
			obj:    &ExampleStruct{NestedStruct: NestedStruct{StartTime: &now, EndTime: &later, Replicas: 1}},
			oldObj: baseOldObj,
		},
		{
			name:   "Nested_CondTrue_Invalid",
			obj:    &ExampleStruct{NestedStruct: NestedStruct{StartTime: &now, EndTime: &later, Replicas: 0}}, // Validation fails
			oldObj: baseOldObj,
		},
		{
			name:   "Nested_CondFalse",
			obj:    &ExampleStruct{NestedStruct: NestedStruct{StartTime: &now, EndTime: &earlier, Replicas: 0}}, // Validation skipped
			oldObj: baseOldObj,
		},
		{
			name:   "Nested_NilTimes",
			obj:    &ExampleStruct{NestedStruct: NestedStruct{StartTime: nil, EndTime: nil, Replicas: 0}}, // Validation skipped
			oldObj: baseOldObj,
		},
	}

	b.ReportAllocs() // Report memory allocations per operation
	b.ResetTimer()   // Start timing after all setup is complete

	// --- The benchmark loop ---
	for i := 0; i < b.N; i++ {
		// Cycle through the different test cases to average performance
		testCase := benchmarkData[i%len(benchmarkData)]

		// Call the function under test.
		// We DO NOT check the error result in a benchmark, only measure time.
		_ = Validate_ExampleStruct(ctx, op, path, testCase.obj, testCase.oldObj)
	}

	// b.StopTimer() // Usually not needed unless there's significant teardown
}
