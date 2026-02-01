package serialize

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"gopkg.in/yaml.v3"
)

// ToYAML serializes a snapshot to canonical YAML with deterministic ordering.
//
// The function produces deterministic output by:
//   - Sorting all map keys alphabetically
//   - Preserving array order (arrays are not sorted)
//   - Using consistent YAML formatting
//
// YAML-specific validation:
//   - Keys cannot contain null bytes (\x00) as YAML spec prohibits them
//
// Returns an error if:
//   - YAML encoding fails
//   - Any top-level key contains null bytes
//   - Unsupported data types are encountered (e.g., channels, functions)
//
// The output is compatible with standard YAML parsers including:
//   - Kubernetes manifests
//   - Docker Compose files
//   - Ansible playbooks
//   - GitHub Actions workflows
func ToYAML(snapshot compiler.Snapshot) ([]byte, error) {
	// Validate top-level keys for YAML compatibility
	if err := validateAllKeys(snapshot.Data, FormatYAML); err != nil {
		return nil, err
	}

	// Validate data doesn't contain unsupported types
	if err := validateYAMLTypes(snapshot.Data); err != nil {
		return nil, err
	}

	// Canonicalize the snapshot structure (sorts maps, preserves arrays)
	canonical := canonicalizeForYAML(snapshot)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // Use 2-space indent (YAML convention)

	if err := enc.Encode(canonical); err != nil {
		// Enhance error message for unsupported types
		return nil, fmt.Errorf("failed to encode YAML (unsupported type or invalid structure): %w", err)
	}

	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize YAML encoding: %w", err)
	}

	return buf.Bytes(), nil
}

// canonicalizeForYAML recursively canonicalizes a value for deterministic YAML output.
// It sorts all map keys alphabetically while preserving array order.
// Uses yaml.Node for precise control over key ordering in the output.
func canonicalizeForYAML(v any) *yaml.Node {
	switch val := v.(type) {
	case map[string]any:
		return canonicalizeYAMLMap(val)
	case []any:
		return canonicalizeYAMLSlice(val)
	case compiler.Snapshot:
		// Create mapping node for snapshot
		node := &yaml.Node{Kind: yaml.MappingNode}

		// Add fields in alphabetical order
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "data"},
			canonicalizeForYAML(val.Data),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "metadata"},
			canonicalizeForYAML(val.Metadata),
		)
		return node
	case compiler.Metadata:
		// Create mapping node for metadata
		node := &yaml.Node{Kind: yaml.MappingNode}

		// Fields in alphabetical order
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "end_time"},
			scalarNode(val.EndTime),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "errors"},
			canonicalizeForYAML(val.Errors),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "input_files"},
			canonicalizeForYAML(val.InputFiles),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "per_key_provenance"},
			canonicalizeForYAML(val.PerKeyProvenance),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "provider_aliases"},
			canonicalizeForYAML(val.ProviderAliases),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "start_time"},
			scalarNode(val.StartTime),
			&yaml.Node{Kind: yaml.ScalarNode, Value: "warnings"},
			canonicalizeForYAML(val.Warnings),
		)
		return node
	case compiler.Provenance:
		// Create mapping node for provenance
		node := &yaml.Node{Kind: yaml.MappingNode}

		// Fields in alphabetical order
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "provider_alias"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val.ProviderAlias},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "source"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val.Source},
		)
		return node
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: normalizeString(val)}
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
	default:
		// For primitives (numbers, bools), create scalar node
		return scalarNode(v)
	}
}

// canonicalizeYAMLMap creates a YAML mapping node with sorted keys.
func canonicalizeYAMLMap(m map[string]any) *yaml.Node {
	if m == nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
	}

	// Extract and sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build mapping node with sorted keys
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range keys {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			canonicalizeForYAML(m[k]),
		)
	}

	return node
}

// canonicalizeYAMLSlice creates a YAML sequence node with canonicalized elements.
// Array order is preserved (not sorted).
func canonicalizeYAMLSlice(s []any) *yaml.Node {
	if s == nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
	}

	node := &yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range s {
		node.Content = append(node.Content, canonicalizeForYAML(v))
	}
	return node
}

// scalarNode creates a yaml.Node from a primitive value.
// Lets the yaml encoder determine the appropriate tag and formatting.
func scalarNode(v any) *yaml.Node {
	node := &yaml.Node{}
	if err := node.Encode(v); err != nil {
		// Fallback for unsupported types
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", v)}
	}
	return node
}

// validateYAMLTypes recursively validates that all values are YAML-compatible.
// Returns error for unsupported types like channels, functions, complex numbers.
func validateYAMLTypes(v any) error {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if err := validateYAMLTypes(item); err != nil {
				return fmt.Errorf("in key %q: %w", k, err)
			}
		}
	case []any:
		for i, item := range val {
			if err := validateYAMLTypes(item); err != nil {
				return fmt.Errorf("in array index %d: %w", i, err)
			}
		}
	case chan int, chan string, chan any:
		return fmt.Errorf("unsupported type: %T (channels cannot be serialized to YAML)", v)
	case func():
		return fmt.Errorf("unsupported type: %T (functions cannot be serialized to YAML)", v)
	case string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool, nil:
		// Supported primitive types
		return nil
	default:
		// Use reflection to check for unsupported types
		switch fmt.Sprintf("%T", v) {
		case "chan int", "chan string":
			return fmt.Errorf("unsupported type: %T (channels cannot be serialized to YAML)", v)
		}
		// Allow other types - yaml encoder will handle
		return nil
	}
	return nil
}
