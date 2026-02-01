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

// TestExtensionHandling_Integration verifies end-to-end file extension handling
// by building and executing the actual CLI binary with various format flags.
func TestExtensionHandling_Integration(t *testing.T) {
	binPath := buildCLI(t)

	tests := []struct {
		name          string
		format        string
		outputPath    string
		expectedFile  string // relative to temp dir
		verifyContent bool
		expectError   bool
	}{
		// Auto-append extension scenarios (T043)
		{
			name:         "json format auto-appends .json extension",
			format:       "json",
			outputPath:   "output",
			expectedFile: "output.json",
		},
		{
			name:         "yaml format auto-appends .yaml extension",
			format:       "yaml",
			outputPath:   "output",
			expectedFile: "output.yaml",
		},
		{
			name:         "tfvars format auto-appends .tfvars extension",
			format:       "tfvars",
			outputPath:   "terraform",
			expectedFile: "terraform.tfvars",
		},

		// Explicit extension preserved scenarios (T044)
		{
			name:         "explicit .yml extension preserved with yaml format",
			format:       "yaml",
			outputPath:   "config.yml",
			expectedFile: "config.yml",
		},
		{
			name:         "explicit .yaml extension preserved with yaml format",
			format:       "yaml",
			outputPath:   "config.yaml",
			expectedFile: "config.yaml",
		},
		{
			name:         "explicit .json extension preserved with json format",
			format:       "json",
			outputPath:   "data.json",
			expectedFile: "data.json",
		},
		{
			name:         "explicit .tfvars extension preserved",
			format:       "tfvars",
			outputPath:   "vars.tfvars",
			expectedFile: "vars.tfvars",
		},

		// Extension mismatch scenarios (T045)
		{
			name:          "yaml format with .json extension preserves user choice",
			format:        "yaml",
			outputPath:    "data.json",
			expectedFile:  "data.json",
			verifyContent: true, // should contain YAML despite .json extension
		},
		{
			name:          "json format with .yaml extension preserves user choice",
			format:        "json",
			outputPath:    "config.yaml",
			expectedFile:  "config.yaml",
			verifyContent: true, // should contain JSON despite .yaml extension
		},
		{
			name:         "tfvars format with .yml extension preserves user choice",
			format:       "tfvars",
			outputPath:   "vars.yml",
			expectedFile: "vars.yml",
		},

		// Nested paths with auto-append
		{
			name:         "nested path with yaml auto-append",
			format:       "yaml",
			outputPath:   "configs/prod/app",
			expectedFile: "configs/prod/app.yaml",
		},
		{
			name:         "nested path with tfvars auto-append",
			format:       "tfvars",
			outputPath:   "environments/staging/terraform",
			expectedFile: "environments/staging/terraform.tfvars",
		},

		// Nested paths with explicit extensions
		{
			name:         "nested path with explicit extension",
			format:       "json",
			outputPath:   "output/deployment/config.json",
			expectedFile: "output/deployment/config.json",
		},

		// Edge cases
		{
			name:         "filename with multiple dots auto-appends",
			format:       "yaml",
			outputPath:   "app.v1.config",
			expectedFile: "app.v1.config.yaml",
		},
		{
			name:         "filename with multiple dots and explicit extension",
			format:       "json",
			outputPath:   "app.v2.config.json",
			expectedFile: "app.v2.config.json",
		},
		{
			name:         "hidden file with auto-append",
			format:       "yaml",
			outputPath:   ".config",
			expectedFile: ".config.yaml",
		},
		{
			name:         "hidden file with explicit extension",
			format:       "json",
			outputPath:   ".settings.json",
			expectedFile: ".settings.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fixtureFile := createTestFixture(t, tmpDir)

			outputPath := filepath.Join(tmpDir, tt.outputPath)
			expectedPath := filepath.Join(tmpDir, tt.expectedFile)

			args := []string{
				"build",
				"-p", fixtureFile,
				"--format", tt.format,
				"-o", outputPath,
			}

			cmd := exec.Command(binPath, args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil\nOutput: %s", output)
				}
				return
			}

			if err != nil {
				t.Fatalf("command failed: %v\nOutput: %s", err, output)
			}

			// Verify expected file exists
			stat, err := os.Stat(expectedPath)
			if err != nil {
				t.Errorf("expected file %q not found: %v", expectedPath, err)
				// List directory contents for debugging
				listDirContents(t, tmpDir)
				return
			}

			if stat.IsDir() {
				t.Errorf("expected file %q is a directory", expectedPath)
			}

			// Verify no incorrect file was created (e.g., output instead of output.yaml)
			if tt.outputPath != tt.expectedFile {
				incorrectPath := filepath.Join(tmpDir, tt.outputPath)
				if stat, err := os.Stat(incorrectPath); err == nil && !stat.IsDir() {
					t.Errorf("unexpected file created at %q, expected only %q", incorrectPath, expectedPath)
				}
			}

			// Verify content format matches if requested
			if tt.verifyContent {
				content, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}

				verifyContentFormat(t, content, tt.format)
			}
		})
	}
}

