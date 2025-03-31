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

package cross_field

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func Test(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Valid case: minI (5) <= i (10)
	st.Value(&Root{Struct: Struct{S: "x", MinI: 5, I: 10}}).ExpectValid()

	// Invalid case: minI (5) > i (2)
	st.Value(&Root{Struct: Struct{S: "xyz", MinI: 5, I: 2}}).ExpectInvalid(
		field.Invalid(field.NewPath("struct"), Struct{S: "xyz", I: 2, MinI: 5, B: false, F: 0}, "expression returned false"),
	)
}

// Benchmarks need adjustment based on actual performance, but update struct definition
// 4394 ns/op (cost disabled, I think?)
func BenchmarkExpression(b *testing.B) {
	// Use a valid struct for benchmarking
	obj := Struct{S: "x", MinI: 5, I: 10}

	// force compile and then reset to ignore compilation cost
	Validate_Struct(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Validate_Struct(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	}
}

// 2.5 ns/op -> Placeholder comment, update if needed
func BenchmarkNative(b *testing.B) {
	// Use a valid struct for benchmarking
	obj := Struct{S: "x", MinI: 5, I: 10}
	for i := 0; i < b.N; i++ {
		Validate_Struct_Native(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	}
}

// Validate_Struct_Native implements the equivalent Go logic for the expression: self.minI <= self.i
// It should return an error if the expression evaluates to false.
func Validate_Struct_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct) (errs field.ErrorList) {
	if !(obj.MinI <= obj.I) {
		errs = field.ErrorList{field.Invalid(nil, obj, "expression returned false")}
	}
	return errs
}
