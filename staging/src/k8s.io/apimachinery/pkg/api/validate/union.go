/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Union verifies that exactly one member of a union is specified.
//
// UnionMembership must define all the members of the union.
//
// For example:
//
//	var abcUnionMembership := schema.NewUnionMembership("a", "b", "c")
//	func ValidateABC(opCtx operation.Context, fldPath *field.Path, in *ABC) (errs fields.ErrorList) {
//		errs = append(errs, Union(opCtx, fldPath, in, abcUnionMembership, in.A, in.B, in.C)...)
//		return errs
//	}
func Union(opCtx operation.Context, fldPath *field.Path, _, _ any, union *UnionMembership, fieldValues ...any) field.ErrorList {
	if len(union.members) != len(fieldValues) {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("unexpected difference in length between fields defined in UnionMembership and fieldValues"))}
	}
	var specifiedMember *string
	for i, fieldValue := range fieldValues {
		rv := reflect.ValueOf(fieldValue)
		if rv.IsValid() && !rv.IsZero() {
			m := union.members[i]
			if specifiedMember != nil && *specifiedMember != m.memberName {
				return field.ErrorList{
					field.Invalid(fldPath, fmt.Sprintf("{%s}", strings.Join(union.specifiedFields(fieldValues), ", ")),
						fmt.Sprintf("must specify exactly one of: %s", strings.Join(union.allFields(), ", "))),
				}
			}
			name := m.memberName
			specifiedMember = &name
		}
	}
	if specifiedMember == nil {
		return field.ErrorList{field.Invalid(fldPath, "",
			fmt.Sprintf("must specify exactly one of: %s",
				strings.Join(union.allFields(), ", ")))}
	}
	return nil
}

// DiscriminatedUnion verifies specified union member matches the discriminator.
//
// UnionMembership must define all the members of the union and the discriminator.
//
// For example:
//
//	var abcUnionMembership := schema.NewDiscriminatedUnionMembership("type", "a", "b", "c")
//	func ValidateABC(opCtx operation.Context, fldPath, *field.Path, in *ABC) (errs fields.ErrorList) {
//		errs = append(errs, DiscriminatedUnion(opCtx, fldPath, in, abcUnionMembership, in.Type, in.A, in.B, in.C)...)
//		return errs
//	}
func DiscriminatedUnion[T ~string](opCtx operation.Context, fldPath *field.Path, _, _ any, union *UnionMembership, discriminatorValue T, fieldValues ...any) (errs field.ErrorList) {
	discriminatorStrValue := string(discriminatorValue)
	if len(union.members) != len(fieldValues) {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("unexpected difference in length between fields defined in UnionMembership and fieldValues"))}
	}
	for i, fieldValue := range fieldValues {
		member := union.members[i]
		isDiscriminatedMember := discriminatorStrValue == member.memberName
		rv := reflect.ValueOf(fieldValue)
		isSpecified := rv.IsValid() && !rv.IsZero()
		if isSpecified && !isDiscriminatedMember {
			errs = append(errs, field.Invalid(fldPath.Child(member.fieldName), "",
				fmt.Sprintf("may only be specified when `%s` is %q", union.discriminatorName, discriminatorValue)))
		} else if !isSpecified && isDiscriminatedMember {
			errs = append(errs, field.Invalid(fldPath.Child(member.fieldName), "",
				fmt.Sprintf("must be specified when `%s` is %q", union.discriminatorName, discriminatorValue)))
		}
	}
	return errs
}

// RequiredIf verifies that a field is required when a CEL condition evaluates to true.
//
// For example:
//
//	func ValidateMyStruct(opCtx operation.Context, obj, oldObj *MyStruct, fldPath *field.Path) field.ErrorList {
//		return RequiredIf(opCtx, fldPath.Child("FieldA"), obj.FieldA, obj, "Type == 'EnumA'")
//	}
// func Union(opCtx operation.Context, fldPath *field.Path, _, _ any, union *UnionMembership, fieldValues ...any) field.ErrorList {

