// Package parser_test contains integration tests for the parser public API.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestSourceSpan_AllNodesHaveSpans tests that all AST nodes include source span information.
func TestSourceSpan_AllNodesHaveSpans(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'folder'
	type:  'folder'
	path:  '../config'

import:folder:filename

reference:folder:config.key

config-section:
	key1: value1
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check AST root has span
	if result.SourceSpan.Filename != "test.csl" {
		t.Errorf("AST span filename: expected 'test.csl', got '%s'", result.SourceSpan.Filename)
	}
	if result.SourceSpan.StartLine != 1 {
		t.Errorf("AST span start line: expected 1, got %d", result.SourceSpan.StartLine)
	}

	// Check each statement has a span
	for i, stmt := range result.Statements {
		span := stmt.Span()
		if span.Filename != "test.csl" {
			t.Errorf("statement[%d] span filename: expected 'test.csl', got '%s'", i, span.Filename)
		}
		if span.StartLine <= 0 {
			t.Errorf("statement[%d] span start line: expected > 0, got %d", i, span.StartLine)
		}
		if span.StartCol <= 0 {
			t.Errorf("statement[%d] span start col: expected > 0, got %d", i, span.StartCol)
		}
		if span.EndLine <= 0 {
			t.Errorf("statement[%d] span end line: expected > 0, got %d", i, span.EndLine)
		}
		if span.EndCol < 0 {
			t.Errorf("statement[%d] span end col: expected >= 0, got %d", i, span.EndCol)
		}
	}
}

// TestSourceSpan_CorrectLineNumbers tests that line numbers are accurate.
func TestSourceSpan_CorrectLineNumbers(t *testing.T) {
	// Arrange - source starts at line 1, import at line 5
	input := `source:
	alias: 'test'
	type:  'folder'

import:test:path
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Statements) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(result.Statements))
	}

	// First statement (source) should start at line 1
	sourceStmt := result.Statements[0]
	if sourceStmt.Span().StartLine != 1 {
		t.Errorf("source statement: expected start line 1, got %d", sourceStmt.Span().StartLine)
	}

	// Second statement (import) should start at line 5
	importStmt := result.Statements[1]
	if importStmt.Span().StartLine != 5 {
		t.Errorf("import statement: expected start line 5, got %d", importStmt.Span().StartLine)
	}
}
