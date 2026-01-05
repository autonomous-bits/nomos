// Package parser provides functions to parse Nomos configuration files (.csl)
// into an Abstract Syntax Tree (AST).
//
// The parser accepts input via ParseFile (for filesystem paths) or Parse
// (for io.Reader). All parse errors include precise source location information.
//
// Example usage:
//
//	ast, err := parser.ParseFile("config.csl")
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Process ast...
package parser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/autonomous-bits/nomos/libs/parser/internal/scanner"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Parser represents a parser instance. It can be reused for multiple parse operations.
// Parser instances are safe for concurrent use and can be pooled via sync.Pool for
// high-throughput scenarios.
//
// The parser stores source text internally during parsing for error context generation,
// but maintains no state between Parse/ParseFile calls.
type Parser struct {
	// sourceText stores the source text for the current parse operation.
	// It is used for error formatting and context generation.
	// This field is set at the start of each Parse/ParseFile call.
	sourceText string
	// Future fields for configuration options can be added here.
	// Examples: strict mode flags, custom error handlers, debug options.
}

// Option is a functional option for configuring a Parser.
// Currently no options are implemented, but this pattern provides a forward-compatible
// extension point for future parser configuration without breaking the API.
//
// Example future usage:
//
//	p := NewParser(WithStrictMode(true), WithMaxDepth(100))
//
// The parser can be used without any options:
//
//	p := NewParser() // Uses default configuration
type Option func(*Parser)

// NewParser creates a new Parser with the given options.
// Currently accepts options for future extensibility but none are implemented yet.
// Parser instances can be reused across multiple Parse/ParseFile calls and are
// safe for concurrent use when each goroutine has its own instance.
func NewParser(opts ...Option) *Parser {
	p := &Parser{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ParseFile parses a Nomos configuration file from the filesystem.
// It returns an AST or an error with precise location information.
func ParseFile(path string) (*ast.AST, error) {
	p := NewParser()
	return p.ParseFile(path)
}

// ParseFile parses a file using this parser instance.
func (p *Parser) ParseFile(path string) (*ast.AST, error) {
	//nolint:gosec // G304: Path is controlled by caller, legitimate API surface for file parsing
	file, err := os.Open(path)
	if err != nil {
		return nil, NewParseError(IOError, path, 0, 0, fmt.Sprintf("failed to open file: %v", err))
	}
	defer func() {
		_ = file.Close() // Explicitly ignore close error on read-only file
	}()

	return p.Parse(file, path)
}

// Parse parses Nomos configuration from an io.Reader.
// The filename parameter is used for error messages and source spans.
func Parse(r io.Reader, filename string) (*ast.AST, error) {
	p := NewParser()
	return p.Parse(r, filename)
}

// Parse parses input using this parser instance.
func (p *Parser) Parse(r io.Reader, filename string) (*ast.AST, error) {
	// Read all input
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, NewParseError(IOError, filename, 0, 0, fmt.Sprintf("failed to read input: %v", err))
	}

	// Store source text for error formatting
	p.sourceText = string(content)

	// Create scanner
	s := scanner.New(p.sourceText, filename)

	// Parse statements
	statements, err := p.parseStatements(s)
	if err != nil {
		return nil, err
	}

	// Build AST
	astNode := &ast.AST{
		Statements: statements,
		SourceSpan: ast.SourceSpan{
			Filename:  filename,
			StartLine: 1,
			StartCol:  1,
			EndLine:   s.Line(),
			EndCol:    s.Column(),
		},
	}

	return astNode, nil
}

// parseStatements parses all statements in the input.
func (p *Parser) parseStatements(s *scanner.Scanner) ([]ast.Stmt, error) {
	var statements []ast.Stmt

	for !s.IsEOF() {
		// Skip whitespace and empty lines
		s.SkipWhitespace()
		if s.IsEOF() {
			break
		}

		stmt, err := p.parseStatement(s)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	return statements, nil
}

// expectColonAfterKeyword validates that a colon follows a keyword and consumes it.
// This helper reduces code duplication in keyword parsing.
func (p *Parser) expectColonAfterKeyword(s *scanner.Scanner, keyword string) error {
	if s.PeekChar() != ':' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			fmt.Sprintf("invalid syntax: '%s' keyword must be followed by ':'", keyword))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
		return err
	}
	_ = s.Expect(':') // Consume colon (already validated)
	return nil
}

