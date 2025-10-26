package converter

import (
	"reflect"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func TestASTToData_EmptyAST(t *testing.T) {
	tree := &ast.AST{
		Statements: []ast.Stmt{},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	result, err := ASTToData(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestASTToData_NilAST(t *testing.T) {
	result, err := ASTToData(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestASTToData_SimpleSectionWithStringLiterals(t *testing.T) {
	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SectionDecl{
				Name: "config",
				Entries: map[string]ast.Expr{
					"host": &ast.StringLiteral{Value: "localhost"},
					"port": &ast.StringLiteral{Value: "8080"},
				},
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	result, err := ASTToData(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"config": map[string]any{
			"host": "localhost",
			"port": "8080",
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ASTToData() = %v, want %v", result, expected)
	}
}

func TestASTToData_MultipleSections(t *testing.T) {
	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SectionDecl{
				Name: "database",
				Entries: map[string]ast.Expr{
					"name": &ast.StringLiteral{Value: "mydb"},
				},
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
			&ast.SectionDecl{
				Name: "server",
				Entries: map[string]ast.Expr{
					"port": &ast.StringLiteral{Value: "9000"},
				},
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	result, err := ASTToData(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"database": map[string]any{
			"name": "mydb",
		},
		"server": map[string]any{
			"port": "9000",
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ASTToData() = %v, want %v", result, expected)
	}
}

func TestASTToData_WithReferenceExpr(t *testing.T) {
	refExpr := &ast.ReferenceExpr{
		Alias:      "network",
		Path:       []string{"vpc", "cidr"},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SectionDecl{
				Name: "config",
				Entries: map[string]ast.Expr{
					"cidr": refExpr,
				},
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	result, err := ASTToData(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ReferenceExpr should be preserved as-is
	configMap := result["config"].(map[string]any)
	if configMap["cidr"] != refExpr {
		t.Errorf("expected ReferenceExpr to be preserved, got %v", configMap["cidr"])
	}
}

func TestASTToData_IgnoresNonSectionStatements(t *testing.T) {
	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.ImportStmt{
				Alias:      "base",
				Path:       "base.csl",
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
			&ast.SectionDecl{
				Name: "config",
				Entries: map[string]ast.Expr{
					"key": &ast.StringLiteral{Value: "value"},
				},
				SourceSpan: ast.SourceSpan{Filename: "test.csl"},
			},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	result, err := ASTToData(tree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have config section, import is ignored
	expected := map[string]any{
		"config": map[string]any{
			"key": "value",
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ASTToData() = %v, want %v", result, expected)
	}
}
