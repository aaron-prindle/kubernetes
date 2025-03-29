// staging/src/k8s.io/code-generator/cmd/validation-gen/validators/field_comparison.go
package validators

import (
	"fmt"
	"strings"

	// Ensure field path package is imported if needed
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field" // Standard field path package
	"k8s.io/gengo/v2/types"
)

// Constants, supportedOperators, init, struct definition, Init, TagName, ValidScopes remain the same...

const (
	fieldComparisonTagName = "k8s:fieldComparison"
)

var supportedOperators = sets.New(
	"==", "!=", "<", "<=", ">", ">=",
)

func init() {
	RegisterTagValidator(&fieldComparisonTagValidator{})
}

type fieldComparisonTagValidator struct {
	validator Validator
}

func (fctv *fieldComparisonTagValidator) Init(cfg Config) {
	fctv.validator = cfg.Validator
}

func (fieldComparisonTagValidator) TagName() string {
	return fieldComparisonTagName
}

var fieldComparisonTagValidScopes = sets.New(ScopeType)

func (fieldComparisonTagValidator) ValidScopes() sets.Set[Scope] {
	return fieldComparisonTagValidScopes
}

// Reference the runtime validation function
var (
	validateFieldComparisonValidateField = types.Name{Package: libValidationPkg, Name: "FieldComparisonValidateField"}
)

// GetValidations parses the tag and generates the validation logic.
func (fctv fieldComparisonTagValidator) GetValidations(context Context, args []string, payload string) (Validations, error) {
	// Ensure the tag is applied to a struct type
	t := realType(context.Type)
	if t == nil || t.Kind != types.Struct {
		return Validations{}, fmt.Errorf("tag %q can only be used on struct types, got %v", fctv.TagName(), context.Type)
	}

	// Validate the number of arguments (expecting 4 paths/operators)
	if len(args) != 4 {
		return Validations{}, fmt.Errorf("tag %q requires exactly four arguments: <field1Path>, <operator>, <field2Path>, <fieldToValidatePath>", fctv.TagName())
	}

	// Extract arguments
	field1PathString := args[0]
	operator := args[1]
	field2PathString := args[2]
	targetFieldPathString := args[3]

	// Validate the operator
	if !supportedOperators.Has(operator) {
		return Validations{}, fmt.Errorf("invalid operator %q for tag %q. Supported operators are: %v", operator, fctv.TagName(), supportedOperators.UnsortedList())
	}

	// Validate the payload (must be a +k8s validator tag)
	if payload == "" || !strings.HasPrefix(payload, "+k8s:") {
		return Validations{}, fmt.Errorf("tag %q requires a validator payload starting with '+k8s:' (e.g., =+k8s:minimum=1), got %q", fctv.TagName(), payload)
	}
	nestedValidatorComment := payload

	// --- Resolve field paths to member sequences ---
	// These helpers find the sequence of Go struct members corresponding to the dot-path.
	field1Members, err := findMembersByPath(t, field1PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field path %q for tag %q: %w", field1PathString, fctv.TagName(), err)
	}
	field1Memb := field1Members[len(field1Members)-1] // The final member in the path

	field2Members, err := findMembersByPath(t, field2PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field path %q for tag %q: %w", field2PathString, fctv.TagName(), err)
	}
	field2Memb := field2Members[len(field2Members)-1]

	targetFieldMembers, err := findMembersByPath(t, targetFieldPathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid target field path %q for tag %q: %w", targetFieldPathString, fctv.TagName(), err)
	}
	targetFieldMemb := targetFieldMembers[len(targetFieldMembers)-1]

	// --- Generate nested validator ---
	// Construct the field.Path for the target field where the nested validator applies.
	targetPathParts := strings.Split(targetFieldPathString, ".")
	var nestedPath *field.Path

	// Check if context.Path is nil (can happen at the root)
	basePath := context.Path
	if basePath == nil {
		basePath = field.NewPath("") // Start with an empty root path if needed
	}

	if len(targetPathParts) > 0 && targetPathParts[0] != "" { // Ensure parts exist and aren't empty
		firstPart := targetPathParts[0]
		remainingParts := targetPathParts[1:]
		nestedPath = basePath.Child(firstPart, remainingParts...) // Construct nested path correctly
	} else {
		// This indicates an invalid path string like "." or empty string, which findMembersByPath should ideally catch.
		// If it reaches here, treat it as an error or default to the base path.
		// Let's return an error for clarity.
		return Validations{}, fmt.Errorf("internal error: target field path %q resulted in invalid parts for field.Path construction", targetFieldPathString)
		// nestedPath = basePath // Fallback, less informative
	}

	// Define the context for extracting the nested validation function.
	// The validation applies to the *type* of the final field in the target path.
	nestedContext := Context{
		Scope:  ScopeField,           // Validation applies to a field scope
		Type:   targetFieldMemb.Type, // The Go type of the target field
		Parent: t,                    // The parent struct type
		Path:   nestedPath,           // The calculated field.Path
	}

	// Extract the validation function(s) defined by the payload tag.
	nestedValidations, err := fctv.validator.ExtractValidations(nestedContext, []string{nestedValidatorComment})
	if err != nil {
		return Validations{}, fmt.Errorf("error extracting nested validations for payload %q on target field path %q: %w", nestedValidatorComment, targetFieldPathString, err)
	}
	// We currently assume the payload generates exactly one applicable function.
	if len(nestedValidations.Functions) == 0 {
		return Validations{}, fmt.Errorf("payload tag %q did not generate any validation functions applicable to the target field %q (type %s)", nestedValidatorComment, targetFieldPathString, targetFieldMemb.Type.String())
	}
	if len(nestedValidations.Functions) > 1 {
		return Validations{}, fmt.Errorf("payload tag %q generated multiple validation functions; %q currently supports only one nested validator for the target field %q", nestedValidatorComment, fctv.TagName(), targetFieldPathString)
	}
	nestedValidatorFunc := nestedValidations.Functions[0] // The field-level validator function

	// --- Generate FieldComparison call with nested accessors ---
	result := Validations{}
	structPtrType := types.PointerTo(t) // e.g., *ExampleStruct

	// Generate Go code snippets to access the potentially nested fields from the base struct 'o'.
	field1Accessor := generateNestedFieldAccessor("o", field1Members)
	field2Accessor := generateNestedFieldAccessor("o", field2Members)
	targetFieldAccessor := generateNestedFieldAccessor("o", targetFieldMembers)

	// Getter function for field1 (returns value Tfield1)
	getField1Fn := FunctionLiteral{
		Parameters: []ParamResult{{"o", structPtrType}},
		Results:    []ParamResult{{"", field1Memb.Type}}, // Result type is the type of the final field
		Body:       fmt.Sprintf("return %s", field1Accessor),
	}
	// Getter function for field2 (returns value Tfield2)
	getField2Fn := FunctionLiteral{
		Parameters: []ParamResult{{"o", structPtrType}},
		Results:    []ParamResult{{"", field2Memb.Type}},
		Body:       fmt.Sprintf("return %s", field2Accessor),
	}
	// Getter function for the target field (returns pointer *Ttarget)
	targetFieldPtrType := types.PointerTo(targetFieldMemb.Type)
	getTargetFieldFn := FunctionLiteral{
		Parameters: []ParamResult{{"o", structPtrType}},
		Results:    []ParamResult{{"", targetFieldPtrType}},        // Returns pointer to the target field's type
		Body:       fmt.Sprintf("return &%s", targetFieldAccessor), // Takes address of the nested field access
	}

	// Create the call to the runtime validation function.
	f := Function(
		fieldComparisonTagName,               // Name for bookkeeping/debugging
		DefaultFlags,                         // Use default generation flags
		validateFieldComparisonValidateField, // The runtime function to call
		field1PathString,                     // Pass original path string for runtime errors
		operator,                             // The comparison operator
		field2PathString,                     // Pass original path string
		targetFieldPathString,                // Pass original path string
		getField1Fn,                          // Generated getter func
		getField2Fn,                          // Generated getter func
		getTargetFieldFn,                     // Generated getter func (returns pointer)
		// Wrap the nested validator function, specifying the type it operates on
		WrapperFunction{nestedValidatorFunc, targetFieldMemb.Type},
	)
	result.Functions = append(result.Functions, f)

	// Include any Go variables needed by the nested validator (e.g., sets for +k8s:enum)
	result.Variables = append(result.Variables, nestedValidations.Variables...)

	return result, nil
}

