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

// This is a test package.
package cross_field

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// --- Original Struct (approx 5 fields) ---

type Root struct {
	TypeMeta int // Needs a TypeMeta field for validation-gen

	Struct Struct `json:"struct"`
}

// +k8s:rule={"expression":"self.minI <= self.i"}
type Struct struct {
	S    string  `json:"s"`
	I    int     `json:"i"`
	MinI int     `json:"minI"`
	B    bool    `json:"b"`
	F    float64 `json:"f"`
}

// --- Struct2 (approx 20 fields) ---

type Root2 struct {
	TypeMeta int // Needs a TypeMeta field for validation-gen

	Struct Struct2 `json:"struct2"`
}

// +k8s:rule={"expression":"self.minI <= self.i"}

type Struct2 struct {
	S    string  `json:"s"`
	I    int     `json:"i"`    // Validation field
	MinI int     `json:"minI"` // Validation field
	B    bool    `json:"b"`
	F    float64 `json:"f"`
	// Add more fields to reach ~20 total
	Field6  string            `json:"field6"`
	Field7  int               `json:"field7"`
	Field8  bool              `json:"field8"`
	Field9  float64           `json:"field9"`
	Field10 *string           `json:"field10,omitempty"`
	Field11 int32             `json:"field11"`
	Field12 int64             `json:"field12"`
	Field13 []byte            `json:"field13"`
	Field14 map[string]string `json:"field14,omitempty"`
	Field15 Struct            `json:"field15"` // Nested struct
	Field16 *int              `json:"field16,omitempty"`
	Field17 []int             `json:"field17,omitempty"`
	Field18 float32           `json:"field18"`
	Field19 uint              `json:"field19"`
	Field20 string            `json:"field20"`
}

// --- Struct3 (approx 100 fields) ---

type Root3 struct {
	TypeMeta int // Needs a TypeMeta field for validation-gen

	Struct Struct3 `json:"struct3"`
}

// +k8s:rule={"expression":"self.minI <= self.i"}
type Struct3 struct {
	S    string  `json:"s"`
	I    int     `json:"i"`    // Validation field
	MinI int     `json:"minI"` // Validation field
	B    bool    `json:"b"`
	F    float64 `json:"f"`
	// Add more fields to reach ~100 total
	Field6   string            `json:"field6"`
	Field7   int               `json:"field7"`
	Field8   bool              `json:"field8"`
	Field9   float64           `json:"field9"`
	Field10  *string           `json:"field10,omitempty"`
	Field11  int32             `json:"field11"`
	Field12  int64             `json:"field12"`
	Field13  []byte            `json:"field13"`
	Field14  map[string]string `json:"field14,omitempty"`
	Field15  Struct            `json:"field15"` // Nested struct
	Field16  *int              `json:"field16,omitempty"`
	Field17  []int             `json:"field17,omitempty"`
	Field18  float32           `json:"field18"`
	Field19  uint              `json:"field19"`
	Field20  string            `json:"field20"`
	Field21  string            `json:"field21"`
	Field22  int               `json:"field22"`
	Field23  bool              `json:"field23"`
	Field24  float64           `json:"field24"`
	Field25  *string           `json:"field25,omitempty"`
	Field26  int32             `json:"field26"`
	Field27  int64             `json:"field27"`
	Field28  []byte            `json:"field28"`
	Field29  map[string]string `json:"field29,omitempty"`
	Field30  Struct            `json:"field30"`
	Field31  *int              `json:"field31,omitempty"`
	Field32  []int             `json:"field32,omitempty"`
	Field33  float32           `json:"field33"`
	Field34  uint              `json:"field34"`
	Field35  string            `json:"field35"`
	Field36  string            `json:"field36"`
	Field37  int               `json:"field37"`
	Field38  bool              `json:"field38"`
	Field39  float64           `json:"field39"`
	Field40  *string           `json:"field40,omitempty"`
	Field41  int32             `json:"field41"`
	Field42  int64             `json:"field42"`
	Field43  []byte            `json:"field43"`
	Field44  map[string]string `json:"field44,omitempty"`
	Field45  Struct            `json:"field45"`
	Field46  *int              `json:"field46,omitempty"`
	Field47  []int             `json:"field47,omitempty"`
	Field48  float32           `json:"field48"`
	Field49  uint              `json:"field49"`
	Field50  string            `json:"field50"`
	Field51  string            `json:"field51"`
	Field52  int               `json:"field52"`
	Field53  bool              `json:"field53"`
	Field54  float64           `json:"field54"`
	Field55  *string           `json:"field55,omitempty"`
	Field56  int32             `json:"field56"`
	Field57  int64             `json:"field57"`
	Field58  []byte            `json:"field58"`
	Field59  map[string]string `json:"field59,omitempty"`
	Field60  Struct            `json:"field60"`
	Field61  *int              `json:"field61,omitempty"`
	Field62  []int             `json:"field62,omitempty"`
	Field63  float32           `json:"field63"`
	Field64  uint              `json:"field64"`
	Field65  string            `json:"field65"`
	Field66  string            `json:"field66"`
	Field67  int               `json:"field67"`
	Field68  bool              `json:"field68"`
	Field69  float64           `json:"field69"`
	Field70  *string           `json:"field70,omitempty"`
	Field71  int32             `json:"field71"`
	Field72  int64             `json:"field72"`
	Field73  []byte            `json:"field73"`
	Field74  map[string]string `json:"field74,omitempty"`
	Field75  Struct            `json:"field75"`
	Field76  *int              `json:"field76,omitempty"`
	Field77  []int             `json:"field77,omitempty"`
	Field78  float32           `json:"field78"`
	Field79  uint              `json:"field79"`
	Field80  string            `json:"field80"`
	Field81  string            `json:"field81"`
	Field82  int               `json:"field82"`
	Field83  bool              `json:"field83"`
	Field84  float64           `json:"field84"`
	Field85  *string           `json:"field85,omitempty"`
	Field86  int32             `json:"field86"`
	Field87  int64             `json:"field87"`
	Field88  []byte            `json:"field88"`
	Field89  map[string]string `json:"field89,omitempty"`
	Field90  Struct            `json:"field90"`
	Field91  *int              `json:"field91,omitempty"`
	Field92  []int             `json:"field92,omitempty"`
	Field93  float32           `json:"field93"`
	Field94  uint              `json:"field94"`
	Field95  string            `json:"field95"`
	Field96  string            `json:"field96"`
	Field97  int               `json:"field97"`
	Field98  bool              `json:"field98"`
	Field99  float64           `json:"field99"`
	Field100 string            `json:"field100"`
}
