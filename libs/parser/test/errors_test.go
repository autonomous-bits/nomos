// Package parser_test contains unit tests for the error model.
package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestParseError_ImplementsError tests that ParseError implements error interface.
func TestParseError_ImplementsError(t *testing.T) {
	err := parser.NewParseError(parser.SyntaxError, "test.csl", 1, 5, "unexpected token")

	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}

	expectedPrefix := "test.csl:1:5:"
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Errorf("Error() should start with %q, got: %s", expectedPrefix, err.Error())
	}
}

// TestParseError_Fields tests that ParseError fields are accessible.
func TestParseError_Fields(t *testing.T) {
	tests := []struct {
		name            string
		kind            parser.ParseErrorKind
		filename        string
		line            int
		col             int
		message         string
		expectedErrText string
	}{
		{
			name:            "syntax error",
			kind:            parser.SyntaxError,
			filename:        "config.csl",
			line:            10,
			col:             5,
			message:         "expected ':'",
			expectedErrText: "config.csl:10:5: expected ':'",
		},
		{
			name:            "lex error",
			kind:            parser.LexError,
			filename:        "input.csl",
			line:            1,
			col:             1,
			message:         "unexpected character",
			expectedErrText: "input.csl:1:1: unexpected character",
		},
		{
			name:            "io error",
			kind:            parser.IOError,
			filename:        "missing.csl",
			line:            0,
			col:             0,
			message:         "file not found",
			expectedErrText: "missing.csl:0:0: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.NewParseError(tt.kind, tt.filename, tt.line, tt.col, tt.message)

			if err.Kind() != tt.kind {
				t.Errorf("Kind() = %v, want %v", err.Kind(), tt.kind)
			}
			if err.Filename() != tt.filename {
				t.Errorf("Filename() = %s, want %s", err.Filename(), tt.filename)
			}
			if err.Line() != tt.line {
				t.Errorf("Line() = %d, want %d", err.Line(), tt.line)
			}
			if err.Column() != tt.col {
				t.Errorf("Column() = %d, want %d", err.Column(), tt.col)
			}
			if err.Message() != tt.message {
				t.Errorf("Message() = %s, want %s", err.Message(), tt.message)
			}
			if err.Error() != tt.expectedErrText {
				t.Errorf("Error() = %s, want %s", err.Error(), tt.expectedErrText)
			}
		})
	}
}

// TestParseError_Span tests that Span() returns correct source location.
func TestParseError_Span(t *testing.T) {
	err := parser.NewParseError(parser.SyntaxError, "test.csl", 5, 10, "test error")

	span := err.Span()

	if span.Filename != "test.csl" {
		t.Errorf("Span().Filename = %s, want test.csl", span.Filename)
	}
	if span.StartLine != 5 {
		t.Errorf("Span().StartLine = %d, want 5", span.StartLine)
	}
	if span.StartCol != 10 {
		t.Errorf("Span().StartCol = %d, want 10", span.StartCol)
	}
	if span.EndLine != 5 {
		t.Errorf("Span().EndLine = %d, want 5", span.EndLine)
	}
	if span.EndCol != 10 {
		t.Errorf("Span().EndCol = %d, want 10", span.EndCol)
	}
}

// TestFormatParseError_BasicFormatting tests basic error formatting.
func TestFormatParseError_BasicFormatting(t *testing.T) {
	sourceText := "source:\n\talias: 'test'\n\ttype: 'folder'"
	err := parser.NewParseError(parser.SyntaxError, "test.csl", 1, 1, "invalid syntax")

	formatted := parser.FormatParseError(err, sourceText)

	// Check for machine-parseable prefix
	if !strings.Contains(formatted, "test.csl:1:1:") {
		t.Errorf("Formatted error should contain file:line:col prefix, got:\n%s", formatted)
	}

	// Check for message
	if !strings.Contains(formatted, "invalid syntax") {
		t.Errorf("Formatted error should contain message, got:\n%s", formatted)
	}

	// Check for snippet with line number
	if !strings.Contains(formatted, "   1 |") {
		t.Errorf("Formatted error should contain line number, got:\n%s", formatted)
	}

	// Check for caret marker
	if !strings.Contains(formatted, "^") {
		t.Errorf("Formatted error should contain caret marker, got:\n%s", formatted)
	}
}

