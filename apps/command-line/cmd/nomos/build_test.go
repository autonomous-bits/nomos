package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/serialize"
	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestSerializeSnapshot_CaseInsensitive verifies that format selection
// is case-insensitive for all supported formats: json, yaml, tfvars.
// This tests Task T034: Case-insensitive format matching.
func TestSerializeSnapshot_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		// JSON variations
		{name: "format json lowercase", format: "json", wantErr: false},
		{name: "format JSON uppercase", format: "JSON", wantErr: false},
		{name: "format Json mixed case", format: "Json", wantErr: false},
		{name: "format jSoN random case", format: "jSoN", wantErr: false},

		// YAML variations
		{name: "format yaml lowercase", format: "yaml", wantErr: false},
		{name: "format YAML uppercase", format: "YAML", wantErr: false},
		{name: "format Yaml mixed case", format: "Yaml", wantErr: false},
		{name: "format YaML random case", format: "YaML", wantErr: false},

		// Tfvars variations
		{name: "format tfvars lowercase", format: "tfvars", wantErr: false},
		{name: "format TFVARS uppercase", format: "TFVARS", wantErr: false},
		{name: "format Tfvars mixed case", format: "Tfvars", wantErr: false},
		{name: "format TfVars random case", format: "TfVars", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal snapshot with simple test data
			snapshot := createMinimalSnapshot()

			output, err := serializeSnapshot(snapshot, tt.format)

			// Verify error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("serializeSnapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For valid formats, verify non-empty output
			if !tt.wantErr {
				if len(output) == 0 {
					t.Error("serializeSnapshot() returned empty output for valid format")
				}

				// Verify output appears to be in the correct format
				if strings.HasPrefix(strings.ToLower(tt.format), "json") && !isJSONLike(output) {
					t.Errorf("serializeSnapshot() output does not look like JSON: %s", truncateForError(output))
				}
				if strings.HasPrefix(strings.ToLower(tt.format), "yaml") && !isYAMLLike(output) {
					t.Errorf("serializeSnapshot() output does not look like YAML: %s", truncateForError(output))
				}
				if strings.HasPrefix(strings.ToLower(tt.format), "tfvars") && !isTfvarsLike(output) {
					t.Errorf("serializeSnapshot() output does not look like Tfvars: %s", truncateForError(output))
				}
			}
		})
	}
}

// TestSerializeSnapshot_InvalidFormat verifies that invalid formats return
// clear, actionable error messages that include the invalid format value
// and list all supported formats.
// This tests Task T035: Invalid format error message.
func TestSerializeSnapshot_InvalidFormat(t *testing.T) {
	tests := []struct {
		name             string
		format           string
		wantErr          bool
		errorMustContain []string
	}{
		{
			name:    "xml format not supported",
			format:  "xml",
			wantErr: true,
			errorMustContain: []string{
				"unsupported format",
				"xml",
				"json",
				"yaml",
				"tfvars",
			},
		},
		{
			name:    "toml format not supported",
			format:  "toml",
			wantErr: true,
			errorMustContain: []string{
				"unsupported format",
				"toml",
				"json",
				"yaml",
				"tfvars",
			},
		},
		{
			name:    "csv format not supported",
			format:  "csv",
			wantErr: true,
			errorMustContain: []string{
				"unsupported format",
				"csv",
				"json",
				"yaml",
				"tfvars",
			},
		},
		{
			name:    "empty string format not supported",
			format:  "",
			wantErr: true,
			errorMustContain: []string{
				"unsupported format",
				"json",
				"yaml",
				"tfvars",
			},
		},
		{
			name:    "random string format not supported",
			format:  "unknown",
			wantErr: true,
			errorMustContain: []string{
				"unsupported format",
				"unknown",
				"json",
				"yaml",
				"tfvars",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal snapshot
			snapshot := createMinimalSnapshot()

			output, err := serializeSnapshot(snapshot, tt.format)

			// Verify error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("serializeSnapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For invalid formats, verify error message quality
			if tt.wantErr && err != nil {
				errorMsg := err.Error()

				// Check that error message contains all required substrings
				for _, mustContain := range tt.errorMustContain {
					if !strings.Contains(errorMsg, mustContain) {
						t.Errorf("serializeSnapshot() error message missing required text %q\nError: %s", mustContain, errorMsg)
					}
				}

				// Verify error message is actionable (contains list of supported formats)
				if !strings.Contains(errorMsg, "supported:") {
					t.Errorf("serializeSnapshot() error message should indicate supported formats\nError: %s", errorMsg)
				}
			}

			// Invalid formats should not produce output
			if tt.wantErr && len(output) > 0 {
				t.Errorf("serializeSnapshot() returned output for invalid format: %v", output)
			}
		})
	}
}

