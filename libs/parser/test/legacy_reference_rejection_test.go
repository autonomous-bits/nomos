// Package parser_test contains tests for legacy reference statement rejection.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestParseLegacyTopLevelReference_Rejected tests that top-level reference: statements
// are rejected with a helpful error message directing users to inline reference syntax.
func TestParseLegacyTopLevelReference_Rejected(t *testing.T) {
	// Arrange
	fixturePath := "../testdata/fixtures/negative/legacy_top_level_reference.csl"

	// Act
	_, err := parser.ParseFile(fixturePath)

	// Assert - should return an error
	if err == nil {
		t.Fatal("expected parse error for legacy top-level reference statement, got nil")
	}

	// Assert - error message should mention migration to inline form
	errMsg := err.Error()
	if !strings.Contains(errMsg, "inline") && !strings.Contains(errMsg, "value position") {
		t.Errorf("expected error message to suggest inline reference form, got: %s", errMsg)
	}
}

// TestParseLegacyTopLevelReference_SingleStatement tests rejection of a single legacy reference.
func TestParseLegacyTopLevelReference_SingleStatement(t *testing.T) {
	// Arrange
	input := `reference:network:vpc.cidr`

	// Act
	_, err := parser.Parse(strings.NewReader(input), "test.csl")

	// Assert
	if err == nil {
		t.Fatal("expected parse error for legacy top-level reference statement, got nil")
	}

	// Error should suggest migration
	errMsg := err.Error()
	if !strings.Contains(errMsg, "inline") && !strings.Contains(errMsg, "value") {
		t.Errorf("expected error message to suggest inline reference syntax, got: %s", errMsg)
	}
}

// TestParseLegacyTopLevelReference_MultipleStatements tests rejection when multiple
// legacy references appear at the top level.
func TestParseLegacyTopLevelReference_MultipleStatements(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'network'
	type: 'folder'
	path: './config'

reference:network:vpc.cidr
reference:network:subnet.id

config:
	key: 'value'
`

	// Act
	_, err := parser.Parse(strings.NewReader(input), "test.csl")

	// Assert - should fail on first legacy reference
	if err == nil {
		t.Fatal("expected parse error for legacy top-level reference statement, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "inline") && !strings.Contains(errMsg, "value") {
		t.Errorf("expected error message to suggest inline reference syntax, got: %s", errMsg)
	}
}
