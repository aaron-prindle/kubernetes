// staging/src/k8s.io/code-generator/cmd/validation-gen/validators/field_comparison.go
package validators

import (
	"fmt"
	"strings" // Import strings

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/v2/types"
	// field "k8s.io/apimachinery/pkg/util/validation/field" // Use field package alias
	// "k8s.io/gengo/v2/generator" // No longer needed
	// "k8s.io/gengo/v2/namer" // No longer needed
)

const (
	fieldComparisonTagName = "k8s:fieldComparison"
)

// Define supported operators
var supportedOperators = sets.New(
	"==", "!=", "<", "<=", ">", ">=",
)

// Helper to check if a type is numeric
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

// NOTE: generatePayloadCallString is NO LONGER NEEDED with WrapperFunction

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

// Name of the runtime helper function we assume exists
var (
	validateFieldComparisonConditional = types.Name{Package: libValidationPkg, Name: "FieldComparisonConditional"}
)

func (fctv fieldComparisonTagValidator) GetValidations(context Context, args []string, payload string) (Validations, error) {
	t := realType(context.Type)
	if t == nil || t.Kind != types.Struct {
		return Validations{}, fmt.Errorf("tag %q can only be used on struct types, got %v", fctv.TagName(), context.Type)
	}

	// --- Argument Parsing (Expecting 4 args) ---
	if len(args) != 4 {
		return Validations{}, fmt.Errorf("tag %q requires exactly four arguments: <field1Path>, <operator>, <field2Path>, <targetFieldPath>", fctv.TagName())
	}
	field1PathString := args[0]
	operator := args[1]
	field2PathString := args[2]
	targetFieldPathString := args[3]

	// --- Validation (Operator, Payload) ---
	if !supportedOperators.Has(operator) {
		return Validations{}, fmt.Errorf("invalid operator %q. Supported: %v", operator, supportedOperators.UnsortedList())
	}
	if payload == "" || !strings.HasPrefix(payload, "+k8s:") {
		return Validations{}, fmt.Errorf("tag %q requires a payload starting with '+k8s:', got %q", fctv.TagName(), payload)
	}
	payloadComment := payload

	// --- Resolve Field Paths ---
	field1Members, err := findMembersByPath(t, field1PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field1 path %q: %w", field1PathString, err)
	}
	field1Memb := field1Members[len(field1Members)-1]
	field1OriginalType := field1Memb.Type
	field1UnderlyingType := realType(field1OriginalType)
	field2Members, err := findMembersByPath(t, field2PathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid field2 path %q: %w", field2PathString, err)
	}
	field2Memb := field2Members[len(field2Members)-1]
	field2OriginalType := field2Memb.Type
	field2UnderlyingType := realType(field2OriginalType)
	targetFieldMembers, err := findMembersByPath(t, targetFieldPathString)
	if err != nil {
		return Validations{}, fmt.Errorf("invalid target field path %q: %w", targetFieldPathString, err)
	}
	targetFieldMemb := targetFieldMembers[len(targetFieldMembers)-1]
	targetFieldOriginalType := targetFieldMemb.Type

	// --- Type Compatibility Check (Comparison Fields) ---
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
	targetFieldContext := Context{Scope: ScopeField, Type: targetFieldOriginalType, Parent: t, Member: targetFieldMemb, Path: context.Path.Child(targetFieldPathString)} // Path for context
	payloadValidations, err := fctv.validator.ExtractValidations(targetFieldContext, []string{payloadComment})
	if err != nil {
		return Validations{}, fmt.Errorf("error extracting payload validations for target field %q (%s): %w", targetFieldPathString, payloadComment, err)
	}
	if len(payloadValidations.Functions) == 0 {
		return Validations{}, fmt.Errorf("payload tag %q did not generate any validation functions for target field %q (type %s)", payloadComment, targetFieldPathString, targetFieldOriginalType.String())
	}
	if len(payloadValidations.Functions) > 1 {
		return Validations{}, fmt.Errorf("payload tag %q generated multiple validation functions for target field %q; only one is supported", payloadComment, targetFieldPathString)
	}
	payloadFuncInfo := payloadValidations.Functions[0]

	// === Define FunctionLiteral for Comparison ===
	var compBodyBuilder strings.Builder
	compBodyBuilder.WriteString("  // Comparison Body\n")
	field1Accessor := generateNestedFieldAccessor("obj", field1Members)
	field2Accessor := generateNestedFieldAccessor("obj", field2Members)
	needsNilCheck1 := field1OriginalType.Kind == types.Pointer
	needsNilCheck2 := field2OriginalType.Kind == types.Pointer
	val1Var := field1Memb.Name + "CompVal"
	val2Var := field2Memb.Name + "CompVal"
	comparisonExprString := ""
	var conditionParts []string
	var comparisonExpr strings.Builder
	if needsNilCheck1 {
		compBodyBuilder.WriteString(fmt.Sprintf("  %s := %s\n", val1Var, field1Accessor))
	}
	if needsNilCheck2 {
		compBodyBuilder.WriteString(fmt.Sprintf("  %s := %s\n", val2Var, field2Accessor))
	}
	compPart1 := field1Accessor
	if needsNilCheck1 {
		compPart1 = "*" + val1Var
	}
	compPart2 := field2Accessor
	if needsNilCheck2 {
		compPart2 = "*" + val2Var
	}
	if operator == "==" {
		if needsNilCheck1 && needsNilCheck2 {
			comparisonExpr.WriteString(fmt.Sprintf("((%s == nil && %s == nil) || (%s != nil && %s != nil && %s == %s))", val1Var, val2Var, val1Var, val2Var, compPart1, compPart2))
		} else if needsNilCheck1 {
			comparisonExpr.WriteString(fmt.Sprintf("(%s != nil && %s == %s)", val1Var, compPart1, compPart2))
		} else if needsNilCheck2 {
			comparisonExpr.WriteString(fmt.Sprintf("(%s != nil && %s == %s)", val2Var, compPart1, compPart2))
		} else {
			comparisonExpr.WriteString(fmt.Sprintf("(%s == %s)", compPart1, compPart2))
		}
	} else if operator == "!=" {
		if needsNilCheck1 && needsNilCheck2 {
			comparisonExpr.WriteString(fmt.Sprintf("((%s == nil && %s != nil) || (%s != nil && %s == nil) || (%s != nil && %s != nil && %s != %s))", val1Var, val2Var, val1Var, val2Var, val1Var, val2Var, compPart1, compPart2))
		} else if needsNilCheck1 {
			comparisonExpr.WriteString(fmt.Sprintf("(%s == nil || %s != %s)", val1Var, compPart1, compPart2))
		} else if needsNilCheck2 {
			comparisonExpr.WriteString(fmt.Sprintf("(%s == nil || %s != %s)", val2Var, compPart1, compPart2))
		} else {
			comparisonExpr.WriteString(fmt.Sprintf("(%s != %s)", compPart1, compPart2))
		}
	} else {
		if needsNilCheck1 {
			conditionParts = append(conditionParts, fmt.Sprintf("%s != nil", val1Var))
		}
		if needsNilCheck2 {
			conditionParts = append(conditionParts, fmt.Sprintf("%s != nil", val2Var))
		}
		comparisonExpr.WriteString(fmt.Sprintf("(%s %s %s)", compPart1, operator, compPart2))
	}
	comparisonExprString = comparisonExpr.String()
	compBodyBuilder.WriteString("  comparisonHolds := false\n")
	if len(conditionParts) > 0 {
		compBodyBuilder.WriteString(fmt.Sprintf("  if %s {\n    comparisonHolds = %s\n  }\n", strings.Join(conditionParts, " && "), comparisonExprString))
	} else {
		compBodyBuilder.WriteString(fmt.Sprintf("  comparisonHolds = %s\n", comparisonExprString))
	}
	compBodyBuilder.WriteString("  return comparisonHolds\n")

	nilableStructType := context.Type
	if !isNilableType(nilableStructType) {
		nilableStructType = types.PointerTo(nilableStructType)
	}

	comparisonFuncLiteral := FunctionLiteral{
		Parameters: []ParamResult{{"obj", nilableStructType}},
		Results:    []ParamResult{{"", types.Bool}},
		Body:       compBodyBuilder.String(),
	}

	// === Define FunctionLiteral for Target Field Getter ===
	nilableTargetFieldType := targetFieldOriginalType
	targetFieldExprPrefix := ""
	if !isNilableType(nilableTargetFieldType) {
		nilableTargetFieldType = types.PointerTo(targetFieldOriginalType)
		targetFieldExprPrefix = "&"
	}
	targetGetFnAccessor := generateNestedFieldAccessor("o", targetFieldMembers)
	targetGetFnBody := fmt.Sprintf("return %s%s", targetFieldExprPrefix, targetGetFnAccessor)

	targetGetFnLiteral := FunctionLiteral{
		Parameters: []ParamResult{{"o", nilableStructType}},
		Results:    []ParamResult{{"", nilableTargetFieldType}},
		Body:       targetGetFnBody,
	}

	// === Create the Main FunctionGen calling the Runtime Helper ===
	result := Validations{}

	// Create FunctionGen
	f := FunctionGen{
		TagName:  fieldComparisonTagName,
		Flags:    payloadFuncInfo.Flags,
		Function: validateFieldComparisonConditional,
		Args: []any{ // Set the regular arguments
			comparisonFuncLiteral,
			targetFieldPathString,
			targetGetFnLiteral,
			WrapperFunction{payloadFuncInfo, targetFieldOriginalType},
		},
	}

	result.Functions = append(result.Functions, f)
	result.Variables = append(result.Variables, payloadValidations.Variables...)

	return result, nil
}

func (fctv fieldComparisonTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:    fctv.TagName(),
		Scopes: fctv.ValidScopes().UnsortedList(),
		Description: fmt.Sprintf("Generates code to conditionally run a payload validation rule on a specified *target field* if the comparison between two other fields (field1 operator field2) is true. "+
			"If the comparison is false, an Invalid error is generated attached to the struct's path. "+
			"Supported operators: %v. Handles basic numeric, string, bool types, and pointers. Requires the `validate.FieldComparisonConditional` runtime helper.", supportedOperators.UnsortedList()),
		Args: []TagArgDoc{},
		Payloads: []TagPayloadDoc{{
			Description: "The validation tag (including '+k8s:') to apply to the <targetFieldPath> if the comparison evaluates to true.",
		}},
	}
	return doc
}
