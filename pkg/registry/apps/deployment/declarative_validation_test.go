package deployment

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	apitesting "k8s.io/kubernetes/pkg/api/testing"
	"k8s.io/kubernetes/pkg/apis/apps"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func TestDeclarativeValidateForDeclarative(t *testing.T) {
	ctx := genericapirequest.WithRequestInfo(genericapirequest.NewDefaultContext(), &genericapirequest.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
	})
	testCases := map[string]struct {
		input        apps.Deployment
		expectedErrs field.ErrorList
	}{
		"valid": {
			input: mkDeployment(),
		},
	}
	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			apitesting.VerifyValidationEquivalence(t, ctx, &tc.input, Strategy.Validate, tc.expectedErrs)
			tc.input.ResourceVersion = "1"
			tc.input.ResourceVersion = "1"
			apitesting.VerifyUpdateValidationEquivalence(t, ctx, &tc.input, &tc.input, Strategy.ValidateUpdate, tc.expectedErrs)
		})
	}
}

func mkDeployment() apps.Deployment {
	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: apps.DeploymentSpec{
			Replicas: 1,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
			Template: api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"a": "b"},
				},
				Spec: api.PodSpec{
					RestartPolicy:                 api.RestartPolicyAlways,
					TerminationGracePeriodSeconds: func(i int64) *int64 { return &i }(30),
					DNSPolicy:                     api.DNSClusterFirst,
					Containers: []api.Container{
						{Name: "c", Image: "img", TerminationMessagePolicy: api.TerminationMessageFallbackToLogsOnError, ImagePullPolicy: api.PullIfNotPresent},
					},
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &apps.RollingUpdateDeployment{
					MaxUnavailable: intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxSurge:       intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
		},
	}
}
