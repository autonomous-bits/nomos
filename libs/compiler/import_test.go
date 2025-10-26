package compiler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/validator"
)

// TestCompile_SimpleImport tests that a file can import another file.
func TestCompile_SimpleImport(t *testing.T) {
	t.Skip("Import resolution not yet implemented")

	// Arrange
	overridePath := filepath.Join("testdata", "imports", "override.csl")
	registry := NewProviderRegistry()

	// Note: When import resolution is implemented, we'll need to register
	// a file provider here to resolve the imports. This will be done in
	// the integration test once the feature is complete.

	opts := Options{
		Path:             overridePath,
		ProviderRegistry: registry,
	}

	// Act
	snapshot, err := Compile(context.Background(), opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load expected golden file
	goldenPath := filepath.Join("testdata", "imports", "expected.golden.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	var expected map[string]any
	if err := json.Unmarshal(goldenData, &expected); err != nil {
		t.Fatalf("failed to unmarshal golden data: %v", err)
	}

	// Compare results
	if !deepEqual(snapshot.Data, expected) {
		actualJSON, _ := json.MarshalIndent(snapshot.Data, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("data mismatch:\nGot:\n%s\n\nExpected:\n%s", actualJSON, expectedJSON)
	}

	// Verify both files are in metadata
	if len(snapshot.Metadata.InputFiles) != 2 {
		t.Errorf("expected 2 input files (base + override), got %d", len(snapshot.Metadata.InputFiles))
	}
}

// TestCompile_ImportCycle tests that circular imports are detected.
func TestCompile_ImportCycle(t *testing.T) {
	t.Skip("TODO: Implement import cycle detection - tracked in GitHub issue")

	// This test will be enabled once we integrate the validator.DependencyGraph
	// with import resolution to detect circular import chains.
	// Arrange - create temp files with circular imports
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "a.csl")
	file2 := filepath.Join(tmpDir, "b.csl")

	// a.csl imports b.csl
	if err := os.WriteFile(file1, []byte("import: b.csl\n\nconfig:\n  value: 'a'"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}

	// b.csl imports a.csl (creates cycle)
	if err := os.WriteFile(file2, []byte("import: a.csl\n\nconfig:\n  value: 'b'"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	registry := NewProviderRegistry()
	opts := Options{
		Path:             file1,
		ProviderRegistry: registry,
	}

	// Act
	_, err := Compile(context.Background(), opts)

	// Assert
	if err == nil {
		t.Fatal("expected error for circular import, got nil")
	}

	// Check if it's a cycle detection error
	var cycleErr *validator.ErrCycleDetected
	if !stderrors.As(err, &cycleErr) {
		t.Errorf("expected *validator.ErrCycleDetected, got %T: %v", err, err)
	}
}

// deepEqual compares two map[string]any structures deeply.
func deepEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v1 := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}

		if !valuesEqual(v1, v2) {
			return false
		}
	}

	return true
}

func valuesEqual(v1, v2 any) bool {
	switch val1 := v1.(type) {
	case map[string]any:
		val2, ok := v2.(map[string]any)
		if !ok {
			return false
		}
		return deepEqual(val1, val2)

	case string:
		val2, ok := v2.(string)
		return ok && val1 == val2

	case float64:
		val2, ok := v2.(float64)
		return ok && val1 == val2

	case int:
		// JSON unmarshals numbers as float64
		val2, ok := v2.(float64)
		return ok && float64(val1) == val2

	default:
		return v1 == v2
	}
}
