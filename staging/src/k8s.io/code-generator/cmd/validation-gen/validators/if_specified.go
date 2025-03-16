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
	t := realType(context.Type)
	if t.Kind != types.Struct {
		return Validations{}, fmt.Errorf("can only be used on struct types")
	}

	if len(args) != 2 {
		return Validations{}, fmt.Errorf("requires exactly two args - the condition field and target field")
	}

	// First arg is the condition field (the one that must be specified)
	conditionField := args[0]

	// Second arg is the target field (the one to validate)
	targetField := args[1]

	// Handle nested field paths with X delimiter
	var condMemb *types.Member
	if strings.Contains(conditionField, "X") {
		parts := strings.Split(conditionField, "X")
		// For nested fields, we can only access fields within the current struct hierarchy
		// The validator cannot access parent structs in the hierarchy

		// For now, just handle the simple case of a direct field reference
		// In a real implementation, you would recursively resolve nested fields
		condMemb = getMemberByJSON(t, parts[len(parts)-1])
	} else {
		condMemb = getMemberByJSON(t, conditionField)
	}

	if condMemb == nil {
		return Validations{}, fmt.Errorf("no field for condition json name %q", conditionField)
	}

	// Handle nested field paths for target field
	var targetMemb *types.Member
	if strings.Contains(targetField, "X") {
		parts := strings.Split(targetField, "X")
		// For now, just handle the simple case of a direct field reference
		targetMemb = getMemberByJSON(t, parts[len(parts)-1])
	} else {
		targetMemb = getMemberByJSON(t, targetField)
	}

	if targetMemb == nil {
		return Validations{}, fmt.Errorf("no field for target json name %q", targetField)
	}

	// Ensure the payload contains a validator
	if payload == "" {
		return Validations{}, fmt.Errorf("requires a validator payload")
	}

	// Parse the payload to get the validator tag
	payloadParts := strings.SplitN(payload, "=", 2)
	if len(payloadParts) != 2 {
		return Validations{}, fmt.Errorf("validator must be in format tag=value")
	}

	conditionalTagName := strings.TrimPrefix(payloadParts[0], "+k8s:")
	conditionalTagValue := payloadParts[1]

	// Create a fake comment for the validator to apply to the target field
	fakeComment := fmt.Sprintf("+k8s:%s=%s", conditionalTagName, conditionalTagValue)
	fakeComments := []string{fakeComment}

	// Create a subcontext for the target field
	targetContext := Context{
		Scope:  ScopeField,
		Type:   targetMemb.Type,
		Parent: t,
		Path:   context.Path.Child(targetField),
	}

	// Extract validations for the target field validator
	validations, err := istv.validator.ExtractValidations(targetContext, fakeComments)
	if err != nil {
		return Validations{}, fmt.Errorf("error extracting validations for target field: %v", err)
	}

	result := Validations{}

	// For each validation function extracted for the target field
	for _, vfn := range validations.Functions {
		// Create the IfSpecified validation function
		// It will check if the condition field is specified before validating the target field
		f := Function(ifSpecifiedTagName, DefaultFlags, validateIfSpecified, conditionField, targetField, WrapperFunction{vfn, targetMemb.Type})
		// f := Function(ifSpecifiedTagName, vfn.Flags(), validateIfSpecified, conditionField, targetField, WrapperFunction{vfn, targetMemb.Type})
		result.Functions = append(result.Functions, f)
	}

	// Include any variables from the conditional validator
	result.Variables = append(result.Variables, validations.Variables...)

	return result, nil
}

func (istv ifSpecifiedTagValidator) Docs() TagDoc {
	doc := TagDoc{
		Tag:         istv.TagName(),
		Scopes:      istv.ValidScopes().UnsortedList(),
		Description: "Conditionally validates a field if another field is specified.",
		Args: []TagArgDoc{
			{
				Description: "<condition-field>",
				// Docs:        "The field that must be specified (non-nil, non-zero) for validation to occur. For nested fields, use X as a delimiter (e.g., ChildXActive).",
			},
			{
				Description: "<target-field>",
				// Docs:        "The field to validate if the condition field is specified. For nested fields, use X as a delimiter (e.g., ChildXName).",
			},
		},
		Docs: "The condition field must be specified (non-nil for pointer fields, non-zero for value fields) for the validator to be applied to the target field.",
		Payloads: []TagPayloadDoc{{
			Description: "<validator>",
			Docs:        "The validator to apply to the target field if the condition field is specified.",
		}},
	}
	return doc
}
