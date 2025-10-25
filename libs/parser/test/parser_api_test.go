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
