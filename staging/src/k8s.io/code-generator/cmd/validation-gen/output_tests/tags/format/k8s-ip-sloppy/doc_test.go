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

package k8sipsloppy

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

const strictIPCIDRValidationOption = "StrictIPCIDRValidation"

func TestK8sIPSloppy(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	st.Value(&Struct{
		IPField:        "010.002.003.004",
		IPPtrField:     ptr.To("010.002.003.004"),
		IPSet:          []string{"010.002.003.004"},
		IPItems:        []IPItem{{IP: "010.002.003.004", Name: "a"}},
		IPTypedefField: "010.002.003.004",
	}).ExpectValid()

	st.Value(&Struct{
		IPField:        "FE80:0:0:0:0:0:0:0abc",
		IPPtrField:     ptr.To("FE80:0:0:0:0:0:0:0abc"),
		IPSet:          []string{"FE80:0:0:0:0:0:0:0abc"},
		IPItems:        []IPItem{{IP: "FE80:0:0:0:0:0:0:0abc", Name: "a"}},
		IPTypedefField: "FE80:0:0:0:0:0:0:0abc",
	}).Opts([]string{strictIPCIDRValidationOption}).ExpectValid()

	invalidStruct := &Struct{
		IPField:        "010.002.003.004",
		IPPtrField:     ptr.To("010.002.003.004"),
		IPSet:          []string{"010.002.003.004"},
		IPItems:        []IPItem{{IP: "010.002.003.004", Name: "a"}},
		IPTypedefField: "010.002.003.004",
	}
	st.Value(invalidStruct).Opts([]string{strictIPCIDRValidationOption}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField().ByOrigin().ByDetailSubstring(), field.ErrorList{
		field.Invalid(field.NewPath("ipField"), nil, "must not have leading 0s").WithOrigin("format=ip-sloppy"),
		field.Invalid(field.NewPath("ipPtrField"), nil, "must not have leading 0s").WithOrigin("format=ip-sloppy"),
		field.Invalid(field.NewPath("ipSet").Index(0), nil, "must not have leading 0s").WithOrigin("format=ip-sloppy"),
		field.Invalid(field.NewPath("ipItems").Index(0).Child("ip"), nil, "must not have leading 0s").WithOrigin("format=ip-sloppy"),
		field.Invalid(field.NewPath("ipTypedefField"), nil, "must not have leading 0s").WithOrigin("format=ip-sloppy"),
	})

	// Test validation ratcheting.
	st.Value(invalidStruct).OldValue(invalidStruct).Opts([]string{strictIPCIDRValidationOption}).ExpectValid()

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
	}).Opts([]string{strictIPCIDRValidationOption}).ExpectValid()
}
