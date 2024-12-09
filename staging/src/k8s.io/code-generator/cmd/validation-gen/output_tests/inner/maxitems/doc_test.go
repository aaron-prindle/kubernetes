package inner

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"
)

func TestInnerValidationWithMaxItems(t *testing.T) {
	cases := []struct {
		name         string
		obj          *T1
		expectedPath string
		expectErrors bool
	}{
		{
			name: "no errors with one item",
			obj: &T1{
				Simple: T1Simple{
					Field: []int{1}, // within limit: MaxItems(1)
				},
			},
			expectedPath: "simple.field",
			expectErrors: false,
		},
		{
			name: "errors when too many items",
			obj: &T1{
				Simple: T1Simple{
					Field: []int{1, 2}, // exceeds limit: MaxItems(1)
				},
			},
			expectedPath: "simple.field",
			expectErrors: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opCtx := operation.Context{}
			errs := Validate_T1(opCtx, tc.obj, tc.obj, nil)
			if tc.expectErrors && len(errs) == 0 {
				t.Errorf("expected validation errors but got none")
			}
			if !tc.expectErrors && len(errs) > 0 {
				t.Errorf("unexpected validation errors: %v", errs)
			}
			for _, err := range errs {
				if err.Field != tc.expectedPath {
					t.Errorf("expected error field path %q but got %q",
						tc.expectedPath, err.Field)
				}
			}
		})
	}
}
