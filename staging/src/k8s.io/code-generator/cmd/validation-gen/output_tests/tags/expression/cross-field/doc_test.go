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

var ctx = context.Background()
var opCreate = operation.Operation{Type: operation.Create}

// --- Benchmarks ---

// -- Struct (Original) --

// Benchmark results are placeholders, update after running
// Example: 4394 ns/op (cost disabled, I think?)
func BenchmarkExpression(b *testing.B) {
	obj := Struct{S: "x", MinI: 5, I: 10}
	// force compile and then reset to ignore compilation cost
	_ = Validate_Struct(ctx, opCreate, nil, &obj, nil) // Use generated function
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Example: 2.5 ns/op
func BenchmarkNative(b *testing.B) {
	obj := Struct{S: "x", MinI: 5, I: 10}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// -- Struct2 (~20 fields) --

// Placeholder benchmark result
func BenchmarkExpression2(b *testing.B) {
	obj := Struct2{S: "x", MinI: 5, I: 10, Field20: "bench"}
	// force compile and then reset to ignore compilation cost
	_ = Validate_Struct2(ctx, opCreate, nil, &obj, nil) // Use generated function
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct2(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Placeholder benchmark result
func BenchmarkNative2(b *testing.B) {
	obj := Struct2{S: "x", MinI: 5, I: 10, Field20: "bench"}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct2_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// -- Struct3 (~100 fields) --

// Placeholder benchmark result
func BenchmarkExpression3(b *testing.B) {
	obj := Struct3{S: "x", MinI: 5, I: 10, Field100: "bench"}
	// force compile and then reset to ignore compilation cost
	errs := Validate_Struct3(ctx, opCreate, nil, &obj, nil) // Use generated function
	if len(errs) != 0 {
		panic("expected no errs")
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct3(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Placeholder benchmark result
func BenchmarkNative3(b *testing.B) {
	obj := Struct3{S: "x", MinI: 5, I: 10, Field100: "bench"}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct3_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// --- Native Validation Implementations ---

// Validate_Struct_Native implements the equivalent Go logic for the expression: self.minI <= self.i
func Validate_Struct_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct) (errs field.ErrorList) {
	// If fldPath is nil, create a root path. Adjust if validation needs specific paths.
	if fldPath == nil {
		fldPath = field.NewPath("struct") // Match the path used in TestStruct
	}
	if !(obj.MinI <= obj.I) {
		// Use fldPath in the error. Pass obj as the value.
		errs = field.ErrorList{field.Invalid(fldPath, obj, "expression returned false")}
	}
	return errs
}

// Validate_Struct2_Native implements the equivalent Go logic for Struct2
func Validate_Struct2_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct2) (errs field.ErrorList) {
	if fldPath == nil {
		fldPath = field.NewPath("struct2") // Match the path used in TestStruct2
	}
	if !(obj.MinI <= obj.I) {
		errs = field.ErrorList{field.Invalid(fldPath, obj, "expression returned false")}
	}
	return errs
}

// Validate_Struct3_Native implements the equivalent Go logic for Struct3
func Validate_Struct3_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct3) (errs field.ErrorList) {
	if fldPath == nil {
		fldPath = field.NewPath("struct3") // Match the path used in TestStruct3
	}
	if !(obj.MinI <= obj.I) {
		errs = field.ErrorList{field.Invalid(fldPath, obj, "expression returned false")}
	}
	return errs
}

// Note: The generated validation functions (Validate_Struct, Validate_Struct2, Validate_Struct3)
// are assumed to be created by the `validation-gen` tool based on the +k8s:expression tags.
// You need to run the generator before these benchmarks will compile and run correctly.
// Example command (adjust paths as needed):
// validation-gen --input-dirs ./... --output-package ./ --output-file-base zz_generated.validation --go-header-file hack/boilerplate.go.txt
