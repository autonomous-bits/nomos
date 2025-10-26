//go:build integration

package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestIntegration_ImportResolution tests end-to-end import resolution using the file provider.
func TestIntegration_ImportResolution(t *testing.T) {
	t.Skip("Import resolution not yet implemented in compiler")

	ctx := context.Background()

	// Setup test fixtures
	tmpDir := t.TempDir()

	// Create base.csl
	baseContent := `database:
  host: 'localhost'
  port: 5432
  name: 'base_db'

server:
  host: '0.0.0.0'
  port: 8080
`
	basePath := filepath.Join(tmpDir, "base.csl")
	if err := os.WriteFile(basePath, []byte(baseContent), 0644); err != nil {
		t.Fatalf("failed to write base.csl: %v", err)
	}

	// Create override.csl
	overrideContent := `source:
  alias: 'files'
  type: 'file'
  baseDir: '.'

import: files

database:
  host: 'remote-host'
  name: 'production_db'

server:
  port: 9090
`
	overridePath := filepath.Join(tmpDir, "override.csl")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write override.csl: %v", err)
	}

	// Create compiler with file provider
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "file", tmpDir); err != nil {
		t.Fatalf("failed to register file provider: %v", err)
	}

	opts := compiler.Options{
		Path:             overridePath,
		ProviderRegistry: registry,
	}

	// Compile
	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify merged result
	expected := map[string]any{
		"database": map[string]any{
			"host": "remote-host",
			"port": float64(5432), // JSON numbers are float64
			"name": "production_db",
		},
		"server": map[string]any{
			"host": "0.0.0.0",
			"port": float64(9090),
		},
	}

	if !deepEqual(snapshot.Data, expected) {
		actualJSON, _ := json.MarshalIndent(snapshot.Data, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("data mismatch:\nGot:\n%s\n\nExpected:\n%s", actualJSON, expectedJSON)
	}

	// Verify both files are tracked
	if len(snapshot.Metadata.InputFiles) != 2 {
		t.Errorf("expected 2 input files (base + override), got %d", len(snapshot.Metadata.InputFiles))
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

		switch val1 := v1.(type) {
		case map[string]any:
			val2, ok := v2.(map[string]any)
			if !ok || !deepEqual(val1, val2) {
				return false
			}
		case []any:
			val2, ok := v2.([]any)
			if !ok || len(val1) != len(val2) {
				return false
			}
			for i := range val1 {
				if !deepEqualValue(val1[i], val2[i]) {
					return false
				}
			}
		default:
			if v1 != v2 {
				return false
			}
		}
	}

	return true
}

func deepEqualValue(a, b any) bool {
	switch v1 := a.(type) {
	case map[string]any:
		v2, ok := b.(map[string]any)
		return ok && deepEqual(v1, v2)
	case []any:
		v2, ok := b.([]any)
		if !ok || len(v1) != len(v2) {
			return false
		}
		for i := range v1 {
			if !deepEqualValue(v1[i], v2[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
