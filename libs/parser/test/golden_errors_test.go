// Package parser_test contains integration tests for error scenarios with golden file verification.
package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/internal/testutil"
)

// TestGolden_ErrorScenarios tests that parsing error cases produces expected error messages.
func TestGolden_ErrorScenarios(t *testing.T) {
	errorsDir := "../testdata/errors"
	goldenDir := "../testdata/golden/errors"

	// Files that are known to NOT trigger errors in current parser implementation
	// These represent cases that are actually valid syntax
	knownValid := map[string]bool{
		"unicode_context.csl": true, // Valid unicode is allowed
	}

	// Find all error test files
	entries, err := os.ReadDir(errorsDir)
	if err != nil {
		t.Fatalf("failed to read errors directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".csl") {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			fixturePath := filepath.Join(errorsDir, name)
			goldenPath := filepath.Join(goldenDir, name+".error.json")

			// Skip files that are actually valid syntax
			if knownValid[name] {
				t.Skipf("VALID: %s is valid syntax, not an error case", name)
				return
			}

			// Parse (expecting error)
			_, err := parser.ParseFile(fixturePath)
			if err == nil {
				t.Fatalf("expected parse error for %s, got nil", name)
			}

			// Serialize error details to JSON
			errorData := map[string]interface{}{
				"filename": fixturePath,
				"error":    err.Error(),
			}

			if parseErr, ok := err.(interface {
				Kind() string
				Line() int
				Column() int
				Message() string
			}); ok {
				errorData["kind"] = parseErr.Kind()
				errorData["line"] = parseErr.Line()
				errorData["column"] = parseErr.Column()
				errorData["message"] = parseErr.Message()
			}

			actualJSON, err := testutil.CanonicalJSON(errorData)
			if err != nil {
				t.Fatalf("failed to serialize error to JSON: %v", err)
			}

			// Read or create golden file
			expectedJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Logf("Golden file not found at %s, writing actual output", goldenPath)
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
					t.Fatalf("failed to create golden directory: %v", err)
				}
				if err := os.WriteFile(goldenPath, actualJSON, 0644); err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				t.Skip("Generated golden file, re-run test to verify")
			}

			// Compare (note: error messages may vary, so we focus on structure)
			if !testutil.CompareJSON(actualJSON, expectedJSON) {
				t.Errorf("Error JSON does not match golden file for %s.\nExpected:\n%s\n\nActual:\n%s",
					name, string(expectedJSON), string(actualJSON))
			}
		})
	}
}

// TestGolden_NegativeFixtures tests negative fixtures and ensures errors are returned.
func TestGolden_NegativeFixtures(t *testing.T) {
	negativeDir := "../testdata/fixtures/negative"

	// Files that represent valid syntax or cases we cannot/should not validate at parse time.
	// These are documented in detail in docs/VALIDATION_GAPS.md
	knownValid := map[string]bool{
		// Duplicate detection requires scope-aware analysis for nested structures,
		// key shadowing, and merge semantics. Implemented in compiler, not parser.
		"duplicate_key.csl": true,

		// Import path is optional: "import:alias" OR "import:alias:path"
		// Path resolution happens in compiler during import resolution phase.
		"incomplete_import.csl": true,

		// Non-indented content after section creates empty section + new top-level statement.
		// Parser does not enforce indentation-based scoping (unlike Python).
		// This is syntactically valid but potentially confusing.
		"invalid_indentation.csl": true,

		// Unknown identifiers (not source/import/reference) are treated as section declarations.
		// This is by design to allow arbitrary user-defined section names.
		// Validation of section names would require schema/provider knowledge (compiler concern).
		"unknown_statement.csl": true,
	}

	entries, err := os.ReadDir(negativeDir)
	if err != nil {
		t.Fatalf("failed to read negative fixtures directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".csl") {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			fixturePath := filepath.Join(negativeDir, name)

			// Skip files that are actually valid or cannot be validated at parse time
			if knownValid[name] {
				t.Skipf("VALID: %s represents valid syntax or deferred validation (see docs/VALIDATION_GAPS.md)", name)
				return
			}

			// Parse (expecting error)
			_, err := parser.ParseFile(fixturePath)
			if err == nil {
				t.Errorf("expected parse error for negative fixture %s, got nil", name)
			} else {
				t.Logf("Expected error for %s: %v", name, err)
			}
		})
	}
}
