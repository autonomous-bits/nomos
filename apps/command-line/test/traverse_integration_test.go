//go:build integration

// Package test provides integration tests for the Nomos CLI.
package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/traverse"
	"github.com/autonomous-bits/nomos/libs/compiler"
)

// mockProviderRegistry implements compiler.ProviderRegistry for testing.
type mockProviderRegistry struct{}

func (m *mockProviderRegistry) Register(alias string, constructor compiler.ProviderConstructor) {}

func (m *mockProviderRegistry) GetProvider(alias string) (compiler.Provider, error) {
	return nil, compiler.ErrProviderNotRegistered
}

func (m *mockProviderRegistry) RegisteredAliases() []string {
	return []string{}
}

// TestTraverseIntegration_OrderedFilesPassedToCompiler verifies that
// files discovered by traverse.DiscoverFiles produce deterministic ordering.
// Note: The compiler currently uses its own file discovery (single-level only),
// so this test validates that traverse.DiscoverFiles provides the correct
// ordered list that could be used for validation or display purposes.
func TestTraverseIntegration_OrderedFilesPassedToCompiler(t *testing.T) {
	// Create test directory structure (single level, matching compiler behavior)
	tmpDir := t.TempDir()

	// Create files in deliberate order to test sorting
	testFiles := map[string]string{
		"3-third.csl":  `third: "value"`,
		"1-first.csl":  `first: "value"`,
		"2-second.csl": `second: "value"`,
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %q: %v", name, err)
		}
	}

	// Act: Discover files using traverse package
	files, err := traverse.DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	// Assert: Files should be in lexicographic order
	expectedOrder := []string{"1-first.csl", "2-second.csl", "3-third.csl"}
	if len(files) != len(expectedOrder) {
		t.Fatalf("expected %d files, got %d", len(expectedOrder), len(files))
	}

	for i, expected := range expectedOrder {
		basename := filepath.Base(files[i])
		if basename != expected {
			t.Errorf("file %d: expected %q, got %q", i, expected, basename)
		}
	}

	// Act: Pass the path to compiler (it will use its own discovery)
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: &mockProviderRegistry{},
		Vars:             make(map[string]any),
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Check for compilation errors or warnings
	if len(snapshot.Metadata.Errors) > 0 {
		t.Logf("Compilation errors: %v", snapshot.Metadata.Errors)
	}
	if len(snapshot.Metadata.Warnings) > 0 {
		t.Logf("Compilation warnings: %v", snapshot.Metadata.Warnings)
	}

	// Assert: Compiler should have discovered all files (single-level directory)
	if len(snapshot.Metadata.InputFiles) != len(expectedOrder) {
		t.Fatalf("expected %d input files in metadata, got %d",
			len(expectedOrder), len(snapshot.Metadata.InputFiles))
	}

	// Verify compiler also sorted them correctly
	for i, expectedName := range expectedOrder {
		basename := filepath.Base(snapshot.Metadata.InputFiles[i])
		if basename != expectedName {
			t.Errorf("metadata file %d: expected %q, got %q",
				i, expectedName, basename)
		}
	}

	// Additional verification: last-wins semantics should be deterministic
	// Debug: check what's actually in the snapshot
	t.Logf("Snapshot data keys: %v", getKeys(snapshot.Data))
	t.Logf("Snapshot data: %+v", snapshot.Data)

	expectedKeys := []string{"first", "second", "third"}
	for _, key := range expectedKeys {
		if _, exists := snapshot.Data[key]; !exists {
			t.Errorf("expected key %q in snapshot data, but not found", key)
		}
	}
}

// getKeys returns the keys of a map as a slice.
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestTraverseIntegration_NestedDirectories verifies that the traverse
// package handles nested directories correctly with deterministic ordering.
// Note: The compiler currently only discovers files in a single directory level,
// but the traverse package supports recursive discovery for future use.
func TestTraverseIntegration_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure:
	// root/
	//   a.csl
	//   subdir/
	//     b.csl
	//     deeper/
	//       c.csl
	files := map[string]string{
		"a.csl":               `{"a": "root"}`,
		"subdir/b.csl":        `{"b": "subdir"}`,
		"subdir/deeper/c.csl": `{"c": "deeper"}`,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create directory for %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file %q: %v", relPath, err)
		}
	}

	// Act: Discover using traverse package (supports recursion)
	discoveredFiles, err := traverse.DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	// Verify lexicographic ordering of full paths
	if len(discoveredFiles) != 3 {
		t.Fatalf("expected 3 files, got %d", len(discoveredFiles))
	}

	// Files should be sorted by full path
	// Expected: root/a.csl, root/subdir/b.csl, root/subdir/deeper/c.csl
	expectedBasenames := []string{"a.csl", "b.csl", "c.csl"}
	for i, basename := range expectedBasenames {
		if filepath.Base(discoveredFiles[i]) != basename {
			t.Errorf("file %d: expected basename %q, got %q",
				i, basename, filepath.Base(discoveredFiles[i]))
		}
	}

	// Note: We're not testing compiler integration for nested directories
	// because the compiler currently only supports single-level discovery.
	// This test validates that traverse.DiscoverFiles works correctly
	// for future enhancement of the compiler or direct CLI use.
}
