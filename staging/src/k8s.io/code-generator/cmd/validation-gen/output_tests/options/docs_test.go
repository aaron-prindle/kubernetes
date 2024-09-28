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

package options

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/sets"
)

func Test(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// validation is skipped if no options are provided
	st.Value(&T1{S: ""}).ExpectValid()

	// validation is skipped if FeatureX flag is not set
	st.Value(&T1{S: ""}).Opts(&operation.ValidationOpts{Flags: sets.New("FeatureY")}).
		ExpectValidateFalse("field T1.S")

	// validation is checked if FeatureX flag is set
	st.Value(&T1{S: ""}).Opts(&operation.ValidationOpts{Flags: sets.New("FeatureX")}).
		ExpectValidateFalse("field T1.S")
}
