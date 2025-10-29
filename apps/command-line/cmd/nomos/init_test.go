package main

import (
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
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act: run init command
	err := runInit([]string{cslPath})

	// Assert: expect error about missing version
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
	if want := "version"; !containsString(err.Error(), want) {
		t.Errorf("expected error to mention 'version', got: %v", err)
	}
}

// TestInitCommand_LocalProviderCopy tests --from flag for local provider installation.
func TestInitCommand_LocalProviderCopy(t *testing.T) {
	// Arrange: create temporary directories
	tmpDir := t.TempDir()
	providerDir := filepath.Join(tmpDir, "provider-binary")
	projectDir := filepath.Join(tmpDir, "project")

	if err := os.MkdirAll(providerDir, 0755); err != nil {
		t.Fatalf("failed to create provider dir: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create a fake provider binary
	providerBinary := filepath.Join(providerDir, "provider")
	if err := os.WriteFile(providerBinary, []byte("#!/bin/sh\necho 'fake provider'"), 0755); err != nil {
		t.Fatalf("failed to create provider binary: %v", err)
	}

	// Create .csl file with version
	cslPath := filepath.Join(projectDir, "test.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './configs'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Change to project directory for init
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Act: run init with --from flag
	err := runInit([]string{"--from", "configs=" + providerBinary, cslPath})

	// Assert: no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Assert: lock file created
	lockPath := filepath.Join(projectDir, ".nomos", "providers.lock.json")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Errorf("lock file not created at %s", lockPath)
	}

	// Assert: binary installed
	installedPath := filepath.Join(projectDir, ".nomos", "providers", "file", "0.2.0")
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		t.Errorf("provider not installed at %s", installedPath)
	}
}

// TestInitCommand_DryRun tests that --dry-run doesn't create files.
func TestInitCommand_DryRun(t *testing.T) {
	// Arrange: create temporary directory with .csl file
	tmpDir := t.TempDir()
	cslPath := filepath.Join(tmpDir, "test.csl")

	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './configs'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Change to project directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Act: run init with --dry-run
	err := runInit([]string{"--dry-run", cslPath})

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
