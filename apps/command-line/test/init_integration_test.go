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

// TestInitCommand_Integration_GitHubProvider tests the full init workflow
// with a GitHub Releases provider. Requires NOMOS_RUN_NETWORK_INTEGRATION=1.
func TestInitCommand_Integration_GitHubProvider(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	// Build the nomos binary
	nomosPath := buildNomosForTest(t)

	// Create temp project directory
	projectDir := t.TempDir()

	// Create .csl file with owner/repo format
	cslPath := filepath.Join(projectDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'autonomous-bits/nomos-provider-file'
	version: '1.0.0'
	directory: './data'

app:
	name: 'test-app'
	env: 'dev'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos init
	cmd := exec.Command(nomosPath, "init", cslPath)
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
			Alias    string                 `json:"alias"`
			Type     string                 `json:"type"`
			Version  string                 `json:"version"`
			OS       string                 `json:"os"`
			Arch     string                 `json:"arch"`
			Path     string                 `json:"path"`
			Checksum string                 `json:"checksum"`
			Source   map[string]interface{} `json:"source"`
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
	if provider.Type != "autonomous-bits/nomos-provider-file" {
		t.Errorf("expected type 'autonomous-bits/nomos-provider-file', got '%s'", provider.Type)
	}
	if provider.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", provider.Version)
	}

	// Verify GitHub source metadata
	github, ok := provider.Source["github"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected github metadata in source, got %T", provider.Source["github"])
	}
	if github["owner"] != "autonomous-bits" {
		t.Errorf("expected owner 'autonomous-bits', got %v", github["owner"])
	}
	if github["repo"] != "nomos-provider-file" {
		t.Errorf("expected repo 'nomos-provider-file', got %v", github["repo"])
	}

	// Verify release_tag is present (should be "v1.0.0" or "1.0.0")
	releaseTag, ok := github["release_tag"].(string)
	if !ok || releaseTag == "" {
		t.Errorf("expected non-empty release_tag, got %v", github["release_tag"])
	}

	// Verify asset name is present
	asset, ok := github["asset"].(string)
	if !ok || asset == "" {
		t.Errorf("expected non-empty asset name, got %v", github["asset"])
	}

	// Verify checksum is present
	if provider.Checksum == "" {
		t.Error("expected non-empty checksum")
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

	// Create .csl file with owner/repo format
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'autonomous-bits/nomos-provider-file'
	version: '1.0.0'
	directory: './data'
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