// TestSerializeSnapshot_DefaultFormat documents and verifies the default
// format behavior. When the --format flag is not provided, it defaults
// to "json" via the Cobra flag default value.
// This tests Task T036: Default format behavior.
func TestSerializeSnapshot_DefaultFormat(t *testing.T) {
	t.Run("json is the default format", func(t *testing.T) {
		// NOTE: The default format is set at the flag level in buildCmd.Flags()
		// as: buildCmd.Flags().StringVarP(&buildFlags.format, "format", "f", "json", ...)
		//
		// This test verifies that "json" format works correctly when explicitly
		// passed. The flag default mechanism ensures that if no --format flag
		// is provided, serializeSnapshot will receive "json" as the format.

		snapshot := createMinimalSnapshot()

		// Test explicit "json" format (what the flag defaults to)
		output, err := serializeSnapshot(snapshot, "json")

		if err != nil {
			t.Errorf("serializeSnapshot() with default format 'json' returned error: %v", err)
			return
		}

		if len(output) == 0 {
			t.Error("serializeSnapshot() with default format 'json' returned empty output")
			return
		}

		// Verify JSON format
		if !isJSONLike(output) {
			t.Errorf("serializeSnapshot() default format should produce JSON output, got: %s", truncateForError(output))
		}

		// Verify JSON can be parsed (basic validation)
		if !strings.Contains(string(output), "{") || !strings.Contains(string(output), "}") {
			t.Error("serializeSnapshot() default format output does not appear to be valid JSON")
		}
	})

	t.Run("empty format string behavior", func(t *testing.T) {
		// When empty string is passed (edge case), it should be treated as
		// an invalid format since it doesn't match any valid format constant.
		// The flag default prevents this in normal usage, but we test the
		// function behavior for completeness.

		snapshot := createMinimalSnapshot()

		output, err := serializeSnapshot(snapshot, "")

		// Empty format should be treated as invalid
		if err == nil {
			t.Error("serializeSnapshot() with empty format string should return error")
			return
		}

		if len(output) > 0 {
			t.Error("serializeSnapshot() with empty format string should not produce output")
		}

		// Error message should be informative
		if !strings.Contains(err.Error(), "unsupported format") {
			t.Errorf("serializeSnapshot() empty format error should mention unsupported format, got: %v", err)
		}
	})
}

// Helper functions for test assertions

// createMinimalSnapshot creates a minimal valid snapshot for testing.
// This includes just enough data to produce valid serialization output
// without requiring complex test fixtures.
func createMinimalSnapshot() compiler.Snapshot {
	return compiler.Snapshot{
		Data: map[string]any{
			"test_key": "test_value",
		},
		Metadata: compiler.Metadata{
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
			InputFiles: []string{
				"test.csl",
			},
			Errors:           []string{},
			Warnings:         []string{},
			ProviderAliases:  []string{},
			PerKeyProvenance: map[string]compiler.Provenance{},
		},
	}
}

