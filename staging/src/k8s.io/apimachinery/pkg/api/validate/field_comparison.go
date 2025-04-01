// staging/src/k8s.io/apimachinery/pkg/api/validate/field_comparison.go (New File or add to existing validation helpers)
package validate // Or your chosen package alias used in generated code

import (
	"context"
	"strings" // Needed for path splitting

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Define ValidateFunc type if not already present in the package
// type ValidateFunc[T any] func(ctx context.Context, op operation.Operation, fldPath *field.Path, newObj, oldObj T) field.ErrorList

// FieldComparisonConditional validates a target field using payloadValidator
// only if the compareFn returns true when run on the new object.
// If compareFn returns false, an Invalid error is returned, attached to the
// struct's path (fldPath), indicating the comparison failure.
//
// Tstruct: The type of the struct containing the fields.
// TfieldPtr: The type of the target field *value* returned by getTargetFieldFn.
//
//	This is often a pointer type, even if the original field is not,
//	as handled by the code generator.
func FieldComparisonConditional[Tstruct any, TfieldPtr any](
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path, // Path to the *struct* being validated
	newObj, oldObj *Tstruct, // The struct instances (oldObj can be nil)
	comparisonName string, // User-friendly name of the comparison (e.g., "minI <= i")
	compareFn func(obj *Tstruct) bool, // Function that performs the comparison logic on the new object
	targetFieldPathString string, // Dot-separated path string *relative* to the struct for the target field
	getTargetFieldFn func(o *Tstruct) TfieldPtr, // Function to extract the target field value
	payloadValidator ValidateFunc[TfieldPtr], // The validation function to run on the target field if comparison holds
) field.ErrorList {
	var errs field.ErrorList

	// Should not happen in standard validation flow, but safeguard.
	if newObj == nil {
		// Or return an internal error? Returning empty list is safer.
		return errs
	}

	// --- Step 1: Run the Comparison ---
	comparisonHolds := compareFn(newObj)

	// --- Step 2: Conditional Validation ---
	if comparisonHolds {
		// --- Step 2a: Comparison True - Run Payload Validation ---

		// Calculate the full path to the target field
		targetPath := fldPath
		for _, part := range strings.Split(targetFieldPathString, ".") {
			// Avoid adding empty segments if path starts/ends with '.' or has '..'
			if part != "" {
				targetPath = targetPath.Child(part)
			}
		}

		// Get the new and old values of the *target* field
		newTargetVal := getTargetFieldFn(newObj)
		var oldTargetVal TfieldPtr
		if oldObj != nil {
			oldTargetVal = getTargetFieldFn(oldObj)
		}

		// Run the payload validator on the target field
		errs = append(errs, payloadValidator(ctx, op, targetPath, newTargetVal, oldTargetVal)...)

	}
	return errs
}
