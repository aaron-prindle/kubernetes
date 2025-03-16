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
package validate

import (
	"context"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// IfSpecified validates a field only if another field in the struct is specified.
// Parameters:
//   - conditionField: Name of the field that must be specified for validation to occur
//     For nested fields, use X as a delimiter (e.g., ChildXActive)
//   - targetField: Name of the field to validate
//     For nested fields, use X as a delimiter (e.g., ChildXName)
//   - validator: The validation function to apply to the target field
func IfSpecified[T any](
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path,
	obj, oldObj T,
	conditionField, targetField string,
	validator ValidateFunc[T],
) field.ErrorList {
	// Skip validation if the object is nil
	if isNil(obj) {
		return field.ErrorList{}
	}

	// Check if the condition field is specified
	if !isFieldSpecified(obj, conditionField) {
		// Condition field not specified, skip validation
		return field.ErrorList{}
	}

	// For the target field path, use the last part after delimiter for the field path
	var fieldPathChild string
	if strings.Contains(targetField, "X") {
		parts := strings.Split(targetField, "X")
		fieldPathChild = parts[len(parts)-1]
	} else {
		fieldPathChild = targetField
	}

	// Condition field is specified, apply the validator
	return validator(ctx, op, fldPath.Child(fieldPathChild), obj, oldObj)
}

// isFieldSpecified checks if a field in an object is specified (non-nil, non-zero)
func isFieldSpecified(obj interface{}, fieldPath string) bool {
	if obj == nil {
		return false
	}

	// Get the value of the object
	v := reflect.ValueOf(obj)

	// Dereference pointers
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	// Must be a struct to access fields
	if v.Kind() != reflect.Struct {
		return false
	}

	// Handle nested fields with X delimiter
	if strings.Contains(fieldPath, "X") {
		parts := strings.Split(fieldPath, "X")
		currentObj := v

		// Reject any paths that include "parent" since we can't navigate up
		for _, part := range parts {
			if part == "parent" {
				return false // Cannot navigate to parent objects
			}
		}

		// Navigate through nested fields
		for _, part := range parts {
			// Try to find the field at current level
			field := currentObj.FieldByName(part)
			if !field.IsValid() {
				return false
			}

			// If this is the last part, check if it's specified
			if part == parts[len(parts)-1] {
				return isSpecified(field)
			}

			// Otherwise, continue traversing if the field is a struct or pointer to struct
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					return false
				}
				field = field.Elem()
			}

			if field.Kind() != reflect.Struct {
				return false
			}

			currentObj = field
		}

		return false // Should not reach here
	}

	// Simple non-nested field
	field := v.FieldByName(fieldPath)
	if !field.IsValid() {
		return false
	}

	// Check if the field is specified based on its type
	return isSpecified(field)
}

// isSpecified checks if a reflect.Value is non-zero/non-nil
func isSpecified(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	// For pointers, slices, maps, etc. - check if nil
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return !v.IsNil()
	case reflect.String:
		// For strings, check if non-empty
		return v.String() != ""
	case reflect.Bool:
		// For booleans, check if true
		return v.Bool()
	default:
		// For all other types, check if non-zero
		return !v.IsZero()
	}
}

// isNil checks if a value is nil
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return val.IsNil()
	default:
		return false
	}
}
