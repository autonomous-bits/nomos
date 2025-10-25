// Package testutil provides utilities for testing the parser.
package testutil

import (
	"bytes"
	"encoding/json"
	"sort"
)

// CanonicalJSON serializes v to JSON with sorted keys for deterministic output.
// This is used for golden file comparisons where order matters.
func CanonicalJSON(v interface{}) ([]byte, error) {
	// First marshal to get the data
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Unmarshal into a generic structure
	var generic interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, err
	}

	// Sort the generic structure
	sorted := sortJSON(generic)

	// Marshal again with indentation for readability
	return json.MarshalIndent(sorted, "", "  ")
}

// sortJSON recursively sorts JSON objects by key.
func sortJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// Sort map keys
		sorted := make(map[string]interface{}, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			sorted[k] = sortJSON(val[k])
		}
		return sorted

	case []interface{}:
		// Recursively sort array elements
		sorted := make([]interface{}, len(val))
		for i, item := range val {
			sorted[i] = sortJSON(item)
		}
		return sorted

	default:
		// Primitive value, return as-is
		return val
	}
}

// CompareJSON compares two JSON byte slices for equality, ignoring whitespace differences.
func CompareJSON(a, b []byte) bool {
	var aData, bData interface{}

	if err := json.Unmarshal(a, &aData); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bData); err != nil {
		return false
	}

	aCanonical, _ := json.Marshal(sortJSON(aData))
	bCanonical, _ := json.Marshal(sortJSON(bData))

	return bytes.Equal(aCanonical, bCanonical)
}
