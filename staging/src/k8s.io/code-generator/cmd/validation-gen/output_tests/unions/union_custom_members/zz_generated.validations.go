//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by validation-gen. DO NOT EDIT.

package union_custom_members

import (
	fmt "fmt"

	operation "k8s.io/apimachinery/pkg/api/operation"
	safe "k8s.io/apimachinery/pkg/api/safe"
	validate "k8s.io/apimachinery/pkg/api/validate"
	field "k8s.io/apimachinery/pkg/util/validation/field"
	testscheme "k8s.io/code-generator/cmd/validation-gen/testscheme"
)

func init() { localSchemeBuilder.Register(RegisterValidations) }

// RegisterValidations adds validation functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterValidations(scheme *testscheme.Scheme) error {
	scheme.AddValidationFunc((*T1)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T1(opCtx, obj.(*T1), safe.Cast[T1](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*U)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_U(opCtx, obj.(*U), safe.Cast[U](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	return nil
}

func Validate_M1(opCtx operation.Context, obj, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
	// type M1
	if obj != nil {
		errs = append(errs, validate.FixedResult(fldPath, *obj, true, "type M1")...)
	}

	// field M1.S
	errs = append(errs,
		func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field M1.S")...)
			return
		}(obj.S, safe.Field(oldObj, func(oldObj M1) *string { return &oldObj.S }), fldPath.Child("s"))...)

	return errs
}

func Validate_M2(opCtx operation.Context, obj, oldObj *M2, fldPath *field.Path) (errs field.ErrorList) {
	// type M2
	if obj != nil {
		errs = append(errs, validate.FixedResult(fldPath, *obj, true, "type M2")...)
	}

	// field M2.S
	errs = append(errs,
		func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field M2.S")...)
			return
		}(obj.S, safe.Field(oldObj, func(oldObj M2) *string { return &oldObj.S }), fldPath.Child("s"))...)

	return errs
}

func Validate_T1(opCtx operation.Context, obj, oldObj *T1, fldPath *field.Path) (errs field.ErrorList) {
	// type T1
	if obj != nil {
		errs = append(errs, validate.FixedResult(fldPath, *obj, true, "type T1")...)
	}

	// field T1.TypeMeta has no validation

	// field T1.LS
	errs = append(errs,
		func(obj []string, oldObj []string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T1.LS")...)
			for i, val := range obj {
				errs = append(errs,
					func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
						if e := validate.Required(fldPath, obj); len(e) != 0 {
							errs = append(errs, e...)
							return // fatal
						}
						errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T1.LS[*]")...)
						return
					}(val, nil, fldPath.Index(i))...)
			}
			return
		}(obj.LS, safe.Field(oldObj, func(oldObj T1) []string { return oldObj.LS }), fldPath.Child("ls"))...)

	return errs
}

var unionMembershipForU = validate.NewUnionMembership([2]string{"m1", "CustomM1"}, [2]string{"m2", "CustomM2"})

func Validate_U(opCtx operation.Context, obj, oldObj *U, fldPath *field.Path) (errs field.ErrorList) {
	// type U
	if obj != nil {
		errs = append(errs, validate.Union(fldPath, *obj, unionMembershipForU, obj.M1, obj.M2)...)
	}

	// field U.TypeMeta has no validation

	// field U.M1
	errs = append(errs,
		func(obj *M1, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
			if obj != nil {
				errs = append(errs, Validate_M1(opCtx, obj, oldObj, fldPath)...)
			}
			return
		}(obj.M1, safe.Field(oldObj, func(oldObj U) *M1 { return oldObj.M1 }), fldPath.Child("m1"))...)

	// field U.M2
	errs = append(errs,
		func(obj *M2, oldObj *M2, fldPath *field.Path) (errs field.ErrorList) {
			if obj != nil {
				errs = append(errs, Validate_M2(opCtx, obj, oldObj, fldPath)...)
			}
			return
		}(obj.M2, safe.Field(oldObj, func(oldObj U) *M2 { return oldObj.M2 }), fldPath.Child("m2"))...)

	// field U.T1
	errs = append(errs,
		func(obj *T1, oldObj *T1, fldPath *field.Path) (errs field.ErrorList) {
			if obj != nil {
				errs = append(errs, Validate_T1(opCtx, obj, oldObj, fldPath)...)
			}
			return
		}(obj.T1, safe.Field(oldObj, func(oldObj U) *T1 { return oldObj.T1 }), fldPath.Child("t1"))...)

	return errs
}
