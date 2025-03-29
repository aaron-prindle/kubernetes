/*
Copyright 2015 The Kubernetes Authors.

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

package gengo

import (
	"bytes"
	"fmt"
	"strings"
	// "unicode" // No longer needed in parseTagArgs after modification
)

// ExtractCommentTags parses comments for lines of the form:
//
//	'marker' + "key=value".
//
// Values are optional; "" is the default.  A tag can be specified more than
// one time and all values are returned.  If the resulting map has an entry for
// a key, the value (a slice) is guaranteed to have at least 1 element.
//
// Example: if you pass "+" for 'marker', and the following lines are in
// the comments:
//
//	+foo=value1
//	+bar
//	+foo=value2
//	+baz="qux"
//
// Then this function will return:
//
//	map[string][]string{"foo":{"value1, "value2"}, "bar": {""}, "baz": {`"qux"`}}
//
// Deprecated: Use ExtractFunctionStyleCommentTags.
func ExtractCommentTags(marker string, lines []string) map[string][]string {
	out := map[string][]string{}
	for _, line := range lines {
		line = strings.Trim(line, " ")
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, marker) {
			continue
		}
		kv := strings.SplitN(line[len(marker):], "=", 2)
		if len(kv) == 2 {
			out[kv[0]] = append(out[kv[0]], kv[1])
		} else if len(kv) == 1 {
			out[kv[0]] = append(out[kv[0]], "")
		}
	}
	return out
}

// ExtractSingleBoolCommentTag parses comments for lines of the form:
//
//	'marker' + "key=value1"
//
// If the tag is not found, the default value is returned.  Values are asserted
// to be boolean ("true" or "false"), and any other value will cause an error
// to be returned.  If the key has multiple values, the first one will be used.
func ExtractSingleBoolCommentTag(marker string, key string, defaultVal bool, lines []string) (bool, error) {
	tags, err := ExtractFunctionStyleCommentTags(marker, []string{key}, lines)
	if err != nil {
		return false, err
	}
	values := tags[key]
	if values == nil {
		return defaultVal, nil
	}
	if len(values) == 0 { // Should not happen based on ExtractFunctionStyleCommentTags guarantees, but be safe.
		return defaultVal, nil
	}
	if values[0].Value == "true" {
		return true, nil
	}
	if values[0].Value == "false" {
		return false, nil
	}
	// Allow empty value to mean defaultVal for boolean flags
	if values[0].Value == "" {
		return defaultVal, nil
	}
	return false, fmt.Errorf("tag value for %q is not boolean: %q", key, values[0].Value)
}

// ExtractFunctionStyleCommentTags parses comments for special metadata tags.
// ... (rest of doc comments unchanged, but reflect the new capabilities implicitly) ...
// This function should be preferred instead of ExtractCommentTags.
func ExtractFunctionStyleCommentTags(marker string, tagNames []string, lines []string) (map[string][]Tag, error) {
	stripTrailingComment := func(in string) string {
		parts := strings.SplitN(in, "//", 2)
		return strings.TrimSpace(parts[0])
	}

	out := map[string][]Tag{}
	for i, line := range lines { // Add line number for error reporting
		lineNum := i + 1
		originalLine := line // Keep original for error messages before trimming
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(line, marker) {
			continue
		}
		line = stripTrailingComment(line) // Apply after prefix check
		if len(line) <= len(marker) {     // Handle case like just "+"
			continue
		}
		content := line[len(marker):] // The part after the marker

		keyPart := content
		valuePart := ""

		// Find the assignment operator '=', respecting parentheses in the key/args part.
		// The first '=' *outside* of balanced parentheses is the delimiter.
		parenDepth := 0
		splitIndex := -1
		for j, r := range content {
			if r == '(' {
				parenDepth++
			} else if r == ')' {
				// Only decrement if we are inside parentheses.
				// Handles malformed input like "key)=value" gracefully.
				if parenDepth > 0 {
					parenDepth--
				}
			} else if r == '=' && parenDepth == 0 { // Found '=' at the top level
				splitIndex = j
				break
			}
		}

		// Check for malformed parentheses (unclosed)
		if parenDepth != 0 {
			// This indicates an unmatched parenthesis before any top-level '=',
			// or at the end if no '=' exists. Treat the whole thing as the key.
			// Alternatively, could return an error here. Let parseTagKey handle arg errors.
			splitIndex = -1 // Effectively ignore the '=' if parens are unbalanced
		}

		if splitIndex != -1 {
			keyPart = content[:splitIndex]
			valuePart = content[splitIndex+1:]
		} // else: no top-level '=' found, keyPart remains 'content', valuePart remains ""

		// Now call parseTagKey with the correctly identified keyPart
		tag := Tag{}
		// Pass line number and original line text for better error messages
		if name, args, err := parseTagKey(keyPart, tagNames, lineNum, originalLine); err != nil {
			// Propagate error with context
			return nil, fmt.Errorf("line %d: failed to parse tag in comment '%s': %w", lineNum, originalLine, err)
		} else if name != "" {
			// Only add if parseTagKey returned a valid name (respecting tagNames filter)
			tag.Name, tag.Args = name, args
			tag.Value = valuePart // Assign the correctly identified valuePart
			out[tag.Name] = append(out[tag.Name], tag)
		}
	}
	return out, nil
}

// Tag represents a single comment tag.
type Tag struct {
	// Name is the name of the tag, potentially including operators if they
	// appear before any parenthesis.
	Name string
	// Args is a list of optional arguments to the tag. Arguments can now
	// contain operators and other symbols.
	Args []string
	// Value is the value of the tag.
	Value string
}

func (t Tag) String() string {
	buf := bytes.Buffer{}
	buf.WriteString(t.Name)
	if len(t.Args) > 0 {
		buf.WriteString("(")
		for i, a := range t.Args {
			if i > 0 {
				buf.WriteString(", ") // Keep space for readability if desired
			}
			buf.WriteString(a)
		}
		buf.WriteString(")")
	}
	// Optionally include value for debugging?
	// if t.Value != "" {
	//     buf.WriteString("=")
	//     buf.WriteString(t.Value)
	// }
	return buf.String()
}

// parseTagKey parses the key part of an extended comment tag, including
// optional arguments. The input is assumed to be the entire text of the
// tag *before* the top-level '=' sign (if any).
//
// The tags argument is an optional list of tag names to match. If it is nil or
// empty, all tags match. Matching is done against the part *before* any '('.
//
// This function returns the key name and arguments, unless tagNames was
// specified and the input did not match, in which case it returns "".
// lineNum and originalLine are added for better error context.
func parseTagKey(input string, tagNames []string, lineNum int, originalLine string) (string, []string, error) {
	parts := strings.SplitN(input, "(", 2)
	key := strings.TrimSpace(parts[0]) // Trim spaces from key name itself

	if key == "" {
		return "", nil, fmt.Errorf("empty tag key found") // Or just ignore? Let's error for now.
	}

	if len(tagNames) > 0 {
		found := false
		for _, tn := range tagNames {
			// Exact match on the part before parenthesis
			if key == tn {
				found = true
				break
			}
		}
		if !found {
			// It's not an error, just a tag we are not asked to extract.
			return "", nil, nil
		}
	}

	var args []string
	if len(parts) == 2 {
		// parts[1] contains everything after the '(', potentially including the closing ')'
		// Need to find the *matching* closing parenthesis for the arguments.
		argsPart := parts[1]
		parenDepth := 1 // Start at 1 because we are already inside the '(' from SplitN
		endParenIndex := -1
		for i, r := range argsPart {
			if r == '(' {
				parenDepth++
			} else if r == ')' {
				parenDepth--
				if parenDepth == 0 {
					endParenIndex = i
					break
				}
			}
		}

		if endParenIndex == -1 {
			// No matching closing parenthesis found for the arguments
			return key, nil, fmt.Errorf("mismatched parenthesis in arguments section: expected ')' after '(' in '%s'", input)
		}

		// Ensure nothing unexpected comes after the closing parenthesis but before the (already handled) top-level '='
		// This part should be empty or just whitespace if the original split was correct.
		// We only pass the content *inside* the parens to parseTagArgs.
		argsContent := argsPart[:endParenIndex]
		remaining := strings.TrimSpace(argsPart[endParenIndex+1:])
		if remaining != "" {
			return key, nil, fmt.Errorf("unexpected content '%s' after arguments '(...)' in tag key '%s'", remaining, input)
		}

		// Add context to errors from parseTagArgs
		if ret, err := parseTagArgs(argsContent); err != nil { // Pass only the content inside parens
			// Error is wrapped now in ExtractFunctionStyleCommentTags
			return key, nil, fmt.Errorf("error parsing args '%s': %w", argsContent, err)
		} else {
			args = ret
		}
	} else {
		// No '(', check if the input string *contains* one just in case SplitN logic changes.
		if strings.Contains(input, "(") {
			return key, nil, fmt.Errorf("mismatched parenthesis in tag '%s'", key)
		}
		// No args if no '(' was found
		args = nil
	}
	return key, args, nil
}

// parseTagArgs parses the arguments part of an extended comment tag.
// The input is assumed to be the text *between* the opening and closing parentheses.
// e.g., for "+tag(a,b=c)", input should be "a,b=c"
//
// This function supports comma-separated arguments.
// Arguments can contain any character except ','. Commas separate arguments.
// Whitespace within an argument is preserved. Whitespace around ',' is trimmed.
func parseTagArgs(input string) ([]string, error) {
	// Handle the case of empty args, e.g., key() -> input=""
	if input == "" {
		return nil, nil // No arguments
	}

	var args []string
	var currentArg strings.Builder
	runes := []rune(input) // Use runes for correct character handling

	for i, r := range runes {
		switch r {
		case ',':
			// End the current argument (trimming whitespace) and start a new one
			args = append(args, strings.TrimSpace(currentArg.String()))
			currentArg.Reset() // Reset for the next argument
		case '(':
			// Parentheses are not allowed *nested* within the argument list itself
			return nil, fmt.Errorf("unexpected nested '(' inside arguments at position %d: %q", i, input)
		case ')':
			// The closing parenthesis of the main arg list should have been handled before calling this function.
			return nil, fmt.Errorf("unexpected ')' inside arguments at position %d: %q", i, input)
		default:
			// Allow any other character as part of the current argument
			currentArg.WriteRune(r)
		}
	}

	// Add the last argument after the loop finishes (trimming whitespace)
	args = append(args, strings.TrimSpace(currentArg.String()))

	// Filter out empty strings that might result from ",," or leading/trailing commas
	// Alternatively, decide if empty arguments are valid. Let's allow them for now.
	// finalArgs := []string{}
	// for _, arg := range args {
	//  if arg != "" {
	//      finalArgs = append(finalArgs, arg)
	//  }
	// }
	// return finalArgs, nil

	return args, nil
}
