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

package multipletags

import (
	fmt "fmt"

	validate "k8s.io/apimachinery/pkg/api/validate"
	runtime "k8s.io/apimachinery/pkg/runtime"
	field "k8s.io/apimachinery/pkg/util/validation/field"
)

func init() { localSchemeBuilder.Register(RegisterValidations) }

// RegisterValidations adds validation functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterValidations(scheme *runtime.Scheme) error {
	scheme.AddValidationFunc(new(T1), func(obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T1(obj.(*T1), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("No validation found for %T, subresources: %v", obj, subresources))}
	})
	return nil
}

func Validate_T1(obj *T1, fldPath *field.Path) (errs field.ErrorList) {
	// field T1.TypeMeta has no validation

	// field T1.S
	errs = append(errs,
		func(obj string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T1.S #1")...)
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T1.S #2")...)
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T1.S #3")...)
			return
		}(obj.S, fldPath.Child("s"))...)

	// field T1.T2
	errs = append(errs,
		func(obj T2, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, Validate_T2(&obj, fldPath)...)
			return
		}(obj.T2, fldPath.Child("t2"))...)

	return errs
}

func Validate_T2(obj *T2, fldPath *field.Path) (errs field.ErrorList) {
	// field T2.S
	errs = append(errs,
		func(obj string, fldPath *field.Path) (errs field.ErrorList) {
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T2.S #1")...)
			errs = append(errs, validate.FixedResult(fldPath, obj, true, "field T2.S #2")...)
			return
		}(obj.S, fldPath.Child("s"))...)

	return errs
}
