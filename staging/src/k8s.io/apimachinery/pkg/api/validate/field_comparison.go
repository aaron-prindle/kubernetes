// staging/src/k8s.io/apimachinery/pkg/api/validate/field_comparison.go
package validate

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateFunc defines the signature for standard validation functions used by generators.
// Crucially, newVal and oldVal are pointers (*T) to match functions like validate.Minimum, etc.
type ValidateFuncPtr2[T any] func(
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path,
	newVal *T, // <<< MUST be *T
	oldVal *T, // <<< MUST be *T
) field.ErrorList

// Other type definitions like GetFieldFunc, GetFieldFuncPtr might also be here...
type GetFieldFunc2[T any, R any] func(*T) R
type GetFieldFuncPtr2[T any, R any] func(*T) *R

// FieldComparisonValidateField validates a target field only if the comparison between two other fields holds true.
// Parameters:
//   - ... (other params) ...
//   - field1PathString:   Dot-separated path string for the first comparison field.
//   - field2PathString:   Dot-separated path string for the second comparison field.
//   - targetFieldPathString: Dot-separated path string for the field to validate.
//   - ... (getters, validator) ...
//
// FieldComparisonValidateField implementation (should now compile correctly with the updated ValidateFunc definition)
func FieldComparisonValidateField[Tstruct any, Tfield1 any, Tfield2 any, Ttarget any](
	ctx context.Context,
	op operation.Operation,
	fldPath *field.Path, // Base path to the struct
	newStruct, oldStruct *Tstruct,
	field1PathString, operator, field2PathString, targetFieldPathString string, // Use full paths
	getField1 GetFieldFunc2[Tstruct, Tfield1], // Returns Tfield1 value
	getField2 GetFieldFunc2[Tstruct, Tfield2], // Returns Tfield2 value
	getTargetField GetFieldFuncPtr2[Tstruct, Ttarget], // Returns *Ttarget pointer
	validator ValidateFuncPtr2[Ttarget], // Expects func(..., newVal *Ttarget, oldVal *Ttarget)
) field.ErrorList {
	if newStruct == nil {
		return nil
	}

	// Get comparison values
	val1 := getField1(newStruct)
	val2 := getField2(newStruct)

	// Perform comparison
	comparisonResult, err := compareValues(val1, val2, operator)
	if err != nil {
		// Construct path for error reporting
		var field1ErrPath *field.Path
		parts1 := strings.Split(field1PathString, ".")
		basePath := fldPath
		if basePath == nil {
			basePath = field.NewPath("") // Handle nil root path if necessary
		}
		if len(parts1) > 0 && parts1[0] != "" {
			field1ErrPath = basePath.Child(parts1[0], parts1[1:]...)
		} else {
			// Handle invalid path for error reporting, maybe return error directly
			return field.ErrorList{field.InternalError(basePath, fmt.Errorf("invalid field1 path %q for comparison error reporting: %w", field1PathString, err))}
		}
		// Report comparison error associated with the first field path
		return field.ErrorList{field.InternalError(field1ErrPath, fmt.Errorf("comparison error between %q and %q: %w", field1PathString, field2PathString, err))}
	}

	// If comparison holds true, validate the target field
	if comparisonResult {
		// Get pointers to the target field's new and old values
		newTargetValPtr := getTargetField(newStruct) // Type is *Ttarget
		var oldTargetValPtr *Ttarget                 // Type is *Ttarget

		if oldStruct != nil {
			// Assignment is *Ttarget = *Ttarget (should be valid)
			oldTargetValPtr = getTargetField(oldStruct)
		}

		// Construct the specific path to the target field for the validator
		var targetPath *field.Path
		partsTarget := strings.Split(targetFieldPathString, ".")
		basePath := fldPath // Re-use basePath logic as above
		if basePath == nil {
			basePath = field.NewPath("")
		}
		if len(partsTarget) > 0 && partsTarget[0] != "" {
			targetPath = basePath.Child(partsTarget[0], partsTarget[1:]...)
		} else {
			// Handle invalid path for validation target
			return field.ErrorList{field.InternalError(basePath, fmt.Errorf("invalid target field path %q for validation", targetFieldPathString))}
		}

		// Call the validator function.
		// Arguments should match: validator expects (*Ttarget, *Ttarget)
		// We pass: newTargetValPtr (*Ttarget), oldTargetValPtr (*Ttarget)
		// This call *should* be type-correct if ValidateFunc definition is right.
		return validator(ctx, op, targetPath, newTargetValPtr, oldTargetValPtr)
	}

	// Comparison was false, skip validation
	return nil
}

