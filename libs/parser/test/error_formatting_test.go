// Package parser_test provides comprehensive tests for error formatting and handling.
package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestFormatParseError_BasicError tests basic error formatting.
func TestFormatParseError_BasicError(t *testing.T) {
	sourceText := `source:
  alias: myConfig
  type: yaml

database:
  host: localhost
`

	// Create a parse error
	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		6,
		3,
		"unexpected token",
	)

	formatted := parser.FormatParseError(err, sourceText)

	// Verify formatted output contains key elements
	if !strings.Contains(formatted, "test.csl:6:3:") {
		t.Errorf("formatted error missing location prefix")
	}
	if !strings.Contains(formatted, "unexpected token") {
		t.Errorf("formatted error missing message")
	}
	if !strings.Contains(formatted, "host: localhost") {
		t.Errorf("formatted error missing source context")
	}
	if !strings.Contains(formatted, "^") {
		t.Errorf("formatted error missing caret marker")
	}

	t.Logf("Formatted error:\n%s", formatted)
}

// TestFormatParseError_AllErrorKinds tests formatting for all error types.
func TestFormatParseError_AllErrorKinds(t *testing.T) {
	sourceText := "line1\nline2\nline3"

	testCases := []struct {
		name string
		kind parser.ParseErrorKind
	}{
		{"LexError", parser.LexError},
		{"SyntaxError", parser.SyntaxError},
		{"IOError", parser.IOError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.NewParseError(tc.kind, "test.csl", 2, 3, "test error")
			formatted := parser.FormatParseError(err, sourceText)

			if !strings.Contains(formatted, "test.csl:2:3:") {
				t.Errorf("formatted error missing location")
			}
			if !strings.Contains(formatted, "line2") {
				t.Errorf("formatted error missing source line")
			}
		})
	}
}

// TestFormatParseError_UTF8Handling tests UTF-8 character handling in snippets.
func TestFormatParseError_UTF8Handling(t *testing.T) {
	testCases := []struct {
		name       string
		sourceText string
		line       int
		col        int
		expectChar string
	}{
		{
			name:       "ASCII only",
			sourceText: "hello world\nfoo bar",
			line:       2,
			col:        5,
			expectChar: "bar",
		},
		{
			name:       "UTF-8 emoji",
			sourceText: "test: 🚀 rocket\nanother: line",
			line:       1,
			col:        7,
			expectChar: "🚀",
		},
		{
			name:       "UTF-8 multibyte characters",
			sourceText: "名前: テスト\n値: データ",
			line:       1,
			col:        4,
			expectChar: "テスト",
		},
		{
			name:       "Mixed ASCII and UTF-8",
			sourceText: "key: café\nhost: localhost",
			line:       1,
			col:        6,
			expectChar: "café",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.NewParseError(
				parser.SyntaxError,
				"test.csl",
				tc.line,
				tc.col,
				"test error",
			)

			formatted := parser.FormatParseError(err, tc.sourceText)

			// Verify the caret appears in the output
			if !strings.Contains(formatted, "^") {
				t.Errorf("formatted error missing caret marker")
			}

			// Verify the expected character appears in the output
			if !strings.Contains(formatted, tc.expectChar) {
				t.Errorf("formatted error missing expected character %q", tc.expectChar)
			}

			t.Logf("Formatted error:\n%s", formatted)
		})
	}
}

// TestFormatParseError_EdgeCases tests edge cases in error formatting.
func TestFormatParseError_EdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		sourceText string
		line       int
		col        int
	}{
		{
			name:       "empty source",
			sourceText: "",
			line:       1,
			col:        1,
		},
		{
			name:       "line out of bounds (too high)",
			sourceText: "line1\nline2",
			line:       10,
			col:        1,
		},
		{
			name:       "line zero",
			sourceText: "line1\nline2",
			line:       0,
			col:        1,
		},
		{
			name:       "column zero",
			sourceText: "line1\nline2",
			line:       1,
			col:        0,
		},
		{
			name:       "single line file",
			sourceText: "only one line",
			line:       1,
			col:        5,
		},
		{
			name:       "last line of file",
			sourceText: "line1\nline2\nline3",
			line:       3,
			col:        3,
		},
		{
			name:       "first line of file",
			sourceText: "line1\nline2\nline3",
			line:       1,
			col:        3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.NewParseError(
				parser.SyntaxError,
				"test.csl",
				tc.line,
				tc.col,
				"test error",
			)

			// This should not panic
			formatted := parser.FormatParseError(err, tc.sourceText)

			// Verify it returns something
			if formatted == "" {
				t.Errorf("FormatParseError returned empty string")
			}

			t.Logf("Formatted error:\n%s", formatted)
		})
	}
}

