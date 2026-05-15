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

package k8sip

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestK8sIP(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	st.Value(&Struct{
		IPField:        "10.9.8.7",
		IPPtrField:     ptr.To("2001:db8::ffff"),
		IPSet:          []string{"10.0.0.1", "2001:db8::1"},
		IPItems:        []IPItem{{IP: "10.0.0.1", Name: "a"}},
		IPTypedefField: "2001:db8::2",
	}).ExpectValid()

	invalidStruct := &Struct{
		IPField:        "010.002.003.004",
		IPPtrField:     ptr.To("010.002.003.004"),
		IPSet:          []string{"010.002.003.004"},
		IPItems:        []IPItem{{IP: "010.002.003.004", Name: "a"}},
		IPTypedefField: "010.002.003.004",
	}
	st.Value(invalidStruct).ExpectMatches(field.ErrorMatcher{}.ByType().ByField().ByOrigin().ByDetailSubstring(), field.ErrorList{
		field.Invalid(field.NewPath("ipField"), nil, "must not have leading 0s").WithOrigin("format=ip-strict"),
		field.Invalid(field.NewPath("ipPtrField"), nil, "must not have leading 0s").WithOrigin("format=ip-strict"),
		field.Invalid(field.NewPath("ipSet").Index(0), nil, "must not have leading 0s").WithOrigin("format=ip-strict"),
		field.Invalid(field.NewPath("ipItems").Index(0).Child("ip"), nil, "must not have leading 0s").WithOrigin("format=ip-strict"),
		field.Invalid(field.NewPath("ipTypedefField"), nil, "must not have leading 0s").WithOrigin("format=ip-strict"),
	})

	// Test validation ratcheting.
	st.Value(invalidStruct).OldValue(invalidStruct).ExpectValid()

	// listType=map correlates by key, so the invalid IP field still ratchets
	// when the item moves or sibling fields change.
	st.Value(&Struct{
		IPField: "10.9.8.7",
		IPItems: []IPItem{
			{IP: "10.0.0.1", Name: "new-valid"},
			{IP: "010.002.003.004", Name: "new-legacy"},
		},
	}).OldValue(&Struct{
		IPField: "10.9.8.7",
		IPItems: []IPItem{
			{IP: "010.002.003.004", Name: "old-legacy"},
			{IP: "10.0.0.1", Name: "old-valid"},
		},
	}).ExpectValid()
}
