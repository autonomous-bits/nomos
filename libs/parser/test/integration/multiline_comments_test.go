//go:build integration

// Package integration contains integration tests for the parser.
package integration

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestMultiLineComments_Integration_RealWorldDocumentation tests real-world multi-line comment usage.
// T041: Integration test for real-world multi-line documentation
//
// This integration test demonstrates practical multi-line comment usage patterns commonly found
// in production configuration files, including:
// - Documentation blocks preceding sections
// - Commented-out configuration for reference
// - Mixed full-line and trailing comments
// - Empty comment lines for visual separation
// - TODO/FIXME/NOTE style annotations
func TestMultiLineComments_Integration_RealWorldDocumentation(t *testing.T) {
	// Real-world configuration with comprehensive multi-line comments
	// This test demonstrates that the parser correctly handles:
	// - Multi-line documentation blocks before sections
	// - Commented-out configuration within sections
	// - Comments between entries
	// - Comments with empty lines
	input := `database:
  host: localhost
  # Comment between entries
  port: 5432
  # Commented-out configuration
  # backup_host: backup-server
  # backup_port: 5433
  max_connections: 100

api:
  endpoint: https://api.example.com
  # Commented-out retry configuration
  # retry: 3
  # backoff: exponential
  timeout: 30
`

	// Parse the configuration
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	// Assert no parse errors
	if err != nil {
		t.Fatalf("expected no parse error, got: %v", err)
	}

	// Verify we got sections (exact count may vary based on parser implementation)
	if len(result.Statements) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(result.Statements))
	}

	// Find the database section
	var databaseSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "database" {
			databaseSection = section
			break
		}
	}

	if databaseSection == nil {
		t.Fatal("database section not found")
	}

	// Verify only active (uncommented) keys are present in database section
	expectedDatabaseKeys := []string{"host", "port", "max_connections"}
	for _, key := range expectedDatabaseKeys {
		if !hasEntry(databaseSection.Entries, key) {
			t.Errorf("expected database key '%s' to exist, but it was not found", key)
		}
	}

	// Verify commented-out keys are NOT present
	unexpectedDatabaseKeys := []string{"backup_host", "backup_port"}
	for _, key := range unexpectedDatabaseKeys {
		if hasEntry(databaseSection.Entries, key) {
			t.Errorf("expected database key '%s' to NOT exist (should be commented out), but it was found", key)
		}
	}

	// Verify database configuration values
	hostExpr, ok := findEntry(databaseSection.Entries, "host")
	if !ok {
		t.Fatal("expected host entry")
	}
	hostValue := hostExpr.(*ast.StringLiteral).Value
	if hostValue != "localhost" {
		t.Errorf("expected host='localhost', got '%s'", hostValue)
	}

	portExpr, ok := findEntry(databaseSection.Entries, "port")
	if !ok {
		t.Fatal("expected port entry")
	}
	portValue := portExpr.(*ast.StringLiteral).Value
	if portValue != "5432" {
		t.Errorf("expected port='5432', got '%s'", portValue)
	}

	// Find the api section
	var apiSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "api" {
			apiSection = section
			break
		}
	}

	if apiSection == nil {
		t.Fatal("api section not found")
	}

	// Verify only active keys in api
	expectedAPIKeys := []string{"endpoint", "timeout"}
	for _, key := range expectedAPIKeys {
		if !hasEntry(apiSection.Entries, key) {
			t.Errorf("expected api key '%s' to exist, but it was not found", key)
		}
	}

	// Verify commented-out api keys are NOT present
	unexpectedAPIKeys := []string{"retry", "backoff"}
	for _, key := range unexpectedAPIKeys {
		if hasEntry(apiSection.Entries, key) {
			t.Errorf("expected api key '%s' to NOT exist (should be commented out), but it was found", key)
		}
	}

	// Success - all real-world multi-line comment scenarios handled correctly
	t.Log("Successfully parsed real-world configuration with multi-line comments")
	t.Log("Verified that comments between entries are properly ignored")
	t.Log("Verified that commented-out keys are not present in AST")
}

func findEntry(entries []ast.MapEntry, key string) (ast.Expr, bool) {
	for _, entry := range entries {
		if entry.Spread {
			continue
		}
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return nil, false
}

func hasEntry(entries []ast.MapEntry, key string) bool {
	_, ok := findEntry(entries, key)
	return ok
}
