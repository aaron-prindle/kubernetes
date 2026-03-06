//go:build fieldsv1string

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

package v1_test

import (
	"encoding/json"
	"testing"
	"unsafe"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFieldsV1StringInterning(t *testing.T) {
	tests := []struct {
		name      string
		isProto   bool
		unmarshal func(t *testing.T, payload []byte) *metav1.FieldsV1
	}{
		{
			name:    "json interning",
			isProto: false,
			unmarshal: func(t *testing.T, payload []byte) *metav1.FieldsV1 {
				var f metav1.FieldsV1
				if err := json.Unmarshal(payload, &f); err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				return &f
			},
		},
		{
			name:    "protobuf interning",
			isProto: true,
			unmarshal: func(t *testing.T, payload []byte) *metav1.FieldsV1 {
				f := &metav1.FieldsV1{}
				if err := f.Unmarshal(payload); err != nil {
					t.Fatalf("Failed to unmarshal Protobuf: %v", err)
				}
				return f
			},
		},
	}

	jsonPayload := `{"f:metadata":{"f:labels":{"f:app":{}}}}`
	orig := metav1.NewFieldsV1(jsonPayload)
	protoPayload, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal original for Protobuf testing: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var payload []byte
			if tc.isProto {
				payload = protoPayload
			} else {
				payload = []byte(jsonPayload)
			}

			// Make two distinct copies of the byte slice to ensure independent starting points.
			b1 := append([]byte(nil), payload...)
			b2 := append([]byte(nil), payload...)

			f1 := tc.unmarshal(t, b1)
			f2 := tc.unmarshal(t, b2)

			if f1.GetRawString() != f2.GetRawString() {
				t.Fatalf("Expected strings to be equal: %q vs %q", f1.GetRawString(), f2.GetRawString())
			}

			// Verify that they point to the exact same underlying memory (interned).
			ptr1 := unsafe.StringData(f1.GetRawString())
			ptr2 := unsafe.StringData(f2.GetRawString())

			if ptr1 != ptr2 {
				t.Errorf("FieldsV1 strings are not interned! Pointers differ: %p != %p", ptr1, ptr2)
			}
		})
	}
}
