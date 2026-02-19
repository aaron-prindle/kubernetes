package v1_test

import (
	"encoding/json"
	"testing"
	"unsafe"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFieldsV1Interning(t *testing.T) {
	jsonPayload1 := []byte(`{"f:metadata":{"f:labels":{"f:app":{}}}}`)
	jsonPayload2 := []byte(`{"f:metadata":{"f:labels":{"f:app":{}}}}`)

	var f1 metav1.FieldsV1
	var f2 metav1.FieldsV1

	if err := json.Unmarshal(jsonPayload1, &f1); err != nil {
		t.Fatalf("Failed to unmarshal first payload: %v", err)
	}

	if err := json.Unmarshal(jsonPayload2, &f2); err != nil {
		t.Fatalf("Failed to unmarshal second payload: %v", err)
	}

	if f1.Raw != f2.Raw {
		t.Fatalf("Expected strings to be equal: %q vs %q", f1.Raw, f2.Raw)
	}

	// Verify that they point to the exact same underlying memory (interned).
	ptr1 := unsafe.StringData(f1.Raw)
	ptr2 := unsafe.StringData(f2.Raw)

	if ptr1 != ptr2 {
		t.Errorf("FieldsV1.Raw strings are not interned! Pointers differ: %p != %p", ptr1, ptr2)
	} else {
		t.Logf("Successfully verified interning! Both strings point to %p", ptr1)
	}
}
