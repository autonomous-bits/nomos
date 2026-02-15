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

// TestFormatFlag_AllFormats tests that the --format flag works correctly for all
// supported output formats (json, yaml, tfvars) using the same input fixture.
//
// T037: Integration test for format flag variations
func TestFormatFlag_AllFormats(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `region: "us-west-2"

database:
  engine: "postgres"
  port: 5432
  multi_az: false

vpc:
  cidr: "10.0.0.0/16"
  enable_dns: true
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	tests := []struct {
		name       string
		format     string
		verifyFunc func(t *testing.T, output []byte)
	}{
		{
			name:   "json format",
			format: "json",
			verifyFunc: func(t *testing.T, output []byte) {
				t.Helper()

				var parsed map[string]any
				if err := json.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, output)
					return
				}

				// Verify expected structure
				if parsed["data"] == nil {
					t.Error("JSON output missing 'data' section")
				}
				if parsed["metadata"] == nil {
					t.Error("JSON output missing 'metadata' section")
				}

				// Verify it's actually JSON format (not YAML or tfvars)
				outputStr := strings.TrimSpace(string(output))
				if !strings.HasPrefix(outputStr, "{") {
					t.Error("JSON output should start with '{'")
				}
			},
		},
		{
			name:   "yaml format",
			format: "yaml",
			verifyFunc: func(t *testing.T, output []byte) {
				t.Helper()

				var parsed map[string]any
				if err := yaml.Unmarshal(output, &parsed); err != nil {
					t.Errorf("failed to parse YAML output: %v\nOutput: %s", err, output)
					return
				}

				// Verify expected structure
				if parsed["data"] == nil {
					t.Error("YAML output missing 'data' section")
				}
				if parsed["metadata"] == nil {
					t.Error("YAML output missing 'metadata' section")
				}

				// Verify it's YAML format (not JSON)
				outputStr := strings.TrimSpace(string(output))
				if strings.HasPrefix(outputStr, "{") {
					t.Error("YAML output should not start with '{' (appears to be JSON)")
				}

				// Verify YAML-specific syntax
				if !strings.Contains(outputStr, "data:") || !strings.Contains(outputStr, "metadata:") {
					t.Error("YAML output missing expected YAML syntax (key: value)")
				}
			},
		},
		{
			name:   "tfvars format",
			format: "tfvars",
			verifyFunc: func(t *testing.T, output []byte) {
				t.Helper()

				outputStr := string(output)

				// Verify tfvars syntax (key = value)
				if !strings.Contains(outputStr, "=") {
					t.Error("tfvars output missing '=' assignment syntax")
				}

				// Verify it's not JSON or YAML
				trimmed := strings.TrimSpace(outputStr)
				if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "data:") {
					t.Error("tfvars output appears to be JSON or YAML format")
				}

				// Verify HCL-style comments or structure
				// tfvars typically has key = value pairs
				if !strings.Contains(outputStr, "\n") {
					t.Error("tfvars output should have multiple lines")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+tt.format)

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", tt.format, "-o", outFile, "--include-metadata")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build command failed: %v\nOutput: %s", err, output)
			}

			// The CLI auto-appends extensions when not provided
			// Determine expected extension based on format
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
			tt.verifyFunc(t, content)
		})
	}
}

// TestFormatFlag_InvalidFormat tests error handling when an invalid format
// is specified via the --format flag.
//
// T037: Integration test for format flag variations
func TestFormatFlag_InvalidFormat(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `config:
  name: "test"
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	tests := []struct {
		name          string
		invalidFormat string
	}{
		{
			name:          "xml format",
			invalidFormat: "xml",
		},
		{
			name:          "invalid format",
			invalidFormat: "invalid",
		},
		{
			name:          "toml format",
			invalidFormat: "toml",
		},
		{
			name:          "empty format",
			invalidFormat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+tt.invalidFormat)

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", tt.invalidFormat, "-o", outFile)
			stdout, stderr, exitCode := runCommand(t, cmd)

			// Should fail with exit code 1 (error) or 2 (usage error)
			if exitCode == 0 {
				t.Errorf("expected non-zero exit code for invalid format %q, got 0\nStdout: %s\nStderr: %s",
					tt.invalidFormat, stdout, stderr)
			}

			// Verify error message mentions supported formats
			combinedOutput := stdout + stderr
			supportedFormats := []string{"json", "yaml", "tfvars"}
			foundSupportedFormat := false
			for _, format := range supportedFormats {
				if strings.Contains(strings.ToLower(combinedOutput), format) {
					foundSupportedFormat = true
					break
				}
			}

			if !foundSupportedFormat {
				t.Errorf("error message should mention supported formats (json, yaml, tfvars)\nStdout: %s\nStderr: %s",
					stdout, stderr)
			}

			// Verify output file was not created
			if _, err := os.Stat(outFile); err == nil {
				t.Error("output file should not be created for invalid format")
			}
		})
	}
}

