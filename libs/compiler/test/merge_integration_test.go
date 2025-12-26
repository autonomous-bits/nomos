//go:build integration
// +build integration

package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestMergeSemantics_Integration demonstrates merge behavior with actual .csl files.
func TestMergeSemantics_Integration(t *testing.T) {
	// Get path to test fixtures
	testdataDir := filepath.Join("..", "testdata", "merge_semantics")

	// Create a simple provider registry (no providers needed for this test)
	registry := compiler.NewProviderRegistry()

	// Compile the directory with base.csl and override.csl
	snapshot, err := compiler.Compile(context.Background(), compiler.Options{
		Path:             testdataDir,
		ProviderRegistry: registry,
	})

	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Verify that files were discovered in correct order
	expectedFiles := []string{
		filepath.Join(testdataDir, "base.csl"),
		filepath.Join(testdataDir, "override.csl"),
	}

	if len(snapshot.Metadata.InputFiles) != 2 {
		t.Errorf("expected 2 input files, got %d", len(snapshot.Metadata.InputFiles))
	}

	for i, expectedFile := range expectedFiles {
		absPath, _ := filepath.Abs(expectedFile)
		if snapshot.Metadata.InputFiles[i] != absPath {
			t.Errorf("file[%d]: expected %s, got %s", i, absPath, snapshot.Metadata.InputFiles[i])
		}
	}

	// Verify deep-merge behavior for database section
	database, ok := snapshot.Data["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database to be a map, got %T", snapshot.Data["database"])
	}

	// host should be from base.csl (not overwritten)
	if database["host"] != "localhost" {
		t.Errorf("database.host: expected 'localhost', got %v", database["host"])
	}

	// port should be from override.csl (last-wins)
	if database["port"] != "5433" {
		t.Errorf("database.port: expected '5433', got %v", database["port"])
	}

	// name should be from base.csl (not overwritten)
	if database["name"] != "myapp" {
		t.Errorf("database.name: expected 'myapp', got %v", database["name"])
	}

	// ssl should be from override.csl (new key)
	if database["ssl"] != "true" {
		t.Errorf("database.ssl: expected 'true', got %v", database["ssl"])
	}

	// Verify server section uses last-wins
	server, ok := snapshot.Data["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be a map, got %T", snapshot.Data["server"])
	}

	// port should be from override.csl
	if server["port"] != "9000" {
		t.Errorf("server.port: expected '9000', got %v", server["port"])
	}

	// host should be from base.csl (not in override)
	if server["host"] != "0.0.0.0" {
		t.Errorf("server.host: expected '0.0.0.0', got %v", server["host"])
	}

	// Verify provenance tracking
	// Both top-level keys should have provenance from override.csl since it touched them
	overridePath, _ := filepath.Abs(filepath.Join(testdataDir, "override.csl"))

	if snapshot.Metadata.PerKeyProvenance["database"].Source != overridePath {
		t.Errorf("database provenance: expected %s, got %s",
			overridePath, snapshot.Metadata.PerKeyProvenance["database"].Source)
	}

	if snapshot.Metadata.PerKeyProvenance["server"].Source != overridePath {
		t.Errorf("server provenance: expected %s, got %s",
			overridePath, snapshot.Metadata.PerKeyProvenance["server"].Source)
	}
}

// TestMergeSemantics_ArrayReplacement demonstrates array replacement behavior.
func TestMergeSemantics_ArrayReplacement(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	baseFile := filepath.Join(tmpDir, "arrays_base.csl")
	overrideFile := filepath.Join(tmpDir, "arrays_override.csl")

	// base file with array
	baseContent := `items:
  tags: 'tag1,tag2,tag3'
`
	if err := os.WriteFile(baseFile, []byte(baseContent), 0644); err != nil { //nolint:gosec // G306: Test fixture file
		t.Fatalf("failed to write base file: %v", err)
	}

	// override file with different array
	overrideContent := `items:
  tags: 'tagA,tagB'
`
	if err := os.WriteFile(overrideFile, []byte(overrideContent), 0644); err != nil { //nolint:gosec // G306: Test fixture file
		t.Fatalf("failed to write override file: %v", err)
	}

	registry := compiler.NewProviderRegistry()
	snapshot, err := compiler.Compile(context.Background(), compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	})

	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	items, ok := snapshot.Data["items"].(map[string]any)
	if !ok {
		t.Fatalf("expected items to be a map, got %T", snapshot.Data["items"])
	}

	// Arrays should be replaced (last-wins)
	if items["tags"] != "tagA,tagB" {
		t.Errorf("items.tags: expected 'tagA,tagB', got %v", items["tags"])
	}
}

// TestMergeSemantics_GoldenOutput validates against a golden JSON file.
func TestMergeSemantics_GoldenOutput(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata", "merge_semantics")

	registry := compiler.NewProviderRegistry()
	snapshot, err := compiler.Compile(context.Background(), compiler.Options{
		Path:             testdataDir,
		ProviderRegistry: registry,
	})

	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Marshal to JSON for comparison
	actualJSON, err := json.MarshalIndent(snapshot.Data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal snapshot data: %v", err)
	}

	goldenPath := filepath.Join(testdataDir, "expected.golden.json")

	// If GOLDEN_UPDATE is set, update the golden file
	if os.Getenv("GOLDEN_UPDATE") == "1" {
		if err := os.WriteFile(goldenPath, actualJSON, 0644); err != nil { //nolint:gosec // G306: Test golden file
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read golden file
	expectedJSON, err := os.ReadFile(goldenPath) //nolint:gosec // G304: Test golden file with known path
	if err != nil {
		t.Fatalf("failed to read golden file: %v (run with GOLDEN_UPDATE=1 to create)", err)
	}

	// Compare
	if string(actualJSON) != string(expectedJSON) {
		t.Errorf("snapshot data doesn't match golden file.\nExpected:\n%s\n\nGot:\n%s",
			string(expectedJSON), string(actualJSON))
	}
}
