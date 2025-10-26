// Package parser_test contains integration tests for the parser public API.
package parser_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestSourceSpan_AllNodesHaveSpans tests that all AST nodes include source span information.
func TestSourceSpan_AllNodesHaveSpans(t *testing.T) {
	// Arrange
	input := `source:
	alias: 'folder'
	type:  'folder'
	path:  '../config'

import:folder:filename

config-section:
	key1: value1
	ref: reference:folder:config.key
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check AST root has span
	if result.SourceSpan.Filename != "test.csl" {
		t.Errorf("AST span filename: expected 'test.csl', got '%s'", result.SourceSpan.Filename)
	}
	if result.SourceSpan.StartLine != 1 {
		t.Errorf("AST span start line: expected 1, got %d", result.SourceSpan.StartLine)
	}

	// Check each statement has a span
	for i, stmt := range result.Statements {
		span := stmt.Span()
		if span.Filename != "test.csl" {
			t.Errorf("statement[%d] span filename: expected 'test.csl', got '%s'", i, span.Filename)
		}
		if span.StartLine <= 0 {
			t.Errorf("statement[%d] span start line: expected > 0, got %d", i, span.StartLine)
		}
		if span.StartCol <= 0 {
			t.Errorf("statement[%d] span start col: expected > 0, got %d", i, span.StartCol)
		}
		if span.EndLine <= 0 {
			t.Errorf("statement[%d] span end line: expected > 0, got %d", i, span.EndLine)
		}
		if span.EndCol < 0 {
			t.Errorf("statement[%d] span end col: expected >= 0, got %d", i, span.EndCol)
		}
	}
}

// TestSourceSpan_CorrectLineNumbers tests that line numbers are accurate.
func TestSourceSpan_CorrectLineNumbers(t *testing.T) {
	// Arrange - source starts at line 1, import at line 5
	input := `source:
	alias: 'test'
	type:  'folder'

import:test:path
`
	reader := strings.NewReader(input)

	// Act
	result, err := parser.Parse(reader, "test.csl")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Statements) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(result.Statements))
	}

	// First statement (source) should start at line 1
	sourceStmt := result.Statements[0]
	if sourceStmt.Span().StartLine != 1 {
		t.Errorf("source statement: expected start line 1, got %d", sourceStmt.Span().StartLine)
	}

	// Second statement (import) should start at line 5
	importStmt := result.Statements[1]
	if importStmt.Span().StartLine != 5 {
		t.Errorf("import statement: expected start line 5, got %d", importStmt.Span().StartLine)
	}
}

// TestReferenceExprSourceSpan_ScalarValue tests that ReferenceExpr nodes
// capture precise source span information for inline references in scalar value positions.
func TestReferenceExprSourceSpan_ScalarValue(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantStartLine  int
		wantStartCol   int
		wantEndLine    int
		wantEndCol     int
		referenceValue string // The expected reference value to find
	}{
		{
			name: "simple inline reference",
			input: `config:
	cidr: reference:network:vpc.cidr
`,
			// "reference:network:vpc.cidr" is 26 chars, starts at col 8 on line 2
			wantStartLine:  2,
			wantStartCol:   8,
			wantEndLine:    2,
			wantEndCol:     33, // 8 + 26 - 1
			referenceValue: "reference:network:vpc.cidr",
		},
		{
			name: "inline reference with single path segment",
			input: `settings:
	value: reference:alias:key
`,
			// "reference:alias:key" is 19 chars, starts at col 9 on line 2
			wantStartLine:  2,
			wantStartCol:   9,
			wantEndLine:    2,
			wantEndCol:     27, // 9 + 19 - 1
			referenceValue: "reference:alias:key",
		},
		{
			name: "inline reference with long dotted path",
			input: `network:
	subnet: reference:vpc:config.network.subnet.cidr
`,
			// "reference:vpc:config.network.subnet.cidr" is 40 chars, starts at col 10 on line 2
			wantStartLine:  2,
			wantStartCol:   10,
			wantEndLine:    2,
			wantEndCol:     49, // 10 + 40 - 1
			referenceValue: "reference:vpc:config.network.subnet.cidr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := parser.Parse(strings.NewReader(tt.input), "test.csl")
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			// Find the ReferenceExpr in the AST
			refExpr := findReferenceExpr(t, result)
			if refExpr == nil {
				t.Fatal("no ReferenceExpr found in parsed AST")
			}

			// Verify the source span
			span := refExpr.Span()

			if span.StartLine != tt.wantStartLine {
				t.Errorf("StartLine: got %d, want %d", span.StartLine, tt.wantStartLine)
			}
			if span.StartCol != tt.wantStartCol {
				t.Errorf("StartCol: got %d, want %d", span.StartCol, tt.wantStartCol)
			}
			if span.EndLine != tt.wantEndLine {
				t.Errorf("EndLine: got %d, want %d", span.EndLine, tt.wantEndLine)
			}
			if span.EndCol != tt.wantEndCol {
				t.Errorf("EndCol: got %d, want %d", span.EndCol, tt.wantEndCol)
			}

			// Verify we can extract the exact text using the span
			extractedText := extractTextFromSpan(tt.input, span)
			if extractedText != tt.referenceValue {
				t.Errorf("extracted text from span: got %q, want %q", extractedText, tt.referenceValue)
			}
		})
	}
}

// TestReferenceExprSourceSpan_MapValue tests precise span capture for inline references
// used as values in configuration maps.
func TestReferenceExprSourceSpan_MapValue(t *testing.T) {
	input := `network:
	vpc: reference:config:vpc.id
	subnet: reference:config:subnet.id
`

	result, err := parser.Parse(strings.NewReader(input), "test.csl")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Find all ReferenceExpr nodes
	refExprs := findAllReferenceExprs(t, result)
	if len(refExprs) != 2 {
		t.Fatalf("expected 2 ReferenceExpr nodes, got %d", len(refExprs))
	}

	// Check that we have both references (order may vary due to map iteration)
	var vpcRef, subnetRef *ast.ReferenceExpr
	for _, ref := range refExprs {
		if ref.Path[len(ref.Path)-1] == "id" && len(ref.Path) == 2 {
			switch ref.Path[0] {
			case "vpc":
				vpcRef = ref
			case "subnet":
				subnetRef = ref
			}
		}
	}

	if vpcRef == nil {
		t.Fatal("vpc reference not found")
	}
	if subnetRef == nil {
		t.Fatal("subnet reference not found")
	}

	// VPC reference: "reference:config:vpc.id" on line 2, col 7
	span1 := vpcRef.Span()
	if span1.StartLine != 2 || span1.StartCol != 7 {
		t.Errorf("vpc reference StartLine:StartCol: got %d:%d, want 2:7", span1.StartLine, span1.StartCol)
	}
	extracted1 := extractTextFromSpan(input, span1)
	if extracted1 != "reference:config:vpc.id" {
		t.Errorf("vpc reference extracted text: got %q, want %q", extracted1, "reference:config:vpc.id")
	}

	// Subnet reference: "reference:config:subnet.id" on line 3, col 10
	span2 := subnetRef.Span()
	if span2.StartLine != 3 || span2.StartCol != 10 {
		t.Errorf("subnet reference StartLine:StartCol: got %d:%d, want 3:10", span2.StartLine, span2.StartCol)
	}
	extracted2 := extractTextFromSpan(input, span2)
	if extracted2 != "reference:config:subnet.id" {
		t.Errorf("subnet reference extracted text: got %q, want %q", extracted2, "reference:config:subnet.id")
	}
}

// Helper functions for ReferenceExpr tests

// findReferenceExpr recursively searches the AST for the first ReferenceExpr node.
func findReferenceExpr(t *testing.T, node interface{}) *ast.ReferenceExpr {
	t.Helper()

	switch n := node.(type) {
	case *ast.AST:
		for _, stmt := range n.Statements {
			if ref := findReferenceExpr(t, stmt); ref != nil {
				return ref
			}
		}
	case *ast.SectionDecl:
		for _, expr := range n.Entries {
			if ref := findReferenceExpr(t, expr); ref != nil {
				return ref
			}
		}
	case *ast.SourceDecl:
		for _, expr := range n.Config {
			if ref := findReferenceExpr(t, expr); ref != nil {
				return ref
			}
		}
	case *ast.ReferenceExpr:
		return n
	}
	return nil
}

// findAllReferenceExprs recursively collects all ReferenceExpr nodes in the AST.
func findAllReferenceExprs(t *testing.T, node interface{}) []*ast.ReferenceExpr {
	t.Helper()

	var refs []*ast.ReferenceExpr

	switch n := node.(type) {
	case *ast.AST:
		for _, stmt := range n.Statements {
			refs = append(refs, findAllReferenceExprs(t, stmt)...)
		}
	case *ast.SectionDecl:
		for _, expr := range n.Entries {
			refs = append(refs, findAllReferenceExprs(t, expr)...)
		}
	case *ast.SourceDecl:
		for _, expr := range n.Config {
			refs = append(refs, findAllReferenceExprs(t, expr)...)
		}
	case *ast.ReferenceExpr:
		refs = append(refs, n)
	}
	return refs
}

// extractTextFromSpan extracts the text from the input string based on the SourceSpan.
// This validates that the span accurately points to the original source text.
func extractTextFromSpan(input string, span ast.SourceSpan) string {
	lines := strings.Split(input, "\n")

	// Handle single-line spans
	if span.StartLine == span.EndLine {
		if span.StartLine > len(lines) {
			return ""
		}
		line := lines[span.StartLine-1] // Convert to 0-indexed
		if span.StartCol > len(line) || span.EndCol > len(line) {
			return ""
		}
		// Extract substring using 1-indexed columns
		return line[span.StartCol-1 : span.EndCol]
	}

	// Multi-line spans (not expected for inline references, but handle anyway)
	var result strings.Builder
	for i := span.StartLine; i <= span.EndLine; i++ {
		if i > len(lines) {
			break
		}
		line := lines[i-1]
		switch i {
		case span.StartLine:
			if span.StartCol <= len(line) {
				result.WriteString(line[span.StartCol-1:])
			}
		case span.EndLine:
			if span.EndCol <= len(line) {
				result.WriteString(line[:span.EndCol])
			}
		default:
			result.WriteString(line)
		}
		if i < span.EndLine {
			result.WriteString("\n")
		}
	}
	return result.String()
}

// TestReferenceExprSourceSpan_EdgeCases tests span accuracy for edge cases.
func TestReferenceExprSourceSpan_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAlias   string
		wantPath    []string
		wantExtract string
		wantLine    int
		wantByteLen int // Expected length in BYTES (not runes)
	}{
		{
			name:        "unicode in surrounding context",
			input:       "unicode:\n  key: reference:net:日本",
			wantAlias:   "net",
			wantPath:    []string{"日本"},
			wantExtract: "reference:net:日本",
			wantLine:    2,
			wantByteLen: 20, // "reference:net:日本" = 20 bytes (日本 is 6 bytes)
		},
		{
			name:        "very long dotted path",
			input:       "section:\n  value: reference:alias:level1.level2.level3.level4.level5.level6.level7",
			wantAlias:   "alias",
			wantPath:    []string{"level1", "level2", "level3", "level4", "level5", "level6", "level7"},
			wantExtract: "reference:alias:level1.level2.level3.level4.level5.level6.level7",
			wantLine:    2,
			wantByteLen: 64,
		},
		{
			name:        "single segment path",
			input:       "section:\n  value: reference:myalias:path",
			wantAlias:   "myalias",
			wantPath:    []string{"path"},
			wantExtract: "reference:myalias:path",
			wantLine:    2,
			wantByteLen: 22,
		},
		{
			name:        "path with numbers and dashes",
			input:       "section:\n  value: reference:alias-1:path-2.sub_3",
			wantAlias:   "alias-1",
			wantPath:    []string{"path-2", "sub_3"},
			wantExtract: "reference:alias-1:path-2.sub_3",
			wantLine:    2,
			wantByteLen: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			ast, err := parser.Parse(strings.NewReader(tt.input), "test.csl")
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			// Find the ReferenceExpr
			ref := findReferenceExpr(t, ast)
			if ref == nil {
				t.Fatalf("no ReferenceExpr found in AST")
			}

			// Verify alias and path
			if ref.Alias != tt.wantAlias {
				t.Errorf("Alias: got %q, want %q", ref.Alias, tt.wantAlias)
			}

			// Compare path slices
			if len(ref.Path) != len(tt.wantPath) {
				t.Errorf("Path length: got %d, want %d", len(ref.Path), len(tt.wantPath))
			} else {
				for i := range ref.Path {
					if ref.Path[i] != tt.wantPath[i] {
						t.Errorf("Path[%d]: got %q, want %q", i, ref.Path[i], tt.wantPath[i])
					}
				}
			}

			// Verify span
			span := ref.Span()
			if span.StartLine != tt.wantLine {
				t.Errorf("StartLine: got %d, want %d", span.StartLine, tt.wantLine)
			}

			// Calculate expected EndCol from StartCol and BYTE length
			// EndCol is inclusive, so: EndCol = StartCol + byteLen - 1
			expectedEndCol := span.StartCol + tt.wantByteLen - 1
			if span.EndCol != expectedEndCol {
				t.Errorf("EndCol: got %d, want %d (StartCol=%d + byteLen=%d - 1)",
					span.EndCol, expectedEndCol, span.StartCol, tt.wantByteLen)
			}

			// Verify extracted text matches expected
			extracted := extractTextFromSpan(tt.input, span)
			if extracted != tt.wantExtract {
				t.Errorf("Extracted text: got %q, want %q", extracted, tt.wantExtract)
			}
		})
	}
}
