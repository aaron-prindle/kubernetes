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

	// When Dependency is specified but validation fails
	st.Value(&SimpleStruct{
		Dependency: "specified",
		IPAddress:  "not-an-ip",   // Invalid IP
		DNSName:    "Invalid DNS", // Invalid DNS label
		Count:      0,             // Invalid (below minimum)
	}).ExpectInvalid(
		// field.Invalid(field.NewPath("ipAddress"), "not-an-ip", "must be a valid IP address (e.g. 10.9.8.7 or 2001:db8::ffff)"),
		// field.Invalid(field.NewPath("dnsName"), "Invalid DNS", "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"),
		field.Invalid(field.NewPath("count"), 0, "must be greater than or equal to 1"),
	)

}
