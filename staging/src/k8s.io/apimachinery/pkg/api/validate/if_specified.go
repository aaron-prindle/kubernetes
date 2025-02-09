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

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetFieldFunc is a function that extracts a field of type R from a struct of type T.
type GetFieldFunc2[T any, R any] func(*T) R

// ValidateFunc is a function that validates a "new" and "old" value (both of type T).
type ValidateFunc2[T any] func(
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path,
	newVal, oldVal *T,
) field.ErrorList

// IfSpecified validates a target field only if the condition field is specified.
// Parameters:
// - condFieldName:    name of the condition field
// - targetFieldName:  name of the target field
// - getCondField:     function to get the condition field's value
// - getTargetField:   function to get the target field's value
// - validator:        validation function to apply to the target field
func IfSpecified[Tstruct any, Tcond any, Ttarget any](
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path,
	newStruct, oldStruct *Tstruct,
	condFieldName, targetFieldName string,
	getCondField GetFieldFunc2[Tstruct, Tcond],
	getTargetField GetFieldFunc2[Tstruct, Ttarget],
	validator ValidateFunc2[Ttarget],
) field.ErrorList {
	// Skip validation if the object is nil
	if newStruct == nil {
		return nil
	}

	// Get the condition field value
	condValue := getCondField(newStruct)

	// Skip validation if the condition field is not "specified"
	if !isValueSpecified(condValue) {
		return nil
	}

	// Extract the old value (if oldStruct is present)
	var oldTargetPtr *Ttarget
	if oldStruct != nil {
		oldTarget := getTargetField(oldStruct)
		oldTargetPtr = &oldTarget
	}

	// Extract the new value
	newTarget := getTargetField(newStruct)

	// Condition field is specified => apply the validator to the *target field*
	return validator(ctx, op, fldPath.Child(targetFieldName), &newTarget, oldTargetPtr)
}

// isValueSpecified checks if a value is specified (non-nil, non-zero, non-empty)
func isValueSpecified(value interface{}) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return !v.IsNil()
	case reflect.String:
		return v.String() != ""
	case reflect.Bool:
		return v.Bool()
	default:
		return !v.IsZero()
	}
}
