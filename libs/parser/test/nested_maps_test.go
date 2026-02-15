package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func TestParseNestedMaps(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *ast.AST)
	}{
		{
			name: "simple nested map",
			input: `
config:
  database:
    host: 'localhost'
    port: 5432
`,
			wantErr: false,
			check: func(t *testing.T, tree *ast.AST) {
				if len(tree.Statements) != 1 {
					t.Fatalf("expected 1 statement, got %d", len(tree.Statements))
				}
				section := tree.Statements[0].(*ast.SectionDecl)
				if section.Name != "config" {
					t.Errorf("expected section name 'config', got %q", section.Name)
				}

				// Check for database key
				dbExpr, ok := findEntry(section.Entries, "database")
				if !ok {
					t.Fatal("expected 'database' key in section")
				}

				// Should be a MapExpr
				mapExpr, ok := dbExpr.(*ast.MapExpr)
				if !ok {
					t.Fatalf("expected MapExpr for database, got %T", dbExpr)
				}

				// Check nested entries
				if len(mapExpr.Entries) != 2 {
					t.Errorf("expected 2 entries in database map, got %d", len(mapExpr.Entries))
				}

				hostExpr, ok := findEntry(mapExpr.Entries, "host")
				if !ok {
					t.Error("expected 'host' in database map")
				}
				hostLit, ok := hostExpr.(*ast.StringLiteral)
				if !ok || hostLit.Value != "localhost" {
					t.Errorf("expected host='localhost', got %v", hostExpr)
				}

				portExpr, ok := findEntry(mapExpr.Entries, "port")
				if !ok {
					t.Error("expected 'port' in database map")
				}
				portLit, ok := portExpr.(*ast.StringLiteral)
				if !ok || portLit.Value != "5432" {
					t.Errorf("expected port='5432', got %v", portExpr)
				}
			},
		},
		{
			name: "deeply nested map",
			input: `
databases:
  primary:
    host: 'primary-db'
    connection:
      max_pool: 20
      timeout: 5000
`,
			wantErr: false,
			check: func(t *testing.T, tree *ast.AST) {
				section := tree.Statements[0].(*ast.SectionDecl)
				primaryExpr, ok := findEntry(section.Entries, "primary")
				if !ok {
					t.Fatal("expected primary entry")
				}
				primaryMap := primaryExpr.(*ast.MapExpr)
				connExpr, ok := findEntry(primaryMap.Entries, "connection")
				if !ok {
					t.Fatal("expected connection entry")
				}
				connMap := connExpr.(*ast.MapExpr)

				if len(connMap.Entries) != 2 {
					t.Errorf("expected 2 entries in connection, got %d", len(connMap.Entries))
				}
			},
		},
		{
			name: "nested map with references",
			input: `
config:
  database:
    host: @infra:config.db.host
    port: 5432
`,
			wantErr: false,
			check: func(t *testing.T, tree *ast.AST) {
				section := tree.Statements[0].(*ast.SectionDecl)
				dbExpr, ok := findEntry(section.Entries, "database")
				if !ok {
					t.Fatal("expected database entry")
				}
				dbMap := dbExpr.(*ast.MapExpr)

				hostExpr, ok := findEntry(dbMap.Entries, "host")
				if !ok {
					t.Fatal("expected host entry")
				}
				hostRef, ok := hostExpr.(*ast.ReferenceExpr)
				if !ok {
					t.Fatalf("expected ReferenceExpr for host, got %T", hostExpr)
				}

				if hostRef.Alias != "infra" {
					t.Errorf("expected alias 'infra', got %q", hostRef.Alias)
				}

				if len(hostRef.Path) != 3 || hostRef.Path[0] != "config" || hostRef.Path[1] != "db" || hostRef.Path[2] != "host" {
					t.Errorf("expected path [config, db, host], got %v", hostRef.Path)
				}
			},
		},
		{
			name: "multiple sibling nested maps",
			input: `
services:
  web:
    host: 'web-server'
    port: 8080
  api:
    host: 'api-server'
    port: 3000
`,
			wantErr: false,
			check: func(t *testing.T, tree *ast.AST) {
				section := tree.Statements[0].(*ast.SectionDecl)

				webExpr, ok := findEntry(section.Entries, "web")
				if !ok {
					t.Fatal("expected MapExpr for web")
				}
				webMap, ok := webExpr.(*ast.MapExpr)
				if !ok {
					t.Fatal("expected MapExpr for web")
				}
				if len(webMap.Entries) != 2 {
					t.Errorf("expected 2 entries in web, got %d", len(webMap.Entries))
				}

				apiExpr, ok := findEntry(section.Entries, "api")
				if !ok {
					t.Fatal("expected MapExpr for api")
				}
				apiMap, ok := apiExpr.(*ast.MapExpr)
				if !ok {
					t.Fatal("expected MapExpr for api")
				}
				if len(apiMap.Entries) != 2 {
					t.Errorf("expected 2 entries in api, got %d", len(apiMap.Entries))
				}
			},
		},
		{
			name: "mixed nested and flat values",
			input: `
app:
  name: 'my-app'
  version: '1.0.0'
  database:
    host: 'localhost'
    port: 5432
  debug: 'true'
`,
			wantErr: false,
			check: func(t *testing.T, tree *ast.AST) {
				section := tree.Statements[0].(*ast.SectionDecl)

				// Flat values
				if entry, ok := findEntry(section.Entries, "name"); !ok || entry == nil {
					t.Error("expected StringLiteral for name")
				}
				if entry, ok := findEntry(section.Entries, "version"); !ok || entry == nil {
					t.Error("expected StringLiteral for version")
				}
				if entry, ok := findEntry(section.Entries, "debug"); !ok || entry == nil {
					t.Error("expected StringLiteral for debug")
				}

				// Nested map
				if entry, ok := findEntry(section.Entries, "database"); !ok || entry == nil {
					t.Error("expected MapExpr for database")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := parser.Parse(strings.NewReader(strings.TrimSpace(tt.input)), "test.csl")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tree == nil {
				t.Fatal("expected non-nil tree")
			}

			if tt.check != nil {
				tt.check(t, tree)
			}
		})
	}
}
