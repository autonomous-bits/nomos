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

	"gopkg.in/yaml.v3"
)

// TestBuild_IncludeMetadataFlag_FileOutput tests that the --include-metadata flag
// causes metadata to be included in file output, and that without the flag,
// only data keys are written at the root level.
//
// T023: Integration test for CLI with --include-metadata flag (file output)
func TestBuild_IncludeMetadataFlag_FileOutput(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `app: "test-app"
version: "1.0.0"
environment: "dev"
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	tests := []struct {
		name       string
		format     string
		useFlag    bool
		verifyFunc func(t *testing.T, output []byte, useFlag bool)
	}{
		{
			name:    "json with --include-metadata",
			format:  "json",
			useFlag: true,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := json.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, output)
					return
				}

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("JSON output with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("JSON output with --include-metadata missing 'metadata' section")
					}

					// Verify data content is nested under "data" key
					dataSection, ok := parsed["data"].(map[string]any)
					if !ok {
						t.Error("'data' section is not a map")
						return
					}
					if dataSection["app"] != "test-app" {
						t.Errorf("expected app='test-app', got %v", dataSection["app"])
					}
				} else {
					// Without flag: expect data keys directly at root (no wrapping)
					if parsed["data"] != nil {
						t.Error("JSON output without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("JSON output without --include-metadata should not have 'metadata' wrapper")
					}

					// Verify data is directly at root
					if parsed["app"] != "test-app" {
						t.Errorf("expected app='test-app' at root, got %v", parsed["app"])
					}
					if parsed["version"] != "1.0.0" {
						t.Errorf("expected version='1.0.0' at root, got %v", parsed["version"])
					}
				}
			},
		},
		{
			name:    "json without --include-metadata (default)",
			format:  "json",
			useFlag: false,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := json.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, output)
					return
				}

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("JSON output with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("JSON output with --include-metadata missing 'metadata' section")
					}
				} else {
					// Without flag: expect data keys directly at root (no wrapping)
					if parsed["data"] != nil {
						t.Error("JSON output without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("JSON output without --include-metadata should not have 'metadata' wrapper")
					}

					// Verify data is directly at root
					if parsed["app"] != "test-app" {
						t.Errorf("expected app='test-app' at root, got %v", parsed["app"])
					}
					if parsed["version"] != "1.0.0" {
						t.Errorf("expected version='1.0.0' at root, got %v", parsed["version"])
					}
					if parsed["environment"] != "dev" {
						t.Errorf("expected environment='dev' at root, got %v", parsed["environment"])
					}
				}
			},
		},
		{
			name:    "yaml with --include-metadata",
			format:  "yaml",
			useFlag: true,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := yaml.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse YAML output: %v\nOutput: %s", err, output)
					return
				}

				outputStr := string(output)

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("YAML output with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("YAML output with --include-metadata missing 'metadata' section")
					}

					// Verify YAML syntax contains data: and metadata: keys
					if !strings.Contains(outputStr, "data:") {
						t.Error("YAML output with --include-metadata should contain 'data:' key")
					}
					if !strings.Contains(outputStr, "metadata:") {
						t.Error("YAML output with --include-metadata should contain 'metadata:' key")
					}
				} else {
					// Without flag: expect data keys directly at root
					if parsed["data"] != nil {
						t.Error("YAML output without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("YAML output without --include-metadata should not have 'metadata' wrapper")
					}

					// Verify data keys are directly at root
					if parsed["app"] != "test-app" {
						t.Errorf("expected app='test-app' at root, got %v", parsed["app"])
					}
				}
			},
		},
		{
			name:    "yaml without --include-metadata (default)",
			format:  "yaml",
			useFlag: false,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := yaml.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse YAML output: %v\nOutput: %s", err, output)
					return
				}

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("YAML output with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("YAML output with --include-metadata missing 'metadata' section")
					}
				} else {
					// Without flag: expect data keys directly at root
					if parsed["data"] != nil {
						t.Error("YAML output without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("YAML output without --include-metadata should not have 'metadata' wrapper")
					}

					// Verify data keys are directly at root
					if parsed["app"] != "test-app" {
						t.Errorf("expected app='test-app' at root, got %v", parsed["app"])
					}
					if parsed["version"] != "1.0.0" {
						t.Errorf("expected version='1.0.0' at root, got %v", parsed["version"])
					}
					if parsed["environment"] != "dev" {
						t.Errorf("expected environment='dev' at root, got %v", parsed["environment"])
					}
				}
			},
		},
		{
			name:    "tfvars with --include-metadata",
			format:  "tfvars",
			useFlag: true,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				outputStr := string(output)

				if useFlag {
					// tfvars format doesn't support nested structures like JSON/YAML
					// So with --include-metadata, we expect comment blocks or special handling
					// For now, verify the output contains expected HCL syntax
					if !strings.Contains(outputStr, "=") {
						t.Error("tfvars output missing '=' assignment syntax")
					}

					// Expect data keys are present
					if !strings.Contains(outputStr, "app") {
						t.Error("tfvars output missing 'app' variable")
					}

					// Note: tfvars format may not support metadata in the same way as JSON/YAML
					// This test documents expected behavior for tfvars + metadata flag
				} else {
					// Without flag: standard tfvars output
					if !strings.Contains(outputStr, "=") {
						t.Error("tfvars output missing '=' assignment syntax")
					}
					if !strings.Contains(outputStr, "app") {
						t.Error("tfvars output missing 'app' variable")
					}
				}
			},
		},
		{
			name:    "tfvars without --include-metadata (default)",
			format:  "tfvars",
			useFlag: false,
			verifyFunc: func(t *testing.T, output []byte, useFlag bool) {
				t.Helper()

				outputStr := string(output)

				// Standard tfvars output
				if !strings.Contains(outputStr, "=") {
					t.Error("tfvars output missing '=' assignment syntax")
				}
				if !strings.Contains(outputStr, "app") {
					t.Error("tfvars output missing 'app' variable")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+tt.name)

			// Build command with or without --include-metadata flag
			args := []string{"build", "-p", fixturePath, "-f", tt.format, "-o", outFile}
			if tt.useFlag {
				args = append(args, "--include-metadata")
			}

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			if exitCode != 0 {
				t.Fatalf("build command failed: exit code %d\nStdout: %s\nStderr: %s",
					exitCode, stdout, stderr)
			}

			// Determine expected output file with extension
			var expectedOutFile string
			switch tt.format {
			case "json":
				expectedOutFile = outFile + ".json"
			case "yaml":
				expectedOutFile = outFile + ".yaml"
			case "tfvars":
				expectedOutFile = outFile + ".tfvars"
			default:
				expectedOutFile = outFile
			}

			// Verify output file was created
			//nolint:gosec // G304: Reading test output file from controlled location
			content, err := os.ReadFile(expectedOutFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			// Verify format-specific structure
			tt.verifyFunc(t, content, tt.useFlag)
		})
	}
}

// TestBuild_IncludeMetadataFlag_StdoutOutput tests that the --include-metadata flag
// causes metadata to be included in stdout output when no output file is specified.
//
// T024: Integration test for CLI with --include-metadata flag (stdout output)
func TestBuild_IncludeMetadataFlag_StdoutOutput(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `app: "stdout-test"
version: "2.0.0"
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	tests := []struct {
		name       string
		format     string
		useFlag    bool
		verifyFunc func(t *testing.T, stdout string, useFlag bool)
	}{
		{
			name:    "json stdout with --include-metadata",
			format:  "json",
			useFlag: true,
			verifyFunc: func(t *testing.T, stdout string, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
					t.Errorf("failed to parse JSON stdout: %v\nStdout: %s", err, stdout)
					return
				}

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("JSON stdout with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("JSON stdout with --include-metadata missing 'metadata' section")
					}

					// Verify data content
					dataSection, ok := parsed["data"].(map[string]any)
					if !ok {
						t.Error("'data' section is not a map")
						return
					}
					if dataSection["app"] != "stdout-test" {
						t.Errorf("expected app='stdout-test', got %v", dataSection["app"])
					}
				} else {
					// Without flag: expect data keys directly at root
					if parsed["data"] != nil {
						t.Error("JSON stdout without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("JSON stdout without --include-metadata should not have 'metadata' wrapper")
					}

					// Verify data is directly at root
					if parsed["app"] != "stdout-test" {
						t.Errorf("expected app='stdout-test' at root, got %v", parsed["app"])
					}
					if parsed["version"] != "2.0.0" {
						t.Errorf("expected version='2.0.0' at root, got %v", parsed["version"])
					}
				}
			},
		},
		{
			name:    "json stdout without --include-metadata (default)",
			format:  "json",
			useFlag: false,
			verifyFunc: func(t *testing.T, stdout string, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
					t.Errorf("failed to parse JSON stdout: %v\nStdout: %s", err, stdout)
					return
				}

				// Without flag: expect data keys directly at root
				if parsed["data"] != nil {
					t.Error("JSON stdout without --include-metadata should not have 'data' wrapper")
				}
				if parsed["metadata"] != nil {
					t.Error("JSON stdout without --include-metadata should not have 'metadata' wrapper")
				}

				// Verify data is directly at root
				if parsed["app"] != "stdout-test" {
					t.Errorf("expected app='stdout-test' at root, got %v", parsed["app"])
				}
				if parsed["version"] != "2.0.0" {
					t.Errorf("expected version='2.0.0' at root, got %v", parsed["version"])
				}
			},
		},
		{
			name:    "yaml stdout with --include-metadata",
			format:  "yaml",
			useFlag: true,
			verifyFunc: func(t *testing.T, stdout string, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := yaml.Unmarshal([]byte(stdout), &parsed); err != nil {
					t.Errorf("failed to parse YAML stdout: %v\nStdout: %s", err, stdout)
					return
				}

				if useFlag {
					// With --include-metadata: expect wrapped structure
					if parsed["data"] == nil {
						t.Error("YAML stdout with --include-metadata missing 'data' section")
					}
					if parsed["metadata"] == nil {
						t.Error("YAML stdout with --include-metadata missing 'metadata' section")
					}

					// Verify YAML syntax
					if !strings.Contains(stdout, "data:") {
						t.Error("YAML stdout with --include-metadata should contain 'data:' key")
					}
					if !strings.Contains(stdout, "metadata:") {
						t.Error("YAML stdout with --include-metadata should contain 'metadata:' key")
					}
				} else {
					// Without flag: expect data keys directly at root
					if parsed["data"] != nil {
						t.Error("YAML stdout without --include-metadata should not have 'data' wrapper")
					}
					if parsed["metadata"] != nil {
						t.Error("YAML stdout without --include-metadata should not have 'metadata' wrapper")
					}
				}
			},
		},
		{
			name:    "yaml stdout without --include-metadata (default)",
			format:  "yaml",
			useFlag: false,
			verifyFunc: func(t *testing.T, stdout string, useFlag bool) {
				t.Helper()

				var parsed map[string]any
				if err := yaml.Unmarshal([]byte(stdout), &parsed); err != nil {
					t.Errorf("failed to parse YAML stdout: %v\nStdout: %s", err, stdout)
					return
				}

				// Without flag: expect data keys directly at root
				if parsed["data"] != nil {
					t.Error("YAML stdout without --include-metadata should not have 'data' wrapper")
				}
				if parsed["metadata"] != nil {
					t.Error("YAML stdout without --include-metadata should not have 'metadata' wrapper")
				}

				// Verify data keys are present at root
				if parsed["app"] != "stdout-test" {
					t.Errorf("expected app='stdout-test' at root, got %v", parsed["app"])
				}
				if parsed["version"] != "2.0.0" {
					t.Errorf("expected version='2.0.0' at root, got %v", parsed["version"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build command WITHOUT -o flag (output to stdout)
			args := []string{"build", "-p", fixturePath, "-f", tt.format}
			if tt.useFlag {
				args = append(args, "--include-metadata")
			}

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			if exitCode != 0 {
				t.Fatalf("build command failed: exit code %d\nStdout: %s\nStderr: %s",
					exitCode, stdout, stderr)
			}

			// Verify stdout contains output
			if strings.TrimSpace(stdout) == "" {
				t.Fatal("expected output on stdout, got empty string")
			}

			// Verify format-specific structure in stdout
			tt.verifyFunc(t, stdout, tt.useFlag)
		})
	}
}

// TestBuild_IncludeMetadataFlag_CompilationError tests that when compilation fails,
// no output file is created regardless of whether --include-metadata flag is used.
//
// T025: Integration test for compilation failure with --include-metadata
func TestBuild_IncludeMetadataFlag_CompilationError(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture with syntax error
	tmpDir := t.TempDir()
	invalidFixturePath := filepath.Join(tmpDir, "invalid.csl")
	invalidContent := `this is not valid Nomos syntax {{{ ]]] syntax error
app: "broken
version: missing quote
`
	//nolint:gosec // G306: Test file with non-sensitive content (intentionally invalid)
	if err := os.WriteFile(invalidFixturePath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to create invalid fixture: %v", err)
	}

	tests := []struct {
		name    string
		format  string
		useFlag bool
	}{
		{
			name:    "json with --include-metadata (compilation error)",
			format:  "json",
			useFlag: true,
		},
		{
			name:    "json without flag (compilation error)",
			format:  "json",
			useFlag: false,
		},
		{
			name:    "yaml with --include-metadata (compilation error)",
			format:  "yaml",
			useFlag: true,
		},
		{
			name:    "yaml without flag (compilation error)",
			format:  "yaml",
			useFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-error-"+tt.name)

			// Build command with or without --include-metadata flag
			args := []string{"build", "-p", invalidFixturePath, "-f", tt.format, "-o", outFile}
			if tt.useFlag {
				args = append(args, "--include-metadata")
			}

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify command failed with non-zero exit code
			if exitCode == 0 {
				t.Errorf("expected non-zero exit code for invalid syntax, got 0\nStdout: %s\nStderr: %s",
					stdout, stderr)
			}

			// Verify exit code is 1 (compilation error)
			if exitCode != 1 {
				t.Errorf("expected exit code 1 for compilation error, got %d\nStderr: %s",
					exitCode, stderr)
			}

			// Verify error message on stderr
			if strings.TrimSpace(stderr) == "" {
				t.Error("expected error message on stderr, got empty string")
			}

			// Verify no output file was created (with any extension)
			possibleFiles := []string{
				outFile,
				outFile + ".json",
				outFile + ".yaml",
				outFile + ".tfvars",
			}

			for _, file := range possibleFiles {
				if _, err := os.Stat(file); err == nil {
					t.Errorf("output file %s should not be created on compilation error", file)
				}
			}
		})
	}
}

