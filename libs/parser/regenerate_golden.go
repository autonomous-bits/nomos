//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// normalizeFilenames walks the AST and normalizes filename paths to match test expectations.
// Tests run from test/ directory and use ../testdata/fixtures/ format.
func normalizeFilenames(node ast.Node, fixturesDir string) {
	if node == nil {
		return
	}

	// Normalize the span filename
	span := node.Span()
	if strings.HasPrefix(span.Filename, fixturesDir) {
		span.Filename = "../" + span.Filename
	}

	// Update the node with normalized span (we need to update the original)
	switch n := node.(type) {
	case *ast.AST:
		n.SourceSpan = span
		for _, stmt := range n.Statements {
			normalizeFilenames(stmt, fixturesDir)
		}
	case *ast.SourceDecl:
		n.SourceSpan = span
		for _, expr := range n.Config {
			normalizeFilenames(expr, fixturesDir)
		}
	case *ast.ImportStmt:
		n.SourceSpan = span
	case *ast.SectionDecl:
		n.SourceSpan = span
		if n.Value != nil {
			normalizeFilenames(n.Value, fixturesDir)
		}
		for _, expr := range n.Entries {
			normalizeFilenames(expr, fixturesDir)
		}
	case *ast.StringLiteral:
		n.SourceSpan = span
	case *ast.ReferenceExpr:
		n.SourceSpan = span
	case *ast.MapExpr:
		n.SourceSpan = span
		for _, expr := range n.Entries {
			normalizeFilenames(expr, fixturesDir)
		}
	case *ast.ListExpr:
		n.SourceSpan = span
		for _, elem := range n.Elements {
			normalizeFilenames(elem, fixturesDir)
		}
	case *ast.PathExpr:
		n.SourceSpan = span
	case *ast.IdentExpr:
		n.SourceSpan = span
	}
}

func main() {
	fixturesDir := "testdata/fixtures"
	goldenDir := "testdata/golden"

	// Walk through all fixture files
	err := filepath.Walk(fixturesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".csl") {
			return nil
		}

		// Skip negative fixtures (they're supposed to error)
		if strings.Contains(path, "/negative/") {
			return nil
		}

		fmt.Printf("Processing: %s\n", path)

		// Parse the fixture file
		ast, parseErr := parser.ParseFile(path)
		if parseErr != nil {
			fmt.Printf("  Skipping (parse error): %v\n", parseErr)
			return nil
		}

		// Normalize paths in the AST to match test expectations
		// Tests run from test/ directory and use ../testdata/fixtures/ format
		normalizeFilenames(ast, fixturesDir)

		// Calculate golden file path
		rel, err := filepath.Rel(fixturesDir, path)
		if err != nil {
			return err
		}
		goldenPath := filepath.Join(goldenDir, rel+".json")

		// Ensure golden directory exists
		goldenDirPath := filepath.Dir(goldenPath)
		if err := os.MkdirAll(goldenDirPath, 0755); err != nil {
			return err
		}

		// Marshal AST to JSON
		jsonData, err := json.MarshalIndent(ast, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal AST for %s: %v", path, err)
		}

		// Write golden file
		if err := os.WriteFile(goldenPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write golden file %s: %v", goldenPath, err)
		}

		fmt.Printf("  ✓ Updated: %s\n", goldenPath)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nGolden files regenerated successfully!")
}
