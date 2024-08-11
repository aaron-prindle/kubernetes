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

// +k8s:validation-gen=TypeMeta

// This is a test package.
package typedef

// +validateTrue="type T1"
type T1 struct {
	TypeMeta int

	// +validateTrue="field T1.MSAMSS"
	// +eachKey=+validateTrue="T1.MSAMSS[keys]"
	// +eachVal=+validateTrue="T1.MSAMSS[vals]"
	MSAMSS map[string]AMSS `json:"msamss"`
}

// +validateTrue="type AMSS"
// +eachKey=+validateTrue="AMSS[keys]"
// +eachVal=+validateTrue="AMSS[vals]"
type AMSS map[string]string
