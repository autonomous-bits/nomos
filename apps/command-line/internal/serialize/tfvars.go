package serialize

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// hclIdentifierRegex defines the pattern for valid HCL identifiers.
// HCL identifiers must start with a letter or underscore, and contain only
// letters, digits, underscores, or hyphens.
var hclIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// ToTfvars serializes a snapshot to HCL .tfvars format with deterministic ordering.
// Only the Data section is serialized (metadata is omitted).
// Maps are serialized with sorted keys using HCL variable syntax.
//
// Returns error if:
//   - Snapshot contains unsupported types (func, chan, complex)
//   - Keys don't match HCL identifier rules ([a-zA-Z_][a-zA-Z0-9_-]*)
//   - HCL encoding fails
//
// Example output:
//
//	count  = 10
//	region = "us-west-2"
//	vpc = {
//	  cidr = "10.0.0.0/16"
//	}
func ToTfvars(snapshot compiler.Snapshot) ([]byte, error) {
	// Validate all keys before serialization
	if err := validateTfvarsKeys(snapshot.Data); err != nil {
		return nil, err
	}

	// Create HCL file
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	// Get sorted keys for deterministic output
	keys := make([]string, 0, len(snapshot.Data))
	for k := range snapshot.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Set attributes in sorted order
	for _, key := range keys {
		value := snapshot.Data[key]

		// Convert to cty.Value
		ctyVal, err := goToCty(value, key)
		if err != nil {
			return nil, err
		}

		// Set attribute
		rootBody.SetAttributeValue(key, ctyVal)
	}

	// Get raw bytes from hclwrite
	output := f.Bytes()

	// Normalize spacing: hclwrite aligns = signs with padding, but tests expect single spaces
	// Replace multiple spaces before = with single space
	output = normalizeHCLSpacing(output)

	return output, nil
}

// normalizeHCLSpacing removes alignment padding that hclwrite adds for readability.
// Converts "key   = value" to "key = value" with exactly one space on each side of =.
func normalizeHCLSpacing(data []byte) []byte {
	// Replace multiple spaces before = with single space
	spacesBeforeEq := regexp.MustCompile(`\s+= `)
	result := spacesBeforeEq.ReplaceAll(data, []byte(" = "))

	// Also handle cases where there might be trailing spaces after = (shouldn't happen but be safe)
	spacesAfterEq := regexp.MustCompile(` =\s+`)
	result = spacesAfterEq.ReplaceAll(result, []byte(" = "))

	return result
}

// validateTfvarsKeys validates that all keys (including nested keys) are valid HCL identifiers.
// HCL identifiers must match the pattern [a-zA-Z_][a-zA-Z0-9_-]*.
//
// Returns an error listing all invalid keys in sorted order for deterministic error messages.
func validateTfvarsKeys(data map[string]any) error {
	invalidKeys := collectInvalidKeys(data, "")

	if len(invalidKeys) > 0 {
		sort.Strings(invalidKeys)
		return fmt.Errorf("invalid keys for HCL identifiers (must match [a-zA-Z_][a-zA-Z0-9_-]*): %v", invalidKeys)
	}

	return nil
}

// collectInvalidKeys recursively collects all invalid keys from the data structure.
// The path parameter tracks the current location in the nested structure.
func collectInvalidKeys(data map[string]any, path string) []string {
	var invalidKeys []string

	for key, value := range data {
		// Check if current key is valid
		if !hclIdentifierRegex.MatchString(key) {
			invalidKeys = append(invalidKeys, key)
		}

		// Recurse into nested maps
		if nested, ok := value.(map[string]any); ok {
			nestedPath := key
			if path != "" {
				nestedPath = path + "." + key
			}
			nestedInvalid := collectInvalidKeys(nested, nestedPath)
			invalidKeys = append(invalidKeys, nestedInvalid...)
		}

		// Recurse into slices
		if slice, ok := value.([]any); ok {
			for i, item := range slice {
				if nestedMap, ok := item.(map[string]any); ok {
					itemPath := fmt.Sprintf("%s[%d]", key, i)
					if path != "" {
						itemPath = path + "." + itemPath
					}
					nestedInvalid := collectInvalidKeys(nestedMap, itemPath)
					invalidKeys = append(invalidKeys, nestedInvalid...)
				}
			}
		}
	}

	return invalidKeys
}

// goToCty converts a Go value to a cty.Value for HCL serialization.
// The path parameter is used for error messages to indicate where unsupported types were found.
//
// Supported types:
//   - string, int, int64, float64, bool, nil
//   - []any (converted to cty tuple)
//   - map[string]any (converted to cty object)
//
// Returns error for unsupported types (func, chan, complex).
func goToCty(v any, path string) (cty.Value, error) {
	switch val := v.(type) {
	case string:
		return cty.StringVal(val), nil
	case int:
		return cty.NumberIntVal(int64(val)), nil
	case int64:
		return cty.NumberIntVal(val), nil
	case float64:
		return cty.NumberFloatVal(val), nil
	case bool:
		return cty.BoolVal(val), nil
	case nil:
		return cty.NullVal(cty.DynamicPseudoType), nil
	case []any:
		return convertSliceToCty(val, path)
	case map[string]any:
		return convertMapToCty(val, path)
	default:
		// Detect unsupported types with better error messages
		typeName := fmt.Sprintf("%T", v)
		return cty.NilVal, fmt.Errorf("unsupported type %s for tfvars serialization at %s", typeName, path)
	}
}

// convertSliceToCty converts a Go slice to a cty tuple value.
// Empty slices are converted to empty list values.
func convertSliceToCty(s []any, path string) (cty.Value, error) {
	if len(s) == 0 {
		return cty.ListValEmpty(cty.DynamicPseudoType), nil
	}

	vals := make([]cty.Value, len(s))
	for i, item := range s {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		ctyVal, err := goToCty(item, itemPath)
		if err != nil {
			return cty.NilVal, err
		}
		vals[i] = ctyVal
	}

	return cty.TupleVal(vals), nil
}

// convertMapToCty converts a Go map to a cty object value.
// Empty maps are converted to empty object values.
func convertMapToCty(m map[string]any, path string) (cty.Value, error) {
	if len(m) == 0 {
		return cty.EmptyObjectVal, nil
	}

	vals := make(map[string]cty.Value)
	for k, v := range m {
		keyPath := k
		if path != "" {
			keyPath = path + "." + k
		}
		ctyVal, err := goToCty(v, keyPath)
		if err != nil {
			return cty.NilVal, err
		}
		vals[k] = ctyVal
	}

	return cty.ObjectVal(vals), nil
}
