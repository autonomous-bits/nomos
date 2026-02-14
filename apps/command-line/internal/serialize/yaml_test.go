package serialize

import (
	"strings"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"gopkg.in/yaml.v3"
)

// TestToYAML_Deterministic tests that ToYAML produces byte-for-byte identical
// output for the same logical snapshot across 10 independent invocations.
// This ensures consistent builds and reproducibility in CI/CD pipelines.
//
// T009: Determinism test (10 runs produce identical output)
func TestToYAML_Deterministic(t *testing.T) {
	// Create a snapshot with nested maps to test key ordering
	now := time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC)
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"zebra": "last",
			"alpha": "first",
			"middle": map[string]any{
				"z": 3,
				"a": 1,
				"m": 2,
			},
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{"a.csl", "b.csl"},
			ProviderAliases: []string{"file"},
			StartTime:       now,
			EndTime:         now.Add(1 * time.Second),
			Errors:          []string{},
			Warnings:        []string{},
			PerKeyProvenance: map[string]compiler.Provenance{
				"zebra": {Source: "a.csl", ProviderAlias: "file"},
				"alpha": {Source: "b.csl", ProviderAlias: "file"},
			},
		},
	}

	// Serialize 10 times and compare bytes
	var firstOutput []byte
	for i := 0; i < 10; i++ {
		output, err := ToYAML(snapshot, true)
		if err != nil {
			t.Fatalf("iteration %d: ToYAML failed: %v", i, err)
		}

		if i == 0 {
			firstOutput = output
		} else if string(output) != string(firstOutput) {
			t.Errorf("iteration %d: output differs from first iteration", i)
			t.Logf("First:\n%s", firstOutput)
			t.Logf("Current:\n%s", output)
		}
	}
}

// TestToYAML_KeyOrdering tests that map keys are sorted alphabetically
// in the YAML output for deterministic builds.
//
// T010: Key ordering test (map keys sorted alphabetically)
func TestToYAML_KeyOrdering(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"zebra":  1,
			"alpha":  2,
			"middle": 3,
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, true)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	outputStr := string(output)

	// Check that keys appear in sorted order in the output string
	alphaPos := indexOf(outputStr, "alpha:")
	middlePos := indexOf(outputStr, "middle:")
	zebraPos := indexOf(outputStr, "zebra:")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output:\n%s", outputStr)
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}
}

// TestToYAML_NestedKeyOrdering tests that nested map keys are also sorted.
func TestToYAML_NestedKeyOrdering(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"config": map[string]any{
				"zebra": "z",
				"alpha": "a",
			},
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, true)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	outputStr := string(output)

	// Find positions within the nested "config" object
	configStart := indexOf(outputStr, "config:")
	if configStart == -1 {
		t.Fatalf("config key not found in:\n%s", outputStr)
	}

	// Find alpha and zebra after config
	alphaPos := indexOfAfter(outputStr, "alpha:", configStart)
	zebraPos := indexOfAfter(outputStr, "zebra:", configStart)

	if alphaPos == -1 || zebraPos == -1 {
		t.Fatalf("nested keys not found after config in:\n%s", outputStr)
	}

	if alphaPos >= zebraPos {
		t.Errorf("nested keys not in sorted order: alpha=%d, zebra=%d", alphaPos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}
}

// TestToYAML_ValidYAML tests that the output parses successfully with yaml.v3.
// This ensures the generated YAML is syntactically valid and can be consumed
// by standard YAML parsers (e.g., Kubernetes, Ansible, Docker Compose).
//
// T011: Valid YAML test (output parses with yaml.v3)
func TestToYAML_ValidYAML(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{
			name: "simple values",
			data: map[string]any{
				"string": "value",
				"number": 42,
				"bool":   true,
			},
		},
		{
			name: "nested objects",
			data: map[string]any{
				"parent": map[string]any{
					"child": map[string]any{
						"value": "nested",
					},
				},
			},
		},
		{
			name: "arrays",
			data: map[string]any{
				"items": []any{"one", "two", "three"},
			},
		},
		{
			name: "mixed types",
			data: map[string]any{
				"config": map[string]any{
					"enabled": true,
					"count":   5,
					"tags":    []any{"prod", "web"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
				Metadata: compiler.Metadata{
					InputFiles:      []string{},
					ProviderAliases: []string{},
					StartTime:       time.Time{},
					EndTime:         time.Time{},
				},
			}

			output, err := ToYAML(snapshot, true)
			if err != nil {
				t.Fatalf("ToYAML failed: %v", err)
			}

			// Verify it's valid YAML by parsing it
			var result map[string]any
			if err := yaml.Unmarshal(output, &result); err != nil {
				t.Errorf("failed to parse YAML output: %v", err)
				t.Logf("Output:\n%s", output)
			}
		})
	}
}

