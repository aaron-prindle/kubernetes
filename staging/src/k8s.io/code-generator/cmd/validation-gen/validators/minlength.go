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

package validators

import (
	"strconv"

	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/types"
)

func init() {
	AddToRegistry(InitMinLengthDeclarativeValidator)
}

func InitMinLengthDeclarativeValidator(c *generator.Context) DeclarativeValidator {
	return &minLengthDeclarativeValidator{}
}

type minLengthDeclarativeValidator struct{}

const (
	minLengthTagName = "minlength" // TODO: also support k8s:minLength
)

var (
	minLengthValidator = types.Name{Package: libValidationPkg, Name: "MinLength"}
)

func (minLengthDeclarativeValidator) ExtractValidations(t *types.Type, comments []string) (Validations, error) {
	vals, ok := gengo.ExtractCommentTags("+", comments)[minLengthTagName]
	if !ok {
		return Validations{}, nil
	}

	var validations Validations
	for _, val := range vals {
		minLength, err := strconv.Atoi(val)
		if err != nil {
			return Validations{}, err // Return an error if parsing fails
		}

		validations.AddFunction(Function(minLengthTagName, Fatal, minLengthValidator, minLength))
	}
	return validations, nil
}

func (minLengthDeclarativeValidator) Docs() []TagDoc {
	return []TagDoc{{
		Tag:         minLengthTagName,
		Description: "Indicates that a string field has a minimum length.",
		Contexts:    []TagContext{TagContextType, TagContextField},
		Payloads: []TagPayloadDoc{{
			Description: "<non-negative integer>",
			Docs:        "This field must be at least X characters long.",
		}},
	}}
}
