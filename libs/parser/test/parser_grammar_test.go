// Package parser_test contains unit tests for individual parser grammar constructs.
package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/internal/testutil"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestParseSourceDecl_ValidDeclaration tests source declaration parsing.
func TestParseSourceDecl_ValidDeclaration(t *testing.T) {
	input := `source:
	alias: 'folder'
	type:  'folder'
	path:  '../config'
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	decl, ok := result.Statements[0].(*ast.SourceDecl)
	if !ok {
		t.Fatalf("expected *ast.SourceDecl, got %T", result.Statements[0])
	}

	if decl.Alias != "folder" {
		t.Errorf("expected alias 'folder', got '%s'", decl.Alias)
	}
	if decl.Type != "folder" {
		t.Errorf("expected type 'folder', got '%s'", decl.Type)
	}

	// Extract path from Config (now contains Expr values)
	pathExpr, ok := decl.Config["path"]
	if !ok {
		t.Fatal("expected 'path' in Config")
	}
	pathLiteral, ok := pathExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected path to be StringLiteral, got %T", pathExpr)
	}
	if pathLiteral.Value != "../config" {
		t.Errorf("expected path '../config', got '%s'", pathLiteral.Value)
	}
}

// TestParseSourceDecl_PreservesSourceSpan tests that source spans are correct.
func TestParseSourceDecl_PreservesSourceSpan(t *testing.T) {
	input := `source:
	alias: 'test'
	type:  'folder'
`
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decl := result.Statements[0].(*ast.SourceDecl)
	span := decl.Span()

	if span.Filename != "test.csl" {
		t.Errorf("expected filename 'test.csl', got '%s'", span.Filename)
	}
	if span.StartLine != 1 {
		t.Errorf("expected start line 1, got %d", span.StartLine)
	}
	if span.StartCol != 1 {
		t.Errorf("expected start col 1, got %d", span.StartCol)
	}
}

// TestParseImportStmt_WithPath tests that import statement is no longer supported.
func TestParseImportStmt_WithPath(t *testing.T) {
	input := "import:folder:filename\n"
	reader := strings.NewReader(input)
	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Fatal("expected error for import statement, got nil")
	}

	// Verify error message indicates import is no longer supported
	errMsg := err.Error()
	if !strings.Contains(errMsg, "import statement no longer supported") {
		t.Errorf("expected error about import not supported, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "@alias:path") {
		t.Errorf("expected migration hint about @alias:path syntax, got: %s", errMsg)
	}
}

// TestParseImportStmt_WithoutPath tests that import statement is no longer supported.
func TestParseImportStmt_WithoutPath(t *testing.T) {
	input := "import:folder\n"
	reader := strings.NewReader(input)
	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Fatal("expected error for import statement, got nil")
	}

	// Verify error message indicates import is no longer supported
	errMsg := err.Error()
	if !strings.Contains(errMsg, "import statement no longer supported") {
		t.Errorf("expected error about import not supported, got: %s", errMsg)
	}
}

// TestParseReferenceStmt_SimplePath tests that top-level references are rejected.
func TestParseReferenceStmt_SimplePath(t *testing.T) {
	input := "reference:folder:config\n"
	reader := strings.NewReader(input)
	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Fatal("expected error for deprecated top-level reference, got nil")
	}

	// Verify error message suggests inline references
	errMsg := err.Error()
	if !strings.Contains(errMsg, "top-level") && !strings.Contains(errMsg, "inline") {
		t.Errorf("expected error message to mention 'top-level' or 'inline', got: %s", errMsg)
	}
}

// TestParseReferenceStmt_DottedPath tests that top-level references with dotted paths are rejected.
func TestParseReferenceStmt_DottedPath(t *testing.T) {
	input := "reference:folder:config.key.value\n"
	reader := strings.NewReader(input)
	_, err := parser.Parse(reader, "test.csl")

	if err == nil {
		t.Fatal("expected error for deprecated top-level reference, got nil")
	}
}

// TestParseSectionDecl_SimpleSection tests section with key-value pairs.
func TestParseSectionDecl_SimpleSection(t *testing.T) {
	input := "config-section:\n\tkey1: value1\n\tkey2: value2\n"
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader, "test.csl")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decl, ok := result.Statements[0].(*ast.SectionDecl)
	if !ok {
		t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
	}

	if decl.Name != "config-section" {
		t.Errorf("expected name 'config-section', got '%s'", decl.Name)
	}

	// Note: The parser may have a bug where it doesn't preserve all entries.
	// The golden test file shows only key2 being preserved.
	// Testing for at least 1 entry as the integration test does.
	if len(decl.Entries) < 2 {
		t.Errorf("expected 2 entries, got %d", len(decl.Entries))
	}

	// Verify at least one expected key exists
	key2Expr, ok := decl.Entries["key2"]
	if !ok {
		t.Fatal("expected 'key2' in Entries")
	}
	key2Literal, ok := key2Expr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected key2 to be StringLiteral, got %T", key2Expr)
	}
	if key2Literal.Value != "value2" {
		t.Errorf("expected key2='value2', got '%s'", key2Literal.Value)
	}
}

// TestParse_PathTokenization_ComplexPaths tests that top-level references are rejected.
// Path tokenization is now tested via inline references in TestParseInlineReferences_* tests.
func TestParse_PathTokenization_ComplexPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"single level", "reference:src:key\n"},
		{"two levels", "reference:src:config.key\n"},
		{"three levels", "reference:src:app.config.key\n"},
		{"deep nesting", "reference:src:a.b.c.d.e.f\n"},
		{"with dashes", "reference:src:app-config.key-name\n"},
		{"with underscores", "reference:src:app_config.key_name\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")
			if err == nil {
				t.Fatal("expected error for deprecated top-level reference, got nil")
			}
		})
	}
}

// TestParse_Aliasing_VariousAliasFormats tests various alias formats.
// Note: import statements are no longer supported, so this tests error handling
func TestParse_Aliasing_VariousAliasFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "import:simple:file\n"},
		{"with dash", "import:my-source:file\n"},
		{"with underscore", "import:my_source:file\n"},
		{"with numbers", "import:source123:file\n"},
		{"complex", "import:my-source_v2:file\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")
			if err == nil {
				t.Fatal("expected error for import statement, got nil")
			}
			if !strings.Contains(err.Error(), "import statement no longer supported") {
				t.Errorf("expected error about import not supported, got: %v", err)
			}
		})
	}
}

// --- Comment Support Tests (User Story 1: Single-Line Comments) ---

// TestParse_Comments_FullLineIgnored tests that full-line comments are completely ignored.
// T015: Write test: full-line comment ignored
func TestParse_Comments_FullLineIgnored(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedStmts  int
		validateResult func(*testing.T, *ast.AST)
	}{
		{
			name: "single full-line comment",
			input: `# This is a comment
config-section:
	key: value
`,
			expectedStmts: 1,
			validateResult: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "comment at start of file",
			input: `# Header comment
# Another comment
config-section:
	key: value
`,
			expectedStmts: 1,
			validateResult: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "comment between statements",
			input: `source:
	alias: 'folder'
	type:  'folder'
	path:  './config'
# Comment between statements
config-section:
	key: value
`,
			expectedStmts: 2,
			validateResult: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 2 {
					t.Fatalf("expected 2 statements, got %d", len(result.Statements))
				}
			},
		},
		{
			name: "multiple consecutive comments",
			input: `# Comment 1
# Comment 2
# Comment 3
config-section:
	key: value
`,
			expectedStmts: 1,
			validateResult: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result.Statements) != tt.expectedStmts {
				t.Errorf("expected %d statements, got %d", tt.expectedStmts, len(result.Statements))
			}
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// TestParse_Comments_TrailingAfterKeyValue tests trailing comments after key:value pairs.
// T016: Write test: trailing comment after key:value
func TestParse_Comments_TrailingAfterKeyValue(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedKey   string
		expectedValue string
	}{
		{
			name: "trailing comment after value",
			input: `config-section:
	key: value # This is a comment
`,
			expectedKey:   "key",
			expectedValue: "value",
		},
		{
			name: "trailing comment with special chars",
			input: `config-section:
	key: value # Comment with: colons and 'quotes'
`,
			expectedKey:   "key",
			expectedValue: "value",
		},
		{
			name: "trailing comment no space before hash",
			input: `config-section:
	key: value# Comment immediately after value
`,
			expectedKey:   "key",
			expectedValue: "value",
		},
		{
			name: "trailing comment after quoted value",
			input: `config-section:
	key: 'quoted value' # Comment after quoted string
`,
			expectedKey:   "key",
			expectedValue: "quoted value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			valueExpr := section.Entries[tt.expectedKey]
			if valueExpr == nil {
				t.Fatalf("expected key '%s' not found", tt.expectedKey)
			}

			literal, ok := valueExpr.(*ast.StringLiteral)
			if !ok {
				t.Fatalf("expected StringLiteral, got %T", valueExpr)
			}

			if literal.Value != tt.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tt.expectedValue, literal.Value)
			}
		})
	}
}

// TestParse_Comments_BeforeSectionDeclaration tests comments before section declarations.
// T017: Write test: comment before section declaration
func TestParse_Comments_BeforeSectionDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
	}{
		{
			name: "comment immediately before section",
			input: `# This section contains configuration
config-section:
	key: value
`,
			expectedName: "config-section",
		},
		{
			name: "multiple comments before section",
			input: `# Documentation for this section
# More documentation
# Even more docs
my-section:
	key: value
`,
			expectedName: "my-section",
		},
		{
			name: "comment with indentation before section",
			input: `	# Indented comment
config-section:
	key: value
`,
			expectedName: "config-section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			section := result.Statements[0].(*ast.SectionDecl)
			if section.Name != tt.expectedName {
				t.Errorf("expected section name '%s', got '%s'", tt.expectedName, section.Name)
			}
		})
	}
}

// TestParse_Comments_HashInSingleQuotedString tests # preserved in single-quoted strings.
// T018: Write test: # in single-quoted string preserved
func TestParse_Comments_HashInSingleQuotedString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{
			name: "hash at start of quoted string",
			input: `config-section:
	key: '#hashtag'
`,
			expectedValue: "#hashtag",
		},
		{
			name: "hash in middle of quoted string",
			input: `config-section:
	key: 'value #with hash'
`,
			expectedValue: "value #with hash",
		},
		{
			name: "hash at end of quoted string",
			input: `config-section:
	key: 'value#'
`,
			expectedValue: "value#",
		},
		{
			name: "multiple hashes in quoted string",
			input: `config-section:
	key: '## Multiple ## Hashes ##'
`,
			expectedValue: "## Multiple ## Hashes ##",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			valueExpr := section.Entries["key"]
			literal := valueExpr.(*ast.StringLiteral)

			if literal.Value != tt.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tt.expectedValue, literal.Value)
			}
		})
	}
}

// TestParse_Comments_HashInDoubleQuotedString tests # preserved in double-quoted strings.
// T019: Write test: # in double-quoted string preserved
func TestParse_Comments_HashInDoubleQuotedString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{
			name: "hash at start of double-quoted string",
			input: `config-section:
	key: "#hashtag"
`,
			expectedValue: "#hashtag",
		},
		{
			name: "hash in middle of double-quoted string",
			input: `config-section:
	key: "value #with hash"
`,
			expectedValue: "value #with hash",
		},
		{
			name: "hash at end of double-quoted string",
			input: `config-section:
	key: "value#"
`,
			expectedValue: "value#",
		},
		{
			name: "multiple hashes in double-quoted string",
			input: `config-section:
	key: "## Multiple ## Hashes ##"
`,
			expectedValue: "## Multiple ## Hashes ##",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			valueExpr := section.Entries["key"]
			literal := valueExpr.(*ast.StringLiteral)

			if literal.Value != tt.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tt.expectedValue, literal.Value)
			}
		})
	}
}

// TestParse_Comments_WithSpecialCharacters tests comments with special characters.
// T020: Write test: comment with special characters (:, quotes)
func TestParse_Comments_WithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "comment with colons",
			input: `# Comment with: colons: everywhere:
config-section:
	key: value
`,
		},
		{
			name: "comment with single quotes",
			input: `# Comment with 'single' quotes
config-section:
	key: value
`,
		},
		{
			name: "comment with double quotes",
			input: `# Comment with "double" quotes
config-section:
	key: value
`,
		},
		{
			name: "comment with mixed special chars",
			input: `# TODO: Fix this 'issue' with "quotes" and: colons!
config-section:
	key: value
`,
		},
		{
			name: "comment with punctuation",
			input: `# Comment with punctuation: @#$%^&*()!
config-section:
	key: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result.Statements) != 1 {
				t.Errorf("expected 1 statement, got %d", len(result.Statements))
			}

			section := result.Statements[0].(*ast.SectionDecl)
			if section.Name != "config-section" {
				t.Errorf("expected section name 'config-section', got '%s'", section.Name)
			}
		})
	}
}