// TestFormatParseError_ErrorUnwrapping tests error unwrapping logic.
func TestFormatParseError_ErrorUnwrapping(t *testing.T) {
	sourceText := "line1\nline2\nline3"

	// Create a wrapped parse error
	innerErr := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		2,
		3,
		"syntax error",
	)
	wrappedErr := errors.New("wrapped: " + innerErr.Error())

	// FormatParseError with a non-ParseError should return the error string
	formatted := parser.FormatParseError(wrappedErr, sourceText)
	if !strings.Contains(formatted, "wrapped:") {
		t.Errorf("expected wrapped error message in output")
	}

	// FormatParseError with actual ParseError should include snippet
	formatted = parser.FormatParseError(innerErr, sourceText)
	if !strings.Contains(formatted, "line2") {
		t.Errorf("expected source context in ParseError output")
	}
	if !strings.Contains(formatted, "^") {
		t.Errorf("expected caret in ParseError output")
	}
}

// TestFormatParseError_WithoutSourceText tests formatting without source text.
func TestFormatParseError_WithoutSourceText(t *testing.T) {
	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		5,
		10,
		"syntax error",
	)

	formatted := parser.FormatParseError(err, "")

	// Should still have the basic error message
	if !strings.Contains(formatted, "test.csl:5:10:") {
		t.Errorf("formatted error missing location prefix")
	}
	if !strings.Contains(formatted, "syntax error") {
		t.Errorf("formatted error missing message")
	}

	// Should not have a snippet
	if strings.Contains(formatted, "^") {
		t.Errorf("formatted error should not have caret without source text")
	}
}

// TestParseError_Methods tests ParseError accessor methods.
func TestParseError_Methods(t *testing.T) {
	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		10,
		25,
		"test message",
	)

	if err.Kind() != parser.SyntaxError {
		t.Errorf("Kind() = %v, want %v", err.Kind(), parser.SyntaxError)
	}

	if err.Filename() != "test.csl" {
		t.Errorf("Filename() = %q, want %q", err.Filename(), "test.csl")
	}

	if err.Line() != 10 {
		t.Errorf("Line() = %d, want %d", err.Line(), 10)
	}

	if err.Column() != 25 {
		t.Errorf("Column() = %d, want %d", err.Column(), 25)
	}

	if err.Message() != "test message" {
		t.Errorf("Message() = %q, want %q", err.Message(), "test message")
	}

	// Test Error() method
	errStr := err.Error()
	if !strings.Contains(errStr, "test.csl:10:25:") {
		t.Errorf("Error() missing location: %q", errStr)
	}
	if !strings.Contains(errStr, "test message") {
		t.Errorf("Error() missing message: %q", errStr)
	}
}

// TestParseError_SpanMethod tests the Span() method.
func TestParseError_SpanMethod(t *testing.T) {
	err := parser.NewParseError(
		parser.LexError,
		"source.csl",
		15,
		30,
		"lex error",
	)

	span := err.Span()

	expected := ast.SourceSpan{
		Filename:  "source.csl",
		StartLine: 15,
		StartCol:  30,
		EndLine:   15,
		EndCol:    30,
	}

	if span != expected {
		t.Errorf("Span() = %+v, want %+v", span, expected)
	}
}

// TestParseError_SetSnippet tests snippet setting and retrieval.
func TestParseError_SetSnippet(t *testing.T) {
	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		5,
		10,
		"test error",
	)

	// Initially should be empty
	if err.Snippet() != "" {
		t.Errorf("initial snippet should be empty, got %q", err.Snippet())
	}

	// Set a snippet
	testSnippet := "   5 | some code here\n       ^"
	err.SetSnippet(testSnippet)

	if err.Snippet() != testSnippet {
		t.Errorf("Snippet() = %q, want %q", err.Snippet(), testSnippet)
	}
}

