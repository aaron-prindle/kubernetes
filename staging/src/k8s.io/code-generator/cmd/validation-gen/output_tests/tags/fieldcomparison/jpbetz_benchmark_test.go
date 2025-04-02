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

package fieldcomparison

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"
	validate "k8s.io/apimachinery/pkg/api/validate"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Benchmarks need adjustment based on actual performance, but update struct definition
// CEL: ~4394 ns/op (cost disabled, I think?) // NOTE: CEL running check that isn't identical, checks less

// fieldComparison (v9): 166 ns/op (doing more also as validateTrue is run)
// fieldComparison-native (v10): 362.7 ns/op (doing more also as validateTrue is run)
func BenchmarkExpression(b *testing.B) {
	// Use a valid struct for benchmarking
	obj := ExampleStruct{S: "x", MinI: 5, I: 10}

	// force compile and then reset to ignore compilation cost
	Validate_ExampleStruct(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Validate_ExampleStruct(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	}
}

// 4.1 ns/op -> Placeholder comment, update if needed
func BenchmarkNative(b *testing.B) {
	// Use a valid struct for benchmarking
	obj := ExampleStruct{S: "x", MinI: 5, I: 10}
	for i := 0; i < b.N; i++ {
		Validate_ExampleStruct_Native(context.Background(), operation.Operation{Type: operation.Create}, nil, &obj, nil)
	}
}

// Validate_ExampleStruct_Native implements the equivalent Go logic for the expression: self.minI <= self.i
// It should return an error if the expression evaluates to false.
func Validate_ExampleStruct_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *ExampleStruct) (errs field.ErrorList) {
	if !(obj.MinI <= obj.I) {
		errs = field.ErrorList{field.Invalid(nil, obj, "expression returned false")}
		return errs
	}
	return validate.FixedResult(ctx, op, fldPath, obj, oldObj, true, "")
}
