/*
Copyright 2025 The Kubernetes Authors.

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
package ifspecified

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// SimpleStruct demonstrates basic ifSpecified validation
type SimpleStruct struct {
	TypeMeta int

	// Regular field that may or may not be specified
	Dependency string `json:"dependency"`

	// This field should only be validated as an IP when Dependency is specified
	// +k8s:ifSpecified(Dependency)=+k8s:format=ip-sloppy
	IPAddress string `json:"ipAddress"`

	// This field should only be validated as a DNS label when Dependency is specified
	// +k8s:ifSpecified(Dependency)=+k8s:format=dns-label
	DNSName string `json:"dnsName"`

	// This field should only be validated for minimum when Dependency is specified
	// +k8s:ifSpecified(Dependency)=+k8s:minimum=1
	Count int `json:"count"`
}
