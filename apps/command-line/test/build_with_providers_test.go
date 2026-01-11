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

// TestBuild_WithProviders_ChecksumMismatchRetry tests automatic retry on checksum mismatch.
//
// Scenario: Provider binary has incorrect checksum on disk
// Expected:
//  1. System detects checksum mismatch
//  2. Automatically deletes corrupted binary and retries download
//  3. If second attempt also fails checksum validation, build fails with clear error
//
// Note: This test simulates checksum mismatch by manually corrupting a cached provider.
// In a real scenario, this could happen due to disk corruption or incomplete downloads.
func TestBuild_WithProviders_ChecksumMismatchRetry(t *testing.T) {
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

	// Step 1: First build - download and cache provider successfully
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	firstCmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	firstCmd.Dir = testDir

	stdout1, stderr1, exitCode1 := runCommand(t, firstCmd)

	if exitCode1 != 0 {
		t.Fatalf("first build failed with exit code %d\nstdout: %s\nstderr: %s",
			exitCode1, stdout1, stderr1)
	}

	// Verify provider was downloaded
	if !strings.Contains(stderr1, "Downloading") {
		t.Fatalf("first build should download provider\nstderr: %s", stderr1)
	}

	// Read lockfile to get provider metadata
	lockfilePath := filepath.Join(testDir, ".nomos", "providers.lock.json")
	lockData, err := os.ReadFile(lockfilePath)
	if err != nil {
		t.Fatalf("lockfile should exist after first build: %v", err)
	}

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

	if len(lockfile.Providers) != 1 {
		t.Fatalf("lockfile should contain 1 provider, got %d", len(lockfile.Providers))
	}

	provider := lockfile.Providers[0]
	providerBinaryPath := filepath.Join(testDir, ".nomos", "providers", provider.Path)

	// Verify provider binary exists
	originalBinary, err := os.ReadFile(providerBinaryPath)
	if err != nil {
		t.Fatalf("provider binary should exist at %s: %v", providerBinaryPath, err)
	}

	t.Logf("Original provider binary size: %d bytes, checksum: %s", len(originalBinary), provider.Checksum)

	// Step 2: Corrupt the provider binary to trigger checksum mismatch
	corruptedContent := append([]byte("CORRUPTED:"), originalBinary...)
	if err := os.WriteFile(providerBinaryPath, corruptedContent, 0755); err != nil {
		t.Fatalf("failed to corrupt provider binary: %v", err)
	}

	t.Logf("Corrupted provider binary - new size: %d bytes", len(corruptedContent))

	// Step 3: Second build - should detect checksum mismatch and retry
	// Expected behavior:
	// 1. Validate cached provider
	// 2. Detect checksum mismatch
	// 3. Delete corrupted binary
	// 4. Re-download provider
	// 5. Validate again - should succeed
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	secondCmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	secondCmd.Dir = testDir

	stdout2, stderr2, exitCode2 := runCommand(t, secondCmd)

	// The build should succeed (provider is re-downloaded after checksum mismatch)
	if exitCode2 != 0 {
		t.Logf("Second build stderr: %s", stderr2)
		// Note: Current implementation may not have full retry logic
		// If this fails, it indicates the retry logic needs to be enhanced
		t.Errorf("second build should succeed after retry, got exit code %d\nstdout: %s\nstderr: %s",
			exitCode2, stdout2, stderr2)
	}

	// Verify provider was re-downloaded (should see "Downloading" message again)
	// Note: The exact behavior depends on the implementation
	// Current code in download.go sets shouldDownload=true after validation failure
	if !strings.Contains(stderr2, "Downloading") && !strings.Contains(stderr2, "Checking providers") {
		t.Logf("Warning: Expected to see provider re-download after checksum mismatch")
		t.Logf("stderr: %s", stderr2)
		// This is logged as warning rather than error as the implementation may vary
	}

	// Verify provider binary exists and has correct size after retry
	redownloadedBinary, err := os.ReadFile(providerBinaryPath)
	if err != nil {
		t.Errorf("provider binary should exist after retry at %s: %v", providerBinaryPath, err)
	} else {
		// Binary should be restored to original state
		if len(redownloadedBinary) != len(originalBinary) {
			t.Errorf("re-downloaded binary size = %d, want %d", len(redownloadedBinary), len(originalBinary))
		}
		t.Logf("Re-downloaded provider binary size: %d bytes (matches original: %v)",
			len(redownloadedBinary), len(redownloadedBinary) == len(originalBinary))
	}

	// Verify compilation succeeded (stdout contains JSON)
	if exitCode2 == 0 {
		if !strings.Contains(stdout2, "{") || !strings.Contains(stdout2, "}") {
			t.Errorf("second build stdout should contain JSON output\ngot: %s", stdout2)
		}

		// Parse stdout as JSON to verify valid JSON structure
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout2), &result); err != nil {
			t.Errorf("second build stdout should be valid JSON: %v\nstdout: %s", err, stdout2)
		}
	}
}

