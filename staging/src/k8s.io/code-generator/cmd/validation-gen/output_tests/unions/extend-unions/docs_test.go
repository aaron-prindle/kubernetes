/*
Copyright 2024 The Kubernetes Authors.

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

package union

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func Test(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Valid when Type is "M1" and M1 is set
	st.Value(&U{Type: "M1", M1: &M1{S: "x"}}).ExpectValid()

	// Invalid when Type is "M1" and M1 is nil
	st.Value(&U{Type: "M1"}).ExpectInvalid(
		field.Required(field.NewPath("<nil>"), "field is required when: Type == 'M1'"),
		// field.Required(field.NewPath("m1"), "field is required when: Type == 'M1'"),
	)

	// Valid when Type is "M2" and M2 is set
	st.Value(&U{Type: "M2", M2: &M2{S: "x"}}).ExpectValid()

	// Invalid when Type is "M2" and M2 is nil
	st.Value(&U{Type: "M2"}).ExpectInvalid(
		field.Required(field.NewPath("<nil>"), "field is required when: Type == 'M2'"),
		// field.Required(field.NewPath("m2"), "field is required when: Type == 'M2'"),
	)

	// Valid when Type is "Other" and neither M1 nor M2 is set
	st.Value(&U{Type: "Other"}).ExpectValid()
}
