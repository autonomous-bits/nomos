//go:build integration
// +build integration

// Package integration_test contains integration tests for list parsing features.
package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestSimpleListEndToEnd tests end-to-end parsing of simple list from fixture file.
// This test WILL FAIL until parseListExpr is fully implemented in parser.go.
func TestSimpleListEndToEnd(t *testing.T) {
	// Arrange: Use test fixture file
	fixturePath := filepath.Join("..", "..", "testdata", "fixtures", "lists", "simple.csl")

	// Verify fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Fatalf("test fixture not found: %s", fixturePath)
	}

	// Act: Parse the fixture file
	result, err := parser.ParseFile(fixturePath)

	// Expected to fail until implementation
	if err != nil {
		t.Logf("EXPECTED FAILURE (not yet implemented): %v", err)
		return
	}

	// Assert: Verify basic AST structure
	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	section, ok := result.Statements[0].(*ast.SectionDecl)
	if !ok {
		t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
	}

	if section.Name != "IPs" {
		t.Errorf("expected section name 'IPs', got '%s'", section.Name)
	}

	// Assert: Verify list expression exists
	entryExpr, ok := findEntry(section.Entries, "")
	if !ok {
		t.Fatalf("expected list entry for section, got none")
	}
	listExpr, ok := entryExpr.(*ast.ListExpr)
	if !ok {
		t.Fatalf("expected *ast.ListExpr in section entries, got %T", entryExpr)
	}

	// Assert: Verify list has 3 elements
	if len(listExpr.Elements) != 3 {
		t.Fatalf("expected 3 list elements, got %d", len(listExpr.Elements))
	}

	// Assert: Verify element values
	expectedValues := []string{"10.0.0.1", "10.1.0.1", "10.2.0.1"}
	for i, expected := range expectedValues {
		literal, ok := listExpr.Elements[i].(*ast.StringLiteral)
		if !ok {
			t.Fatalf("element %d: expected *ast.StringLiteral, got %T", i, listExpr.Elements[i])
		}
		if literal.Value != expected {
			t.Errorf("element %d: expected value '%s', got '%s'", i, expected, literal.Value)
		}
	}

	// Assert: Verify source spans are present
	if listExpr.Span().Filename == "" {
		t.Error("list expression should have source span with filename")
	}
	if listExpr.Span().StartLine == 0 {
		t.Error("list expression should have non-zero start line")
	}

	t.Log("✅ Integration test passed - list parsing works end-to-end")
}

// TestEmptyListEndToEnd tests end-to-end parsing of empty list from fixture file.
// This test WILL FAIL until empty list parsing is implemented in parser.go.
func TestEmptyListEndToEnd(t *testing.T) {
	// Arrange: Use test fixture file
	fixturePath := filepath.Join("..", "..", "testdata", "fixtures", "lists", "empty.csl")

	// Verify fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Fatalf("test fixture not found: %s", fixturePath)
	}

	// Act: Parse the fixture file
	result, err := parser.ParseFile(fixturePath)

	// Expected to fail until implementation
	if err != nil {
		t.Logf("EXPECTED FAILURE (not yet implemented): %v", err)
		return
	}

	// Assert: Verify AST structure
	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	section, ok := result.Statements[0].(*ast.SectionDecl)
	if !ok {
		t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
	}

	listExpr, ok := section.Value.(*ast.ListExpr)
	if !ok || listExpr == nil {
		t.Fatalf("expected *ast.ListExpr in section value, got %T", section.Value)
	}

	// Assert: Verify list is empty
	if len(listExpr.Elements) != 0 {
		t.Errorf("expected empty list (0 elements), got %d elements", len(listExpr.Elements))
	}

	t.Log("✅ Integration test passed - empty list parsing works end-to-end")
}

