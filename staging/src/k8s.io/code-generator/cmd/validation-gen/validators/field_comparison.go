// staging/src/k8s.io/code-generator/cmd/validation-gen/validators/field_comparison.go
package validators

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/v2/types"
	// Need field package for generated code types
)

const (
	fieldComparisonTagName = "k8s:fieldComparison"
	// Import paths for generated code
	libFieldPathPkg = "k8s.io/apimachinery/pkg/util/validation/field"
	libOperationPkg = "k8s.io/apimachinery/pkg/api/operation" // Needed for op type
	libContextPkg   = "context"                               // Needed for context type
	libFmtPkg       = "fmt"                                   // Needed for error message formatting
)

// Define supported operators (unchanged)
var supportedOperators = sets.New(
	"==", "!=", "<", "<=", ">", ">=",
)

// Helper to check if a type is numeric (unchanged)
func isNumericType(t *types.Type) bool {
	if t == nil {
		return false
	}
	t = realType(t)
	if t.Kind != types.Builtin {
		return false
	}
	name := t.Name.Name
	return strings.Contains(name, "int") || strings.Contains(name, "float") || strings.Contains(name, "byte") || strings.Contains(name, "rune")
}

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

// Name of the modified runtime helper function
var (
	validateFieldComparisonConditional = types.Name{Package: libValidationPkg, Name: "FieldComparisonConditional"}
	// Types needed for the generated function literal signature
	errorListType = types.Name{Package: libFieldPathPkg, Name: "ErrorList"}
	fieldInvalid  = types.Name{Package: libFieldPathPkg, Name: "Invalid"} // For error generation
	fmtSprintf    = types.Name{Package: libFmtPkg, Name: "Sprintf"}       // For error message
)