// TestFormatParseError_WithContext tests error formatting with context lines.
func TestFormatParseError_WithContext(t *testing.T) {
	sourceText := `line one
line two with error
line three`
	err := parser.NewParseError(parser.SyntaxError, "test.csl", 2, 6, "unexpected token")

	formatted := parser.FormatParseError(err, sourceText)

	// Should show line 1 (context before)
	if !strings.Contains(formatted, "   1 | line one") {
		t.Errorf("Should include line before error, got:\n%s", formatted)
	}

	// Should show line 2 (error line)
	if !strings.Contains(formatted, "   2 | line two with error") {
		t.Errorf("Should include error line, got:\n%s", formatted)
	}

	// Should show line 3 (context after)
	if !strings.Contains(formatted, "   3 | line three") {
		t.Errorf("Should include line after error, got:\n%s", formatted)
	}

	// Caret should be at column 6
	lines := strings.Split(formatted, "\n")
	var caretLine string
	for _, line := range lines {
		if strings.Contains(line, "^") {
			caretLine = line
			break
		}
	}
	if caretLine == "" {
		t.Fatalf("No caret line found in:\n%s", formatted)
	}

	// Count spaces before caret (should be 5: col 6 means 5 chars before)
	expectedSpaces := "     |      ^"
	if !strings.Contains(caretLine, expectedSpaces) {
		t.Errorf("Caret position incorrect. Expected pattern %q in caret line, got: %q", expectedSpaces, caretLine)
	}
}

// TestFormatParseError_UnicodeSupporttests rune-aware column counting for multi-byte characters.
func TestFormatParseError_UnicodeSupport(t *testing.T) {
	// String with multi-byte UTF-8 characters
	sourceText := "Hello 世界 test"
	//             123456789... columns (rune-based)
	// Position 7 is after "Hello " and before "世"
	err := parser.NewParseError(parser.SyntaxError, "unicode.csl", 1, 7, "unicode error")

	formatted := parser.FormatParseError(err, sourceText)

	t.Logf("Formatted error:\n%s", formatted)

	// Check basic components
	if !strings.Contains(formatted, "unicode.csl:1:7:") {
		t.Errorf("Should contain file:line:col prefix, got:\n%s", formatted)
	}

	// Check for source line
	if !strings.Contains(formatted, "Hello 世界 test") {
		t.Errorf("Should contain source line with unicode, got:\n%s", formatted)
	}

	// Extract caret line
	lines := strings.Split(formatted, "\n")
	var caretLine string
	for _, line := range lines {
		if strings.Contains(line, "^") && !strings.Contains(line, "|") {
			continue // Skip source line if it contains ^
		}
		if strings.Contains(line, "^") {
			caretLine = line
			break
		}
	}

	if caretLine == "" {
		t.Fatalf("No caret line found in:\n%s", formatted)
	}

	// Verify caret is at the correct rune position (6 spaces for column 7)
	// "     | " (7 chars) + 6 spaces + "^"
	expectedPrefix := "     |       ^"
	if !strings.Contains(caretLine, expectedPrefix) {
		t.Errorf("Caret should be at rune position 6 (column 7).\nExpected pattern: %q\nGot: %q", expectedPrefix, caretLine)
	}
}

// TestFormatParseError_NonParseError tests formatting of non-ParseError errors.
func TestFormatParseError_NonParseError(t *testing.T) {
	err := errors.New("generic error")

	formatted := parser.FormatParseError(err, "")

	if formatted != "generic error" {
		t.Errorf("Should return original error string for non-ParseError, got: %s", formatted)
	}
}

// TestFormatParseError_WrappedParseError tests formatting of wrapped ParseError.
func TestFormatParseError_WrappedParseError(t *testing.T) {
	parseErr := parser.NewParseError(parser.SyntaxError, "test.csl", 1, 5, "syntax error")
	wrapped := &wrappedError{parseErr}

	sourceText := "test line"
	formatted := parser.FormatParseError(wrapped, sourceText)

	if !strings.Contains(formatted, "test.csl:1:5:") {
		t.Errorf("Should unwrap and format ParseError, got:\n%s", formatted)
	}
}

// wrappedError is a test helper that wraps an error.
type wrappedError struct {
	err error
}

func (w *wrappedError) Error() string {
	return w.err.Error()
}

func (w *wrappedError) Unwrap() error {
	return w.err
}

// TestParseErrorKind_String tests error kind string representation.
func TestParseErrorKind_String(t *testing.T) {
	tests := []struct {
		kind parser.ParseErrorKind
		want string
	}{
		{parser.LexError, "LexError"},
		{parser.SyntaxError, "SyntaxError"},
		{parser.IOError, "IOError"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("ParseErrorKind.String() = %s, want %s", got, tt.want)
			}
		})
	}
}
