//go:build integration
// +build integration

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildFailsWithMigrationError_NoLockfile tests that build fails with
// actionable error when no lockfile exists, instructing user to run nomos init.
func TestBuildFailsWithMigrationError_NoLockfile(t *testing.T) {
	nomosPath := buildNomosForTest(t)
	tmpDir := t.TempDir()

	// Create a .csl file that uses a provider
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './shared-configs'

app:
	name: 'test-app'
	env: 'dev'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Explicitly ensure NO lockfile exists (should be empty tmpDir anyway)
	lockfilePath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")
	if _, err := os.Stat(lockfilePath); err == nil {
		os.Remove(lockfilePath) // Clean up if somehow exists
	}

	// Run nomos build (should fail with migration error)
	cmd := exec.Command(nomosPath, "build", "-p", cslPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Expect failure
	if err == nil {
		t.Fatalf("expected build to fail without lockfile, but it succeeded\nOutput: %s", string(output))
	}

	outputStr := string(output)

	// Assert: error message mentions external providers
	if !strings.Contains(outputStr, "external provider") && !strings.Contains(outputStr, "provider not found") {
		t.Errorf("expected error to mention external providers or provider not found, got: %s", outputStr)
	}

	// Assert: error message suggests running nomos init
	if !strings.Contains(outputStr, "nomos init") {
		t.Errorf("expected error to suggest 'nomos init', got: %s", outputStr)
	}

	// Assert: error message mentions the provider type
	if !strings.Contains(outputStr, "file") {
		t.Errorf("expected error to mention provider type 'file', got: %s", outputStr)
	}
}

// TestBuildFailsWithMigrationError_MalformedLockfile tests that build fails
// gracefully when lockfile is malformed.
func TestBuildFailsWithMigrationError_MalformedLockfile(t *testing.T) {
	nomosPath := buildNomosForTest(t)
	tmpDir := t.TempDir()

	// Create .nomos directory and malformed lockfile
	nomosDir := filepath.Join(tmpDir, ".nomos")
	if err := os.MkdirAll(nomosDir, 0755); err != nil {
		t.Fatalf("failed to create .nomos dir: %v", err)
	}

	lockfilePath := filepath.Join(nomosDir, "providers.lock.json")
	malformedJSON := `{ "providers": [ this is not valid json ] }`
	if err := os.WriteFile(lockfilePath, []byte(malformedJSON), 0644); err != nil {
		t.Fatalf("failed to create malformed lockfile: %v", err)
	}

	// Create a .csl file
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './shared-configs'

app:
	name: 'test-app'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos build
	cmd := exec.Command(nomosPath, "build", "-p", cslPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Expect failure
	if err == nil {
		t.Fatalf("expected build to fail with malformed lockfile, but it succeeded\nOutput: %s", string(output))
	}

	outputStr := string(output)

	// Assert: error mentions provider not found or external providers required
	// (malformed lockfile causes fallback to empty registry, which then fails during provider creation)
	if !strings.Contains(outputStr, "provider type") && !strings.Contains(outputStr, "external providers") {
		t.Errorf("expected error to mention provider type or external providers, got: %s", outputStr)
	}

	// Assert: suggests running nomos init (key migration message)
	if !strings.Contains(outputStr, "nomos init") {
		t.Errorf("expected error to suggest 'nomos init', got: %s", outputStr)
	}
}

// TestBuildFailsWithMigrationError_MissingBinary tests that build fails
// when lockfile exists but referenced binary is missing.
func TestBuildFailsWithMigrationError_MissingBinary(t *testing.T) {
	nomosPath := buildNomosForTest(t)
	tmpDir := t.TempDir()

	// Create .nomos directory
	nomosDir := filepath.Join(tmpDir, ".nomos")
	if err := os.MkdirAll(nomosDir, 0755); err != nil {
		t.Fatalf("failed to create .nomos dir: %v", err)
	}

	// Create a valid lockfile pointing to non-existent binary
	lockfilePath := filepath.Join(nomosDir, "providers.lock.json")
	lockfileContent := `{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "0.2.0",
      "os": "darwin",
      "arch": "arm64",
      "path": ".nomos/providers/file/0.2.0/darwin-arm64/provider"
    }
  ]
}`
	if err := os.WriteFile(lockfilePath, []byte(lockfileContent), 0644); err != nil {
		t.Fatalf("failed to create lockfile: %v", err)
	}

	// Create a .csl file
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'file'
	version: '0.2.0'
	directory: './shared-configs'

app:
	name: 'test-app'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl file: %v", err)
	}

	// Run nomos build
	cmd := exec.Command(nomosPath, "build", "-p", cslPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Expect failure
	if err == nil {
		t.Fatalf("expected build to fail with missing binary, but it succeeded\nOutput: %s", string(output))
	}

	outputStr := string(output)

	// Assert: error mentions binary not found
	if !strings.Contains(outputStr, "not found") && !strings.Contains(outputStr, "does not exist") {
		t.Errorf("expected error to mention binary not found, got: %s", outputStr)
	}

	// Assert: suggests re-running nomos init
	if !strings.Contains(outputStr, "nomos init") {
		t.Errorf("expected error to suggest 'nomos init', got: %s", outputStr)
	}
}