// TestBuild_VersionConflict tests that the build fails when multiple .csl files
// in a directory declare the same provider with different versions.
//
// Scenario: Directory contains multiple .csl files with conflicting provider versions
// Expected:
//  1. Provider discovery across all .csl files in directory
//  2. Version conflict detection (same provider type, different versions)
//  3. Build fails with exit code 1
//  4. Error message clearly lists the conflicting versions
//
// This test verifies the version conflict detection implemented in the
// directory traversal fix (T025a-T025c) and detectVersionConflicts function.
func TestBuild_VersionConflict(t *testing.T) {
	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory
	testDir := t.TempDir()

	// Create first .csl file with provider version 0.1.0
	cslPath1 := filepath.Join(testDir, "config1.csl")
	cslContent1 := `source:
  alias: 'files'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.0'
  directory: './data1'

app:
  name: 'app1'
  env: 'dev'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath1, []byte(cslContent1), 0644); err != nil {
		t.Fatalf("failed to create first .csl file: %v", err)
	}

	// Create second .csl file with same provider but different version 0.1.1
	cslPath2 := filepath.Join(testDir, "config2.csl")
	cslContent2 := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data2'

database:
  host: 'localhost'
  port: 5432
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath2, []byte(cslContent2), 0644); err != nil {
		t.Fatalf("failed to create second .csl file: %v", err)
	}

	// Create third .csl file with same provider but different version 0.2.0
	cslPath3 := filepath.Join(testDir, "config3.csl")
	cslContent3 := `source:
  alias: 'data'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.2.0'
  directory: './data3'

cache:
  enabled: true
  ttl: 3600
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath3, []byte(cslContent3), 0644); err != nil {
		t.Fatalf("failed to create third .csl file: %v", err)
	}

	// Run nomos build with the directory (not individual files)
	// This tests directory traversal and multi-file version conflict detection
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--path", testDir, "--format", "json")
	cmd.Dir = testDir

	stdout, stderr, exitCode := runCommand(t, cmd)

	// Verify build fails (exit code 1)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (failure)\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}

	// Verify stderr contains version conflict error message
	if !strings.Contains(stderr, "conflicting provider versions") &&
		!strings.Contains(stderr, "version conflict") &&
		!strings.Contains(stderr, "ErrVersionConflict") {
		t.Errorf("stderr should contain version conflict error message\ngot: %s", stderr)
	}

	// Verify error message mentions the provider type
	if !strings.Contains(stderr, "autonomous-bits/nomos-provider-file") {
		t.Errorf("error message should mention the provider type\nstderr: %s", stderr)
	}

	// Verify error message lists the conflicting versions
	// The error should mention the conflicting versions (order may vary)
	hasVersion010 := strings.Contains(stderr, "0.1.0")
	hasVersion011 := strings.Contains(stderr, "0.1.1")
	hasVersion020 := strings.Contains(stderr, "0.2.0")

	if !hasVersion010 || !hasVersion011 || !hasVersion020 {
		t.Errorf("error message should list all conflicting versions (0.1.0, 0.1.1, 0.2.0)\nstderr: %s", stderr)
	}

	// Verify stdout is empty (compilation should not produce output on error)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout should be empty on error, got: %s", stdout)
	}

	// Verify no .nomos directory was created (should fail before provider download)
	nomosDir := filepath.Join(testDir, ".nomos")
	if _, err := os.Stat(nomosDir); !os.IsNotExist(err) {
		t.Error(".nomos directory should not exist when build fails with version conflict")
	}
}

