// Package parser_test contains integration tests for the parser inline reference feature.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseInlineReferences_ValidFixture tests that the parser correctly parses
// inline reference expressions in value positions and creates ReferenceExpr AST nodes.
func TestParseInlineReferences_ValidFixture(t *testing.T) {
	// Arrange
	fixturePath := "../testdata/fixtures/valid_inline_references.csl"

	// Act
	result, err := parser.ParseFile(fixturePath)

	// Assert - parse should succeed
	if err != nil {
		t.Fatalf("expected no error parsing valid inline references, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil AST, got nil")
	}

	// Assert - should have correct number of statements
	// 2 source declarations + 2 section declarations = 4 statements
	if len(result.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(result.Statements))
	}

	// Find the 'infrastructure' section (should be 3rd statement)
	var infraSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "infrastructure" {
			infraSection = section
			break
		}
	}

	if infraSection == nil {
		t.Fatal("expected to find 'infrastructure' section in statements")
	}

	// Assert - check that vpc_cidr entry contains a ReferenceExpr
	vpcCIDRExpr, exists := infraSection.Entries["vpc_cidr"]
	if !exists {
		t.Fatal("expected 'vpc_cidr' entry in infrastructure section")
	}

	// Should be a ReferenceExpr, not a StringLiteral
	vpcCIDRRef, ok := vpcCIDRExpr.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected vpc_cidr to be a ReferenceExpr, got %T", vpcCIDRExpr)
	}

	// Verify ReferenceExpr fields
	if vpcCIDRRef.Alias != "network" {
		t.Errorf("expected vpc_cidr alias to be 'network', got %q", vpcCIDRRef.Alias)
	}

	expectedPath := []string{"vpc", "cidr"}
	if len(vpcCIDRRef.Path) != len(expectedPath) {
		t.Fatalf("expected vpc_cidr path length %d, got %d", len(expectedPath), len(vpcCIDRRef.Path))
	}
	for i, component := range expectedPath {
		if vpcCIDRRef.Path[i] != component {
			t.Errorf("expected path[%d] = %q, got %q", i, component, vpcCIDRRef.Path[i])
		}
	}

	// Verify span is present
	if vpcCIDRRef.SourceSpan.Filename == "" {
		t.Error("expected vpc_cidr ReferenceExpr to have a non-empty source span filename")
	}

	// Check 'region' is a StringLiteral
	regionExpr, exists := infraSection.Entries["region"]
	if !exists {
		t.Fatal("expected 'region' entry in infrastructure section")
	}

	regionLiteral, ok := regionExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected region to be a StringLiteral, got %T", regionExpr)
	}

	if regionLiteral.Value != "us-west-2" {
		t.Errorf("expected region to be 'us-west-2', got %q", regionLiteral.Value)
	}
}

// TestParseInlineReferences_ScalarValue tests inline references in simple scalar value positions.
func TestParseInlineReferences_ScalarValue(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'network'
	type: 'folder'
	path: './config'

config:
	cidr: @network:vpc.cidr
`
	// Act
	result, err := parser.Parse(newReader(input), "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find config section
	var configSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "config" {
			configSection = section
			break
		}
	}

	if configSection == nil {
		t.Fatal("expected to find 'config' section")
	}

	// Verify cidr entry exists and contains reference syntax
	cidrExpr, exists := configSection.Entries["cidr"]
	if !exists {
		t.Fatal("expected 'cidr' entry in config section")
	}

	// Should be a ReferenceExpr
	cidrRef, ok := cidrExpr.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected cidr to be a ReferenceExpr, got %T", cidrExpr)
	}

	if cidrRef.Alias != "network" {
		t.Errorf("expected cidr alias to be 'network', got %q", cidrRef.Alias)
	}

	expectedPath := []string{"vpc", "cidr"}
	if len(cidrRef.Path) != len(expectedPath) {
		t.Fatalf("expected cidr path length %d, got %d", len(expectedPath), len(cidrRef.Path))
	}
	for i, component := range expectedPath {
		if cidrRef.Path[i] != component {
			t.Errorf("expected path[%d] = %q, got %q", i, component, cidrRef.Path[i])
		}
	}
}

// TestParseInlineReferences_MixedWithLiterals tests that inline references can coexist
// with plain string literals in the same section.
func TestParseInlineReferences_MixedWithLiterals(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'base'
	type: 'folder'
	path: './base'

config:
	literal_value: 'plain-string'
	ref_value: @base:config.key
	another_literal: 'another-plain-string'
`
	// Act
	result, err := parser.Parse(newReader(input), "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find config section
	var configSection *ast.SectionDecl
	for _, stmt := range result.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "config" {
			configSection = section
			break
		}
	}

	if configSection == nil {
		t.Fatal("expected to find 'config' section")
	}

	// Verify mixed values
	literalExpr, exists := configSection.Entries["literal_value"]
	if !exists {
		t.Fatal("expected 'literal_value' entry")
	}
	literalValue, ok := literalExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected literal_value to be StringLiteral, got %T", literalExpr)
	}
	if literalValue.Value != "plain-string" {
		t.Errorf("expected literal_value to be 'plain-string', got %q", literalValue.Value)
	}

	refExpr, exists := configSection.Entries["ref_value"]
	if !exists {
		t.Fatal("expected 'ref_value' entry")
	}
	refValue, ok := refExpr.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected ref_value to be ReferenceExpr, got %T", refExpr)
	}
	if refValue.Alias != "base" {
		t.Errorf("expected ref_value alias to be 'base', got %q", refValue.Alias)
	}

	anotherLiteralExpr, exists := configSection.Entries["another_literal"]
	if !exists {
		t.Fatal("expected 'another_literal' entry")
	}
	anotherLiteral, ok := anotherLiteralExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected another_literal to be StringLiteral, got %T", anotherLiteralExpr)
	}
	if anotherLiteral.Value != "another-plain-string" {
		t.Errorf("expected another_literal to be 'another-plain-string', got %q", anotherLiteral.Value)
	}
}

// Helper to create io.Reader from string
func newReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
