// validators/label.go
package validators

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/gengo/v2/generator"
	"k8s.io/gengo/v2/types"
)

var validateLabelConsistency = types.Name{Package: libValidationPkg, Name: "ValidateLabelConsistency"}

func init() {
	AddToRegistry(InitLabelValidator)
}

type labelValidationRule struct {
	Label    string `json:"label"`              // The label key to validate
	Field    string `json:"field"`              // The field to compare against
	Required bool   `json:"required,omitempty"` // Whether the label must exist
}

func InitLabelValidator(c *generator.Context) DeclarativeValidator {
	return &labelValidator{
		universe: c.Universe,
	}
}

type labelValidator struct {
	universe types.Universe
}

const (
	labelConsistencyTagName = "labelConsistency"
)

func (v *labelValidator) ExtractValidations(t *types.Type, comments []string) (Validations, error) {
	var result Validations

	for _, comment := range comments {
		if !strings.HasPrefix(comment, "+"+labelConsistencyTagName) {
			continue
		}

		// Extract JSON content between = and end of string
		parts := strings.SplitN(comment, "=", 2)
		if len(parts) != 2 {
			return result, fmt.Errorf("invalid label validation format in %q: expected JSON after =", comment)
		}

		// Parse the rule
		var rule labelValidationRule
		if err := json.Unmarshal([]byte(parts[1]), &rule); err != nil {
			return result, fmt.Errorf("invalid JSON in label validation: %v", err)
		}

		if rule.Label == "" || rule.Field == "" {
			return result, fmt.Errorf("both label and field must be specified in %q", comment)
		}

		// Create validation function
		fn := Function(labelConsistencyTagName, DefaultFlags, validateLabelConsistency,
			[]any{rule.Label, rule.Field, rule.Required}...)
		result.Functions = append(result.Functions, fn)
	}

	return result, nil
}

func (v *labelValidator) Docs() []TagDoc {
	return []TagDoc{{
		Tag:         labelConsistencyTagName,
		Description: "Validates consistency between a label and a field",
		Contexts:    []TagContext{TagContextType},
		Payloads: []TagPayloadDoc{{
			Description: `={"label": "<key>", "field": "<fieldName>", "required": <bool>}`,
			Docs: `Validates that the label value matches the specified field value.
Example: +labelConsistency={"label": "app", "field": "Name", "required": true}`,
			Schema: []TagPayloadSchema{{
				Key:   "label",
				Value: "string",
				Docs:  "The label key to validate",
			}, {
				Key:   "field",
				Value: "string",
				Docs:  "The field name to compare against",
			}, {
				Key:     "required",
				Value:   "bool",
				Default: "false",
				Docs:    "Whether the label is required",
			}},
		}},
	}}
}
