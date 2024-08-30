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
package crosspkg

import (
	"k8s.io/code-generator/cmd/validation-gen/output_tests/primitives"
	"k8s.io/code-generator/cmd/validation-gen/testscheme"
)

var localSchemeBuilder = testscheme.New()

type T1 struct {
	TypeMeta     int
	PrimitivesT1 primitives.T1 `json:"primitivest1"`
	PrimitivesT2 primitives.T2 `json:"primitivest2"`
	PrimitivesT3 primitives.T3 `json:"primitivest3"`
	// T4 and T5 are not root types in that pkg and are not linked into any
	// root type's transitive graph, so they have no functions.
}