// isJSONLike performs a basic check if output looks like JSON.
// This is a heuristic, not a full JSON validator.
func isJSONLike(data []byte) bool {
	s := string(data)
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

// isYAMLLike performs a basic check if output looks like YAML.
// This is a heuristic, not a full YAML validator.
func isYAMLLike(data []byte) bool {
	s := string(data)
	// YAML typically has key: value format
	// Check for common YAML patterns
	return strings.Contains(s, ":") &&
		!strings.HasPrefix(strings.TrimSpace(s), "{") &&
		!strings.HasPrefix(strings.TrimSpace(s), "[")
}

// isTfvarsLike performs a basic check if output looks like HCL tfvars.
// This is a heuristic, not a full HCL validator.
func isTfvarsLike(data []byte) bool {
	s := string(data)
	// Tfvars has key = value format
	return strings.Contains(s, "=") &&
		!strings.HasPrefix(strings.TrimSpace(s), "{") &&
		!strings.Contains(s, ":")
}

// truncateForError truncates output to 50 characters for error messages.
func truncateForError(output []byte) string {
	if len(output) <= 50 {
		return string(output)
	}
	return string(output[:50])
}

// TestResolveOutputPath_AutoAppendExtension verifies that when a user provides
// an output path without an extension, the system automatically appends the
// correct extension based on the output format.
// This tests Task T043: Extension auto-append (no extension provided).
func TestResolveOutputPath_AutoAppendExtension(t *testing.T) {
	tests := []struct {
		name         string
		outputPath   string
		format       serialize.OutputFormat
		expectedPath string
	}{
		// JSON format auto-append
		{
			name:         "json format without extension",
			outputPath:   "output",
			format:       serialize.FormatJSON,
			expectedPath: "output.json",
		},
		{
			name:         "json format without extension - nested path",
			outputPath:   "build/output",
			format:       serialize.FormatJSON,
			expectedPath: "build/output.json",
		},
		{
			name:         "json format without extension - deep path",
			outputPath:   "dist/prod/config",
			format:       serialize.FormatJSON,
			expectedPath: "dist/prod/config.json",
		},

		// YAML format auto-append
		{
			name:         "yaml format without extension",
			outputPath:   "output",
			format:       serialize.FormatYAML,
			expectedPath: "output.yaml",
		},
		{
			name:         "yaml format without extension - nested path",
			outputPath:   "configs/app",
			format:       serialize.FormatYAML,
			expectedPath: "configs/app.yaml",
		},
		{
			name:         "yaml format without extension - deep path",
			outputPath:   "build/staging/values",
			format:       serialize.FormatYAML,
			expectedPath: "build/staging/values.yaml",
		},

		// Tfvars format auto-append
		{
			name:         "tfvars format without extension",
			outputPath:   "terraform",
			format:       serialize.FormatTfvars,
			expectedPath: "terraform.tfvars",
		},
		{
			name:         "tfvars format without extension - nested path",
			outputPath:   "infrastructure/prod",
			format:       serialize.FormatTfvars,
			expectedPath: "infrastructure/prod.tfvars",
		},
		{
			name:         "tfvars format without extension - deep path",
			outputPath:   "tf/modules/network/vars",
			format:       serialize.FormatTfvars,
			expectedPath: "tf/modules/network/vars.tfvars",
		},

		// Edge cases
		{
			name:         "filename with dots but no extension",
			outputPath:   "config.prod",
			format:       serialize.FormatJSON,
			expectedPath: "config.prod.json",
		},
		{
			name:         "multiple dots in path - no extension",
			outputPath:   "app.v2.config",
			format:       serialize.FormatYAML,
			expectedPath: "app.v2.config.yaml",
		},
		{
			name:         "path with directories containing dots",
			outputPath:   "v1.0/configs/app",
			format:       serialize.FormatJSON,
			expectedPath: "v1.0/configs/app.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, tt.outputPath)
			expectedFullPath := filepath.Join(tmpDir, tt.expectedPath)

			result, err := resolveOutputPath(inputPath, tt.format)

			if err != nil {
				t.Fatalf("resolveOutputPath() unexpected error: %v", err)
			}

			if result != expectedFullPath {
				t.Errorf("resolveOutputPath() = %q; want %q", result, expectedFullPath)
			}

			// Verify the path ends with the correct extension
			expectedExt := tt.format.Extension()
			if !strings.HasSuffix(result, expectedExt) {
				t.Errorf("resolveOutputPath() result %q does not end with expected extension %q", result, expectedExt)
			}
		})
	}
}