// parseStatement parses a single statement.
func (p *Parser) parseStatement(s *scanner.Scanner) (ast.Stmt, error) {
	startLine, startCol := s.Line(), s.Column()

	// Check for invalid characters first
	ch := s.PeekChar()
	if ch == '!' || ch == '@' || ch == '#' || ch == '$' || ch == '%' || ch == '^' || ch == '&' || ch == '*' || ch == '(' || ch == ')' {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: unexpected character '%c'", ch))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	// Peek at the first token to determine statement type
	token := s.PeekToken()

	// Skip empty lines
	if token == "" {
		s.SkipToNextLine()
		return nil, nil
	}

	switch token {
	case "source":
		return p.parseSourceDecl(s, startLine, startCol)
	case "import":
		return p.parseImportStmt(s, startLine, startCol)
	case "reference":
		return nil, p.parseReferenceStmt(s, startLine, startCol)
	default:
		// Try to parse as a section declaration
		if ch != '\n' && ch != '\r' && !s.IsEOF() {
			return p.parseSectionDecl(s, startLine, startCol)
		}
		// Skip unknown/empty lines
		s.SkipToNextLine()
		return nil, nil
	}
}

// parseSourceDecl parses a source declaration.
func (p *Parser) parseSourceDecl(s *scanner.Scanner, startLine, startCol int) (*ast.SourceDecl, error) {
	s.ConsumeToken() // consume "source"

	// Validate colon after keyword
	if err := p.expectColonAfterKeyword(s, "source"); err != nil {
		return nil, err
	}

	// Parse configuration block (indented key-value pairs)
	config, err := p.parseConfigBlock(s)
	if err != nil {
		return nil, err
	}

	// Extract and validate alias field
	aliasExpr, ok := config["alias"]
	if !ok {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			"invalid syntax: 'source' declaration requires an 'alias' field")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	// Alias must be a StringLiteral
	aliasLiteral, ok := aliasExpr.(*ast.StringLiteral)
	if !ok {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			"invalid syntax: 'source' alias must be a string literal, not a reference")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	alias := aliasLiteral.Value
	if alias == "" {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			"invalid syntax: 'source' declaration requires a non-empty 'alias' field")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	// Extract type (optional)
	typeName := ""
	if typeExpr, ok := config["type"]; ok {
		if typeLiteral, ok := typeExpr.(*ast.StringLiteral); ok {
			typeName = typeLiteral.Value
		}
	}

	// Extract and validate version (optional)
	version := ""
	if versionExpr, ok := config["version"]; ok {
		if versionLiteral, ok := versionExpr.(*ast.StringLiteral); ok {
			version = versionLiteral.Value
		}
	}

	// Validate semver format if version is provided
	if err := validateSemver(version); err != nil {
		parseErr := NewParseError(SyntaxError, s.Filename(), startLine, startCol, err.Error())
		parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, parseErr
	}

	// Remove reserved fields from config map (extracted to dedicated fields)
	delete(config, "alias")
	delete(config, "type")
	delete(config, "version")

	endLine, endCol := s.Line(), s.Column()

	return &ast.SourceDecl{
		Alias:   alias,
		Type:    typeName,
		Version: version,
		Config:  config,
		SourceSpan: ast.SourceSpan{
			Filename:  s.Filename(),
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
		},
	}, nil
}

// parseImportStmt parses an import statement.
func (p *Parser) parseImportStmt(s *scanner.Scanner, startLine, startCol int) (*ast.ImportStmt, error) {
	s.ConsumeToken() // consume "import"

	// Validate colon after keyword
	if err := p.expectColonAfterKeyword(s, "import"); err != nil {
		return nil, err
	}

	alias := s.ReadIdentifier()

	// Validate alias is not empty
	if alias == "" {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'import' statement requires an alias")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
		return nil, err
	}

	path := ""

	// Check for optional path
	if s.PeekChar() == ':' {
		s.Advance() // consume ':'
		path = s.ReadIdentifier()
	}

	s.SkipToNextLine()
	endLine, endCol := s.Line(), s.Column()

	return &ast.ImportStmt{
		Alias: alias,
		Path:  path,
		SourceSpan: ast.SourceSpan{
			Filename:  s.Filename(),
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
		},
	}, nil
}

// parseReferenceStmt parses a reference statement.
// NOTE: Top-level reference statements are deprecated (BREAKING CHANGE).
// Users should use inline references in value positions instead.
func (p *Parser) parseReferenceStmt(s *scanner.Scanner, startLine, startCol int) error {
	// Capture the full reference statement text for the error message
	// Save current position to extract the reference statement
	lineStartPos := s.Pos()

	// Read the full reference statement (reference:alias:path)
	referenceText := s.ReadValue()
	if referenceText == "" {
		// Fallback if ReadValue returns empty
		referenceText = "reference"
	}

	// Simple error message - references can only be used inline
	errorMessage := "invalid syntax: references can only be used inline in value positions"

	// Reset scanner position to where we started (before ReadValue)
	// This ensures the error snippet points to the correct location
	// Note: We accept the position has moved but the snippet generation
	// will use startLine/startCol which are correct
	_ = lineStartPos // Mark as intentionally unused

	err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, errorMessage)
	err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
	return err
}

