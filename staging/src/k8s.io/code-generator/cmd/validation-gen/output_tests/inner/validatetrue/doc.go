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
// +k8s:validation-gen-scheme-registry=k8s.io/code-generator/cmd/validation-gen/testscheme.Scheme

// Package inner contains test types for testing inner field validation tags.
package inner

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// T1Simple represents the formerly anonymous struct used in T1.Simple.
type T1Simple struct {
	Field string `json:"field"`
}

// T1NestedInner represents the formerly anonymous inner struct used in T1.Nested.Inner.
type T1NestedInner struct {
	InnerField string `json:"innerField"`
}

// T1Nested represents the formerly anonymous struct used in T1.Nested.
type T1Nested struct {
	OuterField string        `json:"outerField"`
	Inner      T1NestedInner `json:"inner"`
}

// T1PointerStruct represents the formerly anonymous pointer struct used in T1.PointerStruct.
type T1PointerStruct struct {
	Field string `json:"field"`
}

// T1NoValidation represents the formerly anonymous struct used in T1.NoValidation.
type T1NoValidation struct {
	Field string `json:"field"`
}

// T1 demonstrates validations for inner fields of structs.
type T1 struct {
	TypeMeta int `json:"typeMeta"`

	// Simple struct with inner field validations
	// +k8s:inner(Field)=+validateTrue="inner T1.Simple.Field"
	Simple T1Simple `json:"simple"`

	// Nested struct with multiple levels of inner validations
	// +k8s:inner(OuterField)=+validateTrue="inner T1.Nested.OuterField"
	// +k8s:inner(Inner.InnerField)=+validateTrue="inner T1.Nested.Inner.InnerField"
	Nested T1Nested `json:"nested"`

	// Pointer struct demonstrating inner validation on pointer fields
	// +k8s:inner(Field)=+validateTrue="inner T1.PointerStruct.Field"
	PointerStruct *T1PointerStruct `json:"pointerStruct"`

	// NoValidation demonstrates a struct without inner validations
	NoValidation T1NoValidation `json:"noValidation"`
}
