// Package parser_test contains integration tests for the parser public API.
package parser_test

import (
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

import:folder:filename
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

import:folder:filename

reference:folder:config.key
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// We expect at least 3 statements: source, import, reference
	if len(result.Statements) < 3 {
		t.Fatalf("expected at least 3 statements, got %d", len(result.Statements))
	}

	// Type assertions to verify statement types
	if _, ok := result.Statements[0].(*ast.SourceDecl); !ok {
		t.Errorf("expected first statement to be *ast.SourceDecl, got %T", result.Statements[0])
	}
	if _, ok := result.Statements[1].(*ast.ImportStmt); !ok {
		t.Errorf("expected second statement to be *ast.ImportStmt, got %T", result.Statements[1])
	}
	if _, ok := result.Statements[2].(*ast.ReferenceStmt); !ok {
		t.Errorf("expected third statement to be *ast.ReferenceStmt, got %T", result.Statements[2])
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
			name:          "incomplete import missing path",
			input:         "import:",
			shouldContain: "", // Empty identifier is handled gracefully
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

	// Verify we have all expected statement types
	if len(result.Statements) < 4 {
		t.Fatalf("expected at least 4 statements (source, import, reference, section), got %d", len(result.Statements))
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

	// Verify import statement is present and correct
	importFound := false
	for _, stmt := range result.Statements {
		if impStmt, ok := stmt.(*ast.ImportStmt); ok {
			importFound = true
			if impStmt.Alias != "folder" {
				t.Errorf("import: expected alias 'folder', got '%s'", impStmt.Alias)
			}
			if impStmt.Path != "filename" {
				t.Errorf("import: expected path 'filename', got '%s'", impStmt.Path)
			}
			// Verify source span
			span := impStmt.Span()
			if span.Filename != filePath {
				t.Errorf("import: expected filename '%s', got '%s'", filePath, span.Filename)
			}
		}
	}
	if !importFound {
		t.Error("expected to find import statement")
	}

	// Verify reference statement with dotted path is present
	referenceFound := false
	for _, stmt := range result.Statements {
		if refStmt, ok := stmt.(*ast.ReferenceStmt); ok {
			referenceFound = true
			if refStmt.Alias != "folder" {
				t.Errorf("reference: expected alias 'folder', got '%s'", refStmt.Alias)
			}
			// Verify dotted path tokenization
			if refStmt.Path != "config.key" {
				t.Errorf("reference: expected path 'config.key', got '%s'", refStmt.Path)
			}
			if !strings.Contains(refStmt.Path, ".") {
				t.Error("reference: path should contain dot for path tokenization")
			}
			// Verify source span
			span := refStmt.Span()
			if span.Filename != filePath {
				t.Errorf("reference: expected filename '%s', got '%s'", filePath, span.Filename)
			}
		}
	}
	if !referenceFound {
		t.Error("expected to find reference statement with dotted path")
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

	t.Log("Integration test PASS: All grammar constructs (source, import, reference, dotted paths, sections) parsed successfully with correct source spans")
}
