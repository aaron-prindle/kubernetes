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

// IfSpecified validates a field using the provided validator only if the referenced field is specified.
// It follows the same pattern as Subfield but checks if the field is specified before applying validation.
func IfSpecified[T any](ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj T,
	fieldPath string, validator ValidateFunc[T]) field.ErrorList {

	if isNil(obj) {
		return nil
	}

	// Check if the referenced field is specified
	if !isFieldSpecified(obj, fieldPath) {
		// Field not specified, skip validation
		return field.ErrorList{}
	}

	// Field is specified, apply the validator
	return validator(ctx, op, fldPath, obj, oldObj)
}

// isNil checks if a value is nil, handling both interface and non-interface types
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

// isFieldSpecified checks if a field at the given path is specified (non-nil for pointers, non-zero for values)
func isFieldSpecified(obj interface{}, fieldPath string) bool {
	if obj == nil {
		return false
	}

	// Handle special references and field path traversal
	parts := strings.Split(fieldPath, "L")

	// Navigate through the object structure to find the referenced field
	current := obj
	for _, part := range parts {
		if current == nil {
			return false
		}

		// Special reference handling for 'self', 'this', 'parent'
		if part == "self" || part == "this" {
			continue // Stay at current object
		}
		if part == "parent" {
			// In generated code, we don't have a parent reference
			// For manual validation, parent should be passed separately
			return false
		}

		// Use reflection to get the field
		val := reflect.ValueOf(current)

		// Dereference pointers
		for val.Kind() == reflect.Ptr && !val.IsNil() {
			val = val.Elem()
		}

		// Check if the value is a struct
		if val.Kind() != reflect.Struct {
			return false
		}

		// Get the field by name
		field := val.FieldByName(part)
		if !field.IsValid() {
			return false
		}

		// Update current to the field value
		if field.CanInterface() {
			current = field.Interface()
		} else {
			return false
		}
	}

	// Check if the field is specified (non-nil for pointers, non-zero for values)
	return isSpecified(current)
}

// isSpecified checks if a value is specified (non-nil for pointers, non-zero for values)
func isSpecified(value interface{}) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)

	// For pointers, check if nil
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		return !v.IsNil()
	}

	// For other types, check if zero value
	return !reflect.DeepEqual(value, reflect.Zero(v.Type()).Interface())
}