func (fctv fieldComparisonTagValidator) GetValidations(context Context, args []string, payload string) (Validations, error) {
	structType := realType(context.Type)
	if structType == nil || structType.Kind != types.Struct {
		return Validations{}, fmt.Errorf("tag %q can only be used on struct types, got %v", fctv.TagName(), context.Type)
	}

	// Ensure the struct type used in the function literal signature is a pointer
	ptrStructType := types.PointerTo(structType)

	// --- Argument Parsing (Unchanged) ---
	if len(args) != 4 {
		return Validations{}, fmt.Errorf("tag %q requires exactly four arguments: <field1Path>, <operator>, <field2Path>, <targetFieldPath>", fctv.TagName())
	}
	field1PathString := args[0]
	operator := args[1]
	field2PathString := args[2]
	targetFieldPathString := args[3]

	// --- Validation (Operator, Payload) (Unchanged) ---
	if !supportedOperators.Has(operator) {
		return Validations{}, fmt.Errorf("invalid operator %q. Supported: %v", operator, supportedOperators.UnsortedList())
	}
	if payload == "" || !strings.HasPrefix(payload, "+k8s:") {
		return Validations{}, fmt.Errorf("tag %q requires a payload starting with '+k8s:', got %q", fctv.TagName(), payload)
	}
	payloadComment := payload

	// --- Resolve Field Paths (Unchanged) ---
	field1Members, err := findMembersByPath(structType, field1PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field1 path %q: %w", field1PathString, err)
	}
	field1Memb := field1Members[len(field1Members)-1]
	field1OriginalType := field1Memb.Type
	field1UnderlyingType := realType(field1OriginalType)

	field2Members, err := findMembersByPath(structType, field2PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field2 path %q: %w", field2PathString, err)
	}
	field2Memb := field2Members[len(field2Members)-1]
	field2OriginalType := field2Memb.Type
	field2UnderlyingType := realType(field2OriginalType)

	targetFieldMembers, err := findMembersByPath(structType, targetFieldPathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid target field path %q: %w", targetFieldPathString, err)
	}
	targetFieldMemb := targetFieldMembers[len(targetFieldMembers)-1]
	targetFieldOriginalType := targetFieldMemb.Type

	// --- Type Compatibility Check (Comparison Fields) (Unchanged) ---
	canCompareDirectly := false
	if field1UnderlyingType.Kind == field2UnderlyingType.Kind {
		if field1UnderlyingType.Kind == types.Builtin {
			isNumeric := isNumericType(field1UnderlyingType)
			isString := field1UnderlyingType.Name.Name == "string"
			isBool := field1UnderlyingType.Name.Name == "bool"
			if isNumeric || isString {
				canCompareDirectly = true
			} else if isBool && (operator == "==" || operator == "!=") {
				canCompareDirectly = true
			}
		}
	}
	if !canCompareDirectly && isNumericType(field1UnderlyingType) && isNumericType(field2UnderlyingType) {
		canCompareDirectly = true
	}
	if !canCompareDirectly {
		return Validations{}, fmt.Errorf("tag %q: cannot generate direct comparison for operator %q between types %s (%v) and %s (%v)", fctv.TagName(), operator, field1OriginalType.String(), field1UnderlyingType.Kind, field2OriginalType.String(), field2UnderlyingType.Kind)
	}

	// --- Extract Payload Validator (Applied to the *target* field) ---
	// Create context for the target field to extract its validation
	targetFieldContext := Context{
		Scope:  ScopeField,
		Type:   targetFieldOriginalType,
		Parent: structType,
		Member: targetFieldMemb,
		// Use the original struct's path as base for target path context
		Path: context.Path, //.Child(targetFieldPathString) - Path is built inside generated func
	}
	payloadValidations, err := fctv.validator.ExtractValidations(targetFieldContext, []string{payloadComment})
	if err != nil {
		return Validations{}, fmt.Errorf("error extracting payload validations for target field %q (%s): %w", targetFieldPathString, payloadComment, err)
	}
	if len(payloadValidations.Functions) == 0 {
		return Validations{}, fmt.Errorf("payload tag %q did not generate any validation functions for target field %q (type %s)", payloadComment, targetFieldPathString, targetFieldOriginalType.String())
	}
	if len(payloadValidations.Functions) > 1 {
		// Might need to combine them later if this becomes common
		return Validations{}, fmt.Errorf("payload tag %q generated multiple validation functions for target field %q; only one is supported by fieldComparison", payloadComment, targetFieldPathString)
	}
	payloadFuncInfo := payloadValidations.Functions[0]

	// --- Determine Target Field Type for Payload Validator ---
	// This logic needs to align with how WrapperFunction worked. It determines
	// the type expected by the payload validator function (often a pointer).
	payloadArgType := targetFieldOriginalType
	if isNilableType(targetFieldOriginalType) {
		// If the field type itself is a pointer, map, slice, assume validator takes it directly
		payloadArgType = targetFieldOriginalType
	} else {
		// If field type is value type (struct, int, string), assume validator takes a pointer
		payloadArgType = types.PointerTo(targetFieldOriginalType)
		// Exception: If the payload is known to operate on value types, adjust here.
		// This requires more info about the payloadFuncInfo or conventions.
		// Sticking to the pointer-unless-already-pointer logic from original code.
	}
	payloadArgTypeName := payloadArgType.Name.String() // For var declarations

	// === Build the Body of the Combined Validation Logic Function ===
	var logicBodyBuilder strings.Builder
	// TODO(aaron-prindle) FIXME - using fieldpath hack for now
	logicBodyBuilder.WriteString(fmt.Sprintf("  var errs %s\n\n", filepath.Base(errorListType.String()))) // Use qualified name? Assuming import.

	// 1. Generate Comparison Condition String
	field1Accessor := generateNestedFieldAccessor("newObj", field1Members)
	field2Accessor := generateNestedFieldAccessor("newObj", field2Members)
	needsNilCheck1 := field1OriginalType.Kind == types.Pointer
	needsNilCheck2 := field2OriginalType.Kind == types.Pointer
	val1Var := "compVal1" // Local var names inside the function body
	val2Var := "compVal2"

	compPart1 := field1Accessor
	if needsNilCheck1 {
		compPart1 = "*" + val1Var
	}
	compPart2 := field2Accessor
	if needsNilCheck2 {
		compPart2 = "*" + val2Var
	}

	var comparisonConditionBuilder strings.Builder // Builds the condition expression
	var preComparisonStmts []string                // Statements needed before the 'if' (e.g., var decls for nil checks)

	// Handle == and != with explicit nil checks
	if operator == "==" {
		if needsNilCheck1 && needsNilCheck2 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val1Var, field1Accessor))
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val2Var, field2Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("(%s == nil && %s == nil) || (%s != nil && %s != nil && %s == %s)", val1Var, val2Var, val1Var, val2Var, compPart1, compPart2))
		} else if needsNilCheck1 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val1Var, field1Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s != nil && %s == %s", val1Var, compPart1, compPart2))
		} else if needsNilCheck2 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val2Var, field2Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s != nil && %s == %s", val2Var, compPart1, compPart2))
		} else {
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s == %s", compPart1, compPart2))
		}
	} else if operator == "!=" {
		if needsNilCheck1 && needsNilCheck2 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val1Var, field1Accessor))
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val2Var, field2Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("(%s == nil && %s != nil) || (%s != nil && %s == nil) || (%s != nil && %s != nil && %s != %s)", val1Var, val2Var, val1Var, val2Var, val1Var, val2Var, compPart1, compPart2))
		} else if needsNilCheck1 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val1Var, field1Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s == nil || %s != %s", val1Var, compPart1, compPart2))
		} else if needsNilCheck2 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val2Var, field2Accessor))
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s == nil || %s != %s", val2Var, compPart1, compPart2))
		} else {
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s != %s", compPart1, compPart2))
		}
	} else { // <, <=, >, >= operators
		var conditionParts []string // Conditions required *before* the actual comparison
		comparisonExpr := fmt.Sprintf("%s %s %s", compPart1, operator, compPart2)

		if needsNilCheck1 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val1Var, field1Accessor))
			conditionParts = append(conditionParts, fmt.Sprintf("%s != nil", val1Var))
		}
		if needsNilCheck2 {
			preComparisonStmts = append(preComparisonStmts, fmt.Sprintf("%s := %s", val2Var, field2Accessor))
			conditionParts = append(conditionParts, fmt.Sprintf("%s != nil", val2Var))
		}

		if len(conditionParts) > 0 {
			comparisonConditionBuilder.WriteString(fmt.Sprintf("%s && (%s)", strings.Join(conditionParts, " && "), comparisonExpr))
		} else {
			comparisonConditionBuilder.WriteString(comparisonExpr)
		}
	}

	// Add any pre-comparison statements (variable assignments for nil checks)
	if len(preComparisonStmts) > 0 {
		for _, stmt := range preComparisonStmts {
			logicBodyBuilder.WriteString(fmt.Sprintf("  %s\n", stmt))
		}
		logicBodyBuilder.WriteString("\n")
	}

	logicBodyBuilder.WriteString(fmt.Sprintf("  if %s {\n", comparisonConditionBuilder.String()))

	// 2. Calculate Target Field Path
	targetPathVar := "targetPath" // Local var name for the path
	logicBodyBuilder.WriteString(fmt.Sprintf("    %s := fldPath", targetPathVar))
	for _, part := range strings.Split(targetFieldPathString, ".") {
		if part != "" {
			logicBodyBuilder.WriteString(fmt.Sprintf(".Child(%q)", part))
		}
	}
	logicBodyBuilder.WriteString("\n\n")

	// 3. Get New and Old Target Field Values
	newTargetValVar := "newTargetFieldValue"
	oldTargetValVar := "oldTargetFieldValue"
	targetAccessorNew := generateNestedFieldAccessor("newObj", targetFieldMembers)
	targetAccessorOld := generateNestedFieldAccessor("oldObj", targetFieldMembers) // Assumes oldObj exists and has same structure within check

	// Handle getting address (&) if needed by payload validator
	newTargetValExpr := targetAccessorNew
	oldTargetValExpr := targetAccessorOld
	if payloadArgType.Kind == types.Pointer && targetFieldOriginalType.Kind != types.Pointer {
		newTargetValExpr = "&" + newTargetValExpr
		oldTargetValExpr = "&" + oldTargetValExpr
	}
	// Note: Case where payload wants T but field is *T is complex for inline.
	// Assume validators handle nil pointers appropriately or expect pointers.

	logicBodyBuilder.WriteString(fmt.Sprintf("    var %s %s\n", newTargetValVar, payloadArgTypeName))
	logicBodyBuilder.WriteString(fmt.Sprintf("    %s = %s\n\n", newTargetValVar, newTargetValExpr)) // Assign potentially address-of expression

	logicBodyBuilder.WriteString(fmt.Sprintf("    var %s %s\n", oldTargetValVar, payloadArgTypeName))
	logicBodyBuilder.WriteString("    if oldObj != nil {\n")
	// Safe access for old value - simple version assuming direct access ok after nil check
	// A more robust version would use the safe getter logic from the previous attempt if needed for nested pointers in oldObj
	if payloadArgType.Kind == types.Pointer && targetFieldOriginalType.Kind == types.Pointer {
		// If both expect/are pointers, check the field itself before assigning
		logicBodyBuilder.WriteString(fmt.Sprintf("      if %s != nil {\n", targetAccessorOld))
		logicBodyBuilder.WriteString(fmt.Sprintf("          %s = %s\n", oldTargetValVar, oldTargetValExpr))
		logicBodyBuilder.WriteString("      }\n")
	} else {
		// Assign directly or with address-of
		logicBodyBuilder.WriteString(fmt.Sprintf("      %s = %s\n", oldTargetValVar, oldTargetValExpr))
	}
	logicBodyBuilder.WriteString("    }\n\n")

	// 4. Call the Payload Validator
	// Construct the arguments for the payload validator function
	// Assumes signature: func(ctx, op, fldPath, new, old) errs
	payloadArgs := []string{
		"ctx",           // from wrapper func param
		"op",            // from wrapper func param
		targetPathVar,   // calculated path
		newTargetValVar, // retrieved new value
		oldTargetValVar, // retrieved old value
		// TODO(aaron-prindle) need to add stuff here
		// TODO(aaron-prindle) FIXME, HACK - hardcoded for now
		"false",
		"\"field ExampleStruct.B\"",
	}
	// Use the FunctionGen Name information directly
	payloadFuncName := payloadFuncInfo.Function.String() // Assumes Function is types.Name{Package, Name}

	logicBodyBuilder.WriteString("    // Call payload validator\n")
	logicBodyBuilder.WriteString(fmt.Sprintf("    errs = append(errs, %s(%s)...)\n",
		filepath.Base(payloadFuncName), // Use the qualified name
		strings.Join(payloadArgs, ", ")))

	logicBodyBuilder.WriteString("  } else {\n")
	logicBodyBuilder.WriteString("    // --- Comparison False: Generate Invalid Error ---\n")
	errorDetail := fmt.Sprintf("comparison failed: %s %s %s requires %s to be valid", field1PathString, operator, field2PathString, targetFieldPathString)
	// field.Invalid(fldPath *Path, value interface{}, detail string)
	logicBodyBuilder.WriteString(fmt.Sprintf("    errs = append(errs, %s(fldPath, newObj, %s(%q)))\n",
		filepath.Base(fieldInvalid.String()), // Use qualified name
		fmtSprintf.String(),                  // Use qualified name
		errorDetail))
	logicBodyBuilder.WriteString("  }\n\n")

	logicBodyBuilder.WriteString("  return errs\n")

	// === Define the FunctionLiteral ===
	// Define the parameter types using ParseFullyQualifiedName
	ctxType := &types.Type{Name: types.ParseFullyQualifiedName(libContextPkg + ".Context")}
	opType := &types.Type{Name: types.ParseFullyQualifiedName(libOperationPkg + ".Operation")}
	// Define the Path type *before* taking its pointer
	pathType := &types.Type{Name: types.ParseFullyQualifiedName(libFieldPathPkg + ".Path")}
	// Define the ErrorList type
	errListType := &types.Type{Name: types.ParseFullyQualifiedName(libFieldPathPkg + ".ErrorList")}

	combinedLogicFuncLiteral := FunctionLiteral{
		Parameters: []ParamResult{
			{"ctx", ctxType},
			{"op", opType},
			// Correctly pass a POINTER to field.Path type
			{"fldPath", types.PointerTo(pathType)},
			{"newObj", ptrStructType}, // Pass pointer to struct
			{"oldObj", ptrStructType}, // Pass pointer to struct
		},
		Results: []ParamResult{{"", errListType}},
		Body:    logicBodyBuilder.String(),
	}

	// === Create the FunctionGen calling the modified Runtime Helper ===
	result := Validations{}

	f := FunctionGen{
		TagName:  fieldComparisonTagName,
		Flags:    payloadFuncInfo.Flags,              // Propagate flags from payload
		Function: validateFieldComparisonConditional, // Call the *modified* helper
		Args: []any{
			// Pass the single function literal as the argument
			combinedLogicFuncLiteral,
		},
	}

	result.Functions = append(result.Functions, f)
	// Propagate variables and imports from the payload validator
	result.Variables = append(result.Variables, payloadValidations.Variables...)

	return result, nil
}

// Docs need update to reflect the single function approach
func (fctv fieldComparisonTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:    fctv.TagName(),
		Scopes: fctv.ValidScopes().UnsortedList(),
		Description: fmt.Sprintf("Generates code that calls the `validate.FieldComparisonConditional` runtime helper. "+
			"The helper is passed a single generated function containing the logic to: "+
			"1) Evaluate the comparison (%s %s %s). "+
			"2) If true, run the payload validation on the target field (%s). "+
			"3) If false, generate a `field.Invalid` error attached to the struct's path. "+
			"Supported operators: %v.", "field1", "op", "field2", "targetField", supportedOperators.UnsortedList()),
		Args: []TagArgDoc{},
		Payloads: []TagPayloadDoc{{
			Description: "The validation tag (including '+k8s:') to apply to the <targetFieldPath> if the comparison evaluates to true.",
		}},
	}
	return doc
}
