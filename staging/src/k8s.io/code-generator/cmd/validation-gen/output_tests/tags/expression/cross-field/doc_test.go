// ========================
// ====== FILE: ./doc_test.go ======
// ========================
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
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

var ctx = context.Background()
var opCreate = operation.Operation{Type: operation.Create}

// --- Helper for Expected Messages ---
func expectedMsg(sLen, i int) string {
	return fmt.Sprintf("the length of s (%d) must be less than i (%d)", sLen, i)
}

// --- Tests ---

// --- Benchmarks ---

// -- Struct (Original ~4 fields) --

// Benchmark results are placeholders, update after running
func BenchmarkExpression(b *testing.B) {
	obj := Struct{S: "x", I: 10}
	// force compile and then reset to ignore compilation cost
	_ = Validate_Struct(ctx, opCreate, nil, &obj, nil) // Use generated function
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Benchmark results are placeholders, update after running
func BenchmarkNative(b *testing.B) {
	obj := Struct{S: "x", I: 10}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// -- Struct2 (~20 fields) --

// Benchmark results are placeholders, update after running
func BenchmarkExpression2(b *testing.B) {
	obj := Struct2{S: "x", I: 10, Field20: "bench"}
	// force compile and then reset to ignore compilation cost
	_ = Validate_Struct2(ctx, opCreate, nil, &obj, nil) // Use generated function
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct2(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Benchmark results are placeholders, update after running
func BenchmarkNative2(b *testing.B) {
	obj := Struct2{S: "x", I: 10, Field20: "bench"}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct2_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// -- Struct3 (~100 fields, primitives) --

// Benchmark results are placeholders, update after running
func BenchmarkExpression3(b *testing.B) {
	obj := Struct3{S: "x", I: 10, Field100: "bench"}
	// force compile and then reset to ignore compilation cost
	errs := Validate_Struct3(ctx, opCreate, nil, &obj, nil) // Use generated function
	if len(errs) != 0 {
		panic(fmt.Sprintf("expected no errs, got: %v", errs))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct3(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// -- Struct4 (~100 field, primitives+complex-types) --

// Benchmark results are placeholders, update after running
func BenchmarkExpression4(b *testing.B) {
	obj := Struct4{S: "x", I: 10, Field100: "bench"}
	// force compile and then reset to ignore compilation cost
	errs := Validate_Struct4(ctx, opCreate, nil, &obj, nil) // Use generated function
	if len(errs) != 0 {
		panic(fmt.Sprintf("expected no errs, got: %v", errs))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate_Struct4(ctx, opCreate, nil, &obj, nil) // Use generated function
	}
}

// Benchmark results are placeholders, update after running
func BenchmarkNative3(b *testing.B) {
	obj := Struct3{S: "x", I: 10, Field100: "bench"}
	b.ResetTimer() // No compilation cost for native
	for i := 0; i < b.N; i++ {
		_ = Validate_Struct3_Native(ctx, opCreate, nil, &obj, nil)
	}
}

// --- Native Validation Implementations ---

// Validate_Struct_Native implements the equivalent Go logic for the rule: self.s.size() < self.i
func Validate_Struct_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct) (errs field.ErrorList) {
	if fldPath == nil {
		fldPath = field.NewPath("struct") // Match the path used in TestStruct
	}
	// Error if the condition (len(S) < I) is FALSE
	if !(len(obj.S) < obj.I) {
		// Using a generic message for benchmark consistency. Match the expected format if desired.
		// errs = field.ErrorList{field.Invalid(fldPath, obj, expectedMsg(len(obj.S), obj.I))}
		errs = field.ErrorList{field.Invalid(fldPath, obj, "native validation failed: len(S) >= I")} // Keep simple message for clarity
	}
	return errs
}

// Validate_Struct2_Native implements the equivalent Go logic for Struct2
func Validate_Struct2_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct2) (errs field.ErrorList) {
	if fldPath == nil {
		fldPath = field.NewPath("struct2") // Match the path used in TestStruct2
	}
	// Error if the condition (len(S) < I) is FALSE
	if !(len(obj.S) < obj.I) {
		errs = field.ErrorList{field.Invalid(fldPath, obj, "native validation failed: len(S) >= I")}
	}
	return errs
}

// Validate_Struct3_Native implements the equivalent Go logic for Struct3
func Validate_Struct3_Native(ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj *Struct3) (errs field.ErrorList) {
	if fldPath == nil {
		fldPath = field.NewPath("struct3") // Match the path used in TestStruct3
	}
	// Error if the condition (len(S) < I) is FALSE
	if !(len(obj.S) < obj.I) {
		errs = field.ErrorList{field.Invalid(fldPath, obj, "native validation failed: len(S) >= I")}
	}
	return errs
}

// Note: The generated validation functions (Validate_Struct, Validate_Struct2, Validate_Struct3)
// are assumed to be created by the `validation-gen` tool based on the +k8s:rule tags.
// You need to run the generator before these benchmarks will compile and run correctly.
// Example command (adjust paths as needed):
// validation-gen --input-dirs ./<your-package-path> --output-package ./<your-package-path> --output-file-base zz_generated.validation --go-header-file hack/boilerplate.go.txt