// TestResolveOutputPath_PreserveExplicitExtension verifies that when a user
// provides an output path with an explicit extension, the system preserves
// it exactly as provided, regardless of whether it matches the format.
// This tests Task T044: Explicit extension preserved (user provides extension).
func TestResolveOutputPath_PreserveExplicitExtension(t *testing.T) {
	tests := []struct {
		name         string
		outputPath   string
		format       serialize.OutputFormat
		expectedPath string
	}{
		// Matching extensions - preserve user's choice
		{
			name:         "yaml format with .yaml extension",
			outputPath:   "config.yaml",
			format:       serialize.FormatYAML,
			expectedPath: "config.yaml",
		},
		{
			name:         "yaml format with .yml extension",
			outputPath:   "config.yml",
			format:       serialize.FormatYAML,
			expectedPath: "config.yml",
		},
		{
			name:         "json format with .json extension",
			outputPath:   "data.json",
			format:       serialize.FormatJSON,
			expectedPath: "data.json",
		},
		{
			name:         "tfvars format with .tfvars extension",
			outputPath:   "vars.tfvars",
			format:       serialize.FormatTfvars,
			expectedPath: "vars.tfvars",
		},
		{
			name:         "tfvars format with .auto.tfvars extension",
			outputPath:   "terraform.auto.tfvars",
			format:       serialize.FormatTfvars,
			expectedPath: "terraform.auto.tfvars",
		},

		// Non-standard extensions - preserve user's choice
		{
			name:         "json format with .txt extension",
			outputPath:   "output.txt",
			format:       serialize.FormatJSON,
			expectedPath: "output.txt",
		},
		{
			name:         "yaml format with .conf extension",
			outputPath:   "app.conf",
			format:       serialize.FormatYAML,
			expectedPath: "app.conf",
		},
		{
			name:         "tfvars format with .hcl extension",
			outputPath:   "vars.hcl",
			format:       serialize.FormatTfvars,
			expectedPath: "vars.hcl",
		},
		{
			name:         "json format with .data extension",
			outputPath:   "snapshot.data",
			format:       serialize.FormatJSON,
			expectedPath: "snapshot.data",
		},

		// Nested paths with explicit extensions
		{
			name:         "nested path with .json extension",
			outputPath:   "output/build/result.json",
			format:       serialize.FormatJSON,
			expectedPath: "output/build/result.json",
		},
		{
			name:         "nested path with non-standard extension",
			outputPath:   "configs/prod/app.config",
			format:       serialize.FormatYAML,
			expectedPath: "configs/prod/app.config",
		},

		// Multiple dots in filename with explicit extension
		{
			name:         "multiple dots with explicit extension",
			outputPath:   "config.prod.v2.yaml",
			format:       serialize.FormatYAML,
			expectedPath: "config.prod.v2.yaml",
		},
		{
			name:         "version in filename with extension",
			outputPath:   "app.v1.0.json",
			format:       serialize.FormatJSON,
			expectedPath: "app.v1.0.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, tt.outputPath)
			expectedFullPath := filepath.Join(tmpDir, tt.expectedPath)

			result, err := resolveOutputPath(inputPath, tt.format)

			if err != nil {
				t.Fatalf("resolveOutputPath() unexpected error: %v", err)
			}

			if result != expectedFullPath {
				t.Errorf("resolveOutputPath() = %q; want %q", result, expectedFullPath)
			}

			// Verify the original extension is preserved
			if filepath.Ext(result) != filepath.Ext(expectedFullPath) {
				t.Errorf("resolveOutputPath() changed extension from %q to %q",
					filepath.Ext(expectedFullPath), filepath.Ext(result))
			}
		})
	}
}

