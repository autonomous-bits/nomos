// Package parser_test contains integration tests for the parser public API.
package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/internal/testutil"
)

// TestGolden_SimpleFile tests that parsing simple.csl produces the expected AST.
func TestGolden_SimpleFile(t *testing.T) {
	// Arrange
	fixturePath := "../testdata/fixtures/simple.csl"
	goldenPath := "../testdata/golden/simple.csl.json"

	// Act
	result, err := parser.ParseFile(fixturePath)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Serialize to canonical JSON
	actualJSON, err := testutil.CanonicalJSON(result)
	if err != nil {
		t.Fatalf("failed to serialize AST to JSON: %v", err)
	}

	// Read golden file
	//nolint:gosec // G304: goldenPath is controlled test fixture path
	expectedJSON, err := os.ReadFile(goldenPath)
	if err != nil || os.Getenv("UPDATE_GOLDEN") == "true" {
		// If golden file doesn't exist or update requested, write it
		if os.Getenv("UPDATE_GOLDEN") == "true" {
			t.Logf("Updating golden file at %s", goldenPath)
		} else {
			t.Logf("Golden file not found at %s, writing actual output", goldenPath)
		}
		if err := os.WriteFile(goldenPath, actualJSON, 0600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		if os.Getenv("UPDATE_GOLDEN") != "true" {
			t.Skip("Generated golden file, re-run test to verify")
		}
		// If updating, we continue to verify (it should pass now)
		expectedJSON = actualJSON
	}

	// Compare
	if !testutil.CompareJSON(actualJSON, expectedJSON) {
		t.Errorf("AST JSON does not match golden file.\nExpected:\n%s\n\nActual:\n%s",
			string(expectedJSON), string(actualJSON))
	}
}

// TestGolden_AllFixtures runs golden tests for all fixtures in testdata/fixtures/.
func TestGolden_AllFixtures(t *testing.T) {
	fixturesDir := "../testdata/fixtures"
	goldenDir := "../testdata/golden"

	// Find all .csl files
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("failed to read fixtures directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".csl") {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			fixturePath := filepath.Join(fixturesDir, name)
			goldenPath := filepath.Join(goldenDir, name+".json")

			// Parse
			result, err := parser.ParseFile(fixturePath)
			if err != nil {
				t.Fatalf("expected no error parsing %s, got %v", name, err)
			}

			// Serialize
			actualJSON, err := testutil.CanonicalJSON(result)
			if err != nil {
				t.Fatalf("failed to serialize AST to JSON: %v", err)
			}

			// Read or create golden file
			//nolint:gosec // G304: goldenPath is controlled test fixture path
			expectedJSON, err := os.ReadFile(goldenPath)
			if err != nil || os.Getenv("UPDATE_GOLDEN") == "true" {
				// If golden file doesn't exist or update requested, write it
				if os.Getenv("UPDATE_GOLDEN") == "true" {
					t.Logf("Updating golden file at %s", goldenPath)
				} else {
					t.Logf("Golden file not found at %s, writing actual output", goldenPath)
				}
				if err := os.WriteFile(goldenPath, actualJSON, 0600); err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				if os.Getenv("UPDATE_GOLDEN") != "true" {
					t.Skip("Generated golden file, re-run test to verify")
				}
				expectedJSON = actualJSON
			}

			// Compare
			if !testutil.CompareJSON(actualJSON, expectedJSON) {
				t.Errorf("AST JSON does not match golden file for %s.\nExpected:\n%s\n\nActual:\n%s",
					name, string(expectedJSON), string(actualJSON))
			}
		})
	}
}
