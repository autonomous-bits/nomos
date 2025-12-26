//go:build integration
// +build integration

package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestCompile_MetadataPopulation tests that Snapshot.Metadata is populated correctly.
func TestCompile_MetadataPopulation(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test .csl files
	file1 := filepath.Join(tmpDir, "app.csl")
	file2 := filepath.Join(tmpDir, "config.csl")

	//nolint:gosec // G306: Test file with intentional 0644 permissions
	err := os.WriteFile(file1, []byte(`app:
  name: 'test-app'
  port: 8080
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	//nolint:gosec // G306: Test file with intentional 0644 permissions
	err = os.WriteFile(file2, []byte(`database:
  host: 'localhost'
  port: 5432
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a mock provider registry with registered aliases
	registry := compiler.NewProviderRegistry()

	// Register some mock providers
	registry.Register("file", func(_ compiler.ProviderInitOptions) (compiler.Provider, error) {
		return &mockProvider{alias: "file"}, nil
	})
	registry.Register("env", func(_ compiler.ProviderInitOptions) (compiler.Provider, error) {
		return &mockProvider{alias: "env"}, nil
	})

	// Compile
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Test Metadata fields
	t.Run("InputFiles contains all .csl files", func(t *testing.T) {
		if len(snapshot.Metadata.InputFiles) != 2 {
			t.Errorf("Expected 2 input files, got %d", len(snapshot.Metadata.InputFiles))
		}

		// Files should be sorted lexicographically
		expectedFirst := filepath.Join(tmpDir, "app.csl")
		expectedSecond := filepath.Join(tmpDir, "config.csl")

		absFirst, _ := filepath.Abs(expectedFirst)
		absSecond, _ := filepath.Abs(expectedSecond)

		if snapshot.Metadata.InputFiles[0] != absFirst {
			t.Errorf("Expected first file %q, got %q", absFirst, snapshot.Metadata.InputFiles[0])
		}

		if snapshot.Metadata.InputFiles[1] != absSecond {
			t.Errorf("Expected second file %q, got %q", absSecond, snapshot.Metadata.InputFiles[1])
		}
	})

	t.Run("ProviderAliases contains registered providers", func(t *testing.T) {
		// Even if providers weren't used, the registered aliases should be available
		// According to PRD, this should list providers that were available/registered
		if len(snapshot.Metadata.ProviderAliases) == 0 {
			t.Error("Expected ProviderAliases to be populated, got empty slice")
		}

		// Check that registered aliases are present
		aliases := snapshot.Metadata.ProviderAliases
		hasFile := false
		hasEnv := false

		for _, alias := range aliases {
			if alias == "file" {
				hasFile = true
			}
			if alias == "env" {
				hasEnv = true
			}
		}

		if !hasFile {
			t.Error("Expected 'file' provider alias to be in ProviderAliases")
		}

		if !hasEnv {
			t.Error("Expected 'env' provider alias to be in ProviderAliases")
		}
	})

	t.Run("Timestamps are populated", func(t *testing.T) {
		if snapshot.Metadata.StartTime.IsZero() {
			t.Error("Expected StartTime to be non-zero")
		}

		if snapshot.Metadata.EndTime.IsZero() {
			t.Error("Expected EndTime to be non-zero")
		}

		if snapshot.Metadata.EndTime.Before(snapshot.Metadata.StartTime) {
			t.Error("Expected EndTime to be after StartTime")
		}
	})

	t.Run("PerKeyProvenance is populated", func(t *testing.T) {
		if len(snapshot.Metadata.PerKeyProvenance) == 0 {
			t.Error("Expected PerKeyProvenance to be populated")
		}

		// Check that top-level keys have provenance
		if _, ok := snapshot.Metadata.PerKeyProvenance["app"]; !ok {
			t.Error("Expected provenance for 'app' key")
		}

		if _, ok := snapshot.Metadata.PerKeyProvenance["database"]; !ok {
			t.Error("Expected provenance for 'database' key")
		}
	})

	t.Run("Errors and Warnings are initialized", func(t *testing.T) {
		// Should be non-nil even if empty
		if snapshot.Metadata.Errors == nil {
			t.Error("Expected Errors to be non-nil")
		}

		if snapshot.Metadata.Warnings == nil {
			t.Error("Expected Warnings to be non-nil")
		}
	})
}

// mockProvider is a simple mock provider for testing.
type mockProvider struct {
	alias string
}

func (m *mockProvider) Init(_ context.Context, _ compiler.ProviderInitOptions) error {
	return nil
}

func (m *mockProvider) Fetch(_ context.Context, _ []string) (any, error) {
	return nil, nil
}
