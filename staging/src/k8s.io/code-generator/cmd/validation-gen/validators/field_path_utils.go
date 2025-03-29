// staging/src/k8s.io/code-generator/cmd/validation-gen/validators/field_path_utils.go (New file or add to existing utils)
package validators

import (
	"fmt"
	"strings"

	"k8s.io/gengo/v2/types"
)

// findMemberByPath recursively finds the member sequence corresponding to a dot-separated path.
// It returns the sequence of members leading to the final field and the final member itself.
// It uses JSON names for lookup at each step.
func findMembersByPath(baseType *types.Type, path string) ([]*types.Member, error) {
	parts := strings.Split(path, ".")
	currentType := baseType
	members := make([]*types.Member, 0, len(parts))

	for i, part := range parts {
		// Ensure current type is a struct before looking for members
		structType := realType2(currentType) // Handle pointers/aliases
		if structType.Kind != types.Struct {
			return nil, fmt.Errorf("field path part %q accesses non-struct type %q in path %q", part, currentType.String(), path)
		}

		member := getMemberByJSON(structType, part)
		if member == nil {
			return nil, fmt.Errorf("field %q not found in type %q for path %q", part, structType.Name.String(), path)
		}
		members = append(members, member)

		// If not the last part, update currentType for the next iteration
		if i < len(parts)-1 {
			currentType = member.Type
		}
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("invalid empty field path %q", path)
	}
	return members, nil
}

// generateNestedFieldAccessor creates the Go code string to access a nested field.
// Example: members corresponding to "Nested.Field" -> "o.Nested.Field"
func generateNestedFieldAccessor(baseVarName string, members []*types.Member) string {
	var parts []string
	parts = append(parts, baseVarName)
	for _, m := range members {
		parts = append(parts, m.Name) // Use Go field name for access
	}
	return strings.Join(parts, ".")
}

// realType2 dereferences pointers and aliases until it finds the underlying type.
func realType2(t *types.Type) *types.Type {
	if t == nil {
		return nil
	}
	for t.Kind == types.Pointer || t.Kind == types.Alias {
		if t.Kind == types.Pointer {
			t = t.Elem
		} else { // Alias
			t = t.Underlying
		}
		if t == nil { // Should not happen in valid Go code parsed by gengo
			return nil
		}
	}
	return t
}

// getMemberByJSON finds a member in a struct by its JSON tag name.
// (Assuming this helper function already exists or can be added)
// func getMemberByJSON(structType *types.Type, jsonName string) *types.Member { ... }
