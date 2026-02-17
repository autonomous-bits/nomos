package ast

import "testing"

func TestASTNodes(t *testing.T) {
	span := SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10}

	tests := []struct {
		name string
		node Node
	}{
		{"AST", &AST{SourceSpan: span}},
		{"SpreadStmt", &SpreadStmt{SourceSpan: span}},
		{"SourceDecl", &SourceDecl{SourceSpan: span}},
		{"SectionDecl", &SectionDecl{SourceSpan: span}},
		{"PathExpr", &PathExpr{SourceSpan: span}},
		{"IdentExpr", &IdentExpr{SourceSpan: span}},
		{"StringLiteral", &StringLiteral{SourceSpan: span}},
		{"ReferenceExpr", &ReferenceExpr{SourceSpan: span}},
		{"MapExpr", &MapExpr{SourceSpan: span}},
		{"ListExpr", &ListExpr{SourceSpan: span}},
		{"MarkedExpr", &MarkedExpr{SourceSpan: span}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s := tt.node.Span(); s != span {
				t.Errorf("Span() = %v, want %v", s, span)
			}
			tt.node.node() // Exercise the marker method (unexported but accessible in package test)

			// Check specific interface implementations if any
			if _, ok := tt.node.(Stmt); ok {
				tt.node.(Stmt).stmt()
			}
			if _, ok := tt.node.(Expr); ok {
				tt.node.(Expr).expr()
			}
		})
	}
}
