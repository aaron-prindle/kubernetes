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

package listmap_single_key

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
	scheme.AddValidationFunc((*M1)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_M1(opCtx, obj.(*M1), safe.Cast[*M1](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*M2)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_M2(opCtx, obj.(*M2), safe.Cast[*M2](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*M3)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_M3(opCtx, obj.(*M3), safe.Cast[*M3](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*M4)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_M4(opCtx, obj.(*M4), safe.Cast[*M4](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*T1)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T1(opCtx, obj.(*T1), safe.Cast[*T1](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	return nil
}

func Validate_M1(opCtx operation.Context, obj, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
	// field M1.K
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "M1.K")...)
			return
		}(&obj.K, safe.Field(oldObj, func(oldObj *M1) *string { return &oldObj.K }), fldPath.Child("k"))...)

	// field M1.S
	errs = append(errs,
		func(obj, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, oldObj, true, "M1.S")...)
			return
		}(&obj.S, safe.Field(oldObj, func(oldObj *M1) *string { return &oldObj.S }), fldPath.Child("s"))...)

	return errs
}

func Validate_M2(opCtx operation.Context, obj, oldObj *M2, fldPath *field.Path) (errs field.ErrorList) {
	// field M2.M1
	errs = append(errs,
		func(obj, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, Validate_M1(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.M1, safe.Field(oldObj, func(oldObj *M2) *M1 { return &oldObj.M1 }), fldPath.Child("M1"))...)

	return errs
}

func Validate_M3(opCtx operation.Context, obj, oldObj *M3, fldPath *field.Path) (errs field.ErrorList) {
	return errs
}

func Validate_M4(opCtx operation.Context, obj, oldObj *M4, fldPath *field.Path) (errs field.ErrorList) {
	// field M4.M2
	errs = append(errs,
		func(obj, oldObj *M2, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, Validate_M2(opCtx, obj, oldObj, fldPath)...)
			return
		}(&obj.M2, safe.Field(oldObj, func(oldObj *M4) *M2 { return &oldObj.M2 }), fldPath.Child("M2"))...)

	return errs
}

func Validate_T1(opCtx operation.Context, obj, oldObj *T1, fldPath *field.Path) (errs field.ErrorList) {
	// field T1.LM1
	errs = append(errs,
		func(obj, oldObj []M1, fldPath *field.Path) (errs field.ErrorList) {
			oldListMap := safe.NewListMap(oldObj, func(o *M1) any { return [1]any{o.K} })
			for i, val := range obj {
				errs = append(errs,
					func(obj, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, Validate_M1(opCtx, obj, oldObj, fldPath)...)
						return
					}(&val, oldListMap.WithMatchingKey(val), fldPath.Index(i))...)
			}
			return
		}(obj.LM1, safe.Field(oldObj, func(oldObj *T1) []M1 { return oldObj.LM1 }), fldPath.Child("lm1"))...)

	// field T1.LM2
	errs = append(errs,
		func(obj, oldObj []M2, fldPath *field.Path) (errs field.ErrorList) {
			oldListMap := safe.NewListMap(oldObj, func(o *M2) any { return [1]any{o.K} })
			for i, val := range obj {
				errs = append(errs,
					func(obj, oldObj *M2, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, Validate_M2(opCtx, obj, oldObj, fldPath)...)
						return
					}(&val, oldListMap.WithMatchingKey(val), fldPath.Index(i))...)
			}
			return
		}(obj.LM2, safe.Field(oldObj, func(oldObj *T1) []M2 { return oldObj.LM2 }), fldPath.Child("lm2"))...)

	// field T1.LM3
	errs = append(errs,
		func(obj, oldObj []M3, fldPath *field.Path) (errs field.ErrorList) {
			oldListMap := safe.NewListMap(oldObj, func(o *M3) any { return [1]any{o.K} })
			for i, val := range obj {
				errs = append(errs,
					func(obj, oldObj *M3, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, Validate_M3(opCtx, obj, oldObj, fldPath)...)
						return
					}(&val, oldListMap.WithMatchingKey(val), fldPath.Index(i))...)
			}
			return
		}(obj.LM3, safe.Field(oldObj, func(oldObj *T1) []M3 { return oldObj.LM3 }), fldPath.Child("lm3"))...)

	// field T1.LM4
	errs = append(errs,
		func(obj, oldObj []M4, fldPath *field.Path) (errs field.ErrorList) {
			oldListMap := safe.NewListMap(oldObj, func(o *M4) any { return [1]any{o.K} })
			for i, val := range obj {
				errs = append(errs,
					func(obj, oldObj *M4, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, Validate_M4(opCtx, obj, oldObj, fldPath)...)
						return
					}(&val, oldListMap.WithMatchingKey(val), fldPath.Index(i))...)
			}
			return
		}(obj.LM4, safe.Field(oldObj, func(oldObj *T1) []M4 { return oldObj.LM4 }), fldPath.Child("lm4"))...)

	return errs
}