// compareValues performs the comparison between two values using the specified operator.
// It handles basic numeric types, strings, and time.Time.
// Returns true if the comparison holds, false otherwise, or an error for incompatible types/operators.
// compareValues performs the comparison between two values using the specified operator.
// It handles basic numeric types, strings, and time.Time.
// Returns true if the comparison holds, false otherwise, or an error only for truly incompatible types/operators.
func compareValues(val1, val2 interface{}, operator string) (bool, error) {
	// --- Handle potential pointers and zero values ---
	// Get the underlying value if it's a pointer.
	// Treat zero values (except bool false) as 'nil' for comparison purposes,
	// as they often represent an unset state in Kubernetes APIs.
	v1 := reflect.ValueOf(val1)
	if v1.Kind() == reflect.Ptr && !v1.IsNil() {
		v1 = v1.Elem() // Dereference the pointer
	}
	// Check if the value is invalid (e.g., nil interface) or if it's a zero value (but not boolean false)
	if !v1.IsValid() || (v1.Kind() != reflect.Bool && v1.IsZero()) {
		val1 = nil // Represent unset/zero as nil
	} else {
		val1 = v1.Interface() // Use the concrete value
	}

	v2 := reflect.ValueOf(val2)
	if v2.Kind() == reflect.Ptr && !v2.IsNil() {
		v2 = v2.Elem() // Dereference the pointer
	}
	// Check if the value is invalid or if it's a zero value (but not boolean false)
	if !v2.IsValid() || (v2.Kind() != reflect.Bool && v2.IsZero()) {
		val2 = nil // Represent unset/zero as nil
	} else {
		val2 = v2.Interface() // Use the concrete value
	}

	// --- Handle comparisons involving nil ---
	if val1 == nil || val2 == nil {
		switch operator {
		case "==":
			// nil == nil is true; nil == non-nil is false
			return val1 == val2, nil
		case "!=":
			// nil != nil is false; nil != non-nil is true
			return val1 != val2, nil
		default:
			// For inequality comparisons (<, <=, >, >=), if one or both sides are nil,
			// the comparison generally cannot be satisfied. Return false without error.
			// This prevents internal errors when comparing potentially unset optional fields.
			return false, nil // <<< FIX APPLIED HERE
		}
	}

	// --- Handle specific types ---

	// Special handling for time.Time
	t1, ok1 := val1.(time.Time)
	t2, ok2 := val2.(time.Time)
	if ok1 && ok2 {
		return compareTimes(t1, t2, operator)
	}
	// Error if trying to compare time.Time with a different type
	if ok1 || ok2 {
		return false, fmt.Errorf("type mismatch: cannot compare time.Time with non-time.Time (%T vs %T)", val1, val2)
	}

	// Attempt numeric comparison (integers, floats)
	f1, err1 := toFloat64(val1)
	f2, err2 := toFloat64(val2)
	if err1 == nil && err2 == nil { // If both are convertible to float64
		return compareFloats(f1, f2, operator)
	}
	// If only one is numeric, it's a type mismatch for numeric operators
	if (err1 == nil || err2 == nil) && (operator != "==" && operator != "!=") {
		return false, fmt.Errorf("type mismatch: cannot perform numeric comparison '%s' between %T and %T", operator, val1, val2)
	}

	// Attempt string comparison
	s1, ok1 := val1.(string)
	s2, ok2 := val2.(string)
	if ok1 && ok2 {
		return compareStrings(s1, s2, operator)
	}
	// If only one is string, it's a type mismatch for string operators
	if (ok1 || ok2) && (operator != "==" && operator != "!=") {
		return false, fmt.Errorf("type mismatch: cannot perform string comparison '%s' between %T and %T", operator, val1, val2)
	}

	// Attempt boolean comparison (only for ==, !=)
	b1, ok1 := val1.(bool)
	b2, ok2 := val2.(bool)
	if ok1 && ok2 {
		switch operator {
		case "==":
			return b1 == b2, nil
		case "!=":
			return b1 != b2, nil
		default:
			return false, fmt.Errorf("invalid operator '%s' for boolean comparison", operator)
		}
	}

	// --- Fallback for == and != using DeepEqual ---
	// Use DeepEqual only if other specific type comparisons didn't apply,
	// but the types might still be comparable (e.g. two structs of same type).
	// Only applies to == and !=.
	if operator == "==" || operator == "!=" {
		equal := reflect.DeepEqual(val1, val2)
		if operator == "==" {
			return equal, nil
		}
		// operator == "!="
		return !equal, nil
	}

	// --- Final Error Handling ---
	// If we reach here, the operator is likely invalid for the given types
	// (e.g., trying to use '>' on incompatible structs).
	return false, fmt.Errorf("unsupported comparison between types %T and %T for operator %q", val1, val2, operator)
}

// Helper to convert numeric types to float64 for comparison
func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(n).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(n).Uint()), nil
	case float32, float64:
		return reflect.ValueOf(n).Float(), nil
	default:
		return 0, fmt.Errorf("value %v (type %T) is not a standard numeric type", v, v)
	}
}

func compareFloats(f1, f2 float64, op string) (bool, error) {
	switch op {
	case "==":
		return f1 == f2, nil // Note: Floating point equality can be tricky
	case "!=":
		return f1 != f2, nil
	case "<":
		return f1 < f2, nil
	case "<=":
		return f1 <= f2, nil
	case ">":
		return f1 > f2, nil
	case ">=":
		return f1 >= f2, nil
	default:
		return false, fmt.Errorf("invalid numeric operator: %q", op)
	}
}

func compareStrings(s1, s2, op string) (bool, error) {
	switch op {
	case "==":
		return s1 == s2, nil
	case "!=":
		return s1 != s2, nil
	// Lexicographical comparison for inequalities
	case "<":
		return s1 < s2, nil
	case "<=":
		return s1 <= s2, nil
	case ">":
		return s1 > s2, nil
	case ">=":
		return s1 >= s2, nil
	default:
		return false, fmt.Errorf("invalid string operator: %q", op)
	}
}

func compareTimes(t1, t2 time.Time, op string) (bool, error) {
	switch op {
	case "==":
		return t1.Equal(t2), nil
	case "!=":
		return !t1.Equal(t2), nil
	case "<":
		return t1.Before(t2), nil
	case "<=":
		return t1.Before(t2) || t1.Equal(t2), nil
	case ">":
		return t1.After(t2), nil
	case ">=":
		return t1.After(t2) || t1.Equal(t2), nil
	default:
		return false, fmt.Errorf("invalid time operator: %q", op)
	}
}