func RequiredIf(opCtx operation.Context, fldPath *field.Path, obj, oldObj any, condition string, fieldValues ...any) field.ErrorList {
	var errs field.ErrorList

	// Convert obj to a map[string]interface{} for activation
	objMap, err := structToMap(obj)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("failed to convert object to map: %v", err))}
	}

	// Create declarations for the fields in obj
	declsList := []*expr.Decl{}
	for k, v := range objMap {
		var declType *expr.Type
		switch v.(type) {
		case string:
			declType = decls.String
		case int, int32, int64:
			declType = decls.Int
		case float32, float64:
			declType = decls.Double
		case bool:
			declType = decls.Bool
		default:
			declType = decls.Dyn
		}
		declsList = append(declsList, decls.NewVar(k, declType))
	}

	// Create a CEL environment with variable declarations
	env, err := cel.NewEnv(
		cel.Declarations(declsList...),
	)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("failed to create CEL environment: %v", err))}
	}

	// Parse the CEL expression
	ast, issues := env.Parse(condition)
	if issues != nil && issues.Err() != nil {
		return field.ErrorList{field.Invalid(fldPath, condition, fmt.Sprintf("invalid CEL expression: %v", issues.Err()))}
	}

	// Check the type of the expression
	checkedAst, issues := env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return field.ErrorList{field.Invalid(fldPath, condition, fmt.Sprintf("type-check error in CEL expression: %v", issues.Err()))}
	}

	// Programmatically evaluate the expression
	prg, err := env.Program(checkedAst)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("failed to create CEL program: %v", err))}
	}

	// Evaluate the expression with the input variables
	out, _, err := prg.Eval(objMap)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, fmt.Errorf("failed to evaluate CEL expression: %v", err))}
	}

	// Check if the condition is true
	conditionMet, ok := out.Value().(bool)
	if !ok {
		return field.ErrorList{field.Invalid(fldPath, condition, "CEL expression did not return a boolean")}
	}

	if conditionMet {
		rv := reflect.ValueOf(fieldValues[0])
		// rv := reflect.ValueOf(fieldValue)
		if !rv.IsValid() || rv.IsZero() {
			return field.ErrorList{
				field.Required(fldPath, fmt.Sprintf("field is required when: %s", condition)),
			}
		}
	}

	return errs
}

// Helper function to convert a struct to a map[string]interface{}
func structToMap(obj any) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil, fmt.Errorf("obj must be a non-nil pointer to a struct")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("obj must point to a struct")
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name
		// Include only exported fields
		if fieldType.PkgPath != "" {
			continue
		}
		result[fieldName] = fieldVal.Interface()
	}
	return result, nil
}

type member struct {
	fieldName, memberName string
}

// UnionMembership represents an ordered list of field union memberships.
type UnionMembership struct {
	discriminatorName string
	members           []member
}

// NewUnionMembership returns a new UnionMembership for the given list of members.
//
// Each member is a [2]string to provide a fieldName and memberName pair, where
// [0] identifies the field name and [1] identifies the union member Name.
//
// Field names must be unique.
func NewUnionMembership(member ...[2]string) *UnionMembership {
	return NewDiscriminatedUnionMembership("", member...)
}

// NewDiscriminatedUnionMembership returns a new UnionMembership for the given discriminator field and list of members.
// members are provided in the same way as for NewUnionMembership.
func NewDiscriminatedUnionMembership(discriminatorFieldName string, members ...[2]string) *UnionMembership {
	u := &UnionMembership{}
	u.discriminatorName = discriminatorFieldName
	for _, fieldName := range members {
		u.members = append(u.members, member{fieldName: fieldName[0], memberName: fieldName[1]})
	}
	return u
}

// specifiedFields returns a string listing all the field names of the specified fieldValues for use in error reporting.
func (u UnionMembership) specifiedFields(fieldValues []any) []string {
	var membersSpecified []string
	for i, fieldValue := range fieldValues {
		rv := reflect.ValueOf(fieldValue)
		if rv.IsValid() && !rv.IsZero() {
			f := u.members[i]
			membersSpecified = append(membersSpecified, f.fieldName)
		}
	}
	return membersSpecified
}

// allFields returns a string listing all the field names of the member of a union for use in error reporting.
func (u UnionMembership) allFields() []string {
	memberNames := make([]string, 0, len(u.members))
	for _, f := range u.members {
		memberNames = append(memberNames, fmt.Sprintf("`%s`", f.fieldName))
	}
	return memberNames
}

// EvaluateCondition evaluates a condition based on the operator and values provided.
//
// Supported operators: ==, !=, >, >=, <, <=
//
// Both fieldValue and value are treated as strings; if possible, they are converted to numbers for numeric comparisons.
func EvaluateCondition(fieldValue any, operator string, value string) (bool, error) {
	fv := fmt.Sprintf("%v", fieldValue)

	switch operator {
	case "==":
		return fv == value, nil
	case "!=":
		return fv != value, nil
	case ">", ">=", "<", "<=":
		// Attempt to parse both values as numbers
		fvNum, err1 := strconv.ParseFloat(fv, 64)
		valNum, err2 := strconv.ParseFloat(value, 64)
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("non-numeric value in numeric comparison")
		}
		switch operator {
		case ">":
			return fvNum > valNum, nil
		case ">=":
			return fvNum >= valNum, nil
		case "<":
			return fvNum < valNum, nil
		case "<=":
			return fvNum <= valNum, nil
		}
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
	return false, fmt.Errorf("invalid comparison")
}
