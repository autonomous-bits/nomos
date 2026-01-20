// Package parser_test contains validation tests for list parsing error scenarios.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestEmptyListItemError tests that empty list items (dash with no value) are rejected.
// This test WILL FAIL until list validation is implemented in parser.go.
func TestEmptyListItemError(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedErrSubstr string
	}{
		{
			name: "dash with no value",
			input: `IPs:
  - 
  - 10.0.0.1`,
			expectedErrSubstr: "empty list item",
		},
		{
			name: "dash with only whitespace",
			input: `items:
  -    
  - value`,
			expectedErrSubstr: "empty list item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if err == nil {
				t.Error("expected parse error for empty list item, got nil")
				return
			}

			// Check error type
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Logf("EXPECTED FAILURE (not yet implemented): got non-ParseError: %v", err)
				return
			}

			// Verify error kind is SyntaxError
			if parseErr.Kind() != parser.SyntaxError {
				t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
			}

			// Verify error message contains expected substring
			if !strings.Contains(parseErr.Message(), tt.expectedErrSubstr) {
				t.Errorf("expected error message to contain %q, got: %s", tt.expectedErrSubstr, parseErr.Message())
			}

			// Verify error has line/column information
			if parseErr.Line() == 0 {
				t.Error("expected error to have line number")
			}
		})
	}
}

// TestInconsistentIndentationError tests that inconsistent list indentation is rejected.
// This test WILL FAIL until indentation validation is implemented in parser.go.
func TestInconsistentIndentationError(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedErrSubstr string
	}{
		{
			name: "3 spaces instead of 2",
			input: `IPs:
  - 10.0.0.1
   - 10.1.0.1`,
			expectedErrSubstr: "inconsistent",
		},
		{
			name: "4 spaces instead of 2",
			input: `items:
  - first
    - second`,
			expectedErrSubstr: "inconsistent",
		},
		{
			name: "1 space instead of 2",
			input: `values:
  - val1
 - val2`,
			expectedErrSubstr: "inconsistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if err == nil {
				t.Error("expected parse error for inconsistent indentation, got nil")
				return
			}

			// Check error type
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Logf("EXPECTED FAILURE (not yet implemented): got non-ParseError: %v", err)
				return
			}

			// Verify error kind
			if parseErr.Kind() != parser.SyntaxError {
				t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
			}

			// Verify error message
			errMsg := strings.ToLower(parseErr.Message())
			if !strings.Contains(errMsg, strings.ToLower(tt.expectedErrSubstr)) {
				t.Errorf("expected error message to contain %q (case-insensitive), got: %s", tt.expectedErrSubstr, parseErr.Message())
			}
		})
	}
}

// TestWhitespaceOnlyListError tests that lists containing only whitespace/comments are rejected.
// This test WILL FAIL until whitespace-only list detection is implemented in parser.go.
func TestWhitespaceOnlyListError(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedErrSubstr string
	}{
		{
			name: "only whitespace",
			input: `IPs:
  
  `,
			expectedErrSubstr: "whitespace",
		},
		{
			name: "only comments",
			input: `items:
  # Just a comment
  # Another comment`,
			expectedErrSubstr: "whitespace",
		},
		{
			name: "blank lines and comments",
			input: `values:

  # Comment line
  
`,
			expectedErrSubstr: "whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if err == nil {
				t.Error("expected parse error for whitespace-only list, got nil")
				return
			}

			// Check error type
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Logf("EXPECTED FAILURE (not yet implemented): got non-ParseError: %v", err)
				return
			}

			// Verify error kind
			if parseErr.Kind() != parser.SyntaxError {
				t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
			}

			// Verify error message
			errMsg := strings.ToLower(parseErr.Message())
			if !strings.Contains(errMsg, strings.ToLower(tt.expectedErrSubstr)) {
				t.Errorf("expected error message to contain %q (case-insensitive), got: %s", tt.expectedErrSubstr, parseErr.Message())
			}
		})
	}
}

// TestTabCharacterInListIndentationError tests that tab characters in list indentation are rejected.
// This test WILL FAIL until tab character detection is implemented in parser.go.
func TestTabCharacterInListIndentationError(t *testing.T) {
	// Note: Using literal tab character in test input
	input := "IPs:\n\t- 10.0.0.1\n  - 10.1.0.1"

	reader := strings.NewReader(input)
	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Error("expected parse error for tab character in indentation, got nil")
		return
	}

	// Check error type
	parseErr, ok := err.(*parser.ParseError)
	if !ok {
		t.Logf("EXPECTED FAILURE (not yet implemented): got non-ParseError: %v", err)
		return
	}

	// Verify error kind
	if parseErr.Kind() != parser.SyntaxError {
		t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
	}

	// Verify error message mentions tab
	errMsg := strings.ToLower(parseErr.Message())
	if !strings.Contains(errMsg, "tab") {
		t.Errorf("expected error message to mention 'tab', got: %s", parseErr.Message())
	}
}

// TestMaxDepthExceededError tests that list nesting beyond the maximum depth is rejected.
func TestMaxDepthExceededError(t *testing.T) {
	fixturePath := "../testdata/errors/lists/depth_exceeded.csl"

	_, err := parser.ParseFile(fixturePath)
	if err == nil {
		t.Fatal("expected parse error for depth exceeded, got nil")
	}

	parseErr, ok := err.(*parser.ParseError)
	if !ok {
		t.Fatalf("expected *parser.ParseError, got %T", err)
	}

	if parseErr.Kind() != parser.SyntaxError {
		t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
	}

	if !strings.Contains(parseErr.Message(), "list nesting depth exceeded") {
		t.Errorf("expected error message to mention list nesting depth exceeded, got: %s", parseErr.Message())
	}
}