// parseSectionDecl parses a configuration section.
func (p *Parser) parseSectionDecl(s *scanner.Scanner, startLine, startCol int) (*ast.SectionDecl, error) {
	name := s.ReadIdentifier()
	if s.PeekChar() != ':' {
		// Not a valid section declaration - this is invalid syntax
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: expected ':' after identifier '%s'", name))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}
	_ = s.Expect(':') // Error already checked via PeekChar

	// Parse indented entries
	entries, err := p.parseConfigBlock(s)
	if err != nil {
		return nil, err
	}

	endLine, endCol := s.Line(), s.Column()

	return &ast.SectionDecl{
		Name:    name,
		Entries: entries,
		SourceSpan: ast.SourceSpan{
			Filename:  s.Filename(),
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
		},
	}, nil
}

// parseConfigBlock parses an indented block of key-value pairs.
// It can now handle nested map structures.
func (p *Parser) parseConfigBlock(s *scanner.Scanner) (map[string]ast.Expr, error) {
	config := make(map[string]ast.Expr)

	s.SkipToNextLine()

	// Get the base indentation level for this block
	var baseIndent int
	if !s.IsEOF() && s.IsIndented() {
		baseIndent = s.GetIndentLevel()
	}

	// Parse indented key-value pairs
	for !s.IsEOF() {
		// Check if line is indented at the expected level
		if !s.IsIndented() {
			break
		}

		currentIndent := s.GetIndentLevel()
		if currentIndent < baseIndent {
			// Less indented - end of this block
			break
		}

		s.SkipWhitespace()

		if s.IsEOF() || s.PeekChar() == '\n' {
			break
		}

		keyStartLine := s.Line()
		keyStart := s.Column()
		key := s.ReadIdentifier()

		// Validate key is not empty
		if key == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), keyStart,
				"invalid syntax: expected identifier for key")
		}

		// Validate key doesn't contain invalid characters (check first char of what remains)
		ch := s.PeekChar()
		if ch != ':' && ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' && !s.IsEOF() {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				fmt.Sprintf("invalid syntax: invalid character '%c' in key", ch))
		}

		s.SkipWhitespace()
		if err := s.Expect(':'); err != nil {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				"invalid syntax: expected ':' after key")
		}
		s.SkipWhitespace()

		// Check if this is a nested map or a value
		// A nested map is indicated by: key: \n with more indentation following
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			// Newline after colon - might be a nested map
			s.SkipToNextLine()

			// Check if next line is more indented
			if !s.IsEOF() && s.IsIndented() {
				nextIndent := s.GetIndentLevel()
				if nextIndent > currentIndent {
					// This is a nested map
					nestedEntries, err := p.parseNestedMap(s, nextIndent)
					if err != nil {
						return nil, err
					}

					// Create MapExpr
					endLine, endCol := s.Line(), s.Column()
					config[key] = &ast.MapExpr{
						Entries: nestedEntries,
						SourceSpan: ast.SourceSpan{
							Filename:  s.Filename(),
							StartLine: keyStartLine,
							StartCol:  keyStart,
							EndLine:   endLine,
							EndCol:    endCol,
						},
					}
					continue
				}
			}

			// Empty value (newline but no nested content)
			config[key] = &ast.StringLiteral{
				Value: "",
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: keyStartLine,
					StartCol:  keyStart,
					EndLine:   keyStartLine,
					EndCol:    keyStart + len(key) + 1,
				},
			}
			continue
		}

		// Parse value expression (can be StringLiteral, ReferenceExpr, or nested map)
		valueStartLine, valueStartCol := s.Line(), s.Column()
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		config[key] = valueExpr
		s.SkipToNextLine()
	}

	return config, nil
}

