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

// +k8s:validation-gen=TypeMeta

// Package inner contains test types for testing inner field validation tags.
package inner

// T1 demonstrates validations for inner fields of structs.
type T1 struct {
	TypeMeta int

	// Simple struct with inner field validations
	// +inner(Field)=+validateTrue="inner T1.Simple.Field"
	Simple struct {
		Field string `json:"field"`
	} `json:"simple"`

	// Nested struct with multiple levels of inner validations
	// +inner(OuterField)=+validateTrue="inner T1.Nested.OuterField"
	// +inner(Inner.InnerField)=+validateTrue="inner T1.Nested.Inner.InnerField"
	Nested struct {
		OuterField string `json:"outerField"`
		Inner      struct {
			InnerField string `json:"innerField"`
		} `json:"inner"`
	} `json:"nested"`

	// Pointer struct demonstrating inner validation on pointer fields
	// +inner(Field)=+validateTrue="inner T1.PointerStruct.Field"
	PointerStruct *struct {
		Field string `json:"field"`
	} `json:"pointerStruct"`

	// NoValidation demonstrates a struct without inner validations
	NoValidation struct {
		Field string `json:"field"`
	} `json:"noValidation"`
}

// T2 demonstrates a type referenced by T1 that has its own validations
type T2 struct {
	// +validateTrue="field T2.Field"
	Field string `json:"field"`
}
