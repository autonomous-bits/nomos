// Package parser_test contains tests for the root reference syntax feature.
// This file implements T019-T022 (TDD tests written BEFORE implementation).
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseReferenceExpr_RootReference tests parsing of root reference syntax @alias:.
// T019: Test parseReferenceExpr with root syntax @base:.
// T020: Test root reference creates ReferenceExpr with empty Path slice.
// T021: Test root reference with various alias names.
// T022: Test malformed root reference (missing dot) returns error.
func TestParseReferenceExpr_RootReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ast.ReferenceExpr
		wantErr bool
		errMsg  string
	}{
		{
			name:  "T019: root reference with dot",
			input: "@base:.",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{},
			},
			wantErr: false,
		},
		{
			name:  "T020: root reference creates empty Path slice",
			input: "@config:.",
			want: &ast.ReferenceExpr{
				Alias: "config",
				Path:  []string{},
			},
			wantErr: false,
		},
		{
			name:  "T021: root reference with hyphens in names",
			input: "@my-config:.",
			want: &ast.ReferenceExpr{
				Alias: "my-config",
				Path:  []string{},
			},
			wantErr: false,
		},
		{
			name:  "T021: root reference with underscores",
			input: "@base_config:.",
			want: &ast.ReferenceExpr{
				Alias: "base_config",
				Path:  []string{},
			},
			wantErr: false,
		},
		{
			name:  "T021: root reference with numbers",
			input: "@config1:.",
			want: &ast.ReferenceExpr{
				Alias: "config1",
				Path:  []string{},
			},
			wantErr: false,
		},
		{
			name:    "T022: malformed - missing dot for root",
			input:   "@base:",
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "T022: malformed - missing path separator",
			input:   "@base",
			wantErr: true,
			errMsg:  "must use format @alias:path",
		},
		{
			name:    "T022: malformed - extra colon",
			input:   "@base::.",
			wantErr: true,
			errMsg:  "empty segment",
		},
		{
			name:    "T022: malformed - empty alias",
			input:   "@:.",
			wantErr: true,
			errMsg:  "alias cannot be empty",
		},
		{
			name:    "T022: malformed - empty path segment",
			input:   "@base::database",
			wantErr: true,
			errMsg:  "empty segment",
		},
		{
			name:    "T022: malformed - whitespace in reference",
			input:   "@base :.",
			wantErr: true,
			errMsg:  "whitespace not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse a section containing the reference
			input := `config:
	value: ` + tt.input + `
`
			reader := strings.NewReader(input)
			result, err := parser.Parse(reader, "test.csl")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			// Should parse successfully
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			// Find the section
			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			section, ok := result.Statements[0].(*ast.SectionDecl)
			if !ok {
				t.Fatalf("expected SectionDecl, got %T", result.Statements[0])
			}

			// Extract the value expression
			valueExpr, ok := section.Entries["value"]
			if !ok {
				t.Fatal("expected 'value' entry in section")
			}

			// Should be a ReferenceExpr
			refExpr, ok := valueExpr.(*ast.ReferenceExpr)
			if !ok {
				t.Fatalf("expected ReferenceExpr, got %T", valueExpr)
			}

			// Verify fields
			if refExpr.Alias != tt.want.Alias {
				t.Errorf("Alias = %q, want %q", refExpr.Alias, tt.want.Alias)
			}
			if len(refExpr.Path) != len(tt.want.Path) {
				t.Errorf("Path length = %d, want %d", len(refExpr.Path), len(tt.want.Path))
			}
			for i, component := range tt.want.Path {
				if i < len(refExpr.Path) && refExpr.Path[i] != component {
					t.Errorf("Path[%d] = %q, want %q", i, refExpr.Path[i], component)
				}
			}

			// Verify SourceSpan is set
			if refExpr.SourceSpan.Filename == "" {
				t.Error("expected non-empty SourceSpan.Filename")
			}
		})
	}
}

// TestParseReferenceExpr_PropertyPath tests parsing of property paths with new syntax.
// This extends the root reference tests to cover non-root paths.
func TestParseReferenceExpr_PropertyPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ast.ReferenceExpr
		wantErr bool
	}{
		{
			name:  "single property",
			input: "@base:database:host",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "host"},
			},
		},
		{
			name:  "nested property path",
			input: "@base:database:pool.max_connections",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "pool", "max_connections"},
			},
		},
		{
			name:  "deeply nested path",
			input: "@base:config:server.http.tls.cert",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "server", "http", "tls", "cert"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `config:
	value: ` + tt.input + `
`
			reader := strings.NewReader(input)
			result, err := parser.Parse(reader, "test.csl")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			refExpr := section.Entries["value"].(*ast.ReferenceExpr)

			if refExpr.Alias != tt.want.Alias {
				t.Errorf("Alias = %q, want %q", refExpr.Alias, tt.want.Alias)
			}
			if len(refExpr.Path) != len(tt.want.Path) {
				t.Errorf("Path = %v, want %v", refExpr.Path, tt.want.Path)
			}
		})
	}
}
