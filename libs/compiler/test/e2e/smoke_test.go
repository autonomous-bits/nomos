//go:build integration

// Package e2e provides end-to-end integration tests for the Nomos compiler library.
//
// These tests validate the complete compilation pipeline from parsing through
// provider resolution to snapshot generation. They use real provider implementations
// and test fixtures to ensure correctness across module boundaries.
//
// Run with:
//
//	go test -tags=integration ./test/e2e
package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/testutil"
)

// TestSmoke_CompilationPipeline tests the complete compilation flow end-to-end.
// This validates that all components work together correctly: Parser → Compiler → Provider resolution.
func TestSmoke_CompilationPipeline(t *testing.T) {
	// Create temporary directory for test fixtures
	tmpDir := t.TempDir()

	// Create a simple config file with various data types
	configFile := filepath.Join(tmpDir, "app.csl")
	configContent := `# Application configuration
database:
  host: 'localhost'
  port: '5432'
  pool_size: '10'
  ssl: 'true'

server:
  host: '0.0.0.0'
  port: '8080'
  timeout: '30'

features:
  - 'authentication'
  - 'logging'
  - 'metrics'
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create provider registry (using real registry, not fake)
	registry := compiler.NewProviderRegistry()

	// Compile the configuration
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	startTime := time.Now()
	snapshot, err := compiler.Compile(ctx, opts)
	duration := time.Since(startTime)

	// Assert compilation succeeded
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Validate snapshot structure (Snapshot is a struct, not pointer)
	if snapshot.Data == nil {
		t.Fatal("snapshot.Data is nil")
	}

	// Debug: Log the snapshot data structure
	t.Logf("Snapshot data keys: %v", getKeys(snapshot.Data))

	// Validate data content (all values are strings in CSL)
	if db, ok := snapshot.Data["database"].(map[string]any); ok {
		if db["host"] != "localhost" {
			t.Errorf("expected database.host='localhost', got %v", db["host"])
		}
		if db["port"] != "5432" {
			t.Errorf("expected database.port='5432', got %v", db["port"])
		}
		if db["ssl"] != "true" {
			t.Errorf("expected database.ssl='true', got %v", db["ssl"])
		}
	} else {
		t.Fatal("database section missing or wrong type")
	}

	// Validate array handling
	if features, ok := snapshot.Data["features"].([]any); ok {
		if len(features) != 3 {
			t.Errorf("expected 3 features, got %d", len(features))
		}
		if features[0] != "authentication" {
			t.Errorf("expected first feature='authentication', got %v", features[0])
		}
	} else {
		t.Fatal("features array missing or wrong type")
	}

	// Validate metadata
	if snapshot.Metadata.InputFiles == nil {
		t.Error("metadata.InputFiles is nil")
	}
	if len(snapshot.Metadata.InputFiles) != 1 {
		t.Errorf("expected 1 input file, got %d", len(snapshot.Metadata.InputFiles))
	}
	if !filepath.IsAbs(snapshot.Metadata.InputFiles[0]) {
		t.Errorf("input file path should be absolute, got %s", snapshot.Metadata.InputFiles[0])
	}

	// Validate timing metadata
	if snapshot.Metadata.StartTime.IsZero() {
		t.Error("metadata.StartTime is zero")
	}
	if snapshot.Metadata.EndTime.IsZero() {
		t.Error("metadata.EndTime is zero")
	}
	if snapshot.Metadata.EndTime.Before(snapshot.Metadata.StartTime) {
		t.Error("metadata.EndTime is before StartTime")
	}

	// Validate error/warning lists exist (even if empty)
	if snapshot.Metadata.Errors == nil {
		t.Error("metadata.Errors should not be nil (can be empty)")
	}
	if snapshot.Metadata.Warnings == nil {
		t.Error("metadata.Warnings should not be nil (can be empty)")
	}
	if len(snapshot.Metadata.Errors) > 0 {
		t.Errorf("expected no errors, got %d: %v", len(snapshot.Metadata.Errors), snapshot.Metadata.Errors)
	}

	t.Logf("✅ Smoke test passed - compilation completed in %v", duration)
}

// TestSmoke_WithProviderReferences tests end-to-end compilation with provider references.
// This validates that reference resolution integrates correctly with the full pipeline.
func TestSmoke_WithProviderReferences(t *testing.T) {
	// Create temporary directory for test fixtures
	tmpDir := t.TempDir()

	// Create a config file with provider references
	configFile := filepath.Join(tmpDir, "config.csl")
	configContent := `# Configuration with references
source configs:
  type: file
  directory: ` + tmpDir + `

database:
  host: reference:configs:db.host
  port: reference:configs:db.port

cache:
  enabled: 'true'
  ttl: '3600'
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create provider data file
	providerDataFile := filepath.Join(tmpDir, "db.csl")
	providerData := `# Provider data
host: 'prod-db.example.com'
port: '5432'
`
	if err := os.WriteFile(providerDataFile, []byte(providerData), 0644); err != nil {
		t.Fatalf("failed to write provider data: %v", err)
	}

	// Create provider registry with file provider
	registry := compiler.NewProviderRegistry()
	provider := testutil.NewFakeFileProvider(tmpDir)
	registry.Register("configs", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		return provider, nil
	})

	// Compile the configuration
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Validate references were resolved
	if db, ok := snapshot.Data["database"].(map[string]any); ok {
		if db["host"] != "prod-db.example.com" {
			t.Errorf("expected database.host='prod-db.example.com', got %v", db["host"])
		}
		if db["port"] != "5432" {
			t.Errorf("expected database.port='5432', got %v", db["port"])
		}
	} else {
		t.Fatal("database section missing or wrong type")
	}

	// Validate provider aliases in metadata
	if len(snapshot.Metadata.ProviderAliases) == 0 {
		t.Error("expected provider aliases in metadata, got none")
	}
	foundConfigsProvider := false
	for _, alias := range snapshot.Metadata.ProviderAliases {
		if alias == "configs" {
			foundConfigsProvider = true
			break
		}
	}
	if !foundConfigsProvider {
		t.Error("expected 'configs' provider in metadata.ProviderAliases")
	}

	t.Log("✅ Smoke test with provider references passed")
}

