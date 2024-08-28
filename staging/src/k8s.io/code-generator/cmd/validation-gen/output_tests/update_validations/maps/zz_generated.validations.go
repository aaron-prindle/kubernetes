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

package maps

import (
	fmt "fmt"

	operation "k8s.io/apimachinery/pkg/api/operation"
	safe "k8s.io/apimachinery/pkg/api/safe"
	validate "k8s.io/apimachinery/pkg/api/validate"
	runtime "k8s.io/apimachinery/pkg/runtime"
	field "k8s.io/apimachinery/pkg/util/validation/field"
)

func init() { localSchemeBuilder.Register(RegisterValidations) }

// RegisterValidations adds validation functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterValidations(scheme *runtime.Scheme) error {
	scheme.AddValidationFunc((*M1)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_M1(opCtx, obj.(*M1), safe.Cast[M1](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	scheme.AddValidationFunc((*T1)(nil), func(opCtx operation.Context, obj, oldObj interface{}, subresources ...string) field.ErrorList {
		if len(subresources) == 0 {
			return Validate_T1(opCtx, obj.(*T1), safe.Cast[T1](oldObj), nil)
		}
		return field.ErrorList{field.InternalError(nil, fmt.Errorf("no validation found for %T, subresources: %v", obj, subresources))}
	})
	return nil
}

func Validate_M1(opCtx operation.Context, obj, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
	// field M1.S
	errs = append(errs,
		func(obj string, oldObj *string, fldPath *field.Path) (errs field.ErrorList) {
			if opCtx.Operation == operation.Update && oldObj != nil {
				errs = append(errs, validate.FixedResultUpdate(fldPath, obj, *oldObj, true, "T1.M1.S, UpdateOnly")...)
			}
			return
		}(obj.S, safe.Field(oldObj, func(oldObj M1) *string { return &oldObj.S }), fldPath.Child("s"))...)

	return errs
}

func Validate_T1(opCtx operation.Context, obj, oldObj *T1, fldPath *field.Path) (errs field.ErrorList) {
	// field T1.M1
	errs = append(errs,
		func(obj map[string]M1, oldObj map[string]M1, fldPath *field.Path) (errs field.ErrorList) {
			for key, val := range obj {
				errs = append(errs,
					func(obj M1, oldObj *M1, fldPath *field.Path) (errs field.ErrorList) {
						errs = append(errs, Validate_M1(opCtx, &obj, oldObj, fldPath)...)
						return
					}(val, safe.Lookup(oldObj, key), fldPath.Key(key))...)
			}
			return
		}(obj.M1, safe.Field(oldObj, func(oldObj T1) map[string]M1 { return oldObj.M1 }), fldPath.Child("m1"))...)

	return errs
}
