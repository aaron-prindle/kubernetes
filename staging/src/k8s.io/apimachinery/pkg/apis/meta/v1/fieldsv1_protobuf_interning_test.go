package v1_test

import (
	"testing"
	"unsafe"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFieldsV1ProtobufInterning(t *testing.T) {
	// Constructing the payload directly via marshal to ensure consistency.
	orig := &metav1.FieldsV1{Raw: `{"f:metadata":{"f:labels":{"f:app":{}}}}`}
	data, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Make two distinct copies of the byte slice.
	b1 := append([]byte(nil), data...)
	b2 := append([]byte(nil), data...)

	f1 := &metav1.FieldsV1{}
	if err := f1.Unmarshal(b1); err != nil {
		t.Fatalf("Failed to unmarshal first: %v", err)
	}

	f2 := &metav1.FieldsV1{}
	if err := f2.Unmarshal(b2); err != nil {
		t.Fatalf("Failed to unmarshal second: %v", err)
	}

	if f1.Raw != f2.Raw {
		t.Fatalf("Expected strings to be equal: %q vs %q", f1.Raw, f2.Raw)
	}

	ptr1 := unsafe.StringData(f1.Raw)
	ptr2 := unsafe.StringData(f2.Raw)

	if ptr1 != ptr2 {
		t.Errorf("FieldsV1.Raw strings are not interned! Pointers differ: %p != %p", ptr1, ptr2)
	} else {
		t.Logf("Successfully verified Protobuf interning! Both strings point to %p", ptr1)
	}
}