// TestFormatFlag_CaseMixing tests that the --format flag accepts case-insensitive
// values (e.g., json/JSON/Json, yaml/YAML/Yaml, tfvars/TFVARS/TfVars).
//
// T037: Integration test for format flag variations
func TestFormatFlag_CaseMixing(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `config:
  name: "test"
  value: 42
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	tests := []struct {
		name         string
		formatValue  string
		expectedType string // "json", "yaml", or "tfvars"
	}{
		// JSON variations
		{name: "json lowercase", formatValue: "json", expectedType: "json"},
		{name: "JSON uppercase", formatValue: "JSON", expectedType: "json"},
		{name: "Json capitalized", formatValue: "Json", expectedType: "json"},
		{name: "JsOn mixed case", formatValue: "JsOn", expectedType: "json"},

		// YAML variations
		{name: "yaml lowercase", formatValue: "yaml", expectedType: "yaml"},
		{name: "YAML uppercase", formatValue: "YAML", expectedType: "yaml"},
		{name: "Yaml capitalized", formatValue: "Yaml", expectedType: "yaml"},
		{name: "YaML mixed case", formatValue: "YaML", expectedType: "yaml"},

		// tfvars variations
		{name: "tfvars lowercase", formatValue: "tfvars", expectedType: "tfvars"},
		{name: "TFVARS uppercase", formatValue: "TFVARS", expectedType: "tfvars"},
		{name: "TfVars capitalized", formatValue: "TfVars", expectedType: "tfvars"},
		{name: "TfVaRs mixed case", formatValue: "TfVaRs", expectedType: "tfvars"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+tt.formatValue)

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", tt.formatValue, "-o", outFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build command failed for format %q: %v\nOutput: %s", tt.formatValue, err, output)
			}

			// The CLI auto-appends extensions when not provided
			var expectedOutFile string
			switch tt.expectedType {
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
				t.Fatalf("failed to read output file for format %q: %v", tt.formatValue, err)
			}

			// Verify the output is in the expected format
			switch tt.expectedType {
			case "json":
				var parsed map[string]any
				if err := json.Unmarshal(content, &parsed); err != nil {
					t.Errorf("failed to parse as JSON for format %q: %v\nContent: %s", tt.formatValue, err, content)
				}
			case "yaml":
				var parsed map[string]any
				if err := yaml.Unmarshal(content, &parsed); err != nil {
					t.Errorf("failed to parse as YAML for format %q: %v\nContent: %s", tt.formatValue, err, content)
				}
				// Verify it's not JSON
				if strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
					t.Errorf("output for format %q appears to be JSON, not YAML", tt.formatValue)
				}
			case "tfvars":
				outputStr := string(content)
				if !strings.Contains(outputStr, "=") {
					t.Errorf("tfvars output for format %q missing '=' assignment syntax", tt.formatValue)
				}
			}
		})
	}
}

// TestFormatFlag_DefaultFormat tests that the default format (JSON) is used
// when the --format flag is omitted.
//
// T037: Integration test for format flag variations
func TestFormatFlag_DefaultFormat(t *testing.T) {
	binPath := buildCLI(t)

	// Create test fixture
	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	fixtureContent := `config:
  name: "test"
  enabled: true
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	outFile := filepath.Join(tmpDir, "output-default")

	// Run build command WITHOUT -f flag
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-o", outFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build command failed: %v\nOutput: %s", err, output)
	}

	// The CLI auto-appends .json extension when format is omitted and no extension provided
	expectedOutFile := outFile + ".json"

	// Verify output file was created
	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(expectedOutFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Verify output is valid JSON (default format)
	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Errorf("default output should be JSON, but failed to parse: %v\nContent: %s", err, content)
	}

	// Verify it's JSON format
	outputStr := strings.TrimSpace(string(content))
	if !strings.HasPrefix(outputStr, "{") {
		t.Error("default output should be JSON format (start with '{')")
	}

	// Verify expected structure (metadata is opt-in)
	if parsed["config"] == nil {
		t.Error("JSON output missing expected top-level data")
	}
	if parsed["metadata"] != nil {
		t.Error("JSON output should not include 'metadata' by default")
	}
}
