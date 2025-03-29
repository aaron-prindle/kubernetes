// staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/tags/fieldcomparison/doc.go
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
package fieldcomparison

import (
	"time"

	"k8s.io/code-generator/cmd/validation-gen/testscheme"
)

var localSchemeBuilder = testscheme.New()

// ExampleStruct demonstrates the +k8s:fieldComparison tag with 4 arguments.
// If 'NestedStruct.EndTime' >= 'NestedStruct.StartTime', then 'NestedStruct.Replicas' must be >= 1.
// +k8s:fieldComparison(EndTime, >=, StartTime, Replicas)=+k8s:minimum=1
// +k8s:fieldComparison(NestedStruct.EndTime, >=, NestedStruct.StartTime, NestedStruct.Replicas)=+k8s:minimum=1
type ExampleStruct struct {
	TypeMeta int // Required by validation-gen for root types

	Replicas        int          `json:"Replicas"` // Validated for minimum=1 if EndTime >= StartTime
	Selector        string       `json:"Selector"` // Required only if Replicas > 0
	StartTime       *time.Time   `json:"StartTime"`
	EndTime         *time.Time   `json:"EndTime"`
	Mode            string       `json:"Mode"`            // e.g., "Simple", "Advanced"
	AdvancedSetting int          `json:"AdvancedSetting"` // Validated for minimum=11 if Mode == "Advanced"
	NestedStruct    NestedStruct `json:"NestedStruct"`
}

type NestedStruct struct {
	Replicas  int
	StartTime *time.Time
	EndTime   *time.Time
}
