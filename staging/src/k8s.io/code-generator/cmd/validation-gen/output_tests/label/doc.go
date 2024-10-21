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

// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-scheme-registry=k8s.io/code-generator/cmd/validation-gen/testscheme.Scheme

package label

import (
	"k8s.io/code-generator/cmd/validation-gen/testscheme"
)

var localSchemeBuilder = testscheme.New()

// From k8s.io/kubernetes/pkg/apis/apps/validation/validation.go
// if template == nil {
//     allErrs = append(allErrs, field.Required(fldPath, ""))
// } else {
//     if !selector.Empty() {
//         labels := labels.Set(template.Labels)
//         if !selector.Matches(labels) {
//             allErrs = append(allErrs, field.Invalid(fldPath.Child("metadata", "labels"), template.Labels, "`selector` does not match template `labels`"))
//         }
//     }
// }

// +labelConsistency={"label": "app", "field": "SelectorMatchLabel", "required": true}
type ReplicaSetExample struct {
	TypeMeta int `json:"typeMeta"`
	// SelectorMatchLabel represents the expected value for template labels
	SelectorMatchLabel string       `json:"selectorMatchLabel,omitempty"`
	Template           *PodTemplate `json:"template,omitempty"`
	Metadata           ObjectMeta   `json:"metadata,omitempty"`
}

// From k8s.io/kubernetes/pkg/apis/batch/validation/validation.go
// allErrs = append(allErrs, apivalidation.ValidateHasLabel(obj.Spec.Template.ObjectMeta,
//     field.NewPath("spec").Child("template").Child("metadata"),
//     batch.ControllerUidLabel, string(obj.UID))...)
// allErrs = append(allErrs, apivalidation.ValidateHasLabel(obj.Spec.Template.ObjectMeta,
//     field.NewPath("spec").Child("template").Child("metadata"),
//     batch.JobNameLabel, string(obj.Name))...)

// +labelConsistency={"label": "controller-uid", "field": "UID", "required": true}
// +labelConsistency={"label": "job-name", "field": "Name", "required": true}
type JobExample struct {
	TypeMeta int          `json:"typeMeta"`
	Name     string       `json:"name,omitempty"`
	UID      string       `json:"uid,omitempty"`
	Template *PodTemplate `json:"template,omitempty"`
	Metadata ObjectMeta   `json:"metadata,omitempty"`
}

// Supporting types
type PodTemplate struct {
	Metadata ObjectMeta `json:"metadata,omitempty"`
}

type ObjectMeta struct {
	Name   string            `json:"name,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}