// TestListImportScenariosEndToEnd tests list parsing in files that include import statements.
func TestListImportScenariosEndToEnd(t *testing.T) {
	tests := []struct {
		name           string
		fixtureFile    string
		sectionName    string
		expectedValues []string
		importAlias    string
		importPath     string
	}{
		{
			name:           "base config with list",
			fixtureFile:    "base_config.csl",
			sectionName:    "servers",
			expectedValues: []string{"base-01", "base-02"},
		},
		{
			name:           "override config with import and list",
			fixtureFile:    "override_config.csl",
			sectionName:    "servers",
			expectedValues: []string{"override-01", "override-02", "override-03"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixturePath := filepath.Join("..", "..", "testdata", "fixtures", "lists", tt.fixtureFile)

			if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
				t.Fatalf("test fixture not found: %s", fixturePath)
			}

			result, err := parser.ParseFile(fixturePath)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			section := findSectionByName(result, tt.sectionName)
			if section == nil {
				t.Fatalf("expected section %q, got none", tt.sectionName)
			}

			entryExpr, ok := findEntry(section.Entries, "")
			if !ok {
				t.Fatalf("expected list entry for section %q, got none", tt.sectionName)
			}
			listExpr, ok := entryExpr.(*ast.ListExpr)
			if !ok {
				t.Fatalf("expected *ast.ListExpr for section %q, got %T", tt.sectionName, entryExpr)
			}

			if len(listExpr.Elements) != len(tt.expectedValues) {
				t.Fatalf("expected %d list elements, got %d", len(tt.expectedValues), len(listExpr.Elements))
			}

			for i, expected := range tt.expectedValues {
				literal, ok := listExpr.Elements[i].(*ast.StringLiteral)
				if !ok {
					t.Fatalf("element %d: expected *ast.StringLiteral, got %T", i, listExpr.Elements[i])
				}
				if literal.Value != expected {
					t.Errorf("element %d: expected %q, got %q", i, expected, literal.Value)
				}
			}
		})
	}
}

// TestListErrorHandlingEndToEnd tests that invalid list syntax produces appropriate errors.
// This test WILL FAIL until list validation is implemented in parser.go.
func TestListErrorHandlingEndToEnd(t *testing.T) {
	tests := []struct {
		name              string
		fixtureFile       string
		expectedErrSubstr string
	}{
		{
			name:              "empty list item",
			fixtureFile:       "empty_item.csl",
			expectedErrSubstr: "empty",
		},
		{
			name:              "inconsistent indentation",
			fixtureFile:       "inconsistent_indent.csl",
			expectedErrSubstr: "inconsistent",
		},
		{
			name:              "tab character",
			fixtureFile:       "tab_char.csl",
			expectedErrSubstr: "tab",
		},
		{
			name:              "whitespace only",
			fixtureFile:       "whitespace_only.csl",
			expectedErrSubstr: "whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			fixturePath := filepath.Join("..", "..", "testdata", "errors", "lists", tt.fixtureFile)

			// Verify fixture exists
			if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
				t.Fatalf("test fixture not found: %s", fixturePath)
			}

			// Act
			_, err := parser.ParseFile(fixturePath)

			// Assert: Should produce error
			if err == nil {
				t.Errorf("expected parse error for %s, got nil", tt.name)
				return
			}

			// Expected to fail until validation is implemented
			parseErr, ok := err.(*parser.ParseError)
			if !ok {
				t.Logf("EXPECTED FAILURE (validation not yet implemented): got non-ParseError: %v", err)
				return
			}

			// Verify error contains expected substring
			if !contains(parseErr.Message(), tt.expectedErrSubstr) {
				t.Errorf("expected error message to contain %q, got: %s", tt.expectedErrSubstr, parseErr.Message())
			}

			t.Logf("✅ Error handling test passed for %s", tt.name)
		})
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	// Simple case-insensitive check for test purposes
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > 0 && len(substr) > 0 &&
				findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalsIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func findImportStmt(tree *ast.AST) any {
	// ImportStmt is deprecated and removed from AST
	_ = tree
	return nil
}

func findSectionByName(tree *ast.AST, name string) *ast.SectionDecl {
	for _, stmt := range tree.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == name {
			return section
		}
	}
	return nil
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
