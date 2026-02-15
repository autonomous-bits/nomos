// Package integration contains integration tests for the parser error model.
package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestParseError_Integration_InvalidCharacter tests parsing a file with invalid characters.
func TestParseError_Integration_InvalidCharacter(t *testing.T) {
	// Read test fixture
	content, err := os.ReadFile("../../testdata/errors/invalid_character.csl")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Parse and expect error
	reader := strings.NewReader(string(content))
	_, err = parser.Parse(reader, "invalid_character.csl")

	if err == nil {
		t.Fatal("expected parse error for invalid character, got nil")
	}

	// Verify error is a ParseError
	parseErr, ok := err.(*parser.ParseError)
	if !ok {
		t.Fatalf("expected *parser.ParseError, got %T", err)
	}

	// Verify error fields
	if parseErr.Filename() != "invalid_character.csl" {
		t.Errorf("expected filename 'invalid_character.csl', got %s", parseErr.Filename())
	}
	if parseErr.Line() != 1 {
		t.Errorf("expected line 1, got %d", parseErr.Line())
	}
	if parseErr.Kind() != parser.SyntaxError {
		t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
	}

	// Format and verify it contains expected elements
	formatted := parser.FormatParseError(err, string(content))

	// Check for machine-parseable prefix
	if !strings.Contains(formatted, "invalid_character.csl:1:") {
		t.Errorf("formatted error should contain file:line prefix, got:\n%s", formatted)
	}

	// Check for snippet
	if !strings.Contains(formatted, "invalid !!! syntax") {
		t.Errorf("formatted error should contain source line, got:\n%s", formatted)
	}

	// Check for caret
	if !strings.Contains(formatted, "^") {
		t.Errorf("formatted error should contain caret marker, got:\n%s", formatted)
	}

	t.Logf("Formatted error:\n%s", formatted)
}

// TestParseError_Integration_MissingColon tests parsing a file with missing colon.
func TestParseError_Integration_MissingColon(t *testing.T) {
	// The fixture "source\n\talias: 'test'" is actually treated as
	// skipped unknown line since 'source' without ':' is not recognized
	// So let's use inline test instead
	source := "invalid-section\n\tkey: 'value'"

	reader := strings.NewReader(source)
	_, err := parser.Parse(reader, "missing_colon.csl")

	if err == nil {
		t.Fatal("expected parse error for missing colon, got nil")
	}

	// Verify it's a ParseError
	parseErr, ok := err.(*parser.ParseError)
	if !ok {
		t.Fatalf("expected *parser.ParseError, got %T", err)
	}

	// Verify error kind
	if parseErr.Kind() != parser.SyntaxError {
		t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
	}

	// Format error
	formatted := parser.FormatParseError(err, source)

	t.Logf("Formatted error:\n%s", formatted)

	// Verify formatted error contains helpful information
	if !strings.Contains(formatted, "invalid-section") {
		t.Errorf("formatted error should show the problematic identifier, got:\n%s", formatted)
	}
} // TestParseError_Integration_FileNotFound tests I/O error handling.
func TestParseError_Integration_FileNotFound(t *testing.T) {
	_, err := parser.ParseFile("/nonexistent/path/file.csl")

	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	// Verify it's a ParseError with IOError kind
	parseErr, ok := err.(*parser.ParseError)
	if !ok {
		t.Fatalf("expected *parser.ParseError, got %T", err)
	}

	if parseErr.Kind() != parser.IOError {
		t.Errorf("expected IOError, got %v", parseErr.Kind())
	}

	if parseErr.Filename() != "/nonexistent/path/file.csl" {
		t.Errorf("expected filename '/nonexistent/path/file.csl', got %s", parseErr.Filename())
	}
}

// TestParseError_Integration_RealScenario simulates a real-world error scenario.
func TestParseError_Integration_RealScenario(t *testing.T) {
	// A realistic Nomos file with an error
	source := `source:
	alias: 'config'
	type: 'folder'
	path: './configs'

bad-section
	key: 'value'
`

	_, err := parser.Parse(strings.NewReader(source), "app.csl")

	if err == nil {
		t.Fatal("expected parse error, got nil")
	}

	// Format the error as would be shown to a user
	formatted := parser.FormatParseError(err, source)

	t.Logf("User-facing error:\n%s", formatted)

	// Verify error is helpful
	if !strings.Contains(formatted, "app.csl") {
		t.Errorf("error should mention filename")
	}
	if !strings.Contains(formatted, "bad-section") {
		t.Errorf("error should show the problematic line")
	}
	if !strings.Contains(formatted, "^") {
		t.Errorf("error should have caret pointing to problem")
	}
}

// TestParseError_Integration_ProgrammaticInspection tests that errors can be inspected programmatically.
func TestParseError_Integration_ProgrammaticInspection(t *testing.T) {
	source := "invalid !@# syntax"

	_, err := parser.Parse(strings.NewReader(source), "test.csl")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Programmatic error handling
	if parseErr, ok := err.(*parser.ParseError); ok {
		switch parseErr.Kind() {
		case parser.SyntaxError:
			t.Logf("Syntax error at %s:%d:%d: %s",
				parseErr.Filename(), parseErr.Line(), parseErr.Column(), parseErr.Message())
		case parser.LexError:
			t.Errorf("unexpected lex error")
		case parser.IOError:
			t.Errorf("unexpected I/O error")
		default:
			t.Errorf("unexpected error kind: %v", parseErr.Kind())
		}

		// Verify span is accessible
		span := parseErr.Span()
		if span.Filename != "test.csl" {
			t.Errorf("span filename should be 'test.csl', got %s", span.Filename)
		}
	} else {
		t.Errorf("expected *parser.ParseError for programmatic inspection, got %T", err)
	}
}

// TestFormatParseError_Integration_MultipleErrors tests formatting multiple errors.
func TestFormatParseError_Integration_MultipleErrors(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		desc   string
	}{
		{
			name:   "invalid character",
			source: "test !!!",
			desc:   "should show invalid character with caret",
		},
		{
			name:   "identifier without colon",
			source: "standalone\n\tkey: 'value'",
			desc:   "should show missing colon error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.Parse(strings.NewReader(tc.source), tc.name+".csl")

			if err == nil {
				t.Fatalf("%s: expected error, got nil", tc.desc)
			}

			formatted := parser.FormatParseError(err, tc.source)
			t.Logf("%s:\n%s\n", tc.desc, formatted)

			// All errors should have:
			// 1. File:line:col prefix
			// 2. A snippet showing context
			// 3. A caret marker
			if !strings.Contains(formatted, ".csl:") {
				t.Errorf("missing file:line:col prefix in:\n%s", formatted)
			}
			if !strings.Contains(formatted, "^") {
				t.Errorf("missing caret marker in:\n%s", formatted)
			}
		})
	}
}