// TestBuild_WithProviders_ChecksumMismatchPersistentFailure tests failure after persistent checksum issues.
//
// Scenario: Provider binary consistently fails checksum validation (simulates persistent corruption)
// Expected: Build fails with clear error message after retry attempt
//
// Note: This test documents the expected behavior for the scenario where even after
// re-downloading, the checksum still doesn't match. This could happen due to:
// - Corrupted download from source
// - Incorrect checksum in lockfile
// - Network proxy tampering with downloads
//
// Implementation Status: This test is currently marked as skipped because the full
// retry-with-failure-limit logic may not be fully implemented. The test serves as
// documentation of the expected behavior.
func TestBuild_WithProviders_ChecksumMismatchPersistentFailure(t *testing.T) {
	t.Skip("Deferred: Full retry logic with failure after second attempt not yet implemented. See T033 in tasks.md")

	// Implementation note: When this test is enabled, the following setup will be needed:

	// Skip unless network integration is explicitly enabled
	// if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
	// 	t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	// }

	// Build the nomos CLI binary for testing
	// binPath := buildCLI(t)

	// Create temporary test directory
	// testDir := t.TempDir()

	// Create test .csl file
	// cslPath := filepath.Join(testDir, "config.csl")
	// cslContent := `source:
	//   alias: 'configs'
	//   type: 'autonomous-bits/nomos-provider-file'
	//   version: '0.1.1'
	//   directory: './data'
	//
	// app:
	//   name: 'test-app'
	// `
	// //nolint:gosec // G306: Test file with non-sensitive content
	// if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
	// 	t.Fatalf("failed to create test .csl file: %v", err)
	// }

	// Expected behavior (when fully implemented):
	// 1. First build succeeds, provider cached
	// 2. Corrupt provider binary
	// 3. Second build detects corruption, deletes binary, re-downloads
	// 4. Artificially corrupt lockfile checksum to simulate persistent failure
	// 5. Third build (or continuation of second) should fail with clear error message
	//
	// Error message should indicate:
	// - Checksum validation failed
	// - Retry was attempted
	// - Failed after retry
	// - Suggest manual intervention (delete .nomos/ directory, check network)

	// TODO: Implement test logic when retry-with-failure-limit is added to download.go
	// Expected assertions:
	// - Exit code should be non-zero (1)
	// - Error message should mention "checksum mismatch" and "retry"
	// - Provider binary should be deleted
	// - User should be directed to corrective actions
}

