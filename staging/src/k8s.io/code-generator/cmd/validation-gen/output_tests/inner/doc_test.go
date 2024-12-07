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

package inner

import (
	"testing"
)

func TestInnerValidation(t *testing.T) {
	cases := []struct {
		name         string
		obj          *T1
		expectedPath string
		expectErrors bool
	}{
		{
			name: "simple inner validation",
			obj: &T1{
				Simple: struct {
					Field string `json:"field"`
				}{
					Field: "test",
				},
			},
			expectedPath: "simple.field",
			expectErrors: false,
		},
		{
			name: "nested inner validation",
			obj: &T1{
				Nested: struct {
					OuterField string `json:"outerField"`
					Inner      struct {
						InnerField string `json:"innerField"`
					} `json:"inner"`
				}{
					OuterField: "outer",
					Inner: struct {
						InnerField string `json:"innerField"`
					}{
						InnerField: "inner",
					},
				},
			},
			expectedPath: "nested.inner.innerField",
			expectErrors: false,
		},
		{
			name: "pointer struct inner validation",
			obj: &T1{
				PointerStruct: &struct {
					Field string `json:"field"`
				}{
					Field: "test",
				},
			},
			expectedPath: "pointerStruct.field",
			expectErrors: false,
		},
		{
			name: "no validation field",
			obj: &T1{
				NoValidation: struct {
					Field string `json:"field"`
				}{
					Field: "test",
				},
			},
			expectedPath: "noValidation.field",
			expectErrors: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := Validate_T1(tc.obj, nil)
			if tc.expectErrors && len(errs) == 0 {
				t.Error("expected validation errors but got none")
			}
			if !tc.expectErrors && len(errs) > 0 {
				t.Errorf("unexpected validation errors: %v", errs)
			}
			for _, err := range errs {
				if err.Field != tc.expectedPath {
					t.Errorf("expected error path %q but got %q",
						tc.expectedPath, err.Field)
				}
			}
		})
	}
}
