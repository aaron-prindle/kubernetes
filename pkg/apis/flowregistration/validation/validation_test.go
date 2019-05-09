/*
Copyright 2018 The Kubernetes Authors.

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

package validation

// TODO(aaron-prindle) add back validation_test.go
// import (
// 	"strings"
// 	"testing"

// 	"github.com/stretchr/testify/require"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/util/validation/field"
// 	"k8s.io/kubernetes/pkg/apis/flowregistration"
// )

// func TestValidateFlowSchema(t *testing.T) {
// 	testQPS := int64(10)
// 	testURL := "http://localhost"
// 	testCases := []struct {
// 		name   string
// 		conf   flowregistration.FlowSchema
// 		numErr int
// 	}{
// 		{
// 			name: "should pass full config",
// 			conf: flowregistration.FlowSchema{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "myconf",
// 				},
// 				Spec: flowregistration.FlowSchemaSpec{
// 					Policy: flowregistration.Policy{
// 						Level: flowregistration.LevelRequest,
// 						Stages: []flowregistration.Stage{
// 							flowregistration.StageRequestReceived,
// 						},
// 					},
// 					Webhook: flowregistration.Webhook{
// 						Throttle: &flowregistration.WebhookThrottleConfig{
// 							QPS: &testQPS,
// 						},
// 						ClientConfig: flowregistration.WebhookClientConfig{
// 							URL: &testURL,
// 						},
// 					},
// 				},
// 			},
// 			numErr: 0,
// 		},
// 		{
// 			name: "should fail no policy",
// 			conf: flowregistration.FlowSchema{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "myconf",
// 				},
// 				Spec: flowregistration.FlowSchemaSpec{
// 					Webhook: flowregistration.Webhook{
// 						ClientConfig: flowregistration.WebhookClientConfig{
// 							URL: &testURL,
// 						},
// 					},
// 				},
// 			},
// 			numErr: 1,
// 		},
// 		{
// 			name: "should fail no webhook",
// 			conf: flowregistration.FlowSchema{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "myconf",
// 				},
// 				Spec: flowregistration.FlowSchemaSpec{
// 					Policy: flowregistration.Policy{
// 						Level: flowregistration.LevelMetadata,
// 						Stages: []flowregistration.Stage{
// 							flowregistration.StageRequestReceived,
// 						},
// 					},
// 				},
// 			},
// 			numErr: 1,
// 		},
// 	}

// 	for _, test := range testCases {
// 		t.Run(test.name, func(t *testing.T) {
// 			errs := ValidateFlowSchema(&test.conf)
// 			require.Len(t, errs, test.numErr)
// 		})
// 	}
// }

// func TestValidatePolicy(t *testing.T) {
// 	successCases := []flowregistration.Policy{}
// 	successCases = append(successCases, flowregistration.Policy{ // Policy with omitStages and level
// 		Level: flowregistration.LevelRequest,
// 		Stages: []flowregistration.Stage{
// 			flowregistration.Stage("RequestReceived"),
// 			flowregistration.Stage("ResponseStarted"),
// 		},
// 	})
// 	successCases = append(successCases, flowregistration.Policy{Level: flowregistration.LevelNone}) // Policy with none level only

// 	for i, policy := range successCases {
// 		if errs := ValidatePolicy(policy, field.NewPath("policy")); len(errs) != 0 {
// 			t.Errorf("[%d] Expected policy %#v to be valid: %v", i, policy, errs)
// 		}
// 	}

// 	errorCases := []flowregistration.Policy{}
// 	errorCases = append(errorCases, flowregistration.Policy{})                                 // Empty policy                                      // Policy with missing level
// 	errorCases = append(errorCases, flowregistration.Policy{Stages: []flowregistration.Stage{ // Policy with invalid stages
// 		flowregistration.Stage("Bad")}})
// 	errorCases = append(errorCases, flowregistration.Policy{Level: flowregistration.Level("invalid")}) // Policy with bad level
// 	errorCases = append(errorCases, flowregistration.Policy{Level: flowregistration.LevelMetadata})    // Policy without stages

// 	for i, policy := range errorCases {
// 		if errs := ValidatePolicy(policy, field.NewPath("policy")); len(errs) == 0 {
// 			t.Errorf("[%d] Expected policy %#v to be invalid!", i, policy)
// 		}
// 	}
// }

// func strPtr(s string) *string { return &s }

// func TestValidateWebhookConfiguration(t *testing.T) {
// 	tests := []struct {
// 		name          string
// 		config        flowregistration.Webhook
// 		expectedError string
// 	}{
// 		{
// 			name: "both service and URL missing",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{},
// 			},
// 			expectedError: `exactly one of`,
// 		},
// 		{
// 			name: "both service and URL provided",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 					},
// 					URL: strPtr("example.com/k8s/webhook"),
// 				},
// 			},
// 			expectedError: `webhook.clientConfig.url: Required value: exactly one of url or service is required`,
// 		},
// 		{
// 			name: "blank URL",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr(""),
// 				},
// 			},
// 			expectedError: `webhook.clientConfig.url: Invalid value: "": host must be provided`,
// 		},
// 		{
// 			name: "missing host",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr("https:///fancy/webhook"),
// 				},
// 			},
// 			expectedError: `host must be provided`,
// 		},
// 		{
// 			name: "fragment",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr("https://example.com/#bookmark"),
// 				},
// 			},
// 			expectedError: `"bookmark": fragments are not permitted`,
// 		},
// 		{
// 			name: "query",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr("https://example.com?arg=value"),
// 				},
// 			},
// 			expectedError: `"arg=value": query parameters are not permitted`,
// 		},
// 		{
// 			name: "user",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr("https://harry.potter@example.com/"),
// 				},
// 			},
// 			expectedError: `"harry.potter": user information is not permitted`,
// 		},
// 		{
// 			name: "just totally wrong",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					URL: strPtr("arg#backwards=thisis?html.index/port:host//:https"),
// 				},
// 			},
// 			expectedError: `host must be provided`,
// 		},
// 		{
// 			name: "path must start with slash",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("foo/"),
// 					},
// 				},
// 			},
// 			expectedError: `clientConfig.service.path: Invalid value: "foo/": must start with a '/'`,
// 		},
// 		{
// 			name: "path accepts slash",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("/"),
// 					},
// 				},
// 			},
// 			expectedError: ``,
// 		},
// 		{
// 			name: "path accepts no trailing slash",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("/foo"),
// 					},
// 				},
// 			},
// 			expectedError: ``,
// 		},
// 		{
// 			name: "path fails //",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("//"),
// 					},
// 				},
// 			},
// 			expectedError: `clientConfig.service.path: Invalid value: "//": segment[0] may not be empty`,
// 		},
// 		{
// 			name: "path no empty step",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("/foo//bar/"),
// 					},
// 				},
// 			},
// 			expectedError: `clientConfig.service.path: Invalid value: "/foo//bar/": segment[1] may not be empty`,
// 		}, {
// 			name: "path no empty step 2",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("/foo/bar//"),
// 					},
// 				},
// 			},
// 			expectedError: `clientConfig.service.path: Invalid value: "/foo/bar//": segment[2] may not be empty`,
// 		},
// 		{
// 			name: "path no non-subdomain",
// 			config: flowregistration.Webhook{
// 				ClientConfig: flowregistration.WebhookClientConfig{
// 					Service: &flowregistration.ServiceReference{
// 						Namespace: "ns",
// 						Name:      "n",
// 						Path:      strPtr("/apis/foo.bar/v1alpha1/--bad"),
// 					},
// 				},
// 			},
// 			expectedError: `clientConfig.service.path: Invalid value: "/apis/foo.bar/v1alpha1/--bad": segment[3]: a DNS-1123 subdomain`,
// 		},
// 	}
// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			errs := ValidateWebhook(test.config, field.NewPath("webhook"))
// 			err := errs.ToAggregate()
// 			if err != nil {
// 				if e, a := test.expectedError, err.Error(); !strings.Contains(a, e) || e == "" {
// 					t.Errorf("expected to contain \nerr: %s \ngot: %s", e, a)
// 				}
// 			} else {
// 				if test.expectedError != "" {
// 					t.Errorf("unexpected no error, expected to contain %s", test.expectedError)
// 				}
// 			}
// 		})
// 	}
// }