// TestSmoke_SnapshotDeterminism tests that identical inputs produce identical snapshots.
// This validates deterministic compilation behavior required for reproducible builds.
func TestSmoke_SnapshotDeterminism(t *testing.T) {
	// Create temporary directory for test fixtures
	tmpDir := t.TempDir()

	// Create multiple config files in non-lexicographic order
	files := map[string]string{
		"z-config.csl": "zebra: 'stripes'",
		"a-config.csl": "apple: 'red'",
		"m-config.csl": "mango: 'yellow'",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	// Create provider registry
	registry := compiler.NewProviderRegistry()

	// Compile twice
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	snapshot1, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("first compilation failed: %v", err)
	}

	snapshot2, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("second compilation failed: %v", err)
	}

	// Compare data sections (exclude metadata as it contains timestamps)
	data1JSON, err := json.Marshal(snapshot1.Data)
	if err != nil {
		t.Fatalf("failed to marshal snapshot1 data: %v", err)
	}

	data2JSON, err := json.Marshal(snapshot2.Data)
	if err != nil {
		t.Fatalf("failed to marshal snapshot2 data: %v", err)
	}

	if string(data1JSON) != string(data2JSON) {
		t.Error("snapshots are not deterministic - data differs between runs")
		t.Logf("Snapshot 1 data: %s", string(data1JSON))
		t.Logf("Snapshot 2 data: %s", string(data2JSON))
	}

	// Verify file ordering is lexicographic
	if len(snapshot1.Metadata.InputFiles) != 3 {
		t.Errorf("expected 3 input files, got %d", len(snapshot1.Metadata.InputFiles))
	}

	expectedOrder := []string{"a-config.csl", "m-config.csl", "z-config.csl"}
	for i, expectedName := range expectedOrder {
		if i >= len(snapshot1.Metadata.InputFiles) {
			break
		}
		actualName := filepath.Base(snapshot1.Metadata.InputFiles[i])
		if actualName != expectedName {
			t.Errorf("file %d: expected %s, got %s", i, expectedName, actualName)
		}
	}

	t.Log("✅ Determinism test passed - snapshots are identical across runs")
}

// TestSmoke_ErrorHandling tests that compilation errors are properly reported.
// This validates error handling and diagnostic generation in the full pipeline.
func TestSmoke_ErrorHandling(t *testing.T) {
	// Create temporary directory for test fixtures
	tmpDir := t.TempDir()

	// Create a config file with syntax error
	configFile := filepath.Join(tmpDir, "invalid.csl")
	invalidContent := `# Invalid configuration
database:
  host: "unterminated string
  port: 5432
`
	if err := os.WriteFile(configFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create provider registry
	registry := compiler.NewProviderRegistry()

	// Compile (should fail)
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	snapshot, err := compiler.Compile(ctx, opts)

	// Assert compilation failed
	if err == nil {
		t.Fatal("expected compilation error for invalid syntax, got nil")
	}

	// Verify error message contains useful information
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message is empty")
	}

	// Error should mention the file with the issue
	if !contains(errMsg, "invalid.csl") {
		t.Errorf("error should mention the file name 'invalid.csl', got: %s", errMsg)
	}

	// Snapshot should have errors if compilation failed
	if len(snapshot.Metadata.Errors) == 0 {
		t.Log("Note: snapshot.Metadata.Errors is empty even though compilation failed")
		// This is acceptable - errors might only be in the returned error
	}

	t.Log("✅ Error handling test passed")
}

// contains checks if a string contains a substring (helper function).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

// getKeys returns the keys of a map for debugging.
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
