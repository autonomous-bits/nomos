//go:build integration
// +build integration

package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuild_WithProviders_AutoDownload tests automatic provider download on
// first build when no .nomos/ directory exists.
//
// Scenario: Fresh build with no cached providers
// Expected: Providers are automatically discovered, downloaded, cached, and compilation succeeds
func TestBuild_WithProviders_AutoDownload(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory (no .nomos/ directory)
	testDir := t.TempDir()

	// Create test .csl file declaring a provider
	cslPath := filepath.Join(testDir, "config.csl")
	cslContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
  environment: 'development'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test .csl file: %v", err)
	}

	// Verify no .nomos/ directory exists initially
	nomosDir := filepath.Join(testDir, ".nomos")
	if _, err := os.Stat(nomosDir); !os.IsNotExist(err) {
		t.Fatalf(".nomos directory should not exist initially")
	}

	// Run nomos build from the test directory
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	cmd.Dir = testDir

	stdout, stderr, exitCode := runCommand(t, cmd)

	// Verify exit code is 0 (success)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	// Verify stderr contains "Checking providers..." message
	if !strings.Contains(stderr, "Checking providers...") {
		t.Errorf("stderr should contain 'Checking providers...' message\ngot: %s", stderr)
	}

	// Verify stderr contains "Downloading" message (indicates provider download)
	if !strings.Contains(stderr, "Downloading") {
		t.Errorf("stderr should contain 'Downloading' message\ngot: %s", stderr)
	}

	// Verify .nomos/ directory was created
	if _, err := os.Stat(nomosDir); os.IsNotExist(err) {
		t.Error(".nomos directory should exist after build")
	}

	// Verify provider binary exists
	providerDir := filepath.Join(nomosDir, "providers")
	if _, err := os.Stat(providerDir); os.IsNotExist(err) {
		t.Error("providers directory should exist after build")
	}

	// Verify lockfile exists at .nomos/providers.lock.json
	lockfilePath := filepath.Join(nomosDir, "providers.lock.json")
	lockData, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile should exist at %s: %v", lockfilePath, err)
	}

	// Parse lockfile and verify structure
	var lockfile struct {
		Providers []struct {
			Alias    string `json:"alias"`
			Type     string `json:"type"`
			Version  string `json:"version"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Path     string `json:"path"`
			Checksum string `json:"checksum"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(lockData, &lockfile); err != nil {
		t.Fatalf("failed to parse lockfile: %v", err)
	}

	// Verify lockfile contains exactly one provider
	if len(lockfile.Providers) != 1 {
		t.Errorf("lockfile should contain 1 provider, got %d", len(lockfile.Providers))
	}

	// Verify provider metadata in lockfile
	if len(lockfile.Providers) > 0 {
		provider := lockfile.Providers[0]
		if provider.Alias != "configs" {
			t.Errorf("provider alias = %q, want 'configs'", provider.Alias)
		}
		if provider.Type != "autonomous-bits/nomos-provider-file" {
			t.Errorf("provider type = %q, want 'autonomous-bits/nomos-provider-file'", provider.Type)
		}
		if provider.Version != "0.1.1" {
			t.Errorf("provider version = %q, want '0.1.1'", provider.Version)
		}
		if provider.Checksum == "" {
			t.Error("provider checksum should not be empty")
		}

		// Verify provider binary exists at path specified in lockfile
		providerBinaryPath := filepath.Join(testDir, provider.Path)
		info, err := os.Stat(providerBinaryPath)
		if err != nil {
			t.Errorf("provider binary should exist at %s: %v", providerBinaryPath, err)
		} else {
			// Verify binary is executable
			if info.Mode().Perm()&0111 == 0 {
				t.Error("provider binary should be executable")
			}
		}
	}

	// Verify stdout contains JSON output (compilation succeeded)
	if !strings.Contains(stdout, "{") || !strings.Contains(stdout, "}") {
		t.Errorf("stdout should contain JSON output\ngot: %s", stdout)
	}

	// Parse stdout as JSON to verify valid JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("stdout should be valid JSON: %v\nstdout: %s", err, stdout)
	}
}

