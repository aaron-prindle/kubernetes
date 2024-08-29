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

package typedef

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
	return nil
}

func Validate_AMSS(opCtx operation.Context, obj, oldObj *AMSS, fldPath *field.Path) (errs field.ErrorList) {
	// type AMSS
	if obj != nil {
		errs = append(errs, validate.FixedResult(opCtx, fldPath, *obj, *oldObj, true, "type AMSS")...)
	}

	if obj != nil {
		for key, val := range *obj {
			errs = append(errs,
				func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
					errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, *oldObj, true, "AMSS[keys]")...)
					return
				}(key, nil, fldPath)...)
			errs = append(errs,
				func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
					errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, *oldObj, true, "AMSS[vals]")...)
					return
				}(val, safe.Lookup(*oldObj, key), fldPath.Key(key))...)
		}
	}
	return errs
}

func Validate_T1(opCtx operation.Context, obj, oldObj *T1, fldPath *field.Path) (errs field.ErrorList) {
	// type T1
	if obj != nil {
		errs = append(errs, validate.FixedResult(opCtx, fldPath, *obj, *oldObj, true, "type T1")...)
	}

	// field T1.TypeMeta has no validation

	// field T1.MSAMSS
	errs = append(errs,
		func(obj map[string]AMSS, oldObj map[string]AMSS, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, *oldObj, true, "field T1.MSAMSS")...)
			for key, val := range obj {
				errs = append(errs,
					func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, *oldObj, true, "T1.MSAMSS[keys]")...)
						return
					}(key, nil, fldPath)...)
				errs = append(errs,
					func(obj AMSS, oldObj *AMSS, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, validate.FixedResult(opCtx, fldPath, obj, *oldObj, true, "T1.MSAMSS[vals]")...)
						errs = append(errs, Validate_AMSS(opCtx, &obj, oldObj, fldPath)...)
						return
					}(val, safe.Lookup(oldObj, key), fldPath.Key(key))...)
			}
			return
		}(obj.MSAMSS, safe.Field(oldObj, func(oldObj T1) map[string]AMSS { return oldObj.MSAMSS }), fldPath.Child("msamss"))...)

	return errs
}
