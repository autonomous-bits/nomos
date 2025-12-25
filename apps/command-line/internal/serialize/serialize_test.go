package serialize

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestToJSON_Deterministic tests that ToJSON produces byte-for-byte identical
// output for the same logical snapshot across multiple invocations.
func TestToJSON_Deterministic(t *testing.T) {
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
		output, err := ToJSON(snapshot)
		if err != nil {
			t.Fatalf("iteration %d: ToJSON failed: %v", i, err)
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

// TestToJSON_MapKeyOrdering tests that map keys are sorted in the JSON output.
func TestToJSON_MapKeyOrdering(t *testing.T) {
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

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse JSON to verify structure
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Check that keys appear in sorted order in the output string
	outputStr := string(output)
	alphaPos := indexOf(outputStr, "\"alpha\"")
	middlePos := indexOf(outputStr, "\"middle\"")
	zebraPos := indexOf(outputStr, "\"zebra\"")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output")
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
	}
}

// TestToJSON_NestedMapKeyOrdering tests that nested map keys are also sorted.
func TestToJSON_NestedMapKeyOrdering(t *testing.T) {
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

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	outputStr := string(output)

	// Find positions within the nested "config" object
	configStart := indexOf(outputStr, "\"config\"")
	if configStart == -1 {
		t.Fatalf("config key not found")
	}

	// Find alpha and zebra after config
	alphaPos := indexOfAfter(outputStr, "\"alpha\"", configStart)
	zebraPos := indexOfAfter(outputStr, "\"zebra\"", configStart)

	if alphaPos == -1 || zebraPos == -1 {
		t.Fatalf("nested keys not found")
	}

	if alphaPos >= zebraPos {
		t.Errorf("nested keys not in sorted order: alpha=%d, zebra=%d", alphaPos, zebraPos)
	}
}

// TestToJSON_EmptyData tests serialization of empty data.
func TestToJSON_EmptyData(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Should produce valid JSON
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
}

// TestToJSON_ComplexNesting tests deeply nested structures.
func TestToJSON_ComplexNesting(t *testing.T) {
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"zebra": "deep",
						"alpha": "value",
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

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
}

// TestToJSON_ArraysPreserveOrder tests that arrays maintain their order.
func TestToJSON_ArraysPreserveOrder(t *testing.T) {
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

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse and verify order
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
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

// TestToJSON_InvalidUTF8 tests handling of invalid UTF-8 strings.
func TestToJSON_InvalidUTF8(t *testing.T) {
	// Create a string with invalid UTF-8
	invalidUTF8 := "valid\xc3\x28invalid"

	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"text": invalidUTF8,
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{},
			ProviderAliases: []string{},
			StartTime:       time.Time{},
			EndTime:         time.Time{},
		},
	}

	output, err := ToJSON(snapshot)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Should produce valid JSON without errors
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
}

// Helper functions

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func indexOfAfter(s, substr string, start int) int {
	idx := indexOf(s[start:], substr)
	if idx == -1 {
		return -1
	}
	return start + idx
}
