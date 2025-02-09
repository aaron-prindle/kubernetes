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
package validators

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/v2/types"
)

const (
	ifSpecifiedTagName = "k8s:ifSpecified"
)

func init() {
	RegisterTagValidator(&ifSpecifiedTagValidator{})
}

type ifSpecifiedTagValidator struct {
	validator Validator
}

func (istv *ifSpecifiedTagValidator) Init(cfg Config) {
	istv.validator = cfg.Validator
}

func (ifSpecifiedTagValidator) TagName() string {
	return ifSpecifiedTagName
}

var ifSpecifiedTagValidScopes = sets.New(ScopeAny)

func (ifSpecifiedTagValidator) ValidScopes() sets.Set[Scope] {
	return ifSpecifiedTagValidScopes
}

var (
	validateIfSpecified = types.Name{Package: libValidationPkg, Name: "IfSpecified"}
)

func (istv ifSpecifiedTagValidator) GetValidations(context Context, args []string, payload string) (Validations, error) {
	if len(args) != 1 {
		return Validations{}, fmt.Errorf("requires exactly one arg - the field path to check")
	}

	fieldPath := args[0]

	// Ensure the payload contains a conditional validator
	if payload == "" {
		return Validations{}, fmt.Errorf("requires a conditional validator payload")
	}

	// The payload should be another validator tag with its args, parse it
	// Example: +k8s:format=ip-sloppy
	payloadParts := strings.SplitN(payload, "=", 2)
	if len(payloadParts) != 2 {
		return Validations{}, fmt.Errorf("conditional validator must be in format tag=value")
	}

	conditionalTagName := strings.TrimPrefix(payloadParts[0], "+k8s:")
	conditionalTagValue := payloadParts[1]

	// Create a fake comment for the conditional validator
	fakeComment := fmt.Sprintf("+k8s:%s=%s", conditionalTagName, conditionalTagValue)
	fakeComments := []string{fakeComment}

	// Get validations for the conditional validator
	conditionalValidations, err := istv.validator.ExtractValidations(context, fakeComments)
	if err != nil {
		return Validations{}, fmt.Errorf("error extracting validations for conditional validator: %v", err)
	}

	result := Validations{}

	// Create a function to check if the referenced field is specified
	// and apply the conditional validator if it is
	for _, vfn := range conditionalValidations.Functions {
		// Similar to subfieldTagValidator, wrap the validation function in the IfSpecified function
		f := Function(ifSpecifiedTagName, DefaultFlags, validateIfSpecified, fieldPath, WrapperFunction{vfn, context.Type})
		// f := Function(ifSpecifiedTagName, vfn.Flags(), validateIfSpecified, fieldPath, WrapperFunction{vfn, context.Type})
		result.Functions = append(result.Functions, f)
	}

	// Include any variables from the conditional validator
	result.Variables = append(result.Variables, conditionalValidations.Variables...)

	return result, nil
}

func (istv ifSpecifiedTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:         istv.TagName(),
		Scopes:      istv.ValidScopes().UnsortedList(),
		Description: "Applies a validator only if another field is specified.",
		Args: []TagArgDoc{{
			Description: "<field-path>",
			// Docs:        "The path to the field to check if specified. Can use special references like 'parent', 'self', etc. Fields are separated by 'L' (e.g., 'parentLChildLField').",
		}},
		Docs: "The referenced field must be specified (non-nil for pointer fields, non-zero for value fields) for the conditional validator to be applied. Field paths use 'L' as a delimiter.",
		Payloads: []TagPayloadDoc{{
			Description: "<conditional-validator>",
			Docs:        "The validator to apply if the referenced field is specified.",
		}},
	}
	return doc
}
