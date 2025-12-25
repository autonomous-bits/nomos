package ast_test

import (
	"encoding/json"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestReferenceExpr_Constructor verifies ReferenceExpr can be constructed with required fields.
func TestReferenceExpr_Constructor(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		path       []string
		sourceSpan ast.SourceSpan
	}{
		{
			name:  "simple reference",
			alias: "network",
			path:  []string{"vpc", "cidr"},
			sourceSpan: ast.SourceSpan{
				Filename:  "test.csl",
				StartLine: 1,
				StartCol:  10,
				EndLine:   1,
				EndCol:    35,
			},
		},
		{
			name:  "single path component",
			alias: "config",
			path:  []string{"key"},
			sourceSpan: ast.SourceSpan{
				Filename:  "app.csl",
				StartLine: 5,
				StartCol:  5,
				EndLine:   5,
				EndCol:    20,
			},
		},
		{
			name:  "deep path",
			alias: "source",
			path:  []string{"a", "b", "c", "d"},
			sourceSpan: ast.SourceSpan{
				Filename:  "deep.csl",
				StartLine: 10,
				StartCol:  1,
				EndLine:   10,
				EndCol:    25,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &ast.ReferenceExpr{
				Alias:      tt.alias,
				Path:       tt.path,
				SourceSpan: tt.sourceSpan,
			}

			// Verify fields
			if ref.Alias != tt.alias {
				t.Errorf("Alias = %v, want %v", ref.Alias, tt.alias)
			}
			if len(ref.Path) != len(tt.path) {
				t.Fatalf("Path length = %v, want %v", len(ref.Path), len(tt.path))
			}
			for i, component := range tt.path {
				if ref.Path[i] != component {
					t.Errorf("Path[%d] = %v, want %v", i, ref.Path[i], component)
				}
			}
			if ref.SourceSpan.Filename != tt.sourceSpan.Filename {
				t.Errorf("SourceSpan.Filename = %v, want %v", ref.SourceSpan.Filename, tt.sourceSpan.Filename)
			}
		})
	}
}

// TestReferenceExpr_ImplementsNode verifies ReferenceExpr implements the Node interface.
func TestReferenceExpr_ImplementsNode(t *testing.T) {
	span := ast.SourceSpan{
		Filename:  "test.csl",
		StartLine: 1,
		StartCol:  1,
		EndLine:   1,
		EndCol:    10,
	}

	ref := &ast.ReferenceExpr{
		Alias:      "test",
		Path:       []string{"key"},
		SourceSpan: span,
	}

	// Verify it implements Node
	var _ ast.Node = ref

	// Verify Span() returns the correct value
	gotSpan := ref.Span()
	if gotSpan.Filename != span.Filename {
		t.Errorf("Span().Filename = %v, want %v", gotSpan.Filename, span.Filename)
	}
	if gotSpan.StartLine != span.StartLine {
		t.Errorf("Span().StartLine = %v, want %v", gotSpan.StartLine, span.StartLine)
	}
}

// TestReferenceExpr_ImplementsExpr verifies ReferenceExpr implements the Expr interface.
func TestReferenceExpr_ImplementsExpr(_ *testing.T) {
	// Verify it implements Expr (compile-time check)
	var _ ast.Expr = &ast.ReferenceExpr{}
}

// TestExpr_TypeSwitch verifies all Expr variants can be handled in type switches.
func TestExpr_TypeSwitch(t *testing.T) {
	span := ast.SourceSpan{
		Filename:  "test.csl",
		StartLine: 1,
		StartCol:  1,
		EndLine:   1,
		EndCol:    10,
	}

	tests := []struct {
		name     string
		expr     ast.Expr
		wantType string
	}{
		{
			name: "StringLiteral",
			expr: &ast.StringLiteral{
				Value:      "test",
				SourceSpan: span,
			},
			wantType: "StringLiteral",
		},
		{
			name: "ReferenceExpr",
			expr: &ast.ReferenceExpr{
				Alias:      "network",
				Path:       []string{"vpc", "cidr"},
				SourceSpan: span,
			},
			wantType: "ReferenceExpr",
		},
		{
			name: "PathExpr",
			expr: &ast.PathExpr{
				Components: []string{"a", "b"},
				SourceSpan: span,
			},
			wantType: "PathExpr",
		},
		{
			name: "IdentExpr",
			expr: &ast.IdentExpr{
				Name:       "ident",
				SourceSpan: span,
			},
			wantType: "IdentExpr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotType string
			switch tt.expr.(type) {
			case *ast.StringLiteral:
				gotType = "StringLiteral"
			case *ast.ReferenceExpr:
				gotType = "ReferenceExpr"
			case *ast.PathExpr:
				gotType = "PathExpr"
			case *ast.IdentExpr:
				gotType = "IdentExpr"
			default:
				t.Fatalf("unexpected type: %T", tt.expr)
			}

			if gotType != tt.wantType {
				t.Errorf("type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// TestReferenceExpr_JSONSerialization verifies ReferenceExpr marshals and unmarshals correctly.
func TestReferenceExpr_JSONSerialization(t *testing.T) {
	original := &ast.ReferenceExpr{
		Alias: "network",
		Path:  []string{"vpc", "cidr"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 5,
			StartCol:  10,
			EndLine:   5,
			EndCol:    35,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var decoded ast.ReferenceExpr
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if decoded.Alias != original.Alias {
		t.Errorf("Alias = %v, want %v", decoded.Alias, original.Alias)
	}
	if len(decoded.Path) != len(original.Path) {
		t.Fatalf("Path length = %v, want %v", len(decoded.Path), len(original.Path))
	}
	for i, component := range original.Path {
		if decoded.Path[i] != component {
			t.Errorf("Path[%d] = %v, want %v", i, decoded.Path[i], component)
		}
	}
	if decoded.SourceSpan.Filename != original.SourceSpan.Filename {
		t.Errorf("SourceSpan.Filename = %v, want %v", decoded.SourceSpan.Filename, original.SourceSpan.Filename)
	}
	if decoded.SourceSpan.StartLine != original.SourceSpan.StartLine {
		t.Errorf("SourceSpan.StartLine = %v, want %v", decoded.SourceSpan.StartLine, original.SourceSpan.StartLine)
	}
}