// TestResolveOutputPath_ExtensionMismatch verifies that when a user provides
// an output path with an extension that doesn't match the format, the system
// preserves the user's explicit choice. This respects user intent even when
// the extension may seem inconsistent with the format.
// This tests Task T045: Extension mismatch handling.
func TestResolveOutputPath_ExtensionMismatch(t *testing.T) {
	tests := []struct {
		name         string
		outputPath   string
		format       serialize.OutputFormat
		expectedPath string
		description  string
	}{
		// JSON format with non-JSON extensions
		{
			name:         "json format with yaml extension",
			outputPath:   "config.yaml",
			format:       serialize.FormatJSON,
			expectedPath: "config.yaml",
			description:  "User wants JSON content in .yaml file",
		},
		{
			name:         "json format with tfvars extension",
			outputPath:   "data.tfvars",
			format:       serialize.FormatJSON,
			expectedPath: "data.tfvars",
			description:  "User wants JSON content in .tfvars file",
		},
		{
			name:         "json format with txt extension",
			outputPath:   "output.txt",
			format:       serialize.FormatJSON,
			expectedPath: "output.txt",
			description:  "User wants JSON content in .txt file",
		},

		// YAML format with non-YAML extensions
		{
			name:         "yaml format with json extension",
			outputPath:   "config.json",
			format:       serialize.FormatYAML,
			expectedPath: "config.json",
			description:  "User wants YAML content in .json file",
		},
		{
			name:         "yaml format with tfvars extension",
			outputPath:   "vars.tfvars",
			format:       serialize.FormatYAML,
			expectedPath: "vars.tfvars",
			description:  "User wants YAML content in .tfvars file",
		},
		{
			name:         "yaml format with xml extension",
			outputPath:   "data.xml",
			format:       serialize.FormatYAML,
			expectedPath: "data.xml",
			description:  "User wants YAML content in .xml file",
		},

		// Tfvars format with non-tfvars extensions
		{
			name:         "tfvars format with json extension",
			outputPath:   "terraform.json",
			format:       serialize.FormatTfvars,
			expectedPath: "terraform.json",
			description:  "User wants tfvars content in .json file",
		},
		{
			name:         "tfvars format with yaml extension",
			outputPath:   "config.yaml",
			format:       serialize.FormatTfvars,
			expectedPath: "config.yaml",
			description:  "User wants tfvars content in .yaml file",
		},
		{
			name:         "tfvars format with hcl extension",
			outputPath:   "vars.hcl",
			format:       serialize.FormatTfvars,
			expectedPath: "vars.hcl",
			description:  "User wants tfvars content in .hcl file (common alternative)",
		},

		// Nested paths with mismatched extensions
		{
			name:         "nested path - json format with yaml extension",
			outputPath:   "output/prod/config.yaml",
			format:       serialize.FormatJSON,
			expectedPath: "output/prod/config.yaml",
			description:  "Nested path preserves user's extension choice",
		},
		{
			name:         "nested path - yaml format with json extension",
			outputPath:   "configs/staging/app.json",
			format:       serialize.FormatYAML,
			expectedPath: "configs/staging/app.json",
			description:  "Nested path preserves user's extension choice",
		},

		// Edge cases with multiple dots and mismatched extensions
		{
			name:         "multiple dots - json format with yaml extension",
			outputPath:   "config.prod.v2.yaml",
			format:       serialize.FormatJSON,
			expectedPath: "config.prod.v2.yaml",
			description:  "Multiple dots with explicit mismatched extension",
		},
		{
			name:         "version in name - tfvars format with json extension",
			outputPath:   "terraform.v1.0.json",
			format:       serialize.FormatTfvars,
			expectedPath: "terraform.v1.0.json",
			description:  "Version number with mismatched extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, tt.outputPath)
			expectedFullPath := filepath.Join(tmpDir, tt.expectedPath)

			result, err := resolveOutputPath(inputPath, tt.format)

			if err != nil {
				t.Fatalf("resolveOutputPath() unexpected error: %v", err)
			}

			if result != expectedFullPath {
				t.Errorf("resolveOutputPath() = %q; want %q\nReason: %s",
					result, expectedFullPath, tt.description)
			}

			// Verify the original extension is preserved despite mismatch
			originalExt := filepath.Ext(tt.outputPath)
			resultExt := filepath.Ext(result)
			if resultExt != originalExt {
				t.Errorf("resolveOutputPath() changed extension from %q to %q\n"+
					"User's explicit choice should be preserved even when mismatched\nReason: %s",
					originalExt, resultExt, tt.description)
			}

			// Verify we didn't append the format's default extension
			formatExt := tt.format.Extension()
			if strings.HasSuffix(result, formatExt) && originalExt != formatExt {
				t.Errorf("resolveOutputPath() incorrectly appended format extension %q\n"+
					"Original extension %q should be preserved\nReason: %s",
					formatExt, originalExt, tt.description)
			}
		})
	}
}

