//go:build integration
// +build integration

package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestInitCommand_Integration_LocalProvider tests the full init workflow
// with a local provider binary.
func TestInitCommand_Integration_LocalProvider(t *testing.T) {
	// Build the nomos binary
	nomosPath := buildNomosForTest(t)

	// Create temp directories
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
	providerBinary := filepath.Join(providerDir, "nomos-provider-file")
	providerContent := `#!/bin/sh
echo 'fake provider'
`
	if err := os.WriteFile(providerBinary, []byte(providerContent), 0755); err != nil {
		t.Fatalf("failed to create provider binary: %v", err)
	}

	// Create .csl file with version and proper source declaration
	cslPath := filepath.Join(projectDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './configs'

app:
	name: 'test-app'
	env: 'dev'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos init with --from flag
	cmd := exec.Command(nomosPath, "init", "--from", "configs="+providerBinary, cslPath)
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify success message
	if want := "Successfully installed 1 provider"; !containsString(string(output), want) {
		t.Errorf("expected success message, got: %s", string(output))
	}

	// Verify lock file created
	lockPath := filepath.Join(projectDir, ".nomos", "providers.lock.json")
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock file not created: %v", err)
	}

	// Parse and validate lock file structure
	var lockFile struct {
		Providers []struct {
			Alias   string `json:"alias"`
			Type    string `json:"type"`
			Version string `json:"version"`
			OS      string `json:"os"`
			Arch    string `json:"arch"`
			Path    string `json:"path"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(lockData, &lockFile); err != nil {
		t.Fatalf("failed to parse lock file: %v", err)
	}

	if len(lockFile.Providers) != 1 {
		t.Fatalf("expected 1 provider in lock file, got %d", len(lockFile.Providers))
	}

	provider := lockFile.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("expected alias 'configs', got '%s'", provider.Alias)
	}
	if provider.Type != "file" {
		t.Errorf("expected type 'file', got '%s'", provider.Type)
	}
	if provider.Version != "0.2.0" {
		t.Errorf("expected version '0.2.0', got '%s'", provider.Version)
	}

	// Verify binary installed at correct path
	installedPath := filepath.Join(projectDir, provider.Path)
	info, err := os.Stat(installedPath)
	if err != nil {
		t.Fatalf("provider binary not installed at %s: %v", installedPath, err)
	}

	// Verify binary is executable
	if info.Mode().Perm()&0111 == 0 {
		t.Error("provider binary is not executable")
	}
}

// TestInitCommand_Integration_MissingVersion tests that init fails
// gracefully when version is missing.
func TestInitCommand_Integration_MissingVersion(t *testing.T) {
	nomosPath := buildNomosForTest(t)
	tmpDir := t.TempDir()

	// Create .csl file WITHOUT version
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	directory: './configs'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos init
	cmd := exec.Command(nomosPath, "init", cslPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Expect failure
	if err == nil {
		t.Fatal("expected init to fail for missing version, but it succeeded")
	}

	// Verify error message mentions version
	if !containsString(string(output), "version") {
		t.Errorf("expected error about missing version, got: %s", string(output))
	}
}

// TestInitCommand_Integration_DryRun tests that --dry-run doesn't create files.
func TestInitCommand_Integration_DryRun(t *testing.T) {
	nomosPath := buildNomosForTest(t)
	tmpDir := t.TempDir()

	// Create .csl file with version
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './configs'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos init with --dry-run
	cmd := exec.Command(nomosPath, "init", "--dry-run", cslPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify dry-run message
	if !containsString(string(output), "Dry run mode") {
		t.Errorf("expected dry-run message, got: %s", string(output))
	}

	// Verify lock file NOT created
	lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should not be created in dry-run mode")
	}
}

// Helper to build nomos binary for testing.
func buildNomosForTest(t *testing.T) string {
	t.Helper()

	tmpBin := filepath.Join(t.TempDir(), "nomos-test")
	// Build from parent directory (apps/command-line) not from test/
	cmd := exec.Command("go", "build", "-o", tmpBin, "../cmd/nomos")

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build nomos: %v\nOutput: %s", err, string(output))
	}

	return tmpBin
}

// Helper to check if string contains substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
