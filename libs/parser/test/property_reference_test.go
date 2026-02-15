// Package parser_test contains tests for the property reference syntax feature.
// This file implements T058-T060 (User Story 3: Property References - TDD phase).
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseReferenceExpr_PropertyReference tests parsing of property reference syntax @alias:path.to.property.
// T058: Test property reference @base:config.api.url parses correctly
// T059: Test deeply nested property path @base:config.a.b.c.d.e
// T060: Test property reference in value context (assignment)
func TestParseReferenceExpr_PropertyReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ast.ReferenceExpr
		wantErr bool
		errMsg  string
	}{
		{
			name:  "T058: single property reference",
			input: "@base:config.api.url",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "api", "url"},
			},
			wantErr: false,
		},
		{
			name:  "T058: two-level property path",
			input: "@shared:config.database.host",
			want: &ast.ReferenceExpr{
				Alias: "shared",
				Path:  []string{"config", "database", "host"},
			},
			wantErr: false,
		},
		{
			name:  "T058: simple scalar property",
			input: "@base:app.version",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "version"},
			},
			wantErr: false,
		},
		{
			name:  "T059: deeply nested property (3 levels)",
			input: "@base:config.server.http.port",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "server", "http", "port"},
			},
			wantErr: false,
		},
		{
			name:  "T059: deeply nested property (4 levels)",
			input: "@base:config.app.server.tls.cert",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "app", "server", "tls", "cert"},
			},
			wantErr: false,
		},
		{
			name:  "T059: deeply nested property (5 levels)",
			input: "@base:config.a.b.c.d.e",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "a", "b", "c", "d", "e"},
			},
			wantErr: false,
		},
		{
			name:  "T059: ultra-deep nesting (7 levels)",
			input: "@base:config.level1.level2.level3.level4.level5.level6.level7",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "level1", "level2", "level3", "level4", "level5", "level6", "level7"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// T060: Test property reference in value context (assignment)
			// Parse a section containing the reference as a value
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
			valueExpr, ok := findEntry(section.Entries, "value")
			if !ok {
				t.Fatal("expected 'value' entry in section")
			}

			// T060: Should be a ReferenceExpr (property references work in value context)
			refExpr, ok := valueExpr.(*ast.ReferenceExpr)
			if !ok {
				t.Fatalf("expected ReferenceExpr, got %T", valueExpr)
			}

			// Verify fields match expected property path
			if refExpr.Alias != tt.want.Alias {
				t.Errorf("Alias = %q, want %q", refExpr.Alias, tt.want.Alias)
			}
			if len(refExpr.Path) != len(tt.want.Path) {
				t.Errorf("Path length = %d, want %d (Path: %v, want: %v)",
					len(refExpr.Path), len(tt.want.Path), refExpr.Path, tt.want.Path)
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