// parseNestedMap parses a nested map at a specific indentation level.
func (p *Parser) parseNestedMap(s *scanner.Scanner, expectedIndent int) (map[string]ast.Expr, error) {
	entries := make(map[string]ast.Expr)

	for !s.IsEOF() {
		// Check if we're still at the correct indentation level
		if !s.IsIndented() {
			break
		}

		currentIndent := s.GetIndentLevel()
		if currentIndent < expectedIndent {
			// Less indented - end of nested map
			break
		}

		if currentIndent > expectedIndent {
			// More indented than expected - skip this line (might be part of a deeper nesting)
			s.SkipToNextLine()
			continue
		}

		s.SkipWhitespace()

		if s.IsEOF() || s.PeekChar() == '\n' {
			s.SkipToNextLine()
			continue
		}

		keyStartLine := s.Line()
		keyStart := s.Column()
		key := s.ReadIdentifier()

		if key == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), keyStart,
				"invalid syntax: expected identifier for key in nested map")
		}

		ch := s.PeekChar()
		if ch != ':' && ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' && !s.IsEOF() {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				fmt.Sprintf("invalid syntax: invalid character '%c' in key", ch))
		}

		s.SkipWhitespace()
		if err := s.Expect(':'); err != nil {
			return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				"invalid syntax: expected ':' after key")
		}
		s.SkipWhitespace()

		// Check for nested map
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()

			if !s.IsEOF() && s.IsIndented() {
				nextIndent := s.GetIndentLevel()
				if nextIndent > currentIndent {
					// Recursively parse nested map
					nestedEntries, err := p.parseNestedMap(s, nextIndent)
					if err != nil {
						return nil, err
					}

					endLine, endCol := s.Line(), s.Column()
					entries[key] = &ast.MapExpr{
						Entries: nestedEntries,
						SourceSpan: ast.SourceSpan{
							Filename:  s.Filename(),
							StartLine: keyStartLine,
							StartCol:  keyStart,
							EndLine:   endLine,
							EndCol:    endCol,
						},
					}
					continue
				}
			}

			// Empty value
			entries[key] = &ast.StringLiteral{
				Value: "",
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: keyStartLine,
					StartCol:  keyStart,
					EndLine:   keyStartLine,
					EndCol:    keyStart + len(key) + 1,
				},
			}
			continue
		}

		// Parse scalar value
		valueStartLine, valueStartCol := s.Line(), s.Column()
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		entries[key] = valueExpr
		s.SkipToNextLine()
	}

	return entries, nil
}

// parseValueExpr parses a value expression, which can be either a string literal
// or an inline reference expression of the form reference:alias:dotted.path
func (p *Parser) parseValueExpr(s *scanner.Scanner, startLine, startCol int) (ast.Expr, error) {
	// startLine and startCol point to where the value starts (after skipping whitespace)

	// ReadValue() reads the value and trims quotes/whitespace
	valueText := s.ReadValue()

	// Validate that strings are properly terminated (ReadValue only strips matching quotes)
	if len(valueText) > 0 && (valueText[0] == '\'' || valueText[0] == '"') {
		quote := valueText[0]
		return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			fmt.Sprintf("invalid syntax: unterminated string (missing closing %c)", quote))
	}

	// Check if this is an inline reference expression
	if strings.HasPrefix(valueText, "reference:") {
		// Parse reference expression: reference:alias:dotted.path
		parts := strings.SplitN(valueText, ":", 3)
		if len(parts) < 3 {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: inline reference must be 'reference:alias:path'")
		}

		alias := parts[1]
		pathStr := parts[2]

		// Validate alias
		if alias == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: inline reference requires a non-empty alias")
		}

		// Validate path
		if pathStr == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: inline reference requires a non-empty path")
		}

		// Split path by dots
		pathComponents := strings.Split(pathStr, ".")

		// Calculate the end column (1-indexed, inclusive)
		// startCol is the 1-indexed column where the value starts
		// EndCol should point to the last character of the value (inclusive)
		refEndCol := startCol + len(valueText) - 1

		return &ast.ReferenceExpr{
			Alias: alias,
			Path:  pathComponents,
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: startLine,
				StartCol:  startCol,
				EndLine:   startLine, // Inline references are single-line
				EndCol:    refEndCol,
			},
		}, nil
	}

	// Plain string literal (ReadValue has already stripped quotes if any)
	// EndCol is 1-indexed and inclusive (points to last character)
	literalEndCol := startCol + len(valueText) - 1
	if len(valueText) == 0 {
		literalEndCol = startCol
	}

	return &ast.StringLiteral{
		Value: valueText,
		SourceSpan: ast.SourceSpan{
			Filename:  s.Filename(),
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   startLine,
			EndCol:    literalEndCol,
		},
	}, nil
}

// generateSnippetFromSource is a convenience wrapper around generateSnippet from errors.go.
func generateSnippetFromSource(sourceText string, line, col int) string {
	return generateSnippet(sourceText, line, col)
}

// validateSemver validates that a version string is valid semantic versioning format.
// Empty strings are valid (representing unversioned providers).
// Returns an error with actionable guidance if the version is invalid.
func validateSemver(version string) error {
	if version == "" {
		return nil // Empty is valid (unversioned provider)
	}
	_, err := semver.StrictNewVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version format: %q - must be valid semantic version (e.g., \"1.2.3\", \"2.0.0-beta.1\"). See https://semver.org", version)
	}
	return nil
}