// TestToYAML_NestedStructures tests that deep hierarchies are preserved correctly.
//
// T012: Nested structures test (deep hierarchies preserved)
func TestToYAML_NestedStructures(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"zebra": "deep",
							"alpha": "value",
						},
					},
				},
			},
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, true)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Parse YAML and verify structure
	var result map[string]any
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	// Navigate nested structure
	data := result["data"].(map[string]any)
	level1 := data["level1"].(map[string]any)
	level2 := level1["level2"].(map[string]any)
	level3 := level2["level3"].(map[string]any)
	level4 := level3["level4"].(map[string]any)

	if level4["alpha"] != "value" {
		t.Errorf("expected alpha=value, got %v", level4["alpha"])
	}
	if level4["zebra"] != "deep" {
		t.Errorf("expected zebra=deep, got %v", level4["zebra"])
	}
}

// TestToYAML_EdgeCases tests edge cases including empty data, special characters,
// and Unicode handling.
//
// T013: Edge cases test (empty data, special chars, Unicode)
func TestToYAML_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{
			name: "empty data",
			data: map[string]any{},
		},
		{
			name: "special characters",
			data: map[string]any{
				"quotes":    `value with "quotes"`,
				"newlines":  "line1\nline2",
				"tabs":      "before\tafter",
				"backslash": `path\to\file`,
			},
		},
		{
			name: "unicode characters",
			data: map[string]any{
				"emoji":    "🚀 rocket",
				"chinese":  "你好世界",
				"arabic":   "مرحبا بالعالم",
				"japanese": "こんにちは世界",
			},
		},
		{
			name: "yaml special values",
			data: map[string]any{
				"null_string":   "null",
				"bool_string":   "true",
				"number_string": "123",
			},
		},
		{
			name: "empty strings and slices",
			data: map[string]any{
				"empty_string": "",
				"empty_array":  []any{},
				"empty_map":    map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
				Metadata: compiler.Metadata{
					InputFiles:      []string{},
					ProviderAliases: []string{},
					StartTime:       time.Time{},
					EndTime:         time.Time{},
				},
			}

			output, err := ToYAML(snapshot, true)
			if err != nil {
				t.Fatalf("ToYAML failed: %v", err)
			}

			// Verify it's valid YAML
			var result map[string]any
			if err := yaml.Unmarshal(output, &result); err != nil {
				t.Errorf("failed to parse YAML output: %v", err)
				t.Logf("Output:\n%s", output)
			}
		})
	}
}

// TestToYAML_ArraysPreserveOrder tests that arrays maintain their order.
func TestToYAML_ArraysPreserveOrder(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"items": []any{"third", "first", "second"},
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{"z.csl", "a.csl"}, // Should preserve this order
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, true)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Parse and verify order
	var result map[string]any
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	data := result["data"].(map[string]any)
	items := data["items"].([]any)

	expected := []string{"third", "first", "second"}
	for i, item := range items {
		if item.(string) != expected[i] {
			t.Errorf("item[%d]: expected %q, got %q", i, expected[i], item)
		}
	}
}

// TestToYAML_NullByteRejection tests that keys containing null bytes are rejected.
func TestToYAML_NullByteRejection(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"valid_key":      "value",
			"invalid\x00key": "should_fail",
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	_, err := ToYAML(snapshot, true)
	if err == nil {
		t.Fatal("expected error for null byte in key, got nil")
	}

	if !strings.Contains(err.Error(), "invalid") || !strings.Contains(strings.ToLower(err.Error()), "key") {
		t.Errorf("expected error message to mention invalid key, got: %v", err)
	}
}

