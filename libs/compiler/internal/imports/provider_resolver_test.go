package imports

import (
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func TestExtractImports_FromAST(t *testing.T) {
	// Create a simple AST with source and section
	// Note: Import statements have been removed from the language
	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SourceDecl{
				Alias: "files",
				Type:  "file",
				Config: map[string]ast.Expr{
					"directory": &ast.StringLiteral{Value: "./configs"},
				},
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

	// Verify data
	if extracted.Data == nil {
		t.Fatal("expected data, got nil")
	}
	if _, ok := extracted.Data["database"]; !ok {
		t.Error("expected 'database' section in data")
	}
}
