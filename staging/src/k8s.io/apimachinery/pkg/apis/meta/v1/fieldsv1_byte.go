//go:build !(fieldsv1_string || fieldsv1_stringhandle)

/*
Copyright The Kubernetes Authors.

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

package v1

import (
	"bytes"
	"io"
)

// FieldsV1 stores a set of fields in a data structure like a Trie, in JSON format.
//
// Each key is either a '.' representing the field itself, and will always map to an empty set,
// or a string representing a sub-field or item. The string will follow one of these four formats:
// 'f:<name>', where <name> is the name of a field in a struct, or key in a map
// 'v:<value>', where <value> is the exact json formatted value of a list item
// 'i:<index>', where <index> is position of a item in a list
// 'k:<keys>', where <keys> is a map of  a list item's key fields to their unique values
// If a key maps to an empty Fields value, the field that key represents is part of the set.
//
// The exact format is defined in sigs.k8s.io/structured-merge-diff
// +protobuf.options.(gogoproto.goproto_stringer)=false
type FieldsV1 struct {
	// Raw is the underlying serialization of this object.
	// Deprecated: Direct access to this field is deprecated. Use GetRaw, SetRaw, GetRawReader, NewFieldsV1 instead.
	Raw []byte `json:"-" protobuf:"bytes,1,opt,name=Raw"`
}

func (f FieldsV1) String() string {
	return string(f.Raw)
}

func (f FieldsV1) Equal(f2 FieldsV1) bool {
	return bytes.Equal(f.Raw, f2.Raw)
}

type FieldsV1Reader interface {
	io.Reader
	Size() int64
}

func (f *FieldsV1) GetRawReader() FieldsV1Reader {
	if f == nil || len(f.Raw) == 0 {
		return bytes.NewReader(nil)
	}
	return bytes.NewReader(f.Raw)
}

// GetRawBytes returns the raw bytes.
// During the transition to an internally string-backed representation,
// this method returns the bytes directly or converts them if needed, without allocation when possible.
func (f *FieldsV1) GetRawBytes() []byte {
	if f == nil {
		return nil
	}
	return f.Raw
}

// GetRawString returns the raw data as a string.
// During the transition to an internally string-backed representation,
// this method returns the string directly or converts the bytes to a string.
func (f *FieldsV1) GetRawString() string {
	if f == nil {
		return ""
	}
	return string(f.Raw)
}

// SetRawBytes sets the raw bytes.
func (f *FieldsV1) SetRawBytes(b []byte) {
	if f != nil {
		f.Raw = b
	}
}

// SetRawString sets the raw data from a string.
func (f *FieldsV1) SetRawString(s string) {
	if f != nil {
		f.Raw = []byte(s)
	}
}

func (in *FieldsV1) DeepCopyInto(out *FieldsV1) {
	*out = *in
	if in.Raw != nil {
		in, out := &in.Raw, &out.Raw
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

func NewFieldsV1(raw string) *FieldsV1 {
	return &FieldsV1{Raw: []byte(raw)}
}
