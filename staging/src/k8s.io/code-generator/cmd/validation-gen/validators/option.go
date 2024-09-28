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
	"fmt"
	"strings"

	"k8s.io/gengo/v2"
	"k8s.io/gengo/v2/types"
)

func init() {
	AddToRegistry(InitOptionDeclarativeValidator)
}

func InitOptionDeclarativeValidator(cfg *ValidatorConfig) DeclarativeValidator {
	return &optionDeclarativeValidator{cfg: cfg}
}

type optionDeclarativeValidator struct {
	cfg *ValidatorConfig
}

const (
	optionTagName = "option" // TODO: also support k8s:option
)

func (o optionDeclarativeValidator) ExtractValidations(t *types.Type, comments []string) (Validations, error) {
	values, ok := gengo.ExtractCommentTags("+", comments)[optionTagName]
	if !ok {
		return Validations{}, nil
	}
	var functions []FunctionGen
	var variables []VariableGen
	for _, v := range values {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return Validations{}, fmt.Errorf("invalid value %q for option %q", v, parts[0])
		}

		flagName := parts[0]
		embeddedValidation := parts[1]

		validations, err := o.cfg.EmbedValidator.ExtractValidations(t, []string{embeddedValidation})
		if err != nil {
			return Validations{}, err
		}
		for _, fn := range validations.Functions {
			functions = append(functions, WithCondition(fn, Conditions{Flags: []string{flagName}}))
		}
		variables = append(variables, validations.Variables...)
	}
	return Validations{
		Functions: functions,
		Variables: variables,
	}, nil
}

func (optionDeclarativeValidator) Docs() []TagDoc {
	return []TagDoc{{
		Tag:         optionTagName,
		Description: "Declares a validation that only applies when an ValidationOpts flag is set.",
		Contexts:    []TagContext{TagContextType, TagContextField},
		Payloads: []TagPayloadDoc{{
			Description: "<option-name>:<validation-tag>",
			Docs:        "This tag will be evaluated if ValidationOpts has a flag of the same name set to true.",
		}},
	}}
}
