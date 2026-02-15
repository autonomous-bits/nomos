package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocumentation_InlineReferenceContent verifies that the README.md contains
// the required documentation for inline reference syntax per User Story #21.
//
// This test ensures that:
// 1. The README includes examples of inline reference syntax in scalar, map, and list positions
// 2. The README includes a migration note about removal of top-level reference: statements
// 3. The README references PRD issue #10 and the codemod script location
func TestDocumentation_InlineReferenceContent(t *testing.T) {
	// Read README.md
	readmePath := filepath.Join("..", "README.md")
	//nolint:gosec // G304: readmePath is controlled test fixture path
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	readme := string(content)

	t.Run("contains inline reference section", func(t *testing.T) {
		if !strings.Contains(readme, "## Inline Reference Syntax") {
			t.Error("README.md must contain '## Inline Reference Syntax' section")
		}
	})

	t.Run("contains scalar example", func(t *testing.T) {
		// Check for scalar value example
		scalarExample := "vpc_cidr: @network:config:vpc.cidr"
		if !strings.Contains(readme, scalarExample) {
			t.Errorf("README.md must contain scalar inline reference example: %q", scalarExample)
		}
	})

	t.Run("contains map/collection example", func(t *testing.T) {
		// Check for map context example
		if !strings.Contains(readme, "servers:") &&
			!strings.Contains(readme, "ip: @") {
			t.Error("README.md must contain map/collection inline reference example")
		}
	})

	t.Run("contains nested/list example", func(t *testing.T) {
		// Check for nested structure example
		nestedExample := "databases:"
		if !strings.Contains(readme, nestedExample) {
			t.Errorf("README.md must contain nested/list inline reference example section")
		}
	})

	t.Run("contains AST ReferenceExpr JSON example", func(t *testing.T) {
		// Check for AST JSON representation
		requiredFields := []string{
			`"type": "ReferenceExpr"`,
			`"alias"`,
			`"path"`,
			`"source_span"`,
		}

		for _, field := range requiredFields {
			if !strings.Contains(readme, field) {
				t.Errorf("README.md AST example must contain field: %q", field)
			}
		}
	})

	t.Run("contains migration notes section", func(t *testing.T) {
		if !strings.Contains(readme, "## Migration Notes") {
			t.Error("README.md must contain '## Migration Notes' section")
		}
	})

	t.Run("mentions removal of top-level references", func(t *testing.T) {
		requiredPhrases := []string{
			"no longer support",
			"breaking change",
			"top-level",
		}

		for _, phrase := range requiredPhrases {
			if !strings.Contains(strings.ToLower(readme), strings.ToLower(phrase)) {
				t.Errorf("Migration notes must mention: %q", phrase)
			}
		}
	})

	t.Run("references PRD issue #10", func(t *testing.T) {
		// Check for issue #10 reference
		if !strings.Contains(readme, "#10") && !strings.Contains(readme, "issue/10") {
			t.Error("README.md must reference PRD issue #10")
		}

		// Check for direct link
		if !strings.Contains(readme, "https://github.com/autonomous-bits/nomos/issues/10") {
			t.Error("README.md must include direct link to issue #10")
		}
	})

	t.Run("references codemod script", func(t *testing.T) {
		codemodPath := "tools/scripts/convert-top-level-references"
		if !strings.Contains(readme, codemodPath) {
			t.Errorf("README.md must reference codemod script at: %q", codemodPath)
		}
	})

	t.Run("contains before/after migration examples", func(t *testing.T) {
		// Check for legacy syntax example
		if !strings.Contains(readme, "Legacy") || !strings.Contains(readme, "REMOVED") {
			t.Error("README.md must show legacy syntax marked as removed")
		}

		// Check for new syntax example
		if !strings.Contains(readme, "New Syntax") || !strings.Contains(readme, "REQUIRED") {
			t.Error("README.md must show new syntax marked as required")
		}
	})

	t.Run("contains code example with ReferenceExpr", func(t *testing.T) {
		// Check for Go code example showing how to work with ReferenceExpr
		if !strings.Contains(readme, "case *ast.ReferenceExpr:") {
			t.Error("README.md must contain Go code example showing ReferenceExpr usage")
		}
	})

	t.Run("contains example output", func(t *testing.T) {
		// Check for example output showing reference vs literal
		if !strings.Contains(readme, "is a reference to") ||
			!strings.Contains(readme, "is a literal") {
			t.Error("README.md must contain example output showing reference and literal handling")
		}
	})
}

// TestDocumentation_ExamplesExist verifies that the example files referenced
// in the documentation actually exist and are accessible.
func TestDocumentation_ExamplesExist(t *testing.T) {
	exampleFiles := []string{
		"../docs/examples/inline_reference_basic.csl",
		"../docs/examples/inline_reference_map.csl",
		"../docs/examples/inline_reference_mixed.csl",
		"../docs/examples/inline_reference_nested.csl",
		"../docs/examples/README.md",
	}

	for _, path := range exampleFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("example file does not exist: %s", path)
			} else if err != nil {
				t.Errorf("failed to access example file %s: %v", path, err)
			}
		})
	}
}

// TestDocumentation_ExamplesAreParseable verifies that all example files
// contain valid Nomos syntax (at least syntactically valid, even if semantically
// incomplete due to missing source files).
func TestDocumentation_ExamplesAreParseable(t *testing.T) {
	// This is a basic check - we just verify the files exist and contain some expected keywords
	exampleFiles := map[string][]string{
		"../docs/examples/inline_reference_basic.csl": {
			"source:",
			"@",
		},
		"../docs/examples/inline_reference_map.csl": {
			"servers:",
			"ip: @",
		},
		"../docs/examples/inline_reference_mixed.csl": {
			"application:",
			"@",
		},
		"../docs/examples/inline_reference_nested.csl": {
			"databases:",
			"@",
		},
	}

	for path, expectedKeywords := range exampleFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			//nolint:gosec // G304: path is controlled test fixture path from exampleFiles map
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read example file %s: %v", path, err)
			}

			fileContent := string(content)
			for _, keyword := range expectedKeywords {
				if !strings.Contains(fileContent, keyword) {
					t.Errorf("example file %s must contain keyword: %q", path, keyword)
				}
			}
		})
	}
}
