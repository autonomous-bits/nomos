// Package parser_test contains tests for version field parsing and validation.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseSourceDecl_WithVersion tests parsing source declarations with valid semver versions.
func TestParseSourceDecl_WithVersion(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedAlias   string
		expectedType    string
		expectedVersion string
	}{
		{
			name: "simple version",
			input: `source:
	alias: 'terraform'
	type: 'terraform'
	version: '1.2.3'
	path: './providers/terraform'
`,
			expectedAlias:   "terraform",
			expectedType:    "terraform",
			expectedVersion: "1.2.3",
		},
		{
			name: "prerelease version",
			input: `source:
	alias: 'beta-provider'
	type: 'custom'
	version: '2.0.0-beta.1'
	api_key: 'test-key'
`,
			expectedAlias:   "beta-provider",
			expectedType:    "custom",
			expectedVersion: "2.0.0-beta.1",
		},
		{
			name: "version with build metadata",
			input: `source:
	alias: 'provider'
	type: 'test'
	version: '1.0.0+20240101'
`,
			expectedAlias:   "provider",
			expectedType:    "test",
			expectedVersion: "1.0.0+20240101",
		},
		{
			name: "version with prerelease and build",
			input: `source:
	alias: 'provider'
	type: 'test'
	version: '1.0.0-alpha.1+exp.sha.5114f85'
`,
			expectedAlias:   "provider",
			expectedType:    "test",
			expectedVersion: "1.0.0-alpha.1+exp.sha.5114f85",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			decl, ok := result.Statements[0].(*ast.SourceDecl)
			if !ok {
				t.Fatalf("expected *ast.SourceDecl, got %T", result.Statements[0])
			}

			if decl.Alias != tt.expectedAlias {
				t.Errorf("expected alias %q, got %q", tt.expectedAlias, decl.Alias)
			}
			if decl.Type != tt.expectedType {
				t.Errorf("expected type %q, got %q", tt.expectedType, decl.Type)
			}
			if decl.Version != tt.expectedVersion {
				t.Errorf("expected version %q, got %q", tt.expectedVersion, decl.Version)
			}

			// Verify version is NOT in Config map (reserved field extracted)
			if _, hasVersion := decl.Config["version"]; hasVersion {
				t.Error("version should not be in Config map (reserved field)")
			}
		})
	}
}

// TestParseSourceDecl_WithoutVersion tests parsing source declarations without version field.
func TestParseSourceDecl_WithoutVersion(t *testing.T) {
	input := `source:
	alias: 'legacy'
	type: 'file'
	path: './data'
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	decl, ok := result.Statements[0].(*ast.SourceDecl)
	if !ok {
		t.Fatalf("expected *ast.SourceDecl, got %T", result.Statements[0])
	}

	if decl.Alias != "legacy" {
		t.Errorf("expected alias 'legacy', got %q", decl.Alias)
	}
	if decl.Type != "file" {
		t.Errorf("expected type 'file', got %q", decl.Type)
	}
	if decl.Version != "" {
		t.Errorf("expected empty version, got %q", decl.Version)
	}

	// Verify version is NOT in Config map
	if _, hasVersion := decl.Config["version"]; hasVersion {
		t.Error("version should not be in Config map")
	}
}

// TestParseSourceDecl_EmptyVersionString tests that empty version string is valid.
func TestParseSourceDecl_EmptyVersionString(t *testing.T) {
	input := `source:
	alias: 'provider'
	type: 'test'
	version: ''
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error for empty version string, got %v", err)
	}

	decl := result.Statements[0].(*ast.SourceDecl)
	if decl.Version != "" {
		t.Errorf("expected empty version, got %q", decl.Version)
	}
}

// TestParseSourceDecl_InvalidVersion tests that invalid semver versions are rejected.
func TestParseSourceDecl_InvalidVersion(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "non-semver version",
			input: `source:
	alias: 'bad-version'
	type: 'custom'
	version: 'not-a-version'
`,
			expectedError: "invalid version format",
		},
		{
			name: "incomplete version",
			input: `source:
	alias: 'bad'
	type: 'test'
	version: '1.2'
`,
			expectedError: "invalid version format",
		},
		{
			name: "version with v prefix",
			input: `source:
	alias: 'bad'
	type: 'test'
	version: 'v1.2.3'
`,
			expectedError: "invalid version format",
		},
		{
			name: "version with spaces",
			input: `source:
	alias: 'bad'
	type: 'test'
	version: '1.2.3 beta'
`,
			expectedError: "invalid version format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if err == nil {
				t.Fatal("expected error for invalid version, got nil")
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.expectedError) {
				t.Errorf("expected error containing %q, got: %s", tt.expectedError, errMsg)
			}

			// Verify error message provides guidance
			if !strings.Contains(errMsg, "semver.org") || !strings.Contains(errMsg, "semantic version") {
				t.Errorf("expected error message to reference semver.org and provide examples, got: %s", errMsg)
			}
		})
	}
}

// TestParseSourceDecl_VersionFieldRemovalFromConfig tests that version is removed from Config after extraction.
func TestParseSourceDecl_VersionFieldRemovalFromConfig(t *testing.T) {
	input := `source:
	alias: 'provider'
	type: 'test'
	version: '1.0.0'
	custom_field: 'value'
	another_field: 'data'
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decl := result.Statements[0].(*ast.SourceDecl)

	// Verify version field is properly set
	if decl.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", decl.Version)
	}

	// Verify Config contains only non-reserved fields
	expectedConfigFields := []string{"custom_field", "another_field"}
	for _, field := range expectedConfigFields {
		if _, exists := decl.Config[field]; !exists {
			t.Errorf("expected field %q in Config", field)
		}
	}

	// Verify Config does NOT contain reserved fields
	reservedFields := []string{"alias", "type", "version"}
	for _, field := range reservedFields {
		if _, exists := decl.Config[field]; exists {
			t.Errorf("reserved field %q should not be in Config", field)
		}
	}

	// Verify Config has exactly the expected number of fields
	if len(decl.Config) != 2 {
		t.Errorf("expected Config to have 2 fields, got %d", len(decl.Config))
	}
}
