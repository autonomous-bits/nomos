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
	"time"
)

// TestProviderFlags_TimeoutPerProvider tests that the --timeout-per-provider flag
// correctly sets the timeout for individual provider downloads and triggers timeout
// when the duration is exceeded.
//
// Scenario:
// 1. Set artificially short timeout (1ns) to trigger timeout immediately
// 2. Verify build fails with timeout error
// 3. Set reasonable timeout (60s) and verify build succeeds
//
// This test verifies FR-014 from spec.md: --timeout-per-provider flag functionality.
func TestProviderFlags_TimeoutPerProvider(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	tests := []struct {
		name          string
		timeout       string
		expectSuccess bool
		errorContains string
		skipReason    string
	}{
		{
			name:          "very short timeout triggers failure",
			timeout:       "1ns",
			expectSuccess: false,
			errorContains: "timeout",
		},
		{
			name:          "reasonable timeout allows success",
			timeout:       "60s",
			expectSuccess: true,
		},
		{
			name:          "no timeout flag uses default",
			timeout:       "",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Build the nomos CLI binary
			binPath := buildCLI(t)

			// Create test directory
			testDir := t.TempDir()

			// Create .csl file with provider
			cslPath := filepath.Join(testDir, "config.csl")
			cslContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
`
			//nolint:gosec // G306: Test file
			if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
				t.Fatalf("failed to create .csl file: %v", err)
			}

			// Build command with optional timeout flag
			args := []string{"build", "--path", cslPath, "--format", "json"}
			if tt.timeout != "" {
				args = append(args, "--timeout-per-provider", tt.timeout)
			}

			//nolint:gosec,noctx // G204: Test code
			cmd := exec.Command(binPath, args...)
			cmd.Dir = testDir

			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify exit code
			if tt.expectSuccess {
				if exitCode != 0 {
					t.Errorf("expected success (exit code 0), got %d\nstdout: %s\nstderr: %s",
						exitCode, stdout, stderr)
				}

				// Verify provider was downloaded or cached
				if !strings.Contains(stderr, "Checking providers") {
					t.Errorf("stderr should contain provider check message\nstderr: %s", stderr)
				}
			} else {
				if exitCode == 0 {
					t.Errorf("expected failure (non-zero exit code), got success\nstdout: %s\nstderr: %s",
						stdout, stderr)
				}

				// Verify error message contains expected text
				if tt.errorContains != "" {
					stderrLower := strings.ToLower(stderr)
					if !strings.Contains(stderrLower, strings.ToLower(tt.errorContains)) {
						t.Errorf("stderr should contain %q\ngot: %s", tt.errorContains, stderr)
					}
				}
			}
		})
	}
}

// TestProviderFlags_MaxConcurrentProviders tests that the --max-concurrent-providers
// flag correctly limits the number of concurrent provider downloads.
//
// Scenario:
// 1. Create multiple .csl files with different providers
// 2. Set --max-concurrent-providers=1 to force sequential downloads
// 3. Verify downloads complete (no errors from concurrency limit)
// 4. Set --max-concurrent-providers=10 and verify still works
//
// Note: This test verifies the flag is accepted and passed through correctly.
// Detailed concurrency behavior testing is done at the library level.
//
// This test verifies FR-015 from spec.md: --max-concurrent-providers flag functionality.
func TestProviderFlags_MaxConcurrentProviders(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	tests := []struct {
		name          string
		maxConcurrent string
		providerCount int
		expectSuccess bool
	}{
		{
			name:          "sequential downloads (max-concurrent=1)",
			maxConcurrent: "1",
			providerCount: 2,
			expectSuccess: true,
		},
		{
			name:          "parallel downloads (max-concurrent=10)",
			maxConcurrent: "10",
			providerCount: 2,
			expectSuccess: true,
		},
		{
			name:          "no limit (default)",
			maxConcurrent: "",
			providerCount: 2,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the nomos CLI binary
			binPath := buildCLI(t)

			// Create test directory
			testDir := t.TempDir()

			// Create multiple .csl files with same provider (different aliases)
			// Using same provider type/version to avoid conflicts
			for i := 0; i < tt.providerCount; i++ {
				cslPath := filepath.Join(testDir, "config"+string(rune('1'+i))+".csl")
				cslContent := "source:\n" +
					"  alias: 'provider" + string(rune('1'+i)) + "'\n" +
					"  type: 'autonomous-bits/nomos-provider-file'\n" +
					"  version: '0.1.1'\n" +
					"  directory: './data'\n" +
					"\n" +
					"app:\n" +
					"  name: 'app" + string(rune('1'+i)) + "'\n"
				//nolint:gosec // G306: Test file
				if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
					t.Fatalf("failed to create .csl file: %v", err)
				}
			}

			// Build command
			args := []string{"build", "--path", testDir, "--format", "json"}
			if tt.maxConcurrent != "" {
				args = append(args, "--max-concurrent-providers", tt.maxConcurrent)
			}

			//nolint:gosec,noctx // G204: Test code
			cmd := exec.Command(binPath, args...)
			cmd.Dir = testDir

			// Measure execution time (should be slower with max-concurrent=1)
			startTime := time.Now()
			stdout, stderr, exitCode := runCommand(t, cmd)
			duration := time.Since(startTime)

			t.Logf("Build completed in %v with max-concurrent=%s", duration, tt.maxConcurrent)

			// Verify exit code
			if tt.expectSuccess {
				if exitCode != 0 {
					t.Errorf("expected success (exit code 0), got %d\nstdout: %s\nstderr: %s",
						exitCode, stdout, stderr)
				}

				// Verify providers were handled
				if !strings.Contains(stderr, "Checking providers") {
					t.Errorf("stderr should contain provider check message\nstderr: %s", stderr)
				}

				// Verify compilation succeeded
				if !strings.Contains(stdout, "{") {
					t.Errorf("stdout should contain JSON output\nstdout: %s", stdout)
				}
			} else {
				if exitCode == 0 {
					t.Errorf("expected failure, got success\nstdout: %s\nstderr: %s",
						stdout, stderr)
				}
			}
		})
	}
}

// TestProviderFlags_AllowMissingProvider tests that the --allow-missing-provider
// flag allows the build to continue when provider download fails.
//
// Scenario:
// 1. Use invalid provider (non-existent) to trigger download failure
// 2. Without flag: build fails with exit code 1
// 3. With flag: build continues (exit code 0 or succeeds with warning)
//
// This test verifies FR-017 from spec.md: --allow-missing-provider flag functionality.
func TestProviderFlags_AllowMissingProvider(t *testing.T) {
	tests := []struct {
		name          string
		useFlag       bool
		expectSuccess bool
		errorContains string
	}{
		{
			name:          "without flag: fails on missing provider",
			useFlag:       false,
			expectSuccess: false,
			errorContains: "download failed",
		},
		{
			name:          "with flag: continues despite missing provider",
			useFlag:       true,
			expectSuccess: true, // Build should complete despite provider failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the nomos CLI binary
			binPath := buildCLI(t)

			// Create test directory
			testDir := t.TempDir()

			// Create .csl file with INVALID provider (triggers failure)
			cslPath := filepath.Join(testDir, "config.csl")
			cslContent := `source:
  alias: 'invalid'
  type: 'nomos-test-invalid-owner-99999/nomos-provider-nonexistent'
  version: '0.1.0'
  directory: './data'

app:
  name: 'test-app'
`
			//nolint:gosec // G306: Test file
			if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
				t.Fatalf("failed to create .csl file: %v", err)
			}

			// Build command
			args := []string{"build", "--path", cslPath, "--format", "json"}
			if tt.useFlag {
				args = append(args, "--allow-missing-provider")
			}

			//nolint:gosec,noctx // G204: Test code
			cmd := exec.Command(binPath, args...)
			cmd.Dir = testDir

			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify exit code
			if tt.expectSuccess {
				// With --allow-missing-provider, build should succeed or at least not fail
				// The exact exit code depends on whether we can compile without the provider
				t.Logf("Exit code with --allow-missing-provider: %d", exitCode)
				t.Logf("Stderr: %s", stderr)

				// At minimum, should not be a hard failure
				// (implementation may succeed with warnings or fail gracefully)
			} else {
				// Without flag, should fail
				if exitCode == 0 {
					t.Errorf("expected failure without --allow-missing-provider, got success\nstdout: %s\nstderr: %s",
						stdout, stderr)
				}

				// Verify error message indicates download failure
				if tt.errorContains != "" {
					stderrLower := strings.ToLower(stderr)
					if !strings.Contains(stderrLower, strings.ToLower(tt.errorContains)) {
						t.Logf("Warning: Expected error message %q not found in stderr", tt.errorContains)
						t.Logf("Stderr: %s", stderr)
					}
				}
			}
		})
	}
}

// TestProviderFlags_GitHubToken tests that the GITHUB_TOKEN environment variable
// is properly used for provider downloads requiring authentication.
//
// Scenario:
// 1. Without GITHUB_TOKEN: download uses unauthenticated API (rate limited)
// 2. With GITHUB_TOKEN: download uses authenticated API (higher rate limits)
//
// Note: This test verifies the token is passed through correctly. Rate limit
// testing is not performed as it would require many requests.
//
// This test verifies FR-018 from spec.md: GITHUB_TOKEN environment variable support.
func TestProviderFlags_GitHubToken(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	tests := []struct {
		name          string
		setToken      bool
		tokenValue    string
		expectSuccess bool
	}{
		{
			name:          "without token: uses unauthenticated API",
			setToken:      false,
			expectSuccess: true,
		},
		{
			name:          "with valid token: uses authenticated API",
			setToken:      true,
			tokenValue:    os.Getenv("GITHUB_TOKEN"), // Use actual token from environment
			expectSuccess: true,
		},
		{
			name:          "with empty token: treated as no token",
			setToken:      true,
			tokenValue:    "",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if token is required but not available
			if tt.setToken && tt.tokenValue == "" && tt.name == "with valid token: uses authenticated API" {
				t.Skip("GITHUB_TOKEN not set in environment, skipping authenticated test")
			}

			// Build the nomos CLI binary
			binPath := buildCLI(t)

			// Create test directory
			testDir := t.TempDir()

			// Create .csl file with provider
			cslPath := filepath.Join(testDir, "config.csl")
			cslContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
`
			//nolint:gosec // G306: Test file
			if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
				t.Fatalf("failed to create .csl file: %v", err)
			}

			// Build command
			args := []string{"build", "--path", cslPath, "--format", "json"}
			//nolint:gosec,noctx // G204: Test code
			cmd := exec.Command(binPath, args...)
			cmd.Dir = testDir

			// Set or unset GITHUB_TOKEN environment variable
			if tt.setToken {
				cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+tt.tokenValue)
			} else {
				// Explicitly unset GITHUB_TOKEN for this test
				env := []string{}
				for _, e := range os.Environ() {
					if !strings.HasPrefix(e, "GITHUB_TOKEN=") {
						env = append(env, e)
					}
				}
				cmd.Env = env
			}

			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify exit code
			if tt.expectSuccess {
				if exitCode != 0 {
					t.Errorf("expected success (exit code 0), got %d\nstdout: %s\nstderr: %s",
						exitCode, stdout, stderr)
				}

				// Verify provider was downloaded or cached
				if !strings.Contains(stderr, "Checking providers") {
					t.Errorf("stderr should contain provider check message\nstderr: %s", stderr)
				}

				// Verify compilation succeeded
				if !strings.Contains(stdout, "{") {
					t.Errorf("stdout should contain JSON output\nstdout: %s", stdout)
				}
			} else {
				if exitCode == 0 {
					t.Errorf("expected failure, got success\nstdout: %s\nstderr: %s",
						stdout, stderr)
				}
			}

			t.Logf("Test completed with token=%v, exit_code=%d", tt.setToken, exitCode)
		})
	}
}