// TestParseErrorKind_StringMethod tests ParseErrorKind string representation.
func TestParseErrorKind_StringMethod(t *testing.T) {
	testCases := []struct {
		kind parser.ParseErrorKind
		want string
	}{
		{parser.LexError, "LexError"},
		{parser.SyntaxError, "SyntaxError"},
		{parser.IOError, "IOError"},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.kind.String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFormatParseError_ContextLines tests that context lines are properly shown.
func TestFormatParseError_ContextLines(t *testing.T) {
	sourceText := `line1
line2
line3
line4
line5`

	testCases := []struct {
		name            string
		line            int
		col             int
		expectLines     []string
		expectNotInView []string
	}{
		{
			name:            "error on line 1 (no line before)",
			line:            1,
			col:             3,
			expectLines:     []string{"   1 | line1", "   2 | line2"},
			expectNotInView: []string{"line3"},
		},
		{
			name:            "error on line 3 (context before and after)",
			line:            3,
			col:             3,
			expectLines:     []string{"   2 | line2", "   3 | line3", "   4 | line4"},
			expectNotInView: []string{"line1", "line5"},
		},
		{
			name:            "error on last line (no line after)",
			line:            5,
			col:             3,
			expectLines:     []string{"   4 | line4", "   5 | line5"},
			expectNotInView: []string{"line1", "line2", "line3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.NewParseError(
				parser.SyntaxError,
				"test.csl",
				tc.line,
				tc.col,
				"test error",
			)

			formatted := parser.FormatParseError(err, sourceText)

			// Check that expected lines are present
			for _, line := range tc.expectLines {
				if !strings.Contains(formatted, line) {
					t.Errorf("expected line %q not found in formatted output", line)
				}
			}

			// Check that lines not in view are absent
			for _, line := range tc.expectNotInView {
				if strings.Contains(formatted, line) {
					t.Errorf("unexpected line %q found in formatted output", line)
				}
			}

			t.Logf("Formatted error:\n%s", formatted)
		})
	}
}

// TestFormatParseError_LongLines tests handling of very long source lines.
func TestFormatParseError_LongLines(t *testing.T) {
	// Create a very long line
	longLine := strings.Repeat("x", 500)
	sourceText := "short\n" + longLine + "\nshort"

	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		2,
		250,
		"error in long line",
	)

	formatted := parser.FormatParseError(err, sourceText)

	// Should still include the line (even if long)
	if !strings.Contains(formatted, longLine) {
		t.Errorf("formatted error should include long line")
	}

	// Should include caret marker
	if !strings.Contains(formatted, "^") {
		t.Errorf("formatted error should include caret")
	}

	t.Logf("Formatted error length: %d characters", len(formatted))
}

// TestFormatParseError_EmptyLines tests handling of empty lines in source.
func TestFormatParseError_EmptyLines(t *testing.T) {
	sourceText := "line1\n\nline3\n\nline5"

	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		3,
		1,
		"error on line with content after empty line",
	)

	formatted := parser.FormatParseError(err, sourceText)

	// Should show empty line as context
	if !strings.Contains(formatted, "   2 |") {
		t.Errorf("formatted error should show empty line as context")
	}

	if !strings.Contains(formatted, "   3 | line3") {
		t.Errorf("formatted error should show error line")
	}

	t.Logf("Formatted error:\n%s", formatted)
}

// TestFormatParseError_TabCharacters tests handling of tab characters.
func TestFormatParseError_TabCharacters(t *testing.T) {
	sourceText := "line1\n\tindented\nline3"

	err := parser.NewParseError(
		parser.SyntaxError,
		"test.csl",
		2,
		5,
		"error on indented line",
	)

	formatted := parser.FormatParseError(err, sourceText)

	// Should preserve the tab character
	if !strings.Contains(formatted, "\tindented") {
		t.Errorf("formatted error should preserve tab character")
	}

	if !strings.Contains(formatted, "^") {
		t.Errorf("formatted error should include caret")
	}

	t.Logf("Formatted error:\n%s", formatted)
}