// func (fctv fieldComparisonTagValidator) Docs() TagDoc {
// 	doc := TagDoc{
// 		Tag:    fctv.TagName(),
// 		Scopes: fctv.ValidScopes().UnsortedList(),
// 		Description: fmt.Sprintf("Conditionally runs a nested validation rule on a specific target field if the comparison between two other fields is true. "+
// 			"Supported operators: %v.", supportedOperators.UnsortedList()),
// 		Args: []TagArgDoc{
// 			{Name: "<field1>", Description: "The JSON name of the first field in the comparison."},
// 			{Name: "<operator>", Description: "The comparison operator (e.g., '==', '!=', '<', '<=', '>', '>=')."},
// 			{Name: "<field2>", Description: "The JSON name of the second field in the comparison."},
// 			{Name: "<fieldToValidate>", Description: "The JSON name of the field to apply the <validator> payload to if the comparison is true."}, // Added fourth arg
// 		},
// 		Payloads: []TagPayloadDoc{{
// 			Name:        "<validator>",
// 			Description: "The validation tag (including '+k8s:') to apply to the <fieldToValidate> if the comparison `field1 operator field2` evaluates to true.", // Updated description
// 			Examples:    []string{"+k8s:minimum=1", "+k8s:maxLength=10"},
// 		}},
// 		Examples: []string{
// 			"+k8s:fieldComparison(Replicas, >, 0, Selector)=+k8s:required", // If Replicas > 0, validate Selector with +k8s:required
// 			"+k8s:fieldComparison(EndTime, >=, StartTime, Replicas)=+k8s:minimum=1", // If EndTime >= StartTime, validate Replicas with +k8s:minimum=1
// 		},
// 	}
// 	return doc
// }

// TODO(aaron-prindle) FIXME - additional arg added
func (fctv fieldComparisonTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:    fctv.TagName(),
		Scopes: fctv.ValidScopes().UnsortedList(),
		Description: fmt.Sprintf("Conditionally runs a nested validation rule on the struct if the comparison between two fields is true. "+
			"Supported operators: %v.", supportedOperators.UnsortedList()),
		Args: []TagArgDoc{
			{Description: "The JSON name of the first field in the comparison."},
			{Description: "The comparison operator (e.g., '==', '!=', '<', '<=', '>', '>=')."},
			{Description: "The JSON name of the second field in the comparison."},
		},
		Payloads: []TagPayloadDoc{{
			Description: "<validator>",
			Docs:        "The validator to apply to the target field if the condition field is specified.",
		}},
	}
	return doc
}
