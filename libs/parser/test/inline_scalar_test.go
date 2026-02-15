// Package parser_test contains tests for inline scalar value handling.
package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParse_InlineScalarValue tests that inline scalar values are correctly parsed
// without creating empty-string keys in the Entries map.
func TestParse_InlineScalarValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKey  string
		wantType string
	}{
		{
			name:     "string scalar",
			input:    `region: "us-west-2"`,
			wantKey:  "region",
			wantType: "*ast.StringLiteral",
		},
		{
			name:     "string scalar with whitespace",
			input:    `  region:   "us-west-2"  `,
			wantKey:  "region",
			wantType: "*ast.StringLiteral",
		},
		{
			name:     "identifier scalar",
			input:    `enabled: true`,
			wantKey:  "enabled",
			wantType: "*ast.StringLiteral", // Parser treats bare identifiers as strings
		},
		{
			name:     "path expression scalar",
			input:    `value: some.path.expr`,
			wantKey:  "value",
			wantType: "*ast.StringLiteral", // Parser treats dotted paths as strings
		},
		{
			name:     "reference expression scalar",
			input:    `vpc_id: @network:config.vpc.id`,
			wantKey:  "vpc_id",
			wantType: "*ast.ReferenceExpr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(strings.NewReader(tt.input), "test.csl")
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			section, ok := result.Statements[0].(*ast.SectionDecl)
			if !ok {
				t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
			}

			if section.Name != tt.wantKey {
				t.Errorf("section name = %q, want %q", section.Name, tt.wantKey)
			}

			// CRITICAL: Verify NO empty-string keys in Entries
			if len(section.Entries) > 0 {
				if hasEntry(section.Entries, "") {
					t.Error("inline scalar should not create empty-string key in Entries")
				}
			}

			// Verify Value field is set for inline scalar
			if section.Value == nil {
				t.Error("inline scalar should set Value field")
			}

			// Verify Entries is nil for inline scalar (mutually exclusive)
			if section.Entries != nil {
				t.Errorf("inline scalar should have nil Entries, got %d entries", len(section.Entries))
			}

			// Verify correct type
			actualType := fmt.Sprintf("%T", section.Value)
			if actualType != tt.wantType {
				t.Errorf("Value type = %s, want %s", actualType, tt.wantType)
			}
		})
	}
}

// TestParse_NestedMapNotInlineScalar tests that nested maps still use Entries,
// not the Value field.
func TestParse_NestedMapNotInlineScalar(t *testing.T) {
	input := `database:
  host: "localhost"
  port: "5432"`

	result, err := parser.Parse(strings.NewReader(input), "test.csl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	section := result.Statements[0].(*ast.SectionDecl)

	// Nested map should use Entries, not Value
	if section.Value != nil {
		t.Error("nested map should not set Value field")
	}

	if section.Entries == nil {
		t.Fatal("nested map should have non-nil Entries")
	}

	if len(section.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(section.Entries))
	}

	// Verify no empty-string keys
	for _, entry := range section.Entries {
		if entry.Key == "" && !entry.Spread {
			t.Error("nested map should not have empty-string keys")
		}
	}
}

// TestParse_MixedScalarsAndMaps tests a configuration with both inline scalars
// and nested maps.
func TestParse_MixedScalarsAndMaps(t *testing.T) {
	input := `region: "us-west-2"
environment: "production"
database:
  host: "localhost"
  port: "5432"
vpc_id: @network:config.vpc.id`

	result, err := parser.Parse(strings.NewReader(input), "test.csl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(result.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(result.Statements))
	}

	// Check region (inline scalar)
	region := result.Statements[0].(*ast.SectionDecl)
	if region.Name != "region" {
		t.Errorf("statement 0 name = %q, want 'region'", region.Name)
	}
	if region.Value == nil {
		t.Error("region should have Value set")
	}
	if region.Entries != nil {
		t.Error("region should have nil Entries")
	}

	// Check environment (inline scalar)
	env := result.Statements[1].(*ast.SectionDecl)
	if env.Name != "environment" {
		t.Errorf("statement 1 name = %q, want 'environment'", env.Name)
	}
	if env.Value == nil {
		t.Error("environment should have Value set")
	}
	if env.Entries != nil {
		t.Error("environment should have nil Entries")
	}

	// Check database (nested map)
	database := result.Statements[2].(*ast.SectionDecl)
	if database.Name != "database" {
		t.Errorf("statement 2 name = %q, want 'database'", database.Name)
	}
	if database.Value != nil {
		t.Error("database should have nil Value")
	}
	if database.Entries == nil || len(database.Entries) != 2 {
		t.Errorf("database should have 2 Entries, got %v", database.Entries)
	}

	// Check vpc_id (inline scalar reference)
	vpcID := result.Statements[3].(*ast.SectionDecl)
	if vpcID.Name != "vpc_id" {
		t.Errorf("statement 3 name = %q, want 'vpc_id'", vpcID.Name)
	}
	if vpcID.Value == nil {
		t.Error("vpc_id should have Value set")
	}
	if vpcID.Entries != nil {
		t.Error("vpc_id should have nil Entries")
	}

	// Verify NO empty-string keys anywhere
	for i, stmt := range result.Statements {
		section := stmt.(*ast.SectionDecl)
		if section.Entries != nil {
			for _, entry := range section.Entries {
				if entry.Key == "" && !entry.Spread {
					t.Errorf("statement %d (%s) has empty-string key", i, section.Name)
				}
			}
		}
	}
}
