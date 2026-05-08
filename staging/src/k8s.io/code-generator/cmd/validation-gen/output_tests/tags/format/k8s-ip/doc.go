/*
Copyright 2026 The Kubernetes Authors.

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
// +k8s:validation-gen-nolint
package k8sip

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

type Struct struct {
	TypeMeta int

	// +k8s:format=k8s-ip
	IPField string `json:"ipField"`

	// +k8s:format=k8s-ip
	IPPtrField *string `json:"ipPtrField"`

	// +k8s:listType=set
	// +k8s:eachVal=+k8s:format=k8s-ip
	IPSet []string `json:"ipSet"`

	// +k8s:listType=map
	// +k8s:listMapKey=ip
	IPItems []IPItem `json:"ipItems"`

	// Note: no validation here.
	IPTypedefField IPStringType `json:"ipTypedefField"`
}

type IPItem struct {
	// +k8s:format=k8s-ip
	IP string `json:"ip"`

	Name string `json:"name"`
}

// +k8s:format=k8s-ip
type IPStringType string
