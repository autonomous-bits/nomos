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

func TestASTToData_ListExprConversion(t *testing.T) {
	refExpr := &ast.ReferenceExpr{
		Alias:      "network",
		Path:       []string{"vpc", "cidr"},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	listExpr := &ast.ListExpr{
		Elements: []ast.Expr{
			&ast.StringLiteral{Value: "web01"},
			refExpr,
			&ast.MapExpr{
				Entries: map[string]ast.Expr{
					"port": &ast.StringLiteral{Value: "8080"},
				},
			},
			&ast.ListExpr{
				Elements: []ast.Expr{
					&ast.StringLiteral{Value: "nested"},
				},
			},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SectionDecl{
				Name: "servers",
				Entries: map[string]ast.Expr{
					"targets": listExpr,
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

	serversMap := result["servers"].(map[string]any)
	listValue, ok := serversMap["targets"].([]any)
	if !ok {
		t.Fatalf("expected list conversion, got %T", serversMap["targets"])
	}

	if listValue[1] != refExpr {
		t.Errorf("expected ReferenceExpr to be preserved in list, got %v", listValue[1])
	}

	mapValue, ok := listValue[2].(map[string]any)
	if !ok {
		t.Fatalf("expected map conversion in list, got %T", listValue[2])
	}
	if mapValue["port"] != "8080" {
		t.Errorf("expected nested map value '8080', got %v", mapValue["port"])
	}

	nestedList, ok := listValue[3].([]any)
	if !ok {
		t.Fatalf("expected nested list conversion, got %T", listValue[3])
	}
	if len(nestedList) != 1 || nestedList[0] != "nested" {
		t.Errorf("expected nested list to contain 'nested', got %v", nestedList)
	}
}

func TestASTToData_ListExprWithPathAndIdent(t *testing.T) {
	listExpr := &ast.ListExpr{
		Elements: []ast.Expr{
			&ast.PathExpr{Components: []string{"network", "subnet"}},
			&ast.IdentExpr{Name: "region"},
		},
		SourceSpan: ast.SourceSpan{Filename: "test.csl"},
	}

	tree := &ast.AST{
		Statements: []ast.Stmt{
			&ast.SectionDecl{
				Name: "config",
				Entries: map[string]ast.Expr{
					"items": listExpr,
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

	configMap := result["config"].(map[string]any)
	listValue, ok := configMap["items"].([]any)
	if !ok {
		t.Fatalf("expected list conversion, got %T", configMap["items"])
	}

	if len(listValue) != 2 {
		t.Fatalf("expected 2 list elements, got %d", len(listValue))
	}
	if listValue[0] != "network.subnet" {
		t.Errorf("expected path conversion to 'network.subnet', got %v", listValue[0])
	}
	if listValue[1] != "region" {
		t.Errorf("expected ident conversion to 'region', got %v", listValue[1])
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
