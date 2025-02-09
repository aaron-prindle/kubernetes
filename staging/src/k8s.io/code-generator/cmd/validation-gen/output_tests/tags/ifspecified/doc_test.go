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
package ifspecified

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestSimpleStruct(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// When Dependency is specified, all validators should run
	st.Value(&SimpleStruct{
		Dependency: "specified",
		IPAddress:  "192.168.1.1", // Valid IP
		DNSName:    "valid-name",  // Valid DNS label
		Count:      10,            // Valid (above minimum)
	}).ExpectValid()

	// When Dependency is specified but validation fails
	st.Value(&SimpleStruct{
		Dependency: "specified",
		IPAddress:  "not-an-ip",   // Invalid IP
		DNSName:    "Invalid DNS", // Invalid DNS label
		Count:      0,             // Invalid (below minimum)
	}).ExpectInvalid(
		field.Invalid(field.NewPath("IPAddress"), "not-an-ip", "must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)"),
		field.Invalid(field.NewPath("DNSName"), "Invalid DNS", "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		field.Invalid(field.NewPath("Count"), 0, "must be greater than or equal to 1"),
	)

	// When Dependency is not specified, validators should be skipped
	st.Value(&SimpleStruct{
		Dependency: "",            // Empty string (not specified)
		IPAddress:  "not-an-ip",   // Would be invalid if validated
		DNSName:    "Invalid DNS", // Would be invalid if validated
		Count:      0,             // Would be invalid if validated
	}).ExpectValid() // Should be valid because no validation happens
}

func TestNestedStruct(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// When ChildXActive is true, Parent's IP should be validated
	st.Value(&NestedStruct{
		ParentIP: "192.168.1.1", // Valid IP
		Child: ChildStruct{
			Active:    true,
			ChildName: "valid-name", // Valid DNS label
			Priority:  5,            // Valid (above minimum)
		},
	}).ExpectValid()

	// When ChildXActive is true but Parent's IP is invalid
	// st.Value(&NestedStruct{
	// 	ParentIP: "not-an-ip", // Invalid IP
	// 	Child: ChildStruct{
	// 		Active:    true,
	// 		ChildName: "valid-name", // Valid DNS label
	// 		Priority:  5,            // Valid (above minimum)
	// 	},
	// }).ExpectInvalid(
	// 	field.Invalid(field.NewPath("parentIP"), "not-an-ip", "must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)"),
	// )

	// When ChildXActive is false, Parent's IP should not be validated
	st.Value(&NestedStruct{
		ParentIP: "not-an-ip", // Would be invalid if validated
		Child: ChildStruct{
			Active:    false,
			ChildName: "valid-name",
			Priority:  5,
		},
	}).ExpectValid() // Should be valid because no validation happens

	// When ParentIP is specified and Child's Active is true, but Child's name is invalid
	st.Value(&NestedStruct{
		ParentIP: "192.168.1.1", // Valid IP
		Child: ChildStruct{
			Active:    true,
			ChildName: "Invalid DNS", // Invalid DNS label
			Priority:  5,
		},
	}).ExpectInvalid(
		field.Invalid(field.NewPath("Child").Child("ChildName"), "Invalid DNS", "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
	)

	// When Child's Active is true but Priority is invalid
	st.Value(&NestedStruct{
		ParentIP: "192.168.1.1", // Valid IP
		Child: ChildStruct{
			Active:    true,
			ChildName: "valid-name", // Valid DNS label
			Priority:  0,            // Invalid (below minimum)
		},
	}).ExpectInvalid(
		field.Invalid(field.NewPath("Child").Child("Priority"), 0, "must be greater than or equal to 1"),
	)

	// When Child's Active is false, Priority validation should be skipped
	st.Value(&NestedStruct{
		ParentIP: "192.168.1.1", // Valid IP
		Child: ChildStruct{
			Active:    false, // Not specified (false)
			ChildName: "valid-name",
			Priority:  0, // Would be invalid if validated
		},
	}).ExpectValid() // Should be valid because no validation happens
}

func TestPointerStruct(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// When Dependency is specified (non-nil), all validators should run
	dependency := "specified"
	dnsName := "valid-name"
	st.Value(&PointerStruct{
		Dependency: &dependency,
		IPAddress:  "192.168.1.1", // Valid IP
		DNSName:    &dnsName,      // Valid DNS label
	}).ExpectValid()

	// When Dependency is specified but validation fails
	invalidDNS := "Invalid DNS"
	st.Value(&PointerStruct{
		Dependency: &dependency,
		IPAddress:  "not-an-ip", // Invalid IP
		DNSName:    &invalidDNS, // Invalid DNS label
	}).ExpectInvalid(
		field.Invalid(field.NewPath("IPAddress"), "not-an-ip", "must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)"),
		field.Invalid(field.NewPath("DNSName"), "Invalid DNS", "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
	)

	// When Dependency is nil, validators should be skipped
	st.Value(&PointerStruct{
		Dependency: nil,         // Nil (not specified)
		IPAddress:  "not-an-ip", // Would be invalid if validated
		DNSName:    &invalidDNS, // Would be invalid if validated
	}).ExpectValid() // Should be valid because no validation happens
}

func TestTypedefStruct(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// When Enabled is true, IPAddressType should be validated
	st.Value(&TypedefStruct{
		Enabled:       true,
		IPAddressType: "192.168.1.1", // Valid IP
	}).ExpectValid()
	// TODO(aaron-prindle) FAILS:
	// --- FAIL: TestTypedefStruct/*ifspecified.TypedefStruct (0.00s)
	// /tags/ifspecified/doc_test.go:164: want no errors, 
	// got: IPAddressType: Invalid value: "192.168.1.1": expected type ~string or ~*string for IP sloppy validation
    // --- FAIL: TestTypedefStruct/*ifspecified.TypedefStruct#01 (0.00s)

	// When Enabled is true but IPAddressType is invalid
	st.Value(&TypedefStruct{
		Enabled:       true,
		IPAddressType: "not-an-ip", // Invalid IP
	}).ExpectInvalid(
		field.Invalid(field.NewPath("IPAddressType"), "not-an-ip", "must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)"),
	)

	// When Enabled is false, IPAddressType should not be validated
	st.Value(&TypedefStruct{
		Enabled:       false,
		IPAddressType: "not-an-ip", // Would be invalid if validated
	}).ExpectValid() // Should be valid because no validation happens
}
