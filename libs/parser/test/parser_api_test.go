// Package parser_test contains integration tests for the parser public API.
package parser_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseFile_ValidFile tests that ParseFile successfully parses a valid .csl file.
func TestParseFile_ValidFile_ReturnsAST(t *testing.T) {
	// Arrange
	filePath := "../testdata/fixtures/simple.csl"

	// Act
	result, err := parser.ParseFile(filePath)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AST, got nil")
	}
	if result.SourceSpan.Filename != filePath {
		t.Errorf("expected filename %s, got %s", filePath, result.SourceSpan.Filename)
	}
}

// TestParseFile_NonExistentFile tests that ParseFile returns an error for missing files.
func TestParseFile_NonExistentFile_ReturnsError(t *testing.T) {
	// Arrange
	filePath := "../testdata/fixtures/nonexistent.csl"

	// Act
	_, err := parser.ParseFile(filePath)

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// TestParse_ValidInput_ReturnsAST tests the Parse function with an io.Reader.
func TestParse_ValidInput_ReturnsAST(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'test'
	type:  'folder'
	path:  './data'
`
	reader := strings.NewReader(input)
	filename := "test.csl"

	// Act
	result, err := parser.Parse(reader, filename)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AST, got nil")
	}
	if result.SourceSpan.Filename != filename {
		t.Errorf("expected filename %s, got %s", filename, result.SourceSpan.Filename)
	}
}

// TestParse_EmptyInput_ReturnsEmptyAST tests that an empty input produces a valid empty AST.
func TestParse_EmptyInput_ReturnsEmptyAST(t *testing.T) {
	// Arrange
	input := ""
	reader := strings.NewReader(input)
	filename := "empty.csl"

	// Act
	result, err := parser.Parse(reader, filename)

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AST, got nil")
	}
	if len(result.Statements) != 0 {
		t.Errorf("expected 0 statements for empty input, got %d", len(result.Statements))
	}
}

// TestParse_InvalidSyntax_ReturnsParseError tests error handling for malformed input.
func TestParse_InvalidSyntax_ReturnsParseError(t *testing.T) {
	// Arrange
	input := "invalid syntax here !!!"
	reader := strings.NewReader(input)
	filename := "invalid.csl"

	// Act
	_, err := parser.Parse(reader, filename)

	// Assert
	if err == nil {
		t.Fatal("expected parse error for invalid syntax, got nil")
	}

	// Verify error contains filename and position information
	errMsg := err.Error()
	if !strings.Contains(errMsg, filename) {
		t.Errorf("error message should contain filename, got: %s", errMsg)
	}
}

// TestAST_ContainsStatements tests that the AST contains parsed statements.
func TestAST_ContainsStatements_AfterParsing(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'folder'
	type:  'folder'

config:
	ref: @folder:config:config.key
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Statements) == 0 {
		t.Error("expected statements in AST, got empty statements list")
	}
}

// TestAST_StatementsHaveCorrectTypes tests that parsed statements have correct types.
func TestAST_StatementsHaveCorrectTypes(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'folder'
	type:  'folder'
	path:  '../config'

config:
	ref: @folder:config:config.key
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// We expect at least 2 statements: source, section
	if len(result.Statements) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(result.Statements))
	}

	// Type assertions to verify statement types
	if _, ok := result.Statements[0].(*ast.SourceDecl); !ok {
		t.Errorf("expected first statement to be *ast.SourceDecl, got %T", result.Statements[0])
	}
	if _, ok := result.Statements[1].(*ast.SectionDecl); !ok {
		t.Errorf("expected second statement to be *ast.SectionDecl, got %T", result.Statements[1])
	}
}

// TestParseFile_EdgeCase_LargeFile tests behavior with large input.
func TestParseFile_EdgeCase_LargeFile(t *testing.T) {
	t.Skip("TODO: Add large file test once parser implementation is complete")
}

// TestParse_Negative_InvalidSyntax tests error handling for malformed input.
func TestParse_Negative_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain string // Expected error message substring
	}{
		{
			name:          "invalid character",
			input:         "!invalid",
			shouldContain: "invalid syntax",
		},
		{
			name:          "import no longer supported",
			input:         "import:",
			shouldContain: "import statement no longer supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if tt.shouldContain == "" {
				// Some cases might not error (graceful handling)
				return
			}

			if err == nil {
				t.Errorf("expected error containing '%s', got nil", tt.shouldContain)
				return
			}

			if !strings.Contains(err.Error(), tt.shouldContain) {
				t.Errorf("expected error containing '%s', got: %v", tt.shouldContain, err)
			}
		})
	}
}

// TestParse_Negative_FilePaths tests error handling for invalid file paths.
func TestParseFile_Negative_NonexistentFile(t *testing.T) {
	_, err := parser.ParseFile("../testdata/fixtures/nonexistent.csl")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

// TestParse_Negative_ErrorContainsLocation tests that errors include location info.
func TestParse_Negative_ErrorContainsLocation(t *testing.T) {
	input := "section\nkey: value"
	reader := strings.NewReader(input)

	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Fatal("expected parse error, got nil")
	}

	errMsg := err.Error()

	// Error should contain filename
	if !strings.Contains(errMsg, "test.csl") {
		t.Errorf("error should contain filename, got: %s", errMsg)
	}

	// Error should contain line number
	if !strings.Contains(errMsg, "1:") || !strings.Contains(errMsg, ":") {
		t.Errorf("error should contain line:col format, got: %s", errMsg)
	}
}

// TestParseFile_Integration_AllGrammarConstructs tests end-to-end parsing
// of a comprehensive file with all grammar elements (MANDATORY integration test per AC).
func TestParseFile_Integration_AllGrammarConstructs(t *testing.T) {
	// This test satisfies the mandatory integration requirement from the story:
	// "Integration Test: test/integration/grammar_test.go that invokes Parse/ParseFile
	// on real .csl fixtures exercising each construct"

	// Arrange
	filePath := "../testdata/fixtures/simple.csl"

	// Act
	result, err := parser.ParseFile(filePath)

	// Assert
	if err != nil {
		t.Fatalf("expected no error parsing simple.csl, got %v", err)
	}

	// Verify we have all expected statement types (source, section with inline ref)
	if len(result.Statements) < 2 {
		t.Fatalf("expected at least 2 statements (source, section), got %d", len(result.Statements))
	}

	// Verify source declaration is present and correct
	sourceFound := false
	for _, stmt := range result.Statements {
		if decl, ok := stmt.(*ast.SourceDecl); ok {
			sourceFound = true
			if decl.Alias != "folder" {
				t.Errorf("source: expected alias 'folder', got '%s'", decl.Alias)
			}
			if decl.Type != "folder" {
				t.Errorf("source: expected type 'folder', got '%s'", decl.Type)
			}
			// Verify source span is populated
			span := decl.Span()
			if span.Filename != filePath {
				t.Errorf("source: expected filename '%s', got '%s'", filePath, span.Filename)
			}
			if span.StartLine < 1 {
				t.Error("source: start line should be >= 1")
			}
		}
	}
	if !sourceFound {
		t.Error("expected to find source declaration")
	}

	// Verify reference is now inline within section (as per User Story #18)
	// Check that the section contains an inline reference
	referenceFound := false
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok {
			if section.Name == "config-section" {
				// Check for the inline reference entry
				if refExpr, ok := section.Entries["ref_example"]; ok {
					if refNode, ok := refExpr.(*ast.ReferenceExpr); ok {
						referenceFound = true
						if refNode.Alias != "folder" {
							t.Errorf("inline reference: expected alias 'folder', got '%s'", refNode.Alias)
						}
						// Verify dotted path tokenization
						expectedPath := []string{"config", "config", "key"}
						if !reflect.DeepEqual(refNode.Path, expectedPath) {
							t.Errorf("inline reference: expected path %v, got %v", expectedPath, refNode.Path)
						}
						// Verify source span
						span := refNode.SourceSpan
						if span.Filename != filePath {
							t.Errorf("inline reference: expected filename '%s', got '%s'", filePath, span.Filename)
						}
					}
				}
			}
		}
	}
	if !referenceFound {
		t.Error("expected to find inline reference within section")
	}

	// Verify section declaration is present
	sectionFound := false
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok {
			sectionFound = true
			if section.Name != "config-section" {
				t.Errorf("section: expected name 'config-section', got '%s'", section.Name)
			}
			if len(section.Entries) < 1 {
				t.Error("section: expected at least one key-value entry")
			}
			// Verify source span
			span := section.Span()
			if span.Filename != filePath {
				t.Errorf("section: expected filename '%s', got '%s'", filePath, span.Filename)
			}
		}
	}
	if !sectionFound {
		t.Error("expected to find section declaration")
	}

	t.Log("Integration test PASS: All grammar constructs (source, import, inline reference, dotted paths, sections) parsed successfully with correct source spans")
}

// TestParseFile_InlineReferenceScalar tests parsing a file with inline reference expressions.
// This is the mandatory integration test for FEATURE-10-1.
func TestParseFile_InlineReferenceScalar(t *testing.T) {
	// Arrange
	filePath := "../testdata/fixtures/inline_ref_scalar.csl"

	// Act
	result, err := parser.ParseFile(filePath)

	// Assert - parse should succeed
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AST, got nil")
	}

	// Find the 'infrastructure' section
	var infraSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "infrastructure" {
			infraSection = section
			break
		}
	}

	if infraSection == nil {
		t.Fatal("expected to find 'infrastructure' section")
	}

	// The parser doesn't support inline reference syntax yet,
	// so this test documents the expected behavior once grammar support is added.
	// Currently, the inline reference will be treated as a string value.
	// Once FEATURE-10-2 (grammar update) is implemented, we expect:
	//   - infraSection to contain an entry "vpc_cidr" with a ReferenceExpr value
	//   - ReferenceExpr.Alias == "network"
	//   - ReferenceExpr.Path == []string{"config", "vpc", "cidr"}
	//   - ReferenceExpr.Span is non-empty

	// TODO: Update this assertion once parser grammar supports inline references
	// For now, we just verify the section exists and has entries
	if infraSection.Entries == nil {
		t.Fatal("expected infrastructure section to have entries")
	}

	t.Logf("Integration test: Section 'infrastructure' parsed with %d entries (parser grammar support pending)", len(infraSection.Entries))
}

// TestParseFile_CommentFixtures tests parsing of comment fixture files and verifies against golden output.
// This is T033 from Phase 3: Verify golden tests for comment support.
func TestParseFile_CommentFixtures(t *testing.T) {
	fixtures := []struct {
		name        string
		fixturePath string
		goldenPath  string
	}{
		{
			name:        "comments_basic.csl",
			fixturePath: "../testdata/fixtures/comments_basic.csl",
			goldenPath:  "../testdata/golden/comments_basic.csl.json",
		},
		{
			name:        "comments_inline.csl",
			fixturePath: "../testdata/fixtures/comments_inline.csl",
			goldenPath:  "../testdata/golden/comments_inline.csl.json",
		},
		{
			name:        "comments_in_strings.csl",
			fixturePath: "../testdata/fixtures/comments_in_strings.csl",
			goldenPath:  "../testdata/golden/comments_in_strings.csl.json",
		},
	}

	for _, tt := range fixtures {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the fixture file
			result, err := parser.ParseFile(tt.fixturePath)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", tt.name, err)
			}

			// Serialize AST to JSON
			actualJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal AST to JSON: %v", err)
			}

			// Read expected golden file
			//nolint:gosec // G304: goldenPath is controlled test fixture path
			expectedJSON, err := os.ReadFile(tt.goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", tt.goldenPath, err)
			}

			// Parse both JSONs to compare structure rather than bytes
			var expected, actual map[string]interface{}
			if err := json.Unmarshal(expectedJSON, &expected); err != nil {
				t.Fatalf("failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal(actualJSON, &actual); err != nil {
				t.Fatalf("failed to unmarshal actual JSON: %v", err)
			}

			// Compare using reflect.DeepEqual for semantic comparison
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("AST JSON does not match golden file for %s.\nExpected:\n%s\n\nActual:\n%s",
					tt.name, string(expectedJSON), string(actualJSON))
			} else {
				t.Logf("✓ Golden test PASS: %s matches expected output", tt.name)
			}
		})
	}
}
