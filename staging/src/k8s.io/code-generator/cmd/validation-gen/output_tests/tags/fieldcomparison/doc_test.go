// staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/tags/fieldcomparison/doc_test.go
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
	"testing"
)

// TestFieldComparisonExampleStruct tests the rule:
// +k8s:fieldComparison(minI, <=, i, b)=+k8s:validateTrue
// Which means: IF minI <= i THEN field 'b' must be true.
func TestFieldComparisonExampleStruct(t *testing.T) {
	// Get the test helper initialized from localSchemeBuilder in doc.go
	st := localSchemeBuilder.Test(t)

	// --- Test Cases ---

	// Case 1: Condition TRUE (minI <= i), Validation FALSE -> INVALID
	t.Run("ConditionTrue_ValidationPasses", func(t *testing.T) {
		st.Value(&ExampleStruct{
			MinI: 5,
			I:    10,
			B:    true,
		}).ExpectValidateFalseByPath(map[string][]string{
			"b": {"field ExampleStruct.B"},
		})

		st.Value(&ExampleStruct{
			MinI: 5,
			I:    5,
			B:    true,
		}).ExpectValidateFalseByPath(map[string][]string{
			"b": {"field ExampleStruct.B"},
		})
	})
	// Case 2: Condition FALSE (minI > i), Validation Skipped -> VALID
	t.Run("ConditionFalse_ValidationSkipped", func(t *testing.T) {
		st.Value(&ExampleStruct{
			MinI: 10,
			I:    5,
			B:    true,
		}).ExpectValid()

		st.Value(&ExampleStruct{
			MinI: 10,
			I:    5,
			B:    true,
		}).ExpectValid()
	})
}
