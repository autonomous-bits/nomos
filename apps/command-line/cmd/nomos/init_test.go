package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestInitCommand_MissingVersion tests that init fails when source declaration lacks version.
func TestInitCommand_MissingVersion(t *testing.T) {
	// Arrange: create temporary directory with .csl file missing version
	tmpDir := t.TempDir()
	cslPath := filepath.Join(tmpDir, "test.csl")

	cslContent := `source:
	alias: 'configs'
	type: 'file'
	directory: './configs'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act: run init command via Cobra
	rootCmd.SetArgs([]string{"init", cslPath})
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil) // reset for other tests

	// Assert: expect error about missing version
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
	if want := "version"; !containsString(err.Error(), want) {
		t.Errorf("expected error to mention 'version', got: %v", err)
	}
}

// TestInitCommand_ParsesOwnerRepoFormat tests that init correctly parses owner/repo format in type field.
func TestInitCommand_ParsesOwnerRepoFormat(t *testing.T) {
	// Arrange: create temporary directory with .csl file using owner/repo format
	tmpDir := t.TempDir()
	cslPath := filepath.Join(tmpDir, "test.csl")

	// Use owner/repo format for provider type
	cslContent := `source:
	alias: 'configs'
	type: 'autonomous-bits/nomos-provider-file'
	version: '1.0.0'
	directory: './data'
`
	//nolint:gosec // G306: Test fixture file creation, 0644 permissions are appropriate
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Act: run init via Cobra (will fail to download without network, but should parse correctly)
	// For this test, we just verify the error message indicates it tried to download
	rootCmd.SetArgs([]string{"init", cslPath})
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil) // reset for other tests

	// Assert: error mentions GitHub or network (indicates it parsed owner/repo correctly)
	// We expect an error because we don't have a mock GitHub server in unit tests
	if err == nil {
		t.Fatal("expected error for missing GitHub access, got nil")
	}

	// The error should mention GitHub or network, indicating it tried to use the downloader
	errStr := err.Error()
	if !containsString(errStr, "GitHub") && !containsString(errStr, "github") && !containsString(errStr, "resolve") {
		t.Errorf("expected error to mention GitHub or resolve, got: %v", err)
	}
}

// TestInitCommand_DryRun tests that --dry-run doesn't create files.
func TestInitCommand_DryRun(t *testing.T) {
	// Arrange: create temporary directory with .csl file
	tmpDir := t.TempDir()
	cslPath := filepath.Join(tmpDir, "test.csl")

	// Use owner/repo format
	cslContent := `source:
	alias: 'configs'
	type: 'autonomous-bits/nomos-provider-file'
	version: '1.0.0'
	directory: './data'
`
	//nolint:gosec // G306: Test fixture file creation, 0644 permissions are appropriate
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Change to project directory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Act: run init with --dry-run via Cobra
	rootCmd.SetArgs([]string{"init", "--dry-run", cslPath})
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil) // reset for other tests

	// Assert: no error
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Assert: lock file NOT created
	lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("lock file should not be created in dry-run mode")
	}
}

// Helper function to check if string contains substring.
func containsString(s, substr string) bool {
	// Simple contains check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