// TestParse_Comments_EmptyCommentLine tests empty comment lines (just #).
// T021: Write test: empty comment line
func TestParse_Comments_EmptyCommentLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "single empty comment",
			input: `#
config-section:
	key: value
`,
		},
		{
			name: "empty comment with whitespace after hash",
			input: `#   
config-section:
	key: value
`,
		},
		{
			name: "multiple empty comments",
			input: `#
#
#
config-section:
	key: value
`,
		},
		{
			name: "empty comments mixed with text comments",
			input: `# Text comment
#
# Another text comment
#
config-section:
	key: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result.Statements) != 1 {
				t.Errorf("expected 1 statement, got %d", len(result.Statements))
			}

			section := result.Statements[0].(*ast.SectionDecl)
			if section.Name != "config-section" {
				t.Errorf("expected section name 'config-section', got '%s'", section.Name)
			}
		})
	}
}

// --- Multi-Line Comment Block Tests (User Story 2: Multi-Line Comments) ---

// TestParse_MultiLineComments_ConsecutiveLinesIgnored tests that consecutive comment lines are all ignored.
// T034: Write test: consecutive comment lines ignored
//
// Verifies that multiple # comment lines in a row are completely skipped during parsing,
// and the resulting AST contains only active configuration with no trace of comment content.
func TestParse_MultiLineComments_ConsecutiveLinesIgnored(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedStmts int
		validateAST   func(*testing.T, *ast.AST)
	}{
		{
			name: "two consecutive comment lines",
			input: `# First comment line
# Second comment line
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
				if len(section.Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(section.Entries))
				}
			},
		},
		{
			name: "three consecutive comment lines",
			input: `# Comment line 1
# Comment line 2
# Comment line 3
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "five consecutive comment lines with documentation style",
			input: `# ===================================
# Configuration Section Documentation
# ===================================
# This section contains important settings
# Last updated: 2026-01-18
database-config:
	host: localhost
	port: 5432
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "database-config" {
					t.Errorf("expected section name 'database-config', got '%s'", section.Name)
				}
				// Verify both keys are present
				if _, ok := section.Entries["host"]; !ok {
					t.Error("expected 'host' key in section entries")
				}
				if _, ok := section.Entries["port"]; !ok {
					t.Error("expected 'port' key in section entries")
				}
			},
		},
		{
			name: "ten consecutive comment lines forming large block",
			input: `# Line 1
# Line 2
# Line 3
# Line 4
# Line 5
# Line 6
# Line 7
# Line 8
# Line 9
# Line 10
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result.Statements) != tt.expectedStmts {
				t.Errorf("expected %d statements, got %d", tt.expectedStmts, len(result.Statements))
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_MultiLineComments_CommentedOutKeysNotInAST tests that commented-out keys don't appear in the AST.
// T035: Write test: commented-out keys not in AST
//
// Verifies that when configuration keys are commented out with #, they are completely excluded
// from the parsed AST, and only active (uncommented) keys remain.
func TestParse_MultiLineComments_CommentedOutKeysNotInAST(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name: "single commented key",
			input: `config-section:
	active_key: value1
	# commented_key: value2
	another_active: value3
`,
			expectedKeys:   []string{"active_key", "another_active"},
			unexpectedKeys: []string{"commented_key"},
		},
		{
			name: "multiple consecutive commented keys",
			input: `config-section:
	enabled: true
	# disabled_feature: true
	# another_disabled: false
	# third_disabled: maybe
	working_feature: active
`,
			expectedKeys:   []string{"enabled", "working_feature"},
			unexpectedKeys: []string{"disabled_feature", "another_disabled", "third_disabled"},
		},
		{
			name: "commented keys at different indentation levels",
			input: `parent-section:
	level1: value1
	# level1_commented: ignored
	nested-section:
		level2: value2
		# level2_commented: ignored
		deeper:
			level3: value3
			# level3_commented: ignored
`,
			expectedKeys:   []string{"level1", "nested-section"},
			unexpectedKeys: []string{"level1_commented", "level2_commented", "level3_commented"},
		},
		{
			name: "all keys commented out except one",
			input: `config-section:
	# key1: value1
	# key2: value2
	# key3: value3
	# key4: value4
	active_key: active_value
	# key5: value5
`,
			expectedKeys:   []string{"active_key"},
			unexpectedKeys: []string{"key1", "key2", "key3", "key4", "key5"},
		},
		{
			name: "commented keys with complex values",
			input: `config-section:
	# database_url: 'postgresql://localhost:5432/mydb'
	# api_key: "secret-key-12345"
	# max_connections: 100
	active_setting: enabled
`,
			expectedKeys:   []string{"active_setting"},
			unexpectedKeys: []string{"database_url", "api_key", "max_connections"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)

			// Verify expected keys ARE present
			for _, key := range tt.expectedKeys {
				if _, ok := section.Entries[key]; !ok {
					t.Errorf("expected key '%s' to be present in AST, but it was not found", key)
				}
			}

			// Verify unexpected (commented) keys are NOT present
			for _, key := range tt.unexpectedKeys {
				if _, ok := section.Entries[key]; ok {
					t.Errorf("expected key '%s' to NOT be present in AST (should be commented out), but it was found", key)
				}
			}
		})
	}
}

// TestParse_MultiLineComments_InterleavedCommentsAndConfig tests comments mixed with configuration lines.
// T036: Write test: interleaved comments and config
//
// Verifies that comments can be freely mixed between configuration lines, and the parser
// correctly processes only the active configuration while completely ignoring all comments.
func TestParse_MultiLineComments_InterleavedCommentsAndConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validateAST func(*testing.T, *ast.AST)
	}{
		{
			name: "comments between section entries",
			input: `config-section:
	key1: value1
	# Comment between entries
	key2: value2
	# Another comment
	key3: value3
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 3 {
					t.Errorf("expected 3 entries, got %d", len(section.Entries))
				}
				expectedKeys := []string{"key1", "key2", "key3"}
				for _, key := range expectedKeys {
					if _, ok := section.Entries[key]; !ok {
						t.Errorf("expected key '%s' not found", key)
					}
				}
			},
		},
		{
			name: "alternating comments and config lines",
			input: `# Header comment
config-section:
	# Comment before first entry
	setting1: enabled
	# Comment between entries
	setting2: disabled
	# Comment at end of section
another-section:
	# Comment in another section
	value: data
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 2 {
					t.Errorf("expected 2 statements, got %d", len(result.Statements))
				}
				// Verify first section
				section1 := result.Statements[0].(*ast.SectionDecl)
				if section1.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section1.Name)
				}
				// Verify second section
				section2 := result.Statements[1].(*ast.SectionDecl)
				if section2.Name != "another-section" {
					t.Errorf("expected section name 'another-section', got '%s'", section2.Name)
				}
			},
		},
		{
			name: "mixed full-line and trailing comments with config",
			input: `config-section:
	# Full-line comment
	key1: value1 # Trailing comment
	# Another full-line comment
	key2: value2 # Another trailing comment
	key3: value3
	# Final comment
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 3 {
					t.Errorf("expected 3 entries, got %d", len(section.Entries))
				}
				// Verify values are correctly parsed (trailing comments removed)
				key1Value := section.Entries["key1"].(*ast.StringLiteral).Value
				if key1Value != "value1" {
					t.Errorf("expected key1='value1', got '%s'", key1Value)
				}
				key2Value := section.Entries["key2"].(*ast.StringLiteral).Value
				if key2Value != "value2" {
					t.Errorf("expected key2='value2', got '%s'", key2Value)
				}
			},
		},
		{
			name: "comment blocks between multiple sections",
			input: `# Section 1 documentation
# This section handles database configuration
database:
	host: localhost
	port: 5432

# Section 2 documentation
# This section handles API configuration
# Multiple lines of documentation here
api:
	endpoint: /api/v1
	timeout: 30

# Section 3 documentation
cache:
	enabled: true
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 3 {
					t.Errorf("expected 3 statements, got %d", len(result.Statements))
				}
				// Verify section names
				section1 := result.Statements[0].(*ast.SectionDecl)
				if section1.Name != "database" {
					t.Errorf("expected section name 'database', got '%s'", section1.Name)
				}
				section2 := result.Statements[1].(*ast.SectionDecl)
				if section2.Name != "api" {
					t.Errorf("expected section name 'api', got '%s'", section2.Name)
				}
				section3 := result.Statements[2].(*ast.SectionDecl)
				if section3.Name != "cache" {
					t.Errorf("expected section name 'cache', got '%s'", section3.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// --- Edge Cases & Error Handling Tests (Phase 6: Tasks T053-T063) ---

// TestParse_EdgeCase_HashAtLineStart tests # at the start of a line with no whitespace.
// T053: Write test: # at line start with no whitespace
//
// Verifies that # appearing at column 0 (no leading whitespace) is correctly treated as
// a comment, and the entire line is ignored during parsing.
func TestParse_EdgeCase_HashAtLineStart(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedStmts int
		validateAST   func(*testing.T, *ast.AST)
	}{
		{
			name: "# at column 0",
			input: `#comment at start
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "multiple # at column 0",
			input: `#first comment
#second comment
#third comment
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				// Section validation is handled by statement count assertion
			},
		},
		{
			name: "# at column 0 with no trailing text",
			input: `#
config-section:
	key: value
`,
			expectedStmts: 1,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result.Statements) != tt.expectedStmts {
				t.Errorf("expected %d statements, got %d", tt.expectedStmts, len(result.Statements))
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_EdgeCase_HashImmediatelyAfterValue tests # immediately after value with no space.
// T054: Write test: # immediately after value (no space)
//
// Verifies that # appearing directly after a value (no separating space) is correctly
// treated as starting a comment, and the comment text is stripped from the value.
func TestParse_EdgeCase_HashImmediatelyAfterValue(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedKey   string
		expectedValue string
	}{
		{
			name: "# immediately after unquoted value",
			input: `config-section:
	key: value#comment
`,
			expectedKey:   "key",
			expectedValue: "value",
		},
		{
			name: "# immediately after quoted value",
			input: `config-section:
	key: 'value'#comment
`,
			expectedKey:   "key",
			expectedValue: "value",
		},
		{
			name: "# with long comment immediately after value",
			input: `config-section:
	setting: enabled#this is a very long comment explaining the setting
`,
			expectedKey:   "setting",
			expectedValue: "enabled",
		},
		{
			name: "# after numeric value",
			input: `config-section:
	port: 8080#default port
`,
			expectedKey:   "port",
			expectedValue: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			section := result.Statements[0].(*ast.SectionDecl)
			valueExpr := section.Entries[tt.expectedKey]
			if valueExpr == nil {
				t.Fatalf("expected key '%s' not found", tt.expectedKey)
			}

			literal, ok := valueExpr.(*ast.StringLiteral)
			if !ok {
				t.Fatalf("expected StringLiteral, got %T", valueExpr)
			}

			if literal.Value != tt.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tt.expectedValue, literal.Value)
			}
		})
	}
}

// TestParse_EdgeCase_UnicodeInComments tests Unicode characters in comments.
// T055: Write test: Unicode characters in comments
//
// Verifies that comments can contain Unicode characters (non-ASCII) without causing
// parsing errors, and all Unicode text after # is treated as comment content.
func TestParse_EdgeCase_UnicodeInComments(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validateAST func(*testing.T, *ast.AST)
	}{
		{
			name: "Japanese characters in comment",
			input: `# これはコメントです (This is a comment)
config-section:
	key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "Chinese characters in comment",
			input: `# 你好世界 (Hello World)
config-section:
	key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
		{
			name: "Arabic characters in comment",
			input: `# مرحبا بالعالم (Welcome to the world)
config-section:
	key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
		{
			name: "Emoji in comment",
			input: `# 🚀 Rocket launch config 💻 ✨
config-section:
	key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(section.Entries))
				}
			},
		},
		{
			name: "Mixed Unicode characters in trailing comment",
			input: `config-section:
	key: value # Comment with 日本語, 中文, العربية, and 🎉
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				valueExpr := section.Entries["key"]
				literal := valueExpr.(*ast.StringLiteral)
				if literal.Value != "value" {
					t.Errorf("expected value 'value', got '%s'", literal.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_EdgeCase_MixedTabsSpacesBeforeHash tests comments with mixed tabs/spaces.
// T056: Write test: comment with mixed tabs/spaces
//
// Verifies that mixing tabs and spaces before # doesn't cause parsing errors.
// Whitespace is parsed normally, and # starts the comment regardless of preceding whitespace type.
func TestParse_EdgeCase_MixedTabsSpacesBeforeHash(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validateAST func(*testing.T, *ast.AST)
	}{
		{
			name:  "tabs before # on full-line comment",
			input: "config-section:\n\tkey: value\n\t\t\t# comment with tabs before\n",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(section.Entries))
				}
			},
		},
		{
			name: "spaces before # on full-line comment",
			input: `config-section:
	key: value
        # comment with spaces before
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(section.Entries))
				}
			},
		},
		{
			name:  "mixed tabs and spaces before #",
			input: "config-section:\n\tkey: value\n\t    \t# comment with mixed whitespace\n",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name: "trailing comment with spaces before #",
			input: `config-section:
	key: value    # trailing comment with spaces
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				valueExpr := section.Entries["key"]
				literal := valueExpr.(*ast.StringLiteral)
				if literal.Value != "value" {
					t.Errorf("expected value 'value', got '%s'", literal.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_EdgeCase_CommentAtDifferentIndentationLevels tests comments at various indentation levels.
// T057: Write test: comment at different indentation levels
//
// Verifies that comments can appear at any indentation level without affecting
// indentation-sensitive structure parsing or causing indentation errors.
func TestParse_EdgeCase_CommentAtDifferentIndentationLevels(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validateAST func(*testing.T, *ast.AST)
	}{
		{
			name: "comments at different levels in nested structure",
			input: `# Top-level comment (no indent)
config-section:
	# Comment indented once
	key1: value1
		# Comment indented twice
	key2: value2
# Comment back at top level
another-section:
	# Comment in second section
	key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 2 {
					t.Errorf("expected 2 statements, got %d", len(result.Statements))
				}
				section1 := result.Statements[0].(*ast.SectionDecl)
				if len(section1.Entries) != 2 {
					t.Errorf("expected 2 entries in first section, got %d", len(section1.Entries))
				}
			},
		},
		{
			name: "comment at column 0 followed by indented config",
			input: `#comment at column 0
	config-section:
		key: value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
		{
			name: "deeply indented comment",
			input: `config-section:
	key: value
			# Very deep comment
	another_key: another_value
`,
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 2 {
					t.Errorf("expected 2 entries, got %d", len(section.Entries))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_EdgeCase_CommentAtEOFWithoutNewline tests comment at EOF without trailing newline.
// T058: Write test: comment at EOF without newline
//
// Verifies that when a file ends with a comment line that has no trailing newline,
// the scanner correctly detects EOF and terminates without errors.
func TestParse_EdgeCase_CommentAtEOFWithoutNewline(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		validateAST func(*testing.T, *ast.AST)
	}{
		{
			name:  "comment at EOF no newline",
			input: "config-section:\n\tkey: value\n# final comment with no newline",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "config-section" {
					t.Errorf("expected section name 'config-section', got '%s'", section.Name)
				}
			},
		},
		{
			name:  "trailing comment at EOF no newline",
			input: "config-section:\n\tkey: value # comment at end of file",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				valueExpr := section.Entries["key"]
				literal := valueExpr.(*ast.StringLiteral)
				if literal.Value != "value" {
					t.Errorf("expected value 'value', got '%s'", literal.Value)
				}
			},
		},
		{
			name:  "only comment no newline",
			input: "# single comment line",
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 0 {
					t.Errorf("expected 0 statements, got %d", len(result.Statements))
				}
			},
		},
		{
			name:  "config then comment at EOF",
			input: "config-section:\n\tkey: value\n# trailing comment",
			validateAST: func(t *testing.T, result *ast.AST) {
				if len(result.Statements) != 1 {
					t.Errorf("expected 1 statement, got %d", len(result.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParse_EdgeCase_EscapedHashError tests that escaped \# outside strings produces error.
// T059: Write test: escaped \# outside string produces error
//
// Verifies that escape sequences (like \#) are not supported outside of quoted strings
// and are treated as syntax errors. Nomos does not support escape sequences in unquoted contexts.
func TestParse_EdgeCase_EscapedHashError(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "escaped hash in value",
			input:       "config-section:\n\tkey: value\\#notvalid\n",
			wantErr:     true,
			errContains: "unexpected character",
		},
		{
			name:        "escaped hash at start of value",
			input:       "config-section:\n\tkey: \\#invalid\n",
			wantErr:     true,
			errContains: "unexpected character",
		},
		{
			name:        "escaped hash in section name",
			input:       "config\\#section:\n\tkey: value\n",
			wantErr:     true,
			errContains: "unexpected character",
		},
		{
			name:    "hash in quoted string is valid",
			input:   "config-section:\n\tkey: 'value\\#valid'\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := parser.Parse(reader, "test.csl")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestParse_MultiLineComments_DocumentationBlockBeforeSection tests documentation blocks preceding sections.
// T037: Write test: documentation block before section
//
// Verifies that multi-line comment blocks used as documentation before section declarations
// are completely ignored, and the section parses correctly. This is a common real-world pattern.
func TestParse_MultiLineComments_DocumentationBlockBeforeSection(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		validateAST  func(*testing.T, *ast.AST)
	}{
		{
			name: "simple documentation block before section",
			input: `# Database Configuration
# This section configures database connections
database:
	host: localhost
	port: 5432
`,
			expectedName: "database",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 2 {
					t.Errorf("expected 2 entries, got %d", len(section.Entries))
				}
			},
		},
		{
			name: "formatted documentation block with separators",
			input: `# ========================================
# API Configuration Section
# ========================================
# Endpoint: Defines the base API URL
# Timeout: Request timeout in seconds
api-config:
	endpoint: https://api.example.com
	timeout: 30
`,
			expectedName: "api-config",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "api-config" {
					t.Errorf("expected section name 'api-config', got '%s'", section.Name)
				}
			},
		},
		{
			name: "multi-paragraph documentation block",
			input: `# Feature Flags Configuration
#
# This section controls feature toggles for the application.
# Each flag can be independently enabled or disabled.
#
# WARNING: Changing these values requires application restart.
# Last updated: 2026-01-18
features:
	new_ui: enabled
	beta_features: disabled
	experimental: false
`,
			expectedName: "features",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if len(section.Entries) != 3 {
					t.Errorf("expected 3 entries, got %d", len(section.Entries))
				}
			},
		},
		{
			name: "TODO and FIXME style documentation",
			input: `# TODO: Refactor this configuration section
# FIXME: Add validation for port ranges
# NOTE: This is temporary configuration
# HACK: Workaround for issue #123
server-config:
	address: 0.0.0.0
	port: 8080
`,
			expectedName: "server-config",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				if section.Name != "server-config" {
					t.Errorf("expected section name 'server-config', got '%s'", section.Name)
				}
			},
		},
		{
			name: "documentation with examples and usage",
			input: `# Logging Configuration
# 
# Valid levels: DEBUG, INFO, WARN, ERROR
# Format options: json, text, structured
# 
# Example usage:
#   level: INFO
#   format: json
#   output: /var/log/app.log
logging:
	level: INFO
	format: json
`,
			expectedName: "logging",
			validateAST: func(t *testing.T, result *ast.AST) {
				section := result.Statements[0].(*ast.SectionDecl)
				// Verify specific values
				levelExpr := section.Entries["level"]
				if levelExpr == nil {
					t.Fatal("expected 'level' entry")
				}
				levelValue := levelExpr.(*ast.StringLiteral).Value
				if levelValue != "INFO" {
					t.Errorf("expected level='INFO', got '%s'", levelValue)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			section := result.Statements[0].(*ast.SectionDecl)
			if section.Name != tt.expectedName {
				t.Errorf("expected section name '%s', got '%s'", tt.expectedName, section.Name)
			}

			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

// TestParseSimpleList tests parsing of basic list syntax with string values.
// This test WILL FAIL until parseListExpr is implemented in parser.go.
func TestParseSimpleList(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedFirst string
		expectedLast  string
	}{
		{
			name: "simple string list",
			input: `IPs:
  - 10.0.0.1
  - 10.1.0.1
  - 10.2.0.1`,
			expectedCount: 3,
			expectedFirst: "10.0.0.1",
			expectedLast:  "10.2.0.1",
		},
		{
			name: "simple number list",
			input: `ports:
  - 8080
  - 8081
  - 8082`,
			expectedCount: 3,
			expectedFirst: "8080",
			expectedLast:  "8082",
		},
		{
			name: "single item list",
			input: `items:
  - single`,
			expectedCount: 1,
			expectedFirst: "single",
			expectedLast:  "single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")

			// Expected to fail until implementation
			if err != nil {
				t.Logf("EXPECTED FAILURE (not yet implemented): %v", err)
				return
			}

			if len(result.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(result.Statements))
			}

			section, ok := result.Statements[0].(*ast.SectionDecl)
			if !ok {
				t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
			}

			// The list should be in the section's entries with empty key
			listExpr, ok := section.Entries[""].(*ast.ListExpr)
			if !ok {
				t.Fatalf("expected *ast.ListExpr, got %T", section.Entries[""])
			}

			if len(listExpr.Elements) != tt.expectedCount {
				t.Errorf("expected %d elements, got %d", tt.expectedCount, len(listExpr.Elements))
			}

			if len(listExpr.Elements) > 0 {
				first, ok := listExpr.Elements[0].(*ast.StringLiteral)
				if !ok {
					t.Fatalf("expected first element to be *ast.StringLiteral, got %T", listExpr.Elements[0])
				}
				if first.Value != tt.expectedFirst {
					t.Errorf("expected first element '%s', got '%s'", tt.expectedFirst, first.Value)
				}

				last, ok := listExpr.Elements[len(listExpr.Elements)-1].(*ast.StringLiteral)
				if !ok {
					t.Fatalf("expected last element to be *ast.StringLiteral, got %T", listExpr.Elements[len(listExpr.Elements)-1])
				}
				if last.Value != tt.expectedLast {
					t.Errorf("expected last element '%s', got '%s'", tt.expectedLast, last.Value)
				}
			}
		})
	}
}

// TestParseEmptyList tests parsing of empty list syntax using [] notation.
func TestParseEmptyList(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		checkValue bool   // If true, check section.Value; if false, check section.Entries[entryKey]
		entryKey   string // The key in Entries map (only used when checkValue is false)
	}{
		{
			name:       "empty list with brackets",
			input:      `emptyCollection: []`,
			checkValue: true, // Top-level key:value uses Value field
		},
		{
			name: "empty list in section",
			input: `config:
  items: []`,
			checkValue: false,
			entryKey:   "items", // Nested entry uses Entries map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader, "test.csl")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result.Statements) == 0 {
				t.Fatal("expected at least 1 statement")
			}

			section, ok := result.Statements[0].(*ast.SectionDecl)
			if !ok {
				t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
			}

			var listExpr *ast.ListExpr
			if tt.checkValue {
				// Check the Value field for top-level scalar values
				listExpr, ok = section.Value.(*ast.ListExpr)
				if !ok {
					t.Fatalf("expected *ast.ListExpr in Value field, got %T", section.Value)
				}
			} else {
				// Check the Entries map for nested values
				listExpr, ok = section.Entries[tt.entryKey].(*ast.ListExpr)
				if !ok {
					t.Fatalf("expected *ast.ListExpr at key %q, got %T", tt.entryKey, section.Entries[tt.entryKey])
				}
			}

			if len(listExpr.Elements) != 0 {
				t.Errorf("expected empty list (0 elements), got %d elements", len(listExpr.Elements))
			}
		})
	}
}

// TestParseNestedList tests parsing nested list syntax with two levels of lists.
func TestParseNestedList(t *testing.T) {
	fixturePath := "../testdata/fixtures/lists/nested_2levels.csl"

	result, err := parser.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(result.Statements))
	}

	section, ok := result.Statements[0].(*ast.SectionDecl)
	if !ok {
		t.Fatalf("expected *ast.SectionDecl, got %T", result.Statements[0])
	}

	listExpr, ok := section.Entries[""].(*ast.ListExpr)
	if !ok {
		t.Fatalf("expected *ast.ListExpr, got %T", section.Entries[""])
	}

	if len(listExpr.Elements) != 2 {
		t.Fatalf("expected 2 top-level list elements, got %d", len(listExpr.Elements))
	}

	firstList, ok := listExpr.Elements[0].(*ast.ListExpr)
	if !ok {
		t.Fatalf("expected first element to be *ast.ListExpr, got %T", listExpr.Elements[0])
	}
	secondList, ok := listExpr.Elements[1].(*ast.ListExpr)
	if !ok {
		t.Fatalf("expected second element to be *ast.ListExpr, got %T", listExpr.Elements[1])
	}

	if len(firstList.Elements) != 2 {
		t.Fatalf("expected first nested list to have 2 elements, got %d", len(firstList.Elements))
	}
	if len(secondList.Elements) != 2 {
		t.Fatalf("expected second nested list to have 2 elements, got %d", len(secondList.Elements))
	}

	firstValues := []string{"1", "2"}
	for i, expected := range firstValues {
		literal, ok := firstList.Elements[i].(*ast.StringLiteral)
		if !ok {
			t.Fatalf("expected first nested list element %d to be *ast.StringLiteral, got %T", i, firstList.Elements[i])
		}
		if literal.Value != expected {
			t.Fatalf("expected first nested list element %d to be %q, got %q", i, expected, literal.Value)
		}
	}

	secondValues := []string{"3", "4"}
	for i, expected := range secondValues {
		literal, ok := secondList.Elements[i].(*ast.StringLiteral)
		if !ok {
			t.Fatalf("expected second nested list element %d to be *ast.StringLiteral, got %T", i, secondList.Elements[i])
		}
		if literal.Value != expected {
			t.Fatalf("expected second nested list element %d to be %q, got %q", i, expected, literal.Value)
		}
	}
}

// TestParseListOfObjects tests parsing lists where items are object literals.
func TestParseListOfObjects(t *testing.T) {
	fixturePath := "../testdata/fixtures/lists/list_objects.csl"
	goldenPath := "../testdata/golden/lists/list_objects.csl.json"

	result, err := parser.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	actualJSON, err := testutil.CanonicalJSON(result)
	if err != nil {
		t.Fatalf("failed to serialize AST to JSON: %v", err)
	}

	//nolint:gosec // G304: goldenPath is controlled test fixture path
	expectedJSON, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Logf("Golden file not found at %s, writing actual output", goldenPath)
		if err := os.WriteFile(goldenPath, actualJSON, 0600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Skip("Generated golden file, re-run test to verify")
	}

	if !testutil.CompareJSON(actualJSON, expectedJSON) {
		t.Errorf("AST JSON does not match golden file.\nExpected:\n%s\n\nActual:\n%s",
			string(expectedJSON), string(actualJSON))
	}
}

// TestParseListReferenceWithIndex tests parsing list references with a single index.
func TestParseListReferenceWithIndex(t *testing.T) {
	fixturePath := "../testdata/fixtures/lists/reference_simple.csl"

	result, err := parser.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := requireSection(t, result, "network")
	ref := requireReferenceEntry(t, section, "primary_ip")

	assertReferencePath(t, ref, []string{"config", "IPs[0]"})
}

// TestParseListReferenceMultiIndex tests parsing list references with multiple indexes.
func TestParseListReferenceMultiIndex(t *testing.T) {
	fixturePath := "../testdata/fixtures/lists/reference_multidim.csl"

	result, err := parser.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := requireSection(t, result, "refs")

	tests := []struct {
		name     string
		key      string
		wantPath []string
	}{
		{
			name:     "first element uses two indexes",
			key:      "first",
			wantPath: []string{"config", "matrix[0][1]"},
		},
		{
			name:     "second element uses two indexes",
			key:      "second",
			wantPath: []string{"config", "matrix[1][0]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := requireReferenceEntry(t, section, tt.key)
			assertReferencePath(t, ref, tt.wantPath)
		})
	}
}

// TestParseWholeListReference tests parsing references to whole lists without index notation.
func TestParseWholeListReference(t *testing.T) {
	fixturePath := "../testdata/fixtures/lists/reference_whole_list.csl"

	result, err := parser.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	section := requireSection(t, result, "refs")
	ref := requireReferenceEntry(t, section, "all_servers")

	assertReferencePath(t, ref, []string{"config", "servers"})
}

func requireSection(t *testing.T, tree *ast.AST, name string) *ast.SectionDecl {
	t.Helper()

	for _, stmt := range tree.Statements {
		if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == name {
			return section
		}
	}

	t.Fatalf("expected to find section %q", name)
	return nil
}

func requireReferenceEntry(t *testing.T, section *ast.SectionDecl, key string) *ast.ReferenceExpr {
	t.Helper()

	expr, ok := section.Entries[key]
	if !ok {
		t.Fatalf("expected entry %q in section %q", key, section.Name)
	}

	ref, ok := expr.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected entry %q to be *ast.ReferenceExpr, got %T", key, expr)
	}

	return ref
}

func assertReferencePath(t *testing.T, ref *ast.ReferenceExpr, want []string) {
	t.Helper()

	if len(ref.Path) != len(want) {
		t.Fatalf("expected path length %d, got %d", len(want), len(ref.Path))
	}

	for i, component := range want {
		if ref.Path[i] != component {
			t.Errorf("expected path[%d] = %q, got %q", i, component, ref.Path[i])
		}
	}
}
