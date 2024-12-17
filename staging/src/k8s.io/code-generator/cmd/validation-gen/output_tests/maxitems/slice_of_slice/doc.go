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

// This is a test package.
package sliceofslice

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// Maxitems
type M struct {
	TypeMeta int

	// slice-of-slice-of-value
	// +k8s:maxItems=1
	M0 [][]int `json:"m0"`

	// slice-of-typedef-of-slice-of-value
	// +k8s:maxItems=1
	M1 []IntSlice `json:"m1"`

	// slice-of-slice-of-pointer
	// +k8s:maxItems=1
	M2 [][]*int `json:"m2"`

	// slice-of-typedef-of-slice-of-pointer
	// Validation on field only.
	// +k8s:maxItems=1
	M3 []IntPtrSlice `json:"m3"`
}

// Note: no limit here
type IntSlice []int

// Note: no limit here
type IntPtrSlice []*int
