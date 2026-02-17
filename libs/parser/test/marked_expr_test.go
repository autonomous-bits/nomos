package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParse_MarkedExpr_SimpleValue tests basic marked value parsing.
func TestParse_MarkedExpr_SimpleValue(t *testing.T) {
	input := `config:
	secret: "my-secret"!
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := result.Statements[0].(*ast.SectionDecl)
	entryMap := entryMapHelper(section.Entries)

	markedExpr, ok := entryMap["secret"].(*ast.MarkedExpr)
	if !ok {
		t.Fatalf("expected MarkedExpr, got %T", entryMap["secret"])
	}

	literal, ok := markedExpr.Expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral inside MarkedExpr, got %T", markedExpr.Expr)
	}

	if literal.Value != "my-secret" {
		t.Errorf("expected value 'my-secret', got '%s'", literal.Value)
	}
}

// TestParse_MarkedExpr_UnquotedValue tests marked unquoted value parsing.
func TestParse_MarkedExpr_UnquotedValue(t *testing.T) {
	input := `config:
	secret: my-secret!
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := result.Statements[0].(*ast.SectionDecl)
	entryMap := entryMapHelper(section.Entries)

	markedExpr, ok := entryMap["secret"].(*ast.MarkedExpr)
	if !ok {
		t.Fatalf("expected MarkedExpr, got %T", entryMap["secret"])
	}

	literal, ok := markedExpr.Expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral inside MarkedExpr, got %T", markedExpr.Expr)
	}

	if literal.Value != "my-secret" {
		t.Errorf("expected value 'my-secret', got '%s'", literal.Value)
	}
}

// TestParse_MarkedExpr_Reference tests marked reference parsing.
func TestParse_MarkedExpr_Reference(t *testing.T) {
	input := `config:
	secret: @env:SECRET!
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := result.Statements[0].(*ast.SectionDecl)
	entryMap := entryMapHelper(section.Entries)

	markedExpr, ok := entryMap["secret"].(*ast.MarkedExpr)
	if !ok {
		t.Fatalf("expected MarkedExpr, got %T", entryMap["secret"])
	}

	ref, ok := markedExpr.Expr.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected ReferenceExpr inside MarkedExpr, got %T", markedExpr.Expr)
	}

	if ref.Alias != "env" {
		t.Errorf("expected alias 'env', got '%s'", ref.Alias)
	}
}

// TestParse_MarkedExpr_WithSpace tests parsing with space before bang.
func TestParse_MarkedExpr_WithSpace(t *testing.T) {
	input := `config:
	secret: "my-secret" !
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := result.Statements[0].(*ast.SectionDecl)
	entryMap := entryMapHelper(section.Entries)

	markedExpr, ok := entryMap["secret"].(*ast.MarkedExpr)
	if !ok {
		t.Fatalf("expected MarkedExpr, got %T", entryMap["secret"])
	}

	literal, ok := markedExpr.Expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral inside MarkedExpr, got %T", markedExpr.Expr)
	}

	if literal.Value != "my-secret" {
		t.Errorf("expected value 'my-secret', got '%s'", literal.Value)
	}
}

// Helper to convert entries to map for easy testing
func entryMapHelper(entries []ast.MapEntry) map[string]ast.Expr {
	m := make(map[string]ast.Expr)
	for _, entry := range entries {
		if !entry.Spread {
			m[entry.Key] = entry.Value
		}
	}
	return m
}