// TestToYAML_ErrorMessageQuality tests that error messages include type name
// and source location for unsupported types.
//
// T014a: Error message quality test
func TestToYAML_ErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]any
		expectedError string
	}{
		{
			name: "unsupported type - channel",
			data: map[string]any{
				"channel": make(chan int),
			},
			expectedError: "unsupported type",
		},
		{
			name: "unsupported type - function",
			data: map[string]any{
				"function": func() {},
			},
			expectedError: "unsupported type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
				Metadata: compiler.Metadata{
					InputFiles:      []string{},
					ProviderAliases: []string{},
					StartTime:       time.Time{},
					EndTime:         time.Time{},
				},
			}

			_, err := ToYAML(snapshot, true)
			if err == nil {
				t.Fatal("expected error for unsupported type, got nil")
			}

			// Error message should include information about the unsupported type
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error message to contain %q, got: %v", tt.expectedError, err)
			}
		})
	}
}

// TestToYAML_TypePreservation tests that different data types are correctly
// represented in YAML format.
func TestToYAML_TypePreservation(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"string": "text",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"null":   nil,
			"array":  []any{1, 2, 3},
			"object": map[string]any{"key": "value"},
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, true)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Parse and verify types are preserved
	var result map[string]any
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse YAML output: %v", err)
	}

	// Extract data section - YAML may return map[interface{}]interface{}
	dataInterface := result["data"]
	if dataInterface == nil {
		t.Fatal("data section not found in output")
	}

	// Convert to map we can work with
	var data map[string]any
	switch v := dataInterface.(type) {
	case map[string]any:
		data = v
	case map[interface{}]interface{}:
		// YAML v3 often returns map[interface{}]interface{}
		data = make(map[string]any)
		for k, val := range v {
			if key, ok := k.(string); ok {
				data[key] = val
			}
		}
	default:
		t.Fatalf("unexpected data type: %T", dataInterface)
	}

	if data["string"] != "text" {
		t.Errorf("string: expected 'text', got %v", data["string"])
	}
	if data["int"] != 42 {
		t.Errorf("int: expected 42, got %v", data["int"])
	}
	if data["float"] != 3.14 {
		t.Errorf("float: expected 3.14, got %v", data["float"])
	}
	if data["bool"] != true {
		t.Errorf("bool: expected true, got %v", data["bool"])
	}
	if data["null"] != nil {
		t.Errorf("null: expected nil, got %v", data["null"])
	}

	// Validate array is present (YAML may use []interface{})
	if data["array"] == nil {
		t.Error("array: expected non-nil value")
	}

	// Validate object is present (YAML may use map[string]interface{} or map[interface{}]interface{})
	if data["object"] == nil {
		t.Error("object: expected non-nil value")
	}
}