// TestExtensionHandling_FormatValidation verifies that invalid formats are rejected.
func TestExtensionHandling_FormatValidation(t *testing.T) {
	binPath := buildCLI(t)

	tests := []struct {
		name           string
		format         string
		wantErrContain string
	}{
		{
			name:           "invalid format rejected",
			format:         "xml",
			wantErrContain: "unsupported format",
		},
		{
			name:           "empty format rejected",
			format:         "",
			wantErrContain: "format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fixtureFile := createTestFixture(t, tmpDir)
			outputPath := filepath.Join(tmpDir, "output")

			args := []string{
				"build",
				"-p", fixtureFile,
				"--format", tt.format,
				"-o", outputPath,
			}

			cmd := exec.Command(binPath, args...)
			output, err := cmd.CombinedOutput()

			if err == nil {
				t.Fatalf("expected error for format %q, got nil\nOutput: %s", tt.format, output)
			}

			if tt.wantErrContain != "" {
				outputStr := string(output)
				if !strings.Contains(strings.ToLower(outputStr), strings.ToLower(tt.wantErrContain)) {
					t.Errorf("error output missing expected text %q\nGot: %s", tt.wantErrContain, outputStr)
				}
			}
		})
	}
}

// TestExtensionHandling_DirectoryCreation verifies that nested directories
// are created correctly when the output path doesn't exist.
func TestExtensionHandling_DirectoryCreation(t *testing.T) {
	binPath := buildCLI(t)

	tests := []struct {
		name         string
		format       string
		outputPath   string
		expectedFile string
	}{
		{
			name:         "single nested directory created",
			format:       "yaml",
			outputPath:   "configs/app",
			expectedFile: "configs/app.yaml",
		},
		{
			name:         "multiple nested directories created",
			format:       "json",
			outputPath:   "a/b/c/d/config",
			expectedFile: "a/b/c/d/config.json",
		},
		{
			name:         "nested directory with explicit extension",
			format:       "tfvars",
			outputPath:   "environments/prod/terraform.tfvars",
			expectedFile: "environments/prod/terraform.tfvars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fixtureFile := createTestFixture(t, tmpDir)

			outputPath := filepath.Join(tmpDir, tt.outputPath)
			expectedPath := filepath.Join(tmpDir, tt.expectedFile)

			args := []string{
				"build",
				"-p", fixtureFile,
				"--format", tt.format,
				"-o", outputPath,
			}

			cmd := exec.Command(binPath, args...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Fatalf("command failed: %v\nOutput: %s", err, output)
			}

			// Verify file exists at expected path
			if _, err := os.Stat(expectedPath); err != nil {
				t.Errorf("expected file %q not found: %v", expectedPath, err)
				listDirContents(t, tmpDir)
			}

			// Verify all intermediate directories were created
			dir := filepath.Dir(expectedPath)
			stat, err := os.Stat(dir)
			if err != nil {
				t.Errorf("expected directory %q not found: %v", dir, err)
			} else if !stat.IsDir() {
				t.Errorf("expected %q to be a directory", dir)
			}
		})
	}
}

// TestExtensionHandling_DefaultFormat verifies behavior when format flag is omitted.
func TestExtensionHandling_DefaultFormat(t *testing.T) {
	binPath := buildCLI(t)

	tmpDir := t.TempDir()
	fixtureFile := createTestFixture(t, tmpDir)
	outputPath := filepath.Join(tmpDir, "output")
	expectedPath := filepath.Join(tmpDir, "output.json") // Default is JSON

	args := []string{
		"build",
		"-p", fixtureFile,
		"-o", outputPath,
		// No --format flag
	}

	cmd := exec.Command(binPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("command failed: %v\nOutput: %s", err, output)
	}

	// Verify .json file was created (default format)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected file %q not found (default format should be json): %v", expectedPath, err)
		listDirContents(t, tmpDir)
	}
}

// listDirContents lists all files in a directory recursively for debugging.
func listDirContents(t *testing.T, dir string) {
	t.Helper()

	t.Logf("Directory contents of %s:", dir)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(dir, path)
		if info.IsDir() {
			t.Logf("  [DIR]  %s", relPath)
		} else {
			t.Logf("  [FILE] %s (%d bytes)", relPath, info.Size())
		}
		return nil
	})
	if err != nil {
		t.Logf("Failed to list directory: %v", err)
	}
}

// verifyContentFormat checks that file content matches the expected format.
func verifyContentFormat(t *testing.T, content []byte, format string) {
	t.Helper()

	contentStr := string(content)

	switch format {
	case "json":
		// JSON should start with { or [
		trimmed := strings.TrimSpace(contentStr)
		if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
			t.Errorf("expected JSON format but content doesn't start with { or [\nContent: %s", contentStr)
		}

	case "yaml":
		// YAML typically has key: value format
		// Should not start with { (which would indicate JSON)
		trimmed := strings.TrimSpace(contentStr)
		if strings.HasPrefix(trimmed, "{") {
			t.Errorf("expected YAML format but content looks like JSON\nContent: %s", contentStr)
		}
		// YAML should contain ': ' pattern
		if !strings.Contains(contentStr, ": ") {
			t.Errorf("expected YAML format with 'key: value' pattern\nContent: %s", contentStr)
		}

	case "tfvars":
		// Tfvars should have key = value format
		if !strings.Contains(contentStr, " = ") && !strings.Contains(contentStr, "=") {
			t.Errorf("expected tfvars format with 'key = value' pattern\nContent: %s", contentStr)
		}
		// Should not start with { (JSON) or contain ': ' (YAML)
		trimmed := strings.TrimSpace(contentStr)
		if strings.HasPrefix(trimmed, "{") {
			t.Errorf("expected tfvars format but content looks like JSON\nContent: %s", contentStr)
		}
	}
}

// createTestFixture creates a minimal test configuration file in the specified directory.
// Returns the path to the created fixture file.
func createTestFixture(t *testing.T, dir string) string {
	t.Helper()

	fixturePath := filepath.Join(dir, "test.csl")
	content := `config:
  test: value
  number: 42
  nested:
    key: data
`

	if err := os.WriteFile(fixturePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	return fixturePath
}
