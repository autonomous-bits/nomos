package imports

import (
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func TestExtractImports_FromAST(t *testing.T) {
	// Create a simple AST with source, import, and section
	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SourceDecl{
				Alias: "files",
				Type:  "file",
				Config: map[string]ast.Expr{
					"file": &ast.StringLiteral{Value: "base.csl"},
				},
			},
			&ast.ImportStmt{
				Alias: "files",
				Path:  "",
			},
			&ast.SectionDecl{
				Name: "database",
				Entries: map[string]ast.Expr{
					"host": &ast.StringLiteral{Value: "localhost"},
				},
			},
		},
	}

	// Extract
	extracted, err := ExtractImports(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sources
	if len(extracted.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(extracted.Sources))
	}
	if len(extracted.Sources) > 0 {
		src := extracted.Sources[0]
		if src.Alias != "files" {
			t.Errorf("expected alias 'files', got %q", src.Alias)
		}
		if src.Type != "file" {
			t.Errorf("expected type 'file', got %q", src.Type)
		}
	}

	// Verify imports
	if len(extracted.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(extracted.Imports))
	}
	if len(extracted.Imports) > 0 {
		imp := extracted.Imports[0]
		if imp.Alias != "files" {
			t.Errorf("expected import alias 'files', got %q", imp.Alias)
		}
		if len(imp.Path) != 0 {
			t.Errorf("expected empty path (entire file), got %v", imp.Path)
		}
	}

	// Verify data
	if extracted.Data == nil {
		t.Fatal("expected data, got nil")
	}
	if _, ok := extracted.Data["database"]; !ok {
		t.Error("expected 'database' section in data")
	}
}
