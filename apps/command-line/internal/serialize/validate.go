package serialize

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Circular Reference Detection
//
// This package does NOT implement circular reference detection.
// That responsibility belongs to the compiler (libs/compiler).
//
// The compiler's ResolveImports() function validates the import graph
// and returns errors for cycles during the compilation phase, before
// any serialization occurs.
//
// Reference: github.com/autonomous-bits/nomos/libs/compiler.ResolveImports()
//
// The serializer assumes it receives well-formed, acyclic data structures
// from compiler.Snapshot. If a programming error in the compiler allows
// circular references to leak through, the serialization will fail with
// a stack overflow or infinite recursion.

// hclIdentifierPattern defines valid HCL variable identifiers for .tfvars format.
// HCL identifiers must:
//   - Start with a letter (a-z, A-Z) or underscore (_)
//   - Contain only letters, digits (0-9), underscores, and hyphens
//
// Examples:
//   - Valid: "region", "vpc_id", "enable-dns", "_private"
//   - Invalid: "123key" (starts with digit), "my key" (contains space), "key@value" (special char)
var hclIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// validateKeyName validates a single key name according to format-specific rules.
//
// Validation rules by format:
//   - JSON: No restrictions (all string keys are valid)
//   - YAML: Keys cannot contain null bytes (\x00)
//   - Tfvars: Keys must match HCL identifier pattern: ^[a-zA-Z_][a-zA-Z0-9_-]*$
//
// Returns an error if the key is invalid for the specified format.
func validateKeyName(key string, format OutputFormat) error {
	switch format {
	case FormatJSON:
		// JSON allows any string key
		return nil

	case FormatYAML:
		// YAML does not support null bytes in keys
		if strings.Contains(key, "\x00") {
			return fmt.Errorf("YAML key cannot contain null bytes: %q", key)
		}
		return nil

	case FormatTfvars:
		// HCL requires identifiers to match strict pattern
		if !hclIdentifierPattern.MatchString(key) {
			return fmt.Errorf("invalid HCL identifier %q: must start with letter or underscore, contain only letters, digits, underscores, and hyphens", key)
		}
		return nil

	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// validateAllKeys validates all top-level keys in a data map for the specified format.
// Returns an error listing all invalid keys if any are found.
// Invalid keys are reported in sorted order for deterministic error messages.
//
// Returns nil if data is nil or empty, or if all keys are valid.
func validateAllKeys(data map[string]any, format OutputFormat) error {
	if len(data) == 0 {
		return nil
	}

	var invalidKeys []string

	for key := range data {
		if err := validateKeyName(key, format); err != nil {
			invalidKeys = append(invalidKeys, key)
		}
	}

	if len(invalidKeys) > 0 {
		// Sort keys for deterministic error messages
		sort.Strings(invalidKeys)
		return fmt.Errorf("invalid keys for %s format: %v", format, invalidKeys)
	}

	return nil
}