// TestBuild_IncludeMetadataFlag_NonexistentFile tests that when the input file
// doesn't exist, the --include-metadata flag doesn't affect error handling.
//
// T025: Integration test for compilation failure with --include-metadata
func TestBuild_IncludeMetadataFlag_NonexistentFile(t *testing.T) {
	binPath := buildCLI(t)

	tmpDir := t.TempDir()
	nonexistentFile := filepath.Join(tmpDir, "nonexistent.csl")

	tests := []struct {
		name    string
		useFlag bool
	}{
		{
			name:    "with --include-metadata",
			useFlag: true,
		},
		{
			name:    "without --include-metadata",
			useFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-nonexistent-"+tt.name)

			// Build command
			args := []string{"build", "-p", nonexistentFile, "-f", "json", "-o", outFile}
			if tt.useFlag {
				args = append(args, "--include-metadata")
			}

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			// Verify command failed
			if exitCode == 0 {
				t.Errorf("expected non-zero exit code for nonexistent file, got 0\nStdout: %s\nStderr: %s",
					stdout, stderr)
			}

			// Verify error message mentions the file issue
			combinedOutput := stdout + stderr
			if !strings.Contains(strings.ToLower(combinedOutput), "not found") &&
				!strings.Contains(strings.ToLower(combinedOutput), "no such file") &&
				!strings.Contains(strings.ToLower(combinedOutput), "does not exist") &&
				!strings.Contains(strings.ToLower(combinedOutput), "stat") {
				t.Errorf("error message should mention file not found\nStdout: %s\nStderr: %s",
					stdout, stderr)
			}

			// Verify no output file was created
			if _, err := os.Stat(outFile + ".json"); err == nil {
				t.Error("output file should not be created on compilation error")
			}
		})
	}
}
