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

// IfSpecified validates a field using provided validator only if specified field path
// is non-zero or non-nil.
func IfSpecified[T any](ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj T,
	fieldPathStr string, validator ValidateFunc[T]) field.ErrorList {

	// Skip validation if the object is nil
	if isNilValue(reflect.ValueOf(obj)) {
		return field.ErrorList{}
	}

	// Special handling for parent references
	if strings.HasPrefix(fieldPathStr, "parent") {
		// For parentL references in generated code, we need special handling
		// In practice, we won't have the parent object in the generated code context
		// For the specific case of ChildLActive or similar checks, find in the current object
		fieldPathWithoutPrefix := strings.TrimPrefix(fieldPathStr, "parentL")
		if fieldPathWithoutPrefix != fieldPathStr {
			// Handle parentL prefix by resolving within the top-level obj
			if isFieldSpecifiedInternal(obj, fieldPathWithoutPrefix) {
				return validator(ctx, op, fldPath, obj, oldObj)
			}
			return field.ErrorList{}
		}
	} else if strings.HasPrefix(fieldPathStr, "selfL") {
		// For selfL references, resolve within the object itself
		fieldName := strings.TrimPrefix(fieldPathStr, "selfL")
		if fieldName != fieldPathStr && isFieldSpecifiedInternal(obj, fieldName) {
			return validator(ctx, op, fldPath, obj, oldObj)
		}
		return field.ErrorList{}
	} else {
		// Regular field reference - check if the field is specified
		if isFieldSpecifiedInternal(obj, fieldPathStr) {
			return validator(ctx, op, fldPath, obj, oldObj)
		}
		return field.ErrorList{}
	}

	// If we can't determine field state, don't validate
	return field.ErrorList{}
}

// isFieldSpecifiedInternal checks if a field within an object is specified (non-nil for pointers, non-zero for values)
func isFieldSpecifiedInternal(obj interface{}, fieldName string) bool {
	if obj == nil {
		return false
	}

	v := reflect.ValueOf(obj)
	// Dereference pointers
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	// If not a struct, we can't get fields
	if v.Kind() != reflect.Struct {
		return false
	}

	// If the fieldName contains L (our delimiter), split and traverse
	if strings.Contains(fieldName, "L") {
		parts := strings.Split(fieldName, "L")
		currentObj := obj
		for _, part := range parts {
			if !isFieldSpecifiedInternal(currentObj, part) {
				return false
			}

			// Advance to the next object in the path
			currentValue := reflect.ValueOf(currentObj)
			for currentValue.Kind() == reflect.Ptr && !currentValue.IsNil() {
				currentValue = currentValue.Elem()
			}
			if currentValue.Kind() != reflect.Struct {
				return false
			}

			field := currentValue.FieldByName(part)
			if !field.IsValid() {
				return false
			}

			if field.CanInterface() {
				currentObj = field.Interface()
			} else {
				return false
			}
		}
		return true
	}

	// Simple field lookup
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return false
	}

	// Check if field is specified (non-zero, non-nil)
	return isSpecified(field)
}

// isSpecified checks if a reflect.Value is specified (non-nil for pointers, non-zero for values)
func isSpecified(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	// For pointers, check if nil
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface ||
		v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		return !v.IsNil()
	}

	// For string check if empty
	if v.Kind() == reflect.String {
		return v.String() != ""
	}

	// For bool check if true (false is considered not specified)
	if v.Kind() == reflect.Bool {
		return v.Bool()
	}

	// For other types, check if zero value
	return !v.IsZero()
}

// isNilValue checks if a reflect.Value is nil or represents a nil pointer
func isNilValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface ||
		v.Kind() == reflect.Slice || v.Kind() == reflect.Map ||
		v.Kind() == reflect.Chan || v.Kind() == reflect.Func {
		return v.IsNil()
	}
	return false
}