// TestResolveOutputPath_EdgeCases verifies edge case handling for file
// extension resolution including empty paths, paths with only extensions,
// paths with trailing slashes, and other boundary conditions.
func TestResolveOutputPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		outputPath  string
		format      serialize.OutputFormat
		expectError bool
		errorMsg    string
		expectedExt string
	}{
		// Empty and whitespace paths
		{
			name:        "empty path",
			outputPath:  "",
			format:      serialize.FormatJSON,
			expectError: true,
			errorMsg:    "empty",
		},
		{
			name:        "whitespace only path",
			outputPath:  "   ",
			format:      serialize.FormatJSON,
			expectError: true,
			errorMsg:    "empty",
		},

		// Just extension (no basename)
		{
			name:        "only extension - .json",
			outputPath:  ".json",
			format:      serialize.FormatJSON,
			expectError: true,
			errorMsg:    "invalid",
		},
		{
			name:        "only extension - .yaml",
			outputPath:  ".yaml",
			format:      serialize.FormatYAML,
			expectError: true,
			errorMsg:    "invalid",
		},
		{
			name:        "only extension - .tfvars",
			outputPath:  ".tfvars",
			format:      serialize.FormatTfvars,
			expectError: true,
			errorMsg:    "invalid",
		},

		// Hidden files (valid use case - should work)
		{
			name:        "hidden file without extension",
			outputPath:  ".config",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".json",
		},
		{
			name:        "hidden file with extension",
			outputPath:  ".config.yaml",
			format:      serialize.FormatYAML,
			expectError: false,
			expectedExt: ".yaml",
		},

		// Path with trailing slash
		{
			name:        "path with trailing slash",
			outputPath:  "output/",
			format:      serialize.FormatJSON,
			expectError: true,
			errorMsg:    "directory",
		},
		{
			name:        "path with multiple trailing slashes",
			outputPath:  "output///",
			format:      serialize.FormatJSON,
			expectError: true,
			errorMsg:    "directory",
		},

		// Special characters in filename
		{
			name:        "filename with spaces - no extension",
			outputPath:  "my config",
			format:      serialize.FormatYAML,
			expectError: false,
			expectedExt: ".yaml",
		},
		{
			name:        "filename with spaces - with extension",
			outputPath:  "my config.json",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".json",
		},
		{
			name:        "filename with hyphens - no extension",
			outputPath:  "app-config-prod",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".json",
		},
		{
			name:        "filename with underscores - no extension",
			outputPath:  "app_config_prod",
			format:      serialize.FormatTfvars,
			expectError: false,
			expectedExt: ".tfvars",
		},

		// Multiple consecutive dots
		{
			name:        "multiple consecutive dots - no extension",
			outputPath:  "config..backup",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".backup",
		},
		{
			name:        "multiple consecutive dots - with extension",
			outputPath:  "config..backup.json",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".json",
		},

		// Relative path components
		{
			name:        "relative path with ..",
			outputPath:  "../output/config",
			format:      serialize.FormatYAML,
			expectError: false,
			expectedExt: ".yaml",
		},
		{
			name:        "relative path with .",
			outputPath:  "./configs/output",
			format:      serialize.FormatJSON,
			expectError: false,
			expectedExt: ".json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't use tmpDir for error cases as path may be invalid
			var inputPath string
			switch {
			case tt.expectError && (tt.outputPath == "" || strings.TrimSpace(tt.outputPath) == ""):
				inputPath = tt.outputPath
			case tt.expectError && (strings.HasSuffix(tt.outputPath, "/") || strings.HasSuffix(tt.outputPath, string(filepath.Separator))):
				// For trailing slash tests, don't use filepath.Join as it cleans the path
				// and removes trailing slashes. Concatenate manually to preserve the slash.
				tmpDir := t.TempDir()
				inputPath = tmpDir + string(filepath.Separator) + tt.outputPath
			default:
				tmpDir := t.TempDir()
				inputPath = filepath.Join(tmpDir, tt.outputPath)
			}

			result, err := resolveOutputPath(inputPath, tt.format)

			if tt.expectError {
				if err == nil {
					t.Fatalf("resolveOutputPath() expected error containing %q, got nil", tt.errorMsg)
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("resolveOutputPath() error = %q; want error containing %q",
						err.Error(), tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveOutputPath() unexpected error: %v", err)
			}

			// Verify expected extension
			if tt.expectedExt != "" {
				if !strings.HasSuffix(result, tt.expectedExt) {
					t.Errorf("resolveOutputPath() result %q does not end with expected extension %q",
						result, tt.expectedExt)
				}
			}
		})
	}
}