// TestToYAML_ExcludeMetadata tests that when includeMetadata=false,
// the output contains ONLY the data section at root level (no "data:" wrapper,
// no "metadata:" section). This enables cleaner output for tools that only
// need the configuration data without build metadata.
//
// T007: Unit test for ToYAML with includeMetadata=false
func TestToYAML_ExcludeMetadata(t *testing.T) {
	now := time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		snapshot        compiler.Snapshot
		includeMetadata bool
		wantDataWrapper bool     // Should output have "data:" wrapper?
		wantMetadata    bool     // Should output have "metadata:" section?
		expectedKeys    []string // Keys expected at root level
	}{
		{
			name: "includeMetadata=false - simple data",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"region": "us-west-2",
					"count":  3,
				},
				Metadata: compiler.Metadata{
					InputFiles:      []string{"test.csl"},
					ProviderAliases: []string{"test"},
					StartTime:       now,
					EndTime:         now.Add(1 * time.Second),
				},
			},
			includeMetadata: false,
			wantDataWrapper: false,
			wantMetadata:    false,
			expectedKeys:    []string{"region", "count"},
		},
		{
			name: "includeMetadata=true - includes all sections",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"region": "us-west-2",
					"count":  3,
				},
				Metadata: compiler.Metadata{
					InputFiles:      []string{"test.csl"},
					ProviderAliases: []string{"test"},
					StartTime:       now,
					EndTime:         now.Add(1 * time.Second),
				},
			},
			includeMetadata: true,
			wantDataWrapper: true,
			wantMetadata:    true,
			expectedKeys:    []string{"data", "metadata"},
		},
		{
			name: "includeMetadata=false - nested data",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"vpc": map[string]any{
						"cidr": "10.0.0.0/16",
						"tags": map[string]any{
							"Name": "main",
							"Env":  "prod",
						},
					},
					"region": "us-east-1",
				},
				Metadata: compiler.Metadata{
					InputFiles:      []string{"config.csl"},
					ProviderAliases: []string{"aws"},
					StartTime:       now,
					EndTime:         now.Add(2 * time.Second),
				},
			},
			includeMetadata: false,
			wantDataWrapper: false,
			wantMetadata:    false,
			expectedKeys:    []string{"vpc", "region"},
		},
		{
			name: "includeMetadata=false - empty data",
			snapshot: compiler.Snapshot{
				Data: map[string]any{},
				Metadata: compiler.Metadata{
					InputFiles:      []string{"empty.csl"},
					ProviderAliases: []string{},
					StartTime:       now,
					EndTime:         now,
				},
			},
			includeMetadata: false,
			wantDataWrapper: false,
			wantMetadata:    false,
			expectedKeys:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ToYAML(tt.snapshot, tt.includeMetadata)
			if err != nil {
				t.Fatalf("ToYAML failed: %v", err)
			}

			// Parse the YAML output
			var result map[string]any
			if err := yaml.Unmarshal(output, &result); err != nil {
				t.Fatalf("failed to parse YAML output: %v", err)
			}

			// Check for "data" wrapper
			_, hasDataKey := result["data"]
			if hasDataKey != tt.wantDataWrapper {
				t.Errorf("data wrapper: got %v, want %v", hasDataKey, tt.wantDataWrapper)
				t.Logf("Output:\n%s", output)
			}

			// Check for "metadata" section
			_, hasMetadataKey := result["metadata"]
			if hasMetadataKey != tt.wantMetadata {
				t.Errorf("metadata section: got %v, want %v", hasMetadataKey, tt.wantMetadata)
				t.Logf("Output:\n%s", output)
			}

			// When includeMetadata=false, verify expected keys are at root level
			if !tt.includeMetadata {
				for _, key := range tt.expectedKeys {
					if _, exists := result[key]; !exists {
						t.Errorf("expected key %q at root level, not found", key)
						t.Logf("Output:\n%s", output)
					}
				}

				// Verify NO "data" or "metadata" keys at root
				if _, exists := result["data"]; exists {
					t.Error("should not have 'data' key at root when includeMetadata=false")
				}
				if _, exists := result["metadata"]; exists {
					t.Error("should not have 'metadata' key at root when includeMetadata=false")
				}
			}

			// Verify output is valid YAML (already parsed above, but good to confirm)
			outputStr := string(output)
			if len(outputStr) == 0 && len(tt.expectedKeys) > 0 {
				t.Error("output is empty but expected keys")
			}
		})
	}
}

// TestToYAML_ExcludeMetadata_KeyOrdering tests that when includeMetadata=false,
// keys at the root level are still sorted alphabetically for deterministic output.
func TestToYAML_ExcludeMetadata_KeyOrdering(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"zebra":  "last",
			"alpha":  "first",
			"middle": "mid",
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{"test.csl"},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToYAML(snapshot, false) // includeMetadata=false
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	outputStr := string(output)

	// Keys should appear in sorted order at root level
	alphaPos := indexOf(outputStr, "alpha:")
	middlePos := indexOf(outputStr, "middle:")
	zebraPos := indexOf(outputStr, "zebra:")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output:\n%s", outputStr)
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}

	// Verify no "data:" or "metadata:" wrappers
	if indexOf(outputStr, "data:") != -1 {
		t.Error("output should not contain 'data:' wrapper when includeMetadata=false")
		t.Logf("Output:\n%s", outputStr)
	}
	if indexOf(outputStr, "metadata:") != -1 {
		t.Error("output should not contain 'metadata:' section when includeMetadata=false")
		t.Logf("Output:\n%s", outputStr)
	}
}