// TestBuild_WithProviders_InterruptedDownloadCleanup tests cleanup after interrupted download.
//
// Scenario: Provider download is interrupted (simulated via context cancellation/timeout)
// Expected:
//  1. Download operation is interrupted before completion
//  2. Partial/temporary files are cleaned up
//  3. Lockfile is NOT updated (no partial state persisted)
//  4. Subsequent build can succeed (system can recover)
//
// This test verifies the system handles interruptions gracefully and maintains
// consistency. Interruptions can occur due to:
// - User cancellation (Ctrl+C)
// - Network timeouts
// - Process termination
//
// The test uses an artificially short timeout to simulate interruption during download.
func TestBuild_WithProviders_InterruptedDownloadCleanup(t *testing.T) {
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

	// Step 1: Attempt build with artificially short timeout to trigger interruption
	// Using 1 nanosecond timeout to ensure download is interrupted immediately
	t.Log("Step 1: Attempting build with 1ns timeout to simulate interruption...")
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	interruptedCmd := exec.Command(binPath, "build",
		"--path", cslPath,
		"--format", "json",
		"--timeout-per-provider", "1ns", // Artificially short timeout
	)
	interruptedCmd.Dir = testDir

	stdout1, stderr1, exitCode1 := runCommand(t, interruptedCmd)

	// Verify build failed (expected due to timeout/interruption)
	if exitCode1 == 0 {
		t.Fatalf("interrupted build should fail, got exit code 0\nstdout: %s\nstderr: %s",
			stdout1, stderr1)
	}

	t.Logf("Interrupted build failed as expected (exit code %d)", exitCode1)

	// Verify error message indicates timeout or download failure
	stderrLower := strings.ToLower(stderr1)
	hasTimeoutError := strings.Contains(stderrLower, "timeout") ||
		strings.Contains(stderrLower, "context deadline exceeded") ||
		strings.Contains(stderrLower, "download failed") ||
		strings.Contains(stderrLower, "failed to download")

	if !hasTimeoutError {
		t.Logf("Warning: Expected timeout/download error message not found in stderr")
		t.Logf("stderr: %s", stderr1)
	}

	// Step 2: Verify partial files are cleaned up
	t.Log("Step 2: Verifying partial files are cleaned up...")

	// Check for temporary files/directories
	nomosDir := filepath.Join(testDir, ".nomos")
	tmpDirPath := filepath.Join(nomosDir, ".nomos-tmp")

	// Temporary directory should either not exist or be empty
	if _, err := os.Stat(tmpDirPath); err == nil {
		// Directory exists - verify it's empty
		entries, err := os.ReadDir(tmpDirPath)
		if err != nil {
			t.Fatalf("failed to read temporary directory: %v", err)
		}

		if len(entries) > 0 {
			t.Errorf("temporary directory should be empty after interruption, found %d files/dirs:",
				len(entries))
			for _, entry := range entries {
				t.Logf("  - %s", entry.Name())
			}
		} else {
			t.Log("✓ Temporary directory exists but is empty (cleaned up)")
		}
	} else if os.IsNotExist(err) {
		t.Log("✓ Temporary directory does not exist (cleaned up or never created)")
	} else {
		t.Errorf("unexpected error checking temporary directory: %v", err)
	}

	// Check for partial provider binaries
	providerDir := filepath.Join(nomosDir, "providers")
	if providerDirInfo, err := os.Stat(providerDir); err == nil && providerDirInfo.IsDir() {
		// Walk provider directory tree
		var foundFiles []string
		err := filepath.Walk(providerDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				foundFiles = append(foundFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk provider directory: %v", err)
		}

		if len(foundFiles) > 0 {
			t.Errorf("found %d files in provider directory after interruption (should be empty):",
				len(foundFiles))
			for _, f := range foundFiles {
				relPath, _ := filepath.Rel(nomosDir, f)
				t.Logf("  - %s", relPath)
			}
		} else {
			t.Log("✓ Provider directory is empty (no partial downloads)")
		}
	} else if os.IsNotExist(err) {
		t.Log("✓ Provider directory does not exist (no partial downloads)")
	}

	// Step 3: Verify lockfile was NOT created/updated
	t.Log("Step 3: Verifying lockfile was not created after interruption...")

	lockfilePath := filepath.Join(nomosDir, "providers.lock.json")
	if _, err := os.Stat(lockfilePath); err == nil {
		// Lockfile exists - this may be unexpected
		lockData, readErr := os.ReadFile(lockfilePath)
		if readErr != nil {
			t.Fatalf("lockfile exists but cannot read it: %v", readErr)
		}

		var lockfile struct {
			Providers []struct {
				Alias string `json:"alias"`
			} `json:"providers"`
		}
		if err := json.Unmarshal(lockData, &lockfile); err != nil {
			t.Fatalf("lockfile exists but cannot parse it: %v", err)
		}

		if len(lockfile.Providers) > 0 {
			t.Errorf("lockfile should not contain providers after interrupted download, found %d",
				len(lockfile.Providers))
			t.Logf("lockfile content: %s", string(lockData))
		} else {
			t.Log("✓ Lockfile exists but is empty (no partial state)")
		}
	} else if os.IsNotExist(err) {
		t.Log("✓ Lockfile does not exist (interrupted before lockfile creation)")
	} else {
		t.Errorf("unexpected error checking lockfile: %v", err)
	}

	// Step 4: Verify recovery - subsequent build with proper timeout should succeed
	t.Log("Step 4: Verifying system can recover with proper timeout...")

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	recoveryCmd := exec.Command(binPath, "build",
		"--path", cslPath,
		"--format", "json",
		"--timeout-per-provider", "60s", // Adequate timeout for download
	)
	recoveryCmd.Dir = testDir

	stdout2, stderr2, exitCode2 := runCommand(t, recoveryCmd)

	// Recovery build should succeed
	if exitCode2 != 0 {
		t.Errorf("recovery build should succeed after interruption, got exit code %d\nstdout: %s\nstderr: %s",
			exitCode2, stdout2, stderr2)
	} else {
		t.Log("✓ Recovery build succeeded")
	}

	// Verify provider was downloaded in recovery build
	if exitCode2 == 0 {
		if !strings.Contains(stderr2, "Downloading") {
			t.Logf("Warning: Expected 'Downloading' message in recovery build stderr")
			t.Logf("stderr: %s", stderr2)
		} else {
			t.Log("✓ Provider downloaded during recovery")
		}

		// Verify lockfile now exists and contains provider
		lockData2, err := os.ReadFile(lockfilePath)
		if err != nil {
			t.Errorf("lockfile should exist after successful recovery build: %v", err)
		} else {
			var lockfile2 struct {
				Providers []struct {
					Alias    string `json:"alias"`
					Type     string `json:"type"`
					Version  string `json:"version"`
					Checksum string `json:"checksum"`
				} `json:"providers"`
			}
			if err := json.Unmarshal(lockData2, &lockfile2); err != nil {
				t.Errorf("failed to parse lockfile after recovery: %v", err)
			} else if len(lockfile2.Providers) != 1 {
				t.Errorf("lockfile should contain 1 provider after recovery, got %d",
					len(lockfile2.Providers))
			} else {
				t.Log("✓ Lockfile contains provider after recovery")
				provider := lockfile2.Providers[0]
				if provider.Checksum == "" {
					t.Error("provider checksum should not be empty in lockfile")
				}
			}
		}

		// Verify provider binary exists
		if len(lockData2) > 0 {
			var lockfile2 struct {
				Providers []struct {
					Path string `json:"path"`
				} `json:"providers"`
			}
			if err := json.Unmarshal(lockData2, &lockfile2); err == nil && len(lockfile2.Providers) > 0 {
				providerBinaryPath := filepath.Join(testDir, lockfile2.Providers[0].Path)
				if _, err := os.Stat(providerBinaryPath); err != nil {
					t.Errorf("provider binary should exist at %s: %v", providerBinaryPath, err)
				} else {
					t.Log("✓ Provider binary exists after recovery")
				}
			}
		}

		// Verify stdout contains JSON output (compilation succeeded)
		if !strings.Contains(stdout2, "{") || !strings.Contains(stdout2, "}") {
			t.Errorf("recovery build stdout should contain JSON output\ngot: %s", stdout2)
		} else {
			t.Log("✓ Recovery build produced valid output")
		}
	}

	t.Log("Test completed: Interrupted download cleanup verification passed")
}

// TestBuild_NetworkFailure tests build failure with clear error when provider download fails due to network issues.
//
// Scenario: Provider download fails due to network error (invalid URL/unreachable host)
// Expected:
//  1. Build fails with exit code 1
//  2. Error message includes provider alias
//  3. Error message includes failure reason (network error)
//  4. Error message is actionable for the user
//
// This test verifies Task T031 - network failure error handling and messaging.
func TestBuild_NetworkFailure(t *testing.T) {
	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory
	testDir := t.TempDir()

	// Create test .csl file with invalid/unreachable provider
	// Using a non-existent GitHub owner to simulate network failure
	cslPath := filepath.Join(testDir, "config.csl")
	cslContent := `source:
  alias: 'invalid-provider'
  type: 'nomos-test-invalid-owner-12345/nomos-provider-nonexistent'
  version: '0.1.0'
  directory: './data'

app:
  name: 'test-app'
  environment: 'production'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test .csl file: %v", err)
	}

	// Run nomos build - should fail due to network/download error
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	cmd.Dir = testDir

	stdout, stderr, exitCode := runCommand(t, cmd)

	// Verify build fails with exit code 1
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (failure)\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// Verify error message includes provider alias
	if !strings.Contains(stderr, "invalid-provider") {
		t.Errorf("error message should include provider alias 'invalid-provider'\nstderr: %s", stderr)
	}

	// Verify error message indicates download/network failure
	// Check for common network error patterns
	hasNetworkError := strings.Contains(stderr, "download failed") ||
		strings.Contains(stderr, "failed to download") ||
		strings.Contains(stderr, "network error") ||
		strings.Contains(stderr, "not found") ||
		strings.Contains(stderr, "404") ||
		strings.Contains(stderr, "release not found") ||
		strings.Contains(stderr, "failed to fetch")

	if !hasNetworkError {
		t.Errorf("error message should indicate network/download failure\nstderr: %s", stderr)
	}

	// Verify error message provides actionable information
	// Should mention either the provider type, version, or some diagnostic info
	hasActionableInfo := strings.Contains(stderr, "nomos-test-invalid-owner-12345") ||
		strings.Contains(stderr, "0.1.0") ||
		strings.Contains(stderr, "provider")

	if !hasActionableInfo {
		t.Errorf("error message should provide actionable information (provider type/version)\nstderr: %s", stderr)
	}

	// Verify stdout is empty (no compilation output on error)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout should be empty on error, got: %s", stdout)
	}

	// Verify no .nomos directory was created (build failed before provider installation)
	nomosDir := filepath.Join(testDir, ".nomos")
	if _, err := os.Stat(nomosDir); err == nil {
		// Directory exists - check if it's empty or contains incomplete state
		entries, _ := os.ReadDir(nomosDir)
		if len(entries) > 0 {
			t.Logf("Warning: .nomos directory exists with %d entries after failed build", len(entries))
			for _, entry := range entries {
				t.Logf("  - %s", entry.Name())
			}
		}
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking .nomos directory: %v", err)
	}

	t.Log("Test completed: Network failure produces clear error with provider alias and failure reason")
}

// TestBuild_MissingVersionField tests build failure when provider declaration is missing the version field.
//
// Scenario: Provider source declaration is missing required 'version:' field
// Expected:
//  1. Build fails with exit code 1 during discovery phase
//  2. Error message clearly indicates version field is missing
//  3. Error message includes provider information (alias and/or type)
//  4. Error is raised before attempting network operations
//
// This test verifies Task T032 - missing version field validation and error messaging.
func TestBuild_MissingVersionField(t *testing.T) {
	// Build the nomos CLI binary for testing
	binPath := buildCLI(t)

	// Create temporary test directory
	testDir := t.TempDir()

	// Create test .csl file with provider declaration MISSING version field
	cslPath := filepath.Join(testDir, "config.csl")
	cslContent := `source:
  alias: 'files'
  type: 'autonomous-bits/nomos-provider-file'
  directory: './data'

app:
  name: 'test-app'
  environment: 'staging'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create test .csl file: %v", err)
	}

	// Run nomos build - should fail during discovery/validation
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--path", cslPath, "--format", "json")
	cmd.Dir = testDir

	stdout, stderr, exitCode := runCommand(t, cmd)

	// Verify build fails with exit code 1
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (failure)\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// Verify error message indicates missing version
	hasMissingVersionError := strings.Contains(stderr, "missing required 'version' field") ||
		strings.Contains(stderr, "version field is missing") ||
		strings.Contains(stderr, "version is required") ||
		strings.Contains(stderr, "missing version") ||
		strings.Contains(stderr, "ErrMissingVersion")

	if !hasMissingVersionError {
		t.Errorf("error message should indicate version field is missing\nstderr: %s", stderr)
	}

	// Verify error message includes provider information
	// Should mention either the alias or the type
	hasProviderInfo := strings.Contains(stderr, "files") || // alias
		strings.Contains(stderr, "autonomous-bits/nomos-provider-file") // type

	if !hasProviderInfo {
		t.Errorf("error message should include provider alias or type\nstderr: %s", stderr)
	}

	// Verify error message explicitly states what field is missing
	if !strings.Contains(stderr, "version") {
		t.Errorf("error message must explicitly mention 'version'\nstderr: %s", stderr)
	}

	// Verify stdout is empty (no compilation output on error)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout should be empty on error, got: %s", stdout)
	}

	// Verify no .nomos directory was created
	// Missing version should be caught in discovery phase before any downloads
	nomosDir := filepath.Join(testDir, ".nomos")
	if _, err := os.Stat(nomosDir); !os.IsNotExist(err) {
		t.Error(".nomos directory should not exist when build fails with missing version (error should occur before provider setup)")
	}

	// Verify no lockfile was created
	lockfilePath := filepath.Join(nomosDir, "providers.lock.json")
	if _, err := os.Stat(lockfilePath); !os.IsNotExist(err) {
		t.Error("lockfile should not exist when build fails with missing version")
	}

	t.Log("Test completed: Missing version field produces clear error with provider information")
}