// TestResolveOutputPath_IntegrationWithFormats verifies that resolveOutputPath
// works correctly with all supported OutputFormat values and validates format
// compatibility.
func TestResolveOutputPath_IntegrationWithFormats(t *testing.T) {
	tests := []struct {
		name        string
		outputPath  string
		format      serialize.OutputFormat
		expectedExt string
		expectError bool
	}{
		// Valid formats
		{
			name:        "json format",
			outputPath:  "output",
			format:      serialize.FormatJSON,
			expectedExt: ".json",
		},
		{
			name:        "yaml format",
			outputPath:  "output",
			format:      serialize.FormatYAML,
			expectedExt: ".yaml",
		},
		{
			name:        "tfvars format",
			outputPath:  "output",
			format:      serialize.FormatTfvars,
			expectedExt: ".tfvars",
		},

		// Invalid format (empty)
		{
			name:        "invalid format - empty string",
			outputPath:  "output",
			format:      serialize.OutputFormat(""),
			expectError: true,
		},

		// Invalid format (unsupported)
		{
			name:        "invalid format - xml",
			outputPath:  "output",
			format:      serialize.OutputFormat("xml"),
			expectError: true,
		},
		{
			name:        "invalid format - toml",
			outputPath:  "output",
			format:      serialize.OutputFormat("toml"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, tt.outputPath)

			result, err := resolveOutputPath(inputPath, tt.format)

			if tt.expectError {
				if err == nil {
					t.Fatal("resolveOutputPath() expected error for invalid format, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveOutputPath() unexpected error: %v", err)
			}

			if !strings.HasSuffix(result, tt.expectedExt) {
				t.Errorf("resolveOutputPath() result %q does not end with expected extension %q",
					result, tt.expectedExt)
			}
		})
	}
}
