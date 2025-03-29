// staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/tags/fieldcomparison/doc_test.go
/*
Copyright 2025 The Kubernetes Authors.
// ... (license header) ...
*/
package fieldcomparison

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Test case for: +k8s:fieldComparison(EndTime, >=, StartTime, Replicas)=+k8s:minimum=1
func TestFieldComparison_TimeOrderReplicas(t *testing.T) {
	st := localSchemeBuilder.Test(t)
	now := time.Now()
	later := now.Add(1 * time.Hour)
	earlier := now.Add(-1 * time.Hour)

	// --- Case 1: Valid cases ---
	st.Value(&ExampleStruct{
		StartTime: &now,
		EndTime:   &later,
		Replicas:  1,
	}).ExpectValid()
	st.Value(&ExampleStruct{
		StartTime: &now,
		EndTime:   &now,
		Replicas:  5,
	}).ExpectValid()

	// --- Case 2: Invalid case ---
	st.Value(&ExampleStruct{
		StartTime: &now,
		EndTime:   &later,
		Replicas:  0, // Fails minimum=1
	}).ExpectInvalid(
		// *** FIX: Remove .Index(0) ***
		field.Invalid(field.NewPath("[].Replicas"), 0, "must be greater than or equal to 1"),
	)

	// --- Case 3 & 4: Condition false or nil, validation skipped ---
	st.Value(&ExampleStruct{
		StartTime: &now,
		EndTime:   &earlier,
		Replicas:  0,
	}).ExpectValid()
	st.Value(&ExampleStruct{
		StartTime: nil,
		EndTime:   nil,
		Replicas:  0,
	}).ExpectValid()
	st.Value(&ExampleStruct{
		StartTime: nil,
		EndTime:   &now,
		Replicas:  0,
	}).ExpectValid()
	st.Value(&ExampleStruct{
		StartTime: &now,
		EndTime:   nil,
		Replicas:  0,
	}).ExpectValid()
}

// TestFieldComparisonNestedTimeReplicas tests the rule:
// +k8s:fieldComparison(NestedStruct.EndTime, >=, NestedStruct.StartTime, NestedStruct.Replicas)=+k8s:minimum=1
func TestFieldComparisonNestedTimeReplicas(t *testing.T) {
	st := localSchemeBuilder.Test(t)
	now := time.Now()
	later := now.Add(1 * time.Hour)
	earlier := now.Add(-1 * time.Hour)

	// --- Scenario 1: Condition True, Validation Passes ---
	t.Run("ConditionTrue_ValidationPasses", func(t *testing.T) {
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &later,
				Replicas:  1,
			},
		}).ExpectValid()
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &now,
				Replicas:  5,
			},
		}).ExpectValid()
	})

	// --- Scenario 2: Condition True, Validation Fails ---
	t.Run("ConditionTrue_ValidationFails", func(t *testing.T) {
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &later,
				Replicas:  0, // Invalid value
			},
		}).ExpectInvalid(
			// *** FIX: Remove .Index(0) ***
			field.Invalid(field.NewPath("[].NestedStruct", "Replicas"), 0, "must be greater than or equal to 1"),
		)
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &now,
				Replicas:  -1, // Invalid value
			},
		}).ExpectInvalid(
			// *** FIX: Remove .Index(0) ***
			field.Invalid(field.NewPath("[].NestedStruct", "Replicas"), -1, "must be greater than or equal to 1"),
		)
	})

	// --- Scenarios 3 & 4: Condition False or Nil, Validation Skipped ---
	t.Run("ConditionFalse_ValidationSkipped", func(t *testing.T) {
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &earlier,
				Replicas:  0,
			},
		}).ExpectValid()
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   &earlier,
				Replicas:  2,
			},
		}).ExpectValid()
	})
	t.Run("NilTimes_ConditionFalse_ValidationSkipped", func(t *testing.T) {
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: nil,
				EndTime:   nil,
				Replicas:  0,
			},
		}).ExpectValid()
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: nil,
				EndTime:   &now,
				Replicas:  0,
			},
		}).ExpectValid()
		st.Value(&ExampleStruct{
			NestedStruct: NestedStruct{
				StartTime: &now,
				EndTime:   nil,
				Replicas:  0,
			},
		}).ExpectValid()
	})
}
