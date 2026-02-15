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
				Entries: []ast.MapEntry{
					{Key: "host", Value: &ast.StringLiteral{Value: "localhost"}},
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

// TestExprToValue tests the expression to value conversion logic
func TestExprToValue(t *testing.T) {
	tests := []struct {
		name      string
		expr      ast.Expr
		want      any
		wantErr   bool
		errSubstr string
	}{
		{
			name: "StringLiteral",
			expr: &ast.StringLiteral{Value: "hello"},
			want: "hello",
		},
		{
			name: "IdentExpr",
			expr: &ast.IdentExpr{Name: "foo"},
			want: "foo",
		},
		{
			name: "PathExpr_Single",
			expr: &ast.PathExpr{Components: []string{"foo"}},
			want: "foo",
		},
		{
			name: "PathExpr_Multiple",
			expr: &ast.PathExpr{Components: []string{"foo", "bar"}},
			want: "foo.bar",
		},
		{
			name: "PathExpr_Empty",
			expr: &ast.PathExpr{Components: []string{}},
			want: "",
		},
		{
			name: "ListExpr_Empty",
			expr: &ast.ListExpr{Elements: []ast.Expr{}},
			want: []any{}, // Empty slice, check specifically in verify
		},
		{
			name: "ListExpr_Values",
			expr: &ast.ListExpr{Elements: []ast.Expr{
				&ast.StringLiteral{Value: "a"},
				&ast.StringLiteral{Value: "b"},
			}},
			want: []any{"a", "b"},
		},
		{
			name: "MapExpr_Empty",
			expr: &ast.MapExpr{Entries: []ast.MapEntry{}},
			want: map[string]any{},
		},
		{
			name: "MapExpr_Values",
			expr: &ast.MapExpr{Entries: []ast.MapEntry{
				{Key: "k1", Value: &ast.StringLiteral{Value: "v1"}},
				{Key: "k2", Value: &ast.StringLiteral{Value: "v2"}},
			}},
			want: map[string]any{"k1": "v1", "k2": "v2"},
		},
		{
			name:      "NilExpr",
			expr:      nil,
			wantErr:   true,
			errSubstr: "nil expression",
		},
		{
			name: "MapExpr_SpreadError",
			expr: &ast.MapExpr{Entries: []ast.MapEntry{
				{Spread: true, Value: &ast.IdentExpr{Name: "foo"}},
			}},
			wantErr:   true,
			errSubstr: "spread entries are not supported",
		},
		{
			name: "MapExpr_EmptyKeyError",
			expr: &ast.MapExpr{Entries: []ast.MapEntry{
				{Key: "", Value: &ast.StringLiteral{Value: "val"}},
			}},
			wantErr:   true,
			errSubstr: "map entry key cannot be empty",
		},
		{
			name: "MapExpr_ValueConversionError",
			expr: &ast.MapExpr{Entries: []ast.MapEntry{
				{Key: "k", Value: &ast.MapExpr{Entries: []ast.MapEntry{
					{Spread: true, Value: &ast.IdentExpr{Name: "foo"}}, // valid AST but fails conversion
				}}},
			}},
			wantErr:   true,
			errSubstr: "failed to convert map key",
		},
		{
			name: "ListExpr_ElementConversionError",
			expr: &ast.ListExpr{Elements: []ast.Expr{
				&ast.MapExpr{Entries: []ast.MapEntry{
					{Spread: true, Value: &ast.IdentExpr{Name: "foo"}}, // valid AST but fails conversion
				}},
			}},
			wantErr:   true,
			errSubstr: "failed to convert list element",
		},
		{
			name: "ReferenceExpr",
			expr: &ast.ReferenceExpr{Alias: "aws", Path: []string{"region"}},
			// We check specific type in verify as references are returned as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exprToValue(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("exprToValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != nil && tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("exprToValue() error = %v, want substring %q", err, tt.errSubstr)
				}
				return
			}

			// Special handling for ReferenceExpr
			if _, ok := tt.expr.(*ast.ReferenceExpr); ok {
				if _, ok := got.(*ast.ReferenceExpr); !ok {
					t.Errorf("exprToValue() = %T, want *ast.ReferenceExpr", got)
				}
				return
			}

			if !deepEqual(got, tt.want) {
				t.Errorf("exprToValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// deepEqual checks if two values are deeply equal for our limited types
func deepEqual(a, b any) bool {
	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !deepEqual(v, vb[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
