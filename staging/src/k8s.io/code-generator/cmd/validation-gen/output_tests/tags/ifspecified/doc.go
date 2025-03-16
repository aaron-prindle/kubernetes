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
// +k8s:ifSpecified(Dependency,IPAddress)=+k8s:format=ip-sloppy
// +k8s:ifSpecified(Dependency,DNSName)=+k8s:format=dns-label
// +k8s:ifSpecified(Dependency,Count)=+k8s:minimum=1
type SimpleStruct struct {
	TypeMeta int

	// Regular field that may or may not be specified
	Dependency string `json:"dependency"`

	// This field should only be validated as an IP when Dependency is specified
	IPAddress string `json:"ipAddress"`

	// This field should only be validated as a DNS label when Dependency is specified
	DNSName string `json:"dnsName"`

	// This field should only be validated for minimum when Dependency is specified
	Count int `json:"count"`
}

// NestedStruct demonstrates ifSpecified with nested field paths
// +k8s:ifSpecified(ChildXActive,ParentIP)=+k8s:format=ip-sloppy
type NestedStruct struct {
	TypeMeta int

	// Parent field
	ParentIP string `json:"parentIP"`

	// Child struct to test nested references
	Child ChildStruct `json:"child"`
}

// ChildStruct is nested within NestedStruct
// +k8s:ifSpecified(Active,ChildName)=+k8s:format=dns-label
// +k8s:ifSpecified(Active,Priority)=+k8s:minimum=1
type ChildStruct struct {
	// Simple boolean to indicate whether validation should occur
	Active bool `json:"active"`

	// This field should only be validated when Active is true
	ChildName string `json:"childName"`

	// This field should only be validated when Active is true
	Priority int `json:"priority"`
}

// PointerStruct demonstrates ifSpecified with pointer fields
// +k8s:ifSpecified(Dependency,IPAddress)=+k8s:format=ip-sloppy
// +k8s:ifSpecified(Dependency,DNSName)=+k8s:format=dns-label
type PointerStruct struct {
	TypeMeta int

	// Optional dependency field
	Dependency *string `json:"dependency"`

	// This field should only be validated when Dependency is non-nil
	IPAddress string `json:"ipAddress"`

	// This pointer field should only be validated when Dependency is non-nil
	DNSName *string `json:"dnsName"`
}

// TypedefStruct demonstrates ifSpecified with type definitions
// +k8s:ifSpecified(Enabled,IPAddressType)=+k8s:format=ip-sloppy
type TypedefStruct struct {
	TypeMeta int

	// Flag to control validation
	Enabled bool `json:"enabled"`

	// This field should only be validated when Enabled is true
	IPAddressType IPAddress `json:"ipAddressType"`
}

// IPAddress is a typedef
type IPAddress string

// +k8s:format=ip-sloppy
type ValidatedIPAddress string
