package parse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseFile_ValidFile tests parsing a valid .csl file.
func TestParseFile_ValidFile(t *testing.T) {
	// Arrange - create a temporary valid .csl file
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid.csl")
	validContent := `simple-section:
  key: value
`
	if err := os.WriteFile(validPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	ast, diags, err := ParseFile(validPath)

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if ast == nil {
		t.Error("expected AST to be non-nil")
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diags))
	}
	if ast != nil && len(ast.Statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(ast.Statements))
	}
}

// TestParseFile_SyntaxError tests parsing a file with syntax errors.
func TestParseFile_SyntaxError(t *testing.T) {
	// Arrange - create a temporary invalid .csl file
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.csl")
	invalidContent := `section
  key value
`
	if err := os.WriteFile(invalidPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	ast, diags, err := ParseFile(invalidPath)

	// Assert
	// Parser errors should be captured as diagnostics, not fatal errors
	if err != nil {
		t.Errorf("expected no fatal error, got %v", err)
	}
	if len(diags) == 0 {
		t.Error("expected diagnostics for syntax error")
	}

	// Verify diagnostic structure
	if len(diags) > 0 {
		diag := diags[0]
		if !diag.IsError() {
			t.Errorf("expected diagnostic severity to be error, got %s", diag.Severity)
		}
		if diag.SourceSpan.Filename != invalidPath {
			t.Errorf("expected filename %s, got %s", invalidPath, diag.SourceSpan.Filename)
		}
		if diag.SourceSpan.StartLine == 0 {
			t.Error("expected non-zero line number in SourceSpan")
		}
		if diag.Message == "" {
			t.Error("expected non-empty message")
		}
	}

	// AST may be nil or partial on syntax error
	_ = ast
}

// TestParseFile_FormattedError tests that formatted error messages include caret snippets.
func TestParseFile_FormattedError(t *testing.T) {
	// Arrange - create a temporary file with a specific syntax error
	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "bad.csl")
	badContent := `config:
  @invalid: value
`
	if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	_, diags, err := ParseFile(badPath)

	// Assert
	if err != nil {
		t.Errorf("expected no fatal error, got %v", err)
	}
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}

	// Verify formatted message contains caret snippet
	diag := diags[0]
	if diag.FormattedMessage == "" {
		t.Error("expected FormattedMessage to be populated")
	}
	// Should contain line number prefix and caret
	if !strings.Contains(diag.FormattedMessage, "|") {
		t.Errorf("expected formatted message to contain line prefix '|', got: %s", diag.FormattedMessage)
	}
	if !strings.Contains(diag.FormattedMessage, "^") {
		t.Errorf("expected formatted message to contain caret '^', got: %s", diag.FormattedMessage)
	}
}

// TestParseFile_FileNotFound tests handling of missing files.
func TestParseFile_FileNotFound(t *testing.T) {
	// Arrange
	nonexistentPath := "/nonexistent/file.csl"

	// Act
	_, diags, err := ParseFile(nonexistentPath)

	// Assert
	// File I/O errors should be returned as diagnostics or fatal errors
	if err == nil && len(diags) == 0 {
		t.Error("expected either error or diagnostic for missing file")
	}
}

// TestParseReader_ValidInput tests parsing valid content from a reader.
func TestParseReader_ValidInput(t *testing.T) {
	// Arrange
	validContent := `simple-section:
  key: value
`
	reader := strings.NewReader(validContent)

	// Act
	ast, diags, err := ParseReader(reader, "test.csl")

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if ast == nil {
		t.Error("expected AST to be non-nil")
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d", len(diags))
	}
	if ast != nil && len(ast.Statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(ast.Statements))
	}
}

// TestParseReader_SyntaxError tests parsing invalid content from a reader.
func TestParseReader_SyntaxError(t *testing.T) {
	// Arrange
	invalidContent := `section
  key value
`
	reader := strings.NewReader(invalidContent)

	// Act
	_, diags, err := ParseReader(reader, "test.csl")

	// Assert
	// Parser errors should be captured as diagnostics, not fatal errors
	if err != nil {
		t.Errorf("expected no fatal error, got %v", err)
	}
	if len(diags) == 0 {
		t.Error("expected diagnostics for syntax error")
	}

	// Verify diagnostic structure
	if len(diags) > 0 {
		diag := diags[0]
		if !diag.IsError() {
			t.Errorf("expected diagnostic severity to be error, got %s", diag.Severity)
		}
		if diag.SourceSpan.Filename != "test.csl" {
			t.Errorf("expected filename test.csl, got %s", diag.SourceSpan.Filename)
		}
		if diag.Message == "" {
			t.Error("expected non-empty message")
		}
		// Verify formatted message includes caret
		if !strings.Contains(diag.FormattedMessage, "^") {
			t.Errorf("expected formatted message to contain caret, got: %s", diag.FormattedMessage)
		}
	}
}
