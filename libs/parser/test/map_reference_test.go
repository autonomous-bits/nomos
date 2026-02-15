// Package parser_test contains tests for the map reference syntax feature.
// This file implements T042-T043 (User Story 2: Map References).
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseReferenceExpr_MapReference tests parsing of map reference syntax @alias:path.to.map.
// T042: Test parseReferenceExpr with map path in libs/parser/parser_test.go
// T043: Test map reference creates ReferenceExpr with correct Path segments
func TestParseReferenceExpr_MapReference(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ast.ReferenceExpr
		wantErr bool
		errMsg  string
	}{
		{
			name:  "T042: single level map path",
			input: "@base:config:database",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "database"},
			},
			wantErr: false,
		},
		{
			name:  "T042: two-level map path",
			input: "@base:config:app.server",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "app", "server"},
			},
			wantErr: false,
		},
		{
			name:  "T042: deep nested map path (3+ segments)",
			input: "@base:config:app.server.config",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "app", "server", "config"},
			},
			wantErr: false,
		},
		{
			name:  "T043: verify Path field contains correct segments",
			input: "@prod:settings:infra.network.vpc",
			want: &ast.ReferenceExpr{
				Alias: "prod",
				Path:  []string{"settings", "infra", "network", "vpc"},
			},
			wantErr: false,
		},
		{
			name:  "T043: single segment path",
			input: "@env:vars:database",
			want: &ast.ReferenceExpr{
				Alias: "env",
				Path:  []string{"vars", "database"},
			},
			wantErr: false,
		},
		{
			name:  "T043: four-level nesting",
			input: "@cfg:app:server.http.tls.certificates",
			want: &ast.ReferenceExpr{
				Alias: "cfg",
				Path:  []string{"app", "server", "http", "tls", "certificates"},
			},
			wantErr: false,
		},
		{
			name:  "map path with hyphens",
			input: "@base:config:app-settings.server-config",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "app-settings", "server-config"},
			},
			wantErr: false,
		},
		{
			name:  "map path with underscores",
			input: "@base:config:app_config.server_settings",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "app_config", "server_settings"},
			},
			wantErr: false,
		},
		{
			name:  "map path with numbers",
			input: "@base:config:db1.pool2",
			want: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"config", "db1", "pool2"},
			},
			wantErr: false,
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

			// T043: Verify Path field contains the correct segments in order
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

			// T043: Verify SourceSpan is correct
			if refExpr.SourceSpan.Filename == "" {
				t.Error("expected non-empty SourceSpan.Filename")
			}
			if refExpr.SourceSpan.StartLine == 0 {
				t.Error("expected non-zero SourceSpan.StartLine")
			}
		})
	}
}

// TestParseReferenceExpr_MapReference_NestedLevels tests map references at various nesting depths.
// T043: Test various nesting levels (1, 2, 3+ segments)
func TestParseReferenceExpr_MapReference_NestedLevels(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPath   []string
		wantLevels int
	}{
		{
			name:       "1-level nesting",
			input:      "@base:config:database",
			wantPath:   []string{"config", "database"},
			wantLevels: 2,
		},
		{
			name:       "2-level nesting",
			input:      "@base:config:app.server",
			wantPath:   []string{"config", "app", "server"},
			wantLevels: 3,
		},
		{
			name:       "3-level nesting",
			input:      "@base:config:app.server.config",
			wantPath:   []string{"config", "app", "server", "config"},
			wantLevels: 4,
		},
		{
			name:       "4-level nesting",
			input:      "@base:config:infra.network.vpc.subnets",
			wantPath:   []string{"config", "infra", "network", "vpc", "subnets"},
			wantLevels: 5,
		},
		{
			name:       "5-level nesting",
			input:      "@base:config:app.server.http.tls.certificates",
			wantPath:   []string{"config", "app", "server", "http", "tls", "certificates"},
			wantLevels: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `config:
	value: ` + tt.input + `
`
			reader := strings.NewReader(input)
			result, err := parser.Parse(reader, "test.csl")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			refExpr := section.Entries["value"].(*ast.ReferenceExpr)

			// Verify nesting level
			if len(refExpr.Path) != tt.wantLevels {
				t.Errorf("nesting levels = %d, want %d", len(refExpr.Path), tt.wantLevels)
			}

			// Verify each path segment
			for i, want := range tt.wantPath {
				if i >= len(refExpr.Path) {
					t.Errorf("Path missing segment at index %d (want %q)", i, want)
					continue
				}
				if refExpr.Path[i] != want {
					t.Errorf("Path[%d] = %q, want %q", i, refExpr.Path[i], want)
				}
			}
		})
	}
}