// TestBuild_WithProviders_AllCached tests build with providers already cached.
//
// Scenario: Second build with all providers cached
// Expected: Build reuses cached providers without re-downloading
func TestBuild_WithProviders_AllCached(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory
	testDir := t.TempDir()

	// Create test .csl file declaring a provider
	cslPath := filepath.Join(testDir, "config.csl")
	cslContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
  environment: 'development'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test .csl file: %v", err)
	}

	// First build: download and cache providers
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	firstCmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	firstCmd.Dir = testDir

	stdout1, stderr1, exitCode1 := runCommand(t, firstCmd)

	if exitCode1 != 0 {
		t.Fatalf("first build failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode1, stdout1, stderr1)
	}

	// Verify first build downloaded the provider
	if !strings.Contains(stderr1, "Downloading") {
		t.Fatalf("first build should download provider\nstderr: %s", stderr1)
	}

	// Second build: should use cached providers
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	secondCmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	secondCmd.Dir = testDir

	stdout2, stderr2, exitCode2 := runCommand(t, secondCmd)

	// Verify second build succeeds
	if exitCode2 != 0 {
		t.Errorf("second build exit code = %d, want 0\nstdout: %s\nstderr: %s",
			exitCode2, stdout2, stderr2)
	}

	// Verify stderr contains "Checking providers..." message
	if !strings.Contains(stderr2, "Checking providers...") {
		t.Errorf("second build stderr should contain 'Checking providers...'\ngot: %s", stderr2)
	}

	// Verify stderr contains "(all cached)" message
	if !strings.Contains(stderr2, "(all cached)") {
		t.Errorf("second build stderr should contain '(all cached)'\ngot: %s", stderr2)
	}

	// Verify stderr does NOT contain "Downloading" message
	if strings.Contains(stderr2, "Downloading") {
		t.Errorf("second build should NOT download provider (all cached)\nstderr: %s", stderr2)
	}

	// Verify stdout contains JSON output (compilation succeeded)
	if !strings.Contains(stdout2, "{") || !strings.Contains(stdout2, "}") {
		t.Errorf("second build stdout should contain JSON output\ngot: %s", stdout2)
	}

	// Parse stdout as JSON to verify valid JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout2), &result); err != nil {
		t.Errorf("second build stdout should be valid JSON: %v\nstdout: %s", err, stdout2)
	}
}