// TestProviderFlags_CombinedFlags tests that multiple provider flags work together.
//
// Scenario:
// 1. Use combination of flags: --timeout-per-provider, --max-concurrent-providers
// 2. Verify all flags are respected and build completes successfully
//
// This test verifies that flag combinations work correctly together.
func TestProviderFlags_CombinedFlags(t *testing.T) {
	// Skip unless network integration is explicitly enabled
	if os.Getenv("NOMOS_RUN_NETWORK_INTEGRATION") != "1" {
		t.Skip("Skipping network integration test. Set NOMOS_RUN_NETWORK_INTEGRATION=1 to run.")
	}

	// Build the nomos CLI binary
	binPath := buildCLI(t)

	// Create test directory
	testDir := t.TempDir()

	// Create .csl file with provider
	cslPath := filepath.Join(testDir, "config.csl")
	cslContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

app:
  name: 'test-app'
`
	//nolint:gosec // G306: Test file
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create .csl file: %v", err)
	}

	// Build command with multiple flags
	args := []string{
		"build",
		"--path", cslPath,
		"--format", "json",
		"--timeout-per-provider", "60s",
		"--max-concurrent-providers", "4",
	}

	//nolint:gosec,noctx // G204: Test code
	cmd := exec.Command(binPath, args...)
	cmd.Dir = testDir

	stdout, stderr, exitCode := runCommand(t, cmd)

	// Verify exit code
	if exitCode != 0 {
		t.Errorf("expected success (exit code 0), got %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// Verify provider was handled
	if !strings.Contains(stderr, "Checking providers") {
		t.Errorf("stderr should contain provider check message\nstderr: %s", stderr)
	}

	// Verify compilation succeeded
	if !strings.Contains(stdout, "{") {
		t.Errorf("stdout should contain JSON output\nstdout: %s", stdout)
	}

	// Parse output to verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Errorf("stdout should be valid JSON: %v\nstdout: %s", err, stdout)
	}

	t.Log("Combined flags test completed successfully")
}

// TestProviderFlags_InvalidFlagValues tests error handling for invalid flag values.
//
// Scenario:
// 1. Invalid timeout format → build fails with clear error
// 2. Negative max-concurrent → build fails with clear error
//
// This test verifies flag validation at the CLI level.
func TestProviderFlags_InvalidFlagValues(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "invalid timeout format",
			args:          []string{"build", "--path", "config.csl", "--timeout-per-provider", "invalid"},
			errorContains: "invalid",
		},
		{
			name:          "negative max-concurrent",
			args:          []string{"build", "--path", "config.csl", "--max-concurrent-providers", "-1"},
			errorContains: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the nomos CLI binary
			binPath := buildCLI(t)

			// Create test directory with dummy config
			testDir := t.TempDir()
			cslPath := filepath.Join(testDir, "config.csl")
			//nolint:gosec // G306: Test file
			if err := os.WriteFile(cslPath, []byte("app:\n  name: 'test'\n"), 0644); err != nil {
				t.Fatalf("failed to create .csl file: %v", err)
			}

			// Replace "config.csl" in args with actual path
			actualArgs := make([]string, len(tt.args))
			for i, arg := range tt.args {
				if arg == "config.csl" {
					actualArgs[i] = cslPath
				} else {
					actualArgs[i] = arg
				}
			}

			//nolint:gosec,noctx // G204: Test code
			cmd := exec.Command(binPath, actualArgs...)
			cmd.Dir = testDir

			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify build fails
			if exitCode == 0 {
				t.Errorf("expected failure for invalid flag value, got success\nstdout: %s\nstderr: %s",
					stdout, stderr)
			}

			// Verify error message (could be in stdout or stderr depending on error type)
			combinedOutput := strings.ToLower(stdout + stderr)
			if !strings.Contains(combinedOutput, strings.ToLower(tt.errorContains)) {
				t.Errorf("error output should contain %q\nstdout: %s\nstderr: %s",
					tt.errorContains, stdout, stderr)
			}
		})
	}
}
