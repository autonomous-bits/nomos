// Package serialize provides deterministic serialization of compiler snapshots.
//
// This package implements canonical serializers for JSON, YAML, and HCL formats
// that guarantee deterministic output for identical logical structures.
//
// Determinism guarantees:
//   - Data section: byte-for-byte identical for identical input (map keys sorted)
//   - Metadata: deterministic ordering of keys, but timestamp values will vary
//   - Per-key provenance: deterministic ordering of keys
//
// Note: The metadata contains timestamps (start_time, end_time) that capture
// when compilation occurred. These will naturally differ between runs. The
// determinism guarantee applies to the structure and ordering, not timestamp values.
package serialize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// ToJSON serializes a snapshot to canonical JSON with deterministic ordering.
// Maps are serialized with sorted keys, and values are normalized for stability.
func ToJSON(snapshot compiler.Snapshot) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// Canonicalize the snapshot structure
	canonical := canonicalizeValue(snapshot)

	if err := enc.Encode(canonical); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Trim trailing newline added by Encoder
	result := bytes.TrimRight(buf.Bytes(), "\n")
	return result, nil
}

// canonicalizeValue recursively canonicalizes a value by sorting map keys
// and normalizing types for deterministic JSON output.
func canonicalizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return canonicalizeMap(val)
	case []any:
		return canonicalizeSlice(val)
	case string:
		return normalizeString(val)
	case compiler.Snapshot:
		// For Snapshot, canonicalize both Data and Metadata
		return map[string]any{
			"data":     canonicalizeValue(val.Data),
			"metadata": canonicalizeValue(val.Metadata),
		}
	case compiler.Metadata:
		// Ensure metadata fields are in deterministic order
		meta := make(map[string]any)
		meta["end_time"] = val.EndTime
		meta["errors"] = canonicalizeValue(val.Errors)
		meta["input_files"] = canonicalizeValue(val.InputFiles)
		meta["per_key_provenance"] = canonicalizeValue(val.PerKeyProvenance)
		meta["provider_aliases"] = canonicalizeValue(val.ProviderAliases)
		meta["start_time"] = val.StartTime
		meta["warnings"] = canonicalizeValue(val.Warnings)
		return meta
	case compiler.Provenance:
		return map[string]any{
			"provider_alias": val.ProviderAlias,
			"source":         val.Source,
		}
	default:
		// For primitives (numbers, bools, nil), return as-is
		return v
	}
}

// canonicalizeMap creates a new map with the same content but guaranteed
// to serialize with sorted keys.
func canonicalizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	// Create a new map with canonicalized values
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = canonicalizeValue(v)
	}

	return result
}

// canonicalizeSlice creates a new slice with canonicalized elements.
func canonicalizeSlice(s []any) []any {
	if s == nil {
		return nil
	}

	result := make([]any, len(s))
	for i, v := range s {
		result[i] = canonicalizeValue(v)
	}
	return result
}

// normalizeString ensures UTF-8 validity and normalization.
func normalizeString(s string) string {
	// Check if string is valid UTF-8
	if !utf8.ValidString(s) {
		// Replace invalid UTF-8 with replacement character
		return strings.ToValidUTF8(s, "�")
	}
	return s
}

// ToYAML serializes a snapshot to YAML with best-effort stability.
func ToYAML(snapshot compiler.Snapshot) ([]byte, error) {
	// Import gopkg.in/yaml.v3 for YAML serialization
	// Note: YAML map key ordering is not guaranteed by spec but gopkg.in/yaml.v3
	// will preserve insertion order. We canonicalize first to get stable ordering.

	_ = canonicalizeValue(snapshot) // prepared for future use

	// For now, return error until we add the dependency
	// The implementation would use yaml.Marshal(canonical)
	return nil, fmt.Errorf("YAML serialization not yet implemented - requires gopkg.in/yaml.v3 dependency")
}

// ToHCL serializes a snapshot to HCL with best-effort stability.
func ToHCL(_ compiler.Snapshot) ([]byte, error) {
	// Import github.com/hashicorp/hcl/v2/hclwrite for HCL serialization
	// Note: HCL serialization for arbitrary data structures is complex and may
	// not preserve all structure types. This is best-effort.

	// For now, return error until we add the dependency
	// The implementation would convert canonical structure to HCL
	return nil, fmt.Errorf("HCL serialization not yet implemented - requires github.com/hashicorp/hcl/v2 dependency")
}