// TestBuild_WithProviders_LockfileUpdated tests lockfile creation and updates.
//
// Scenario: Add a second provider and verify lockfile merges correctly
// Expected: Lockfile contains both providers with correct metadata
func TestBuild_WithProviders_LockfileUpdated(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory
	testDir := t.TempDir()

	// Step 1: Create .csl file with provider A
	cslPath := filepath.Join(testDir, "config.csl")
	cslContentA := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
  environment: 'development'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContentA), 0644); err != nil {
		t.Fatalf("failed to create test .csl file: %v", err)
	}

	// Run first build - should install provider A
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	firstCmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	firstCmd.Dir = testDir

	stdout1, stderr1, exitCode1 := runCommand(t, firstCmd)

	if exitCode1 != 0 {
		t.Fatalf("first build failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode1, stdout1, stderr1)
	}

	// Read lockfile after first build
	lockfilePath := filepath.Join(testDir, ".nomos", "providers.lock.json")
	lockData1, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile should exist after first build: %v", err)
	}

	var lockfile1 struct {
		Providers []struct {
			Alias    string `json:"alias"`
			Type     string `json:"type"`
			Version  string `json:"version"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Path     string `json:"path"`
			Checksum string `json:"checksum"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(lockData1, &lockfile1); err != nil {
		t.Fatalf("failed to parse lockfile after first build: %v", err)
	}

	// Verify lockfile contains provider A
	if len(lockfile1.Providers) != 1 {
		t.Fatalf("lockfile should contain 1 provider after first build, got %d", len(lockfile1.Providers))
	}

	providerA := lockfile1.Providers[0]
	if providerA.Type != "autonomous-bits/nomos-provider-file" {
		t.Errorf("provider A type = %q, want 'autonomous-bits/nomos-provider-file'", providerA.Type)
	}
	if providerA.Version != "0.1.1" {
		t.Errorf("provider A version = %q, want '0.1.1'", providerA.Version)
	}

	// Store provider A metadata for later comparison
	providerAChecksum := providerA.Checksum
	providerAPath := providerA.Path

	// Step 2: Update .csl file to add provider B
	// Note: Using a different version of the same provider type would cause a version conflict,
	// so we'll use the same version. In a real scenario, we'd use a different provider type.
	// For this test, we'll create a second config file to simulate multiple providers.
	cslPath2 := filepath.Join(testDir, "config2.csl")
	cslContentB := `source:
  alias: 'configs2'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data2'

database:
  host: 'localhost'
  port: 5432
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath2, []byte(cslContentB), 0644); err != nil {
		t.Fatalf("failed to create second .csl file: %v", err)
	}

	// Run second build with both .csl files (directory traversal)
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	secondCmd := exec.Command(binPath, "build", "--path", testDir, "--format", "json")
	secondCmd.Dir = testDir

	stdout2, stderr2, exitCode2 := runCommand(t, secondCmd)

	if exitCode2 != 0 {
		t.Fatalf("second build failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode2, stdout2, stderr2)
	}

	// Read lockfile after second build
	lockData2, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile should exist after second build: %v", err)
	}

	var lockfile2 struct {
		Providers []struct {
			Alias    string `json:"alias"`
			Type     string `json:"type"`
			Version  string `json:"version"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Path     string `json:"path"`
			Checksum string `json:"checksum"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(lockData2, &lockfile2); err != nil {
		t.Fatalf("failed to parse lockfile after second build: %v", err)
	}

	// Verify lockfile contains two providers
	if len(lockfile2.Providers) != 2 {
		t.Fatalf("lockfile should contain 2 providers after second build, got %d", len(lockfile2.Providers))
	}

	// Find provider A and B in the lockfile (order may vary)
	var foundA, foundB bool
	for _, provider := range lockfile2.Providers {
		switch provider.Alias {
		case "configs":
			foundA = true

			// Verify provider A metadata unchanged
			if provider.Type != "autonomous-bits/nomos-provider-file" {
				t.Errorf("provider A type changed to %q", provider.Type)
			}
			if provider.Version != "0.1.1" {
				t.Errorf("provider A version changed to %q", provider.Version)
			}
			if provider.Checksum != providerAChecksum {
				t.Errorf("provider A checksum changed from %q to %q", providerAChecksum, provider.Checksum)
			}
			if provider.Path != providerAPath {
				t.Errorf("provider A path changed from %q to %q", providerAPath, provider.Path)
			}
		case "configs2":
			foundB = true

			// Verify provider B has correct metadata
			if provider.Type != "autonomous-bits/nomos-provider-file" {
				t.Errorf("provider B type = %q, want 'autonomous-bits/nomos-provider-file'", provider.Type)
			}
			if provider.Version != "0.1.1" {
				t.Errorf("provider B version = %q, want '0.1.1'", provider.Version)
			}
			if provider.OS == "" {
				t.Error("provider B OS should not be empty")
			}
			if provider.Arch == "" {
				t.Error("provider B Arch should not be empty")
			}
			if provider.Checksum == "" {
				t.Error("provider B Checksum should not be empty")
			}
			if provider.Path == "" {
				t.Error("provider B Path should not be empty")
			}
		}
	}

	if !foundA {
		t.Error("provider A (configs) not found in lockfile after second build")
	}
	if !foundB {
		t.Error("provider B (configs2) not found in lockfile after second build")
	}

	// Verify stdout contains JSON output (compilation succeeded)
	if !strings.Contains(stdout2, "{") || !strings.Contains(stdout2, "}") {
		t.Errorf("second build stdout should contain JSON output\ngot: %s", stdout2)
	}
}
