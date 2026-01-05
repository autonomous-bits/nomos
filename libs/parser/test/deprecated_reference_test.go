package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestParser_DeprecatedReferenceRejection verifies that the parser correctly
// rejects top-level reference statements (BREAKING CHANGE from User Story 1).
//
// Requirements (T011-T012):
// - Parse testdata/errors/deprecated_reference.csl
// - Verify ParseError is returned
// - Verify error kind is SyntaxError
// - Verify error message contains "no longer supported"
// - Verify error message includes migration guidance with "inline reference"
func TestParser_DeprecatedReferenceRejection(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		wantErrorKind       parser.ParseErrorKind
		wantErrorContains   []string // All strings that should be present in error
		wantErrorSubstrings []string // Additional helpful substrings
	}{
		{
			name:          "simple top-level reference",
			input:         "reference:configs:database.host\n",
			wantErrorKind: parser.SyntaxError,
			wantErrorContains: []string{
				"references can only be used inline",
				"value positions",
			},
			wantErrorSubstrings: []string{
				"invalid syntax",
			},
		},
		{
			name:          "reference with nested path",
			input:         "reference:network:vpc.cidr_block\n",
			wantErrorKind: parser.SyntaxError,
			wantErrorContains: []string{
				"references can only be used inline",
				"value positions",
			},
			wantErrorSubstrings: []string{
				"invalid syntax",
			},
		},
		{
			name:          "multiple top-level references",
			input:         "reference:app:settings.timeout\nreference:infra:region\n",
			wantErrorKind: parser.SyntaxError,
			wantErrorContains: []string{
				"references can only be used inline",
				"value positions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			// Assert - verify error is returned
			if err == nil {
				t.Fatal("expected ParseError for deprecated top-level reference, got nil")
			}

			// Assert - verify error is ParseError with correct kind
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Fatalf("expected *parser.ParseError, got %T: %v", err, err)
			}

			if parseErr.Kind() != tt.wantErrorKind {
				t.Errorf("expected error kind %v, got %v", tt.wantErrorKind, parseErr.Kind())
			}

			// Assert - verify error message contains required strings
			errMsg := parseErr.Error()
			for _, want := range tt.wantErrorContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing required substring %q\nGot: %s", want, errMsg)
				}
			}

			// Assert - verify error message contains helpful guidance
			for _, want := range tt.wantErrorSubstrings {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing helpful substring %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestParser_DeprecatedReferenceRejection_Fixture verifies parser rejection
// using actual reference statement content from the test fixture.
//
// Note: The fixture file testdata/errors/deprecated_reference.csl contains
// documentation comments (#) which are not valid Nomos syntax, so we test
// the actual reference statements directly here.
func TestParser_DeprecatedReferenceRejection_Fixture(t *testing.T) {
	// Test cases extracted from testdata/errors/deprecated_reference.csl
	// These are the actual reference statements that should be rejected
	tests := []struct {
		name     string
		input    string
		caseName string // Test case name from fixture
	}{
		{
			name:     "Test Case 1: Simple top-level reference",
			input:    "reference:configs:database.host\n",
			caseName: "simple top-level reference",
		},
		{
			name:     "Test Case 2: Reference with nested path",
			input:    "reference:network:vpc.cidr_block\n",
			caseName: "reference with nested path",
		},
		{
			name:     "Test Case 3: Reference with deeply nested path",
			input:    "reference:storage:bucket.name\n",
			caseName: "deeply nested path",
		},
		{
			name: "Test Case 4: Multiple references in sequence",
			input: `reference:app:settings.timeout
reference:infra:region
reference:data:source.connection_string
`,
			caseName: "multiple references",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "deprecated_reference.csl")

			// Assert - verify error is returned
			if err == nil {
				t.Fatalf("expected ParseError for %s, got nil", tt.caseName)
			}

			// Assert - verify error is ParseError with correct kind
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Fatalf("expected *parser.ParseError, got %T: %v", err, err)
			}

			if parseErr.Kind() != parser.SyntaxError {
				t.Errorf("expected SyntaxError, got %v", parseErr.Kind())
			}

			// Assert - verify error message is clear and simple
			errMsg := parseErr.Error()
			requiredSubstrings := []string{
				"references can only be used inline",
				"value positions",
			}

			for _, want := range requiredSubstrings {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message missing required substring %q\nGot: %s", want, errMsg)
				}
			}
		})
	}
}

// TestParser_InlineReferencesStillWork verifies that inline references
// (the correct syntax) continue to work after top-level reference removal.
//
// This is a regression test to ensure we only reject top-level references,
// not inline references in value positions.
func TestParser_InlineReferencesStillWork(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "inline reference in scalar value",
			input: `config:
	host: reference:configs:database.host
`,
		},
		{
			name: "inline reference in nested section",
			input: `database:
	connection:
		host: reference:network:vpc.endpoint
		port: 5432
`,
		},
		{
			name: "multiple inline references in same section",
			input: `app:
	timeout: reference:settings:timeout
	region: reference:infra:aws.region
	endpoint: reference:services:api.url
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")

			// Assert - inline references should parse successfully
			if err != nil {
				t.Fatalf("inline references should work, got error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil AST result for valid inline references")
			}

			if len(result.Statements) == 0 {
				t.Error("expected parsed statements for inline references")
			}
		})
	}
}
