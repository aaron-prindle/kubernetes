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

package pod

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/version"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	podtest "k8s.io/kubernetes/pkg/api/pod/testing"
	apitesting "k8s.io/kubernetes/pkg/api/testing"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/features"
)

func TestDeclarativeValidatePodStatusUpdatePodIPs(t *testing.T) {
	ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
		APIGroup:          "",
		APIVersion:        "v1",
		Resource:          "pods",
		Subresource:       "status",
		Name:              "valid-pod",
		IsResourceRequest: true,
	})

	testCases := map[string]struct {
		strictIPCIDRValidation bool
		oldPodIPs              []api.PodIP
		newPodIPs              []api.PodIP
		expectedErrs           field.ErrorList
	}{
		"valid IP": {
			strictIPCIDRValidation: true,
			newPodIPs:              []api.PodIP{{IP: "10.0.0.1"}},
		},
		"legacy IP allowed when strict validation is disabled": {
			strictIPCIDRValidation: false,
			newPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
		},
		"new legacy IP rejected when strict validation is enabled": {
			strictIPCIDRValidation: true,
			newPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			expectedErrs: field.ErrorList{
				field.Invalid(field.NewPath("status", "podIPs").Index(0), nil, "").WithOrigin("format=ip-sloppy").MarkAlpha(),
			},
		},
		"unchanged legacy IP ratchets when strict validation is enabled": {
			strictIPCIDRValidation: true,
			oldPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			newPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
		},
		"legacy IP can be deleted when strict validation is enabled": {
			strictIPCIDRValidation: true,
			oldPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			newPodIPs:              nil,
		},
		"legacy IP can be replaced by canonical equivalent when strict validation is enabled": {
			strictIPCIDRValidation: true,
			oldPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			newPodIPs:              []api.PodIP{{IP: "10.0.0.1"}},
		},
		"different legacy IP rejected when strict validation is enabled": {
			strictIPCIDRValidation: true,
			oldPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			newPodIPs:              []api.PodIP{{IP: "010.000.000.002"}},
			expectedErrs: field.ErrorList{
				field.Invalid(field.NewPath("status", "podIPs").Index(0), nil, "").WithOrigin("format=ip-sloppy").MarkAlpha(),
			},
		},
		"moved legacy IP ratchets by list map key when strict validation is enabled": {
			strictIPCIDRValidation: true,
			oldPodIPs:              []api.PodIP{{IP: "010.000.000.001"}},
			newPodIPs:              []api.PodIP{{IP: "fd00::1"}, {IP: "010.000.000.001"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.StrictIPCIDRValidation, tc.strictIPCIDRValidation)

			oldPod := podtest.MakePod("valid-pod",
				podtest.SetResourceVersion("1"),
				podtest.SetStatus(api.PodStatus{
					PodIPs: tc.oldPodIPs,
				}),
			)
			newPod := oldPod.DeepCopy()
			newPod.ResourceVersion = "2"
			newPod.Status.PodIPs = tc.newPodIPs

			apitesting.VerifyUpdateValidationEquivalence(t, ctx, newPod, oldPod, StatusStrategy, tc.expectedErrs,
				apitesting.WithMinEmulationVersion(version.MustParse("1.36")),
				apitesting.WithNormalizationRules(podDeclarativeValidationNormalizationRules...),
			)
		})
	}
}
