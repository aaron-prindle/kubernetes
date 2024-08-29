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

package structs

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
	scheme.AddValidationFunc((*T00)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T00(opCtx, obj.(*T00), safe.Cast[T00](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*T01)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T01(opCtx, obj.(*T01), safe.Cast[T01](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*T02)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T02(opCtx, obj.(*T02), safe.Cast[T02](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*T03)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T03(opCtx, obj.(*T03), safe.Cast[T03](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*TMultiple)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_TMultiple(opCtx, obj.(*TMultiple), safe.Cast[TMultiple](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	return nil
}

func Validate_T00(opCtx operation.Context, obj, oldObj *T00, fldPath *field.Path) (errs field.ErrorList) {
	// field T00.TypeMeta has no validation
	// field T00.S has no validation
	// field T00.PS has no validation

	// field T00.T
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.T, safe.Field(oldObj, func(oldObj *T00) *Tother { return &oldObj.T }), fldPath.Child("t"))...)

	// field T00.PT
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(obj.PT, safe.Field(oldObj, func(oldObj *T00) *Tother { return oldObj.PT }), fldPath.Child("pt"))...)

	return errs
}

func Validate_T01(opCtx operation.Context, obj, oldObj *T01, fldPath *field.Path) (errs field.ErrorList) {
	// type T01
	errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T01, no flags")...)

	// field T01.TypeMeta has no validation

	// field T01.S
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T01.S, no flags")...)
			return
		}(&obj.S, safe.Field(oldObj, func(oldObj *T01) *string { return &oldObj.S }), fldPath.Child("s"))...)

	// field T01.PS
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T01.PS, no flags")...)
			return
		}(obj.PS, safe.Field(oldObj, func(oldObj *T01) *string { return oldObj.PS }), fldPath.Child("ps"))...)

	// field T01.T
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T01.T, no flags")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.T, safe.Field(oldObj, func(oldObj *T01) *Tother { return &oldObj.T }), fldPath.Child("t"))...)

	// field T01.PT
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T01.PT, no flags")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(obj.PT, safe.Field(oldObj, func(oldObj *T01) *Tother { return oldObj.PT }), fldPath.Child("pt"))...)

	return errs
}

func Validate_T02(opCtx operation.Context, obj, oldObj *T02, fldPath *field.Path) (errs field.ErrorList) {
	// type T02
	if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T02, IsFatal"); len(e) != 0 {
		errs = append(errs, e...)
		return // fatal
	}

	// field T02.TypeMeta has no validation

	// field T02.S
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T02.S, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			return
		}(&obj.S, safe.Field(oldObj, func(oldObj *T02) *string { return &oldObj.S }), fldPath.Child("s"))...)

	// field T02.PS
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T02.PS, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			return
		}(obj.PS, safe.Field(oldObj, func(oldObj *T02) *string { return oldObj.PS }), fldPath.Child("ps"))...)

	// field T02.T
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T02.T, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.T, safe.Field(oldObj, func(oldObj *T02) *Tother { return &oldObj.T }), fldPath.Child("t"))...)

	// field T02.PT
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T02.PT, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(obj.PT, safe.Field(oldObj, func(oldObj *T02) *Tother { return oldObj.PT }), fldPath.Child("pt"))...)

	return errs
}

func Validate_T03(opCtx operation.Context, obj, oldObj *T03, fldPath *field.Path) (errs field.ErrorList) {
	// type T03
	if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03, IsFatal"); len(e) != 0 {
		errs = append(errs, e...)
		return // fatal
	}
	errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03, no flags")...)

	// field T03.TypeMeta has no validation

	// field T03.S
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.S, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.S, no flags")...)
			return
		}(&obj.S, safe.Field(oldObj, func(oldObj *T03) *string { return &oldObj.S }), fldPath.Child("s"))...)

	// field T03.PS
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.PS, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.PS, no flags")...)
			return
		}(obj.PS, safe.Field(oldObj, func(oldObj *T03) *string { return oldObj.PS }), fldPath.Child("ps"))...)

	// field T03.T
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.T, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.T, no flags")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.T, safe.Field(oldObj, func(oldObj *T03) *Tother { return &oldObj.T }), fldPath.Child("t"))...)

	// field T03.PT
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.PT, IsFatal"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T03.PT, no flags")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(obj.PT, safe.Field(oldObj, func(oldObj *T03) *Tother { return oldObj.PT }), fldPath.Child("pt"))...)

	return errs
}

func Validate_TMultiple(opCtx operation.Context, obj, oldObj *TMultiple, fldPath *field.Path) (errs field.ErrorList) {
	// type TMultiple
	if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple, IsFatal 1"); len(e) != 0 {
		errs = append(errs, e...)
		return // fatal
	}
	if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple, IsFatal 2"); len(e) != 0 {
		errs = append(errs, e...)
		return // fatal
	}
	errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple, no flags 1")...)
	errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T0, string payload")...)
	errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple, no flags 2")...)

	// field TMultiple.TypeMeta has no validation

	// field TMultiple.S
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.S, IsFatal 1"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.S, IsFatal 2"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.S, no flags 1")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T0, string payload")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.S, no flags 2")...)
			return
		}(&obj.S, safe.Field(oldObj, func(oldObj *TMultiple) *string { return &oldObj.S }), fldPath.Child("s"))...)

	// field TMultiple.PS
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PS, IsFatal 1"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PS, IsFatal 2"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PS, no flags 1")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T0, string payload")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PS, no flags 2")...)
			return
		}(obj.PS, safe.Field(oldObj, func(oldObj *TMultiple) *string { return oldObj.PS }), fldPath.Child("ps"))...)

	// field TMultiple.T
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.T, IsFatal 1"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.T, IsFatal 2"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.T, no flags 1")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T0, string payload")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.T, no flags 2")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.T, safe.Field(oldObj, func(oldObj *TMultiple) *Tother { return &oldObj.T }), fldPath.Child("t"))...)

	// field TMultiple.PT
	errs = append(errs,
		func(obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PT, IsFatal 1"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			if e := validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PT, IsFatal 2"); len(e) != 0 {
				errs = append(errs, e...)
				return // fatal
			}
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PT, no flags 1")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "T0, string payload")...)
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "TMultiple.PT, no flags 2")...)
			errs = append(errs, Validate_Tother(opCtx, obj, oldObj, fldPath)...)
			return
		}(obj.PT, safe.Field(oldObj, func(oldObj *TMultiple) *Tother { return oldObj.PT }), fldPath.Child("pt"))...)

	return errs
}

func Validate_Tother(opCtx operation.Context, obj, oldObj *Tother, fldPath *field.Path) (errs field.ErrorList) {
	// field Tother.OS
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "Tother, no flags")...)
			return
		}(&obj.OS, safe.Field(oldObj, func(oldObj *Tother) *string { return &oldObj.OS }), fldPath.Child("os"))...)

	return errs
}
