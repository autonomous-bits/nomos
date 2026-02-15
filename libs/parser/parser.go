// Package parser provides functions to parse Nomos configuration files (.csl)
// into an Abstract Syntax Tree (AST).
//
// The parser accepts input via ParseFile (for filesystem paths) or Parse
// (for io.Reader). All parse errors include precise source location information.
//
// The parser supports YAML-style comments using the '#' notation. Comments extend
// from the '#' character to the end of the line and are ignored during parsing.
// The '#' character is treated as a comment delimiter only when it appears outside
// quoted strings; within strings, '#' is preserved as literal content.
//
// Example usage:
//
//	ast, err := parser.ParseFile("config.csl")
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Process ast...
package parser //nolint:revive // public package name is intentional and descriptive

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

	// Check for comments first - skip comment lines
	ch := s.PeekChar()
	if s.IsCommentStart() {
		s.SkipComment()
		s.SkipToNextLine()
		return nil, nil
	}

	// Check for invalid characters (@ is now valid for references)
	if ch == '!' || ch == '$' || ch == '%' || ch == '^' || ch == '&' || ch == '*' || ch == '(' || ch == ')' {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: unexpected character '%c'", ch))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	if ch == '@' {
		refExpr, err := p.parseValueExpr(s, startLine, startCol)
		if err != nil {
			return nil, err
		}
		ref, ok := refExpr.(*ast.ReferenceExpr)
		if !ok {
			parseErr := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: standalone references must use @alias:path")
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
			return nil, parseErr
		}
		s.SkipToNextLine()
		return &ast.SpreadStmt{
			Reference:  ref,
			SourceSpan: ref.SourceSpan,
		}, nil
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
		// Import statement no longer supported - return clear error
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			"import statement no longer supported; use @alias:path syntax instead")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
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
	configEntries, err := p.parseConfigBlock(s)
	if err != nil {
		return nil, err
	}

	config := make(map[string]ast.Expr)
	for _, entry := range configEntries {
		if entry.Spread || entry.Key == "" {
			parseErr := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: 'source' declaration does not allow spread or empty keys")
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
			return nil, parseErr
		}
		config[entry.Key] = entry.Value
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

// parseReferenceStmt parses a reference statement.
// NOTE: Top-level reference statements are deprecated (BREAKING CHANGE).
// Users should use inline references in value positions instead.
func (p *Parser) parseReferenceStmt(s *scanner.Scanner, startLine, startCol int) error {
	// Simple error message - references can only be used inline
	errorMessage := "invalid syntax: references can only be used inline in value positions"

	err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, errorMessage)
	err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
	return err
}

// parseSectionDecl parses a configuration section.
func (p *Parser) parseSectionDecl(s *scanner.Scanner, startLine, startCol int) (*ast.SectionDecl, error) {
	name := s.ReadIdentifier()

	// Check for unexpected characters after identifier (FR-014)
	ch := s.PeekChar()
	if ch == '\\' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			fmt.Sprintf("invalid syntax: unexpected character '%c'", ch))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
		return nil, err
	}

	if ch != ':' {
		// Not a valid section declaration - this is invalid syntax
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: expected ':' after identifier '%s'", name))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}
	_ = s.Expect(':') // Error already checked via PeekChar

	s.SkipWhitespace()
	if !s.IsEOF() && s.PeekChar() != '\n' && s.PeekChar() != '\r' && !s.IsCommentStart() {
		valueStartLine, valueStartCol := s.Line(), s.Column()
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		endLine, endCol := s.Line(), s.Column()
		s.SkipToNextLine()

		// Inline scalar value - set Value field, not Entries
		return &ast.SectionDecl{
			Name:    name,
			Value:   valueExpr, // Direct scalar value, no empty-string key
			Entries: nil,       // nil indicates inline scalar, not nested map
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: startLine,
				StartCol:  startCol,
				EndLine:   endLine,
				EndCol:    endCol,
			},
		}, nil
	}

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
// It can now handle nested map structures and direct lists (when a section contains only list items).
func (p *Parser) parseConfigBlock(s *scanner.Scanner) ([]ast.MapEntry, error) {
	config := make([]ast.MapEntry, 0)

	s.SkipToNextLine()
	if p.isWhitespaceOnlyIndentedBlock(s) {
		startLine, startCol := s.Line(), s.Column()
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, listWhitespaceOnlyErrorMessage())
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	baseIndent, hasIndent := p.findBaseIndent(s)
	if !hasIndent {
		if p.isWhitespaceOnlyIndentedBlock(s) {
			startLine, startCol := s.Line(), s.Column()
			err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, listWhitespaceOnlyErrorMessage())
			err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
			return nil, err
		}
		return config, nil
	}

	listOnly := p.isListOnlyBlock(s, baseIndent)
	if listOnly {
		if !p.seekToIndentedContent(s, baseIndent) {
			return config, nil
		}
		listLine, listCol := s.Line(), s.Column()
		listExpr, err := p.parseListExpr(s, baseIndent, 1, listLine, listCol)
		if err != nil {
			return nil, err
		}
		config = append(config, ast.MapEntry{
			Key:   "",
			Value: listExpr,
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: listLine,
				StartCol:  listCol,
				EndLine:   listExpr.Span().EndLine,
				EndCol:    listExpr.Span().EndCol,
			},
		})
		return config, nil
	}

	listSnapshot := s.Snapshot()
	if p.seekToIndentedContent(s, baseIndent) && p.isListItemMarker(s) {
		listLine, listCol := s.Line(), s.Column()
		listExpr, err := p.parseListExpr(s, baseIndent, 1, listLine, listCol)
		if err != nil {
			return nil, err
		}
		config = append(config, ast.MapEntry{
			Key:   "",
			Value: listExpr,
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: listLine,
				StartCol:  listCol,
				EndLine:   listExpr.Span().EndLine,
				EndCol:    listExpr.Span().EndCol,
			},
		})
		return config, nil
	}
	s.Restore(listSnapshot)

	// Parse indented key-value pairs
	for !s.IsEOF() {
		// Check if line is indented at the expected level
		if !s.IsIndented() {
			break
		}

		s.SkipWhitespace()
		currentIndent := s.Column() - 1
		if currentIndent < baseIndent {
			// Less indented - end of this block
			break
		}

		if s.IsEOF() || s.PeekChar() == '\n' {
			break
		}

		// Skip comment lines inside section body
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}

		if p.isListItemMarker(s) {
			listLine, listCol := s.Line(), s.Column()
			if _, err := p.parseListExpr(s, currentIndent, 1, listLine, listCol); err != nil {
				return nil, err
			}
			parseErr := NewParseError(SyntaxError, s.Filename(), listLine, listCol,
				"invalid syntax: list items must be nested under a key")
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, listLine, listCol))
			return nil, parseErr
		}

		if s.PeekChar() == '@' {
			refStartLine, refStartCol := s.Line(), s.Column()
			refExpr, err := p.parseValueExpr(s, refStartLine, refStartCol)
			if err != nil {
				return nil, err
			}
			ref, ok := refExpr.(*ast.ReferenceExpr)
			if !ok {
				parseErr := NewParseError(SyntaxError, s.Filename(), refStartLine, refStartCol,
					"invalid syntax: standalone references must use @alias:path")
				parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, refStartLine, refStartCol))
				return nil, parseErr
			}
			config = append(config, ast.MapEntry{
				Value:  ref,
				Spread: true,
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: ref.SourceSpan.StartLine,
					StartCol:  ref.SourceSpan.StartCol,
					EndLine:   ref.SourceSpan.EndLine,
					EndCol:    ref.SourceSpan.EndCol,
				},
			})
			s.SkipToNextLine()
			continue
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

		// Check if this is a nested map, list, or a value
		// A nested map is indicated by: key: \n with more indentation following
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			// Newline after colon - might be a nested map or list
			s.SkipToNextLine()

			// Find the next non-empty, non-comment line
			foundContent := false
			for !s.IsEOF() && s.IsIndented() {
				s.SkipWhitespace()
				if s.IsCommentStart() {
					s.SkipComment()
					s.SkipToNextLine()
					continue
				}
				if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
					s.SkipToNextLine()
					continue
				}
				foundContent = true
				break
			}

			if foundContent {
				nextIndent := s.Column() - 1
				if nextIndent > currentIndent {
					if p.isListItemMarker(s) {
						listExpr, err := p.parseListExpr(s, nextIndent, 1, keyStartLine, keyStart)
						if err != nil {
							return nil, err
						}
						config = append(config, ast.MapEntry{
							Key:   key,
							Value: listExpr,
							SourceSpan: ast.SourceSpan{
								Filename:  s.Filename(),
								StartLine: keyStartLine,
								StartCol:  keyStart,
								EndLine:   listExpr.Span().EndLine,
								EndCol:    listExpr.Span().EndCol,
							},
						})
						continue
					}

					nestedEntries, err := p.parseNestedMap(s, nextIndent)
					if err != nil {
						return nil, err
					}

					endLine := s.Line()
					endCol := p.mapEndColumn(s)
					config = append(config, ast.MapEntry{
						Key: key,
						Value: &ast.MapExpr{
							Entries: nestedEntries,
							SourceSpan: ast.SourceSpan{
								Filename:  s.Filename(),
								StartLine: keyStartLine,
								StartCol:  keyStart,
								EndLine:   endLine,
								EndCol:    endCol,
							},
						},
						SourceSpan: ast.SourceSpan{
							Filename:  s.Filename(),
							StartLine: keyStartLine,
							StartCol:  keyStart,
							EndLine:   endLine,
							EndCol:    endCol,
						},
					})
					continue
				}
			}

			// Empty value (newline but no nested content)
			config = append(config, ast.MapEntry{
				Key: key,
				Value: &ast.StringLiteral{
					Value: "",
					SourceSpan: ast.SourceSpan{
						Filename:  s.Filename(),
						StartLine: keyStartLine,
						StartCol:  keyStart,
						EndLine:   keyStartLine,
						EndCol:    keyStart + len(key) + 1,
					},
				},
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: keyStartLine,
					StartCol:  keyStart,
					EndLine:   keyStartLine,
					EndCol:    keyStart + len(key) + 1,
				},
			})
			continue
		}

		// Parse value expression (can be StringLiteral, ReferenceExpr, or nested map)
		valueStartLine, valueStartCol := s.Line(), s.Column()
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		config = append(config, ast.MapEntry{
			Key:   key,
			Value: valueExpr,
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: keyStartLine,
				StartCol:  keyStart,
				EndLine:   valueExpr.Span().EndLine,
				EndCol:    valueExpr.Span().EndCol,
			},
		})
		s.SkipToNextLine()
	}

	return config, nil
}

// parseNestedMap parses a nested map at a specific indentation level.
func (p *Parser) parseNestedMap(s *scanner.Scanner, expectedIndent int) ([]ast.MapEntry, error) {
	return p.parseNestedMapWithListDepth(s, expectedIndent, 1)
}

// parseNestedMapWithListDepth parses a nested map at a specific indentation level,
// using listDepth for any lists encountered within the map.
func (p *Parser) parseNestedMapWithListDepth(s *scanner.Scanner, expectedIndent int, listDepth int) ([]ast.MapEntry, error) {
	entries := make([]ast.MapEntry, 0)

	for !s.IsEOF() {
		// Check if we're still at the correct indentation level
		if !s.IsIndented() {
			break
		}

		s.SkipWhitespace()
		currentIndent := s.Column() - 1
		if currentIndent < expectedIndent {
			// Less indented - end of nested map
			break
		}

		if currentIndent > expectedIndent {
			// More indented than expected - stop and let the caller handle deeper nesting
			break
		}

		if s.IsEOF() || s.PeekChar() == '\n' {
			s.SkipToNextLine()
			continue
		}

		// Skip comment lines inside nested map
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}

		if p.isListItemMarker(s) {
			listLine, listCol := s.Line(), s.Column()
			if _, err := p.parseListExpr(s, currentIndent, listDepth, listLine, listCol); err != nil {
				return nil, err
			}
			parseErr := NewParseError(SyntaxError, s.Filename(), listLine, listCol,
				"invalid syntax: list items must be nested under a key")
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, listLine, listCol))
			return nil, parseErr
		}

		if s.PeekChar() == '@' {
			refStartLine, refStartCol := s.Line(), s.Column()
			refExpr, err := p.parseValueExpr(s, refStartLine, refStartCol)
			if err != nil {
				return nil, err
			}
			ref, ok := refExpr.(*ast.ReferenceExpr)
			if !ok {
				parseErr := NewParseError(SyntaxError, s.Filename(), refStartLine, refStartCol,
					"invalid syntax: standalone references must use @alias:path")
				parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, refStartLine, refStartCol))
				return nil, parseErr
			}
			entries = append(entries, ast.MapEntry{
				Value:  ref,
				Spread: true,
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: ref.SourceSpan.StartLine,
					StartCol:  ref.SourceSpan.StartCol,
					EndLine:   ref.SourceSpan.EndLine,
					EndCol:    ref.SourceSpan.EndCol,
				},
			})
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

		// Check for nested map or list
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()

			foundContent := false
			for !s.IsEOF() && s.IsIndented() {
				s.SkipWhitespace()
				if s.IsCommentStart() {
					s.SkipComment()
					s.SkipToNextLine()
					continue
				}
				if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
					s.SkipToNextLine()
					continue
				}
				foundContent = true
				break
			}

			if foundContent {
				nextIndent := s.Column() - 1
				if nextIndent > currentIndent {
					if p.isListItemMarker(s) {
						listExpr, err := p.parseListExpr(s, nextIndent, listDepth, keyStartLine, keyStart)
						if err != nil {
							return nil, err
						}
						entries = append(entries, ast.MapEntry{
							Key:   key,
							Value: listExpr,
							SourceSpan: ast.SourceSpan{
								Filename:  s.Filename(),
								StartLine: keyStartLine,
								StartCol:  keyStart,
								EndLine:   listExpr.Span().EndLine,
								EndCol:    listExpr.Span().EndCol,
							},
						})
						continue
					}
					nestedEntries, err := p.parseNestedMapWithListDepth(s, nextIndent, listDepth)
					if err != nil {
						return nil, err
					}

					endLine := s.Line()
					endCol := p.mapEndColumn(s)
					entries = append(entries, ast.MapEntry{
						Key: key,
						Value: &ast.MapExpr{
							Entries: nestedEntries,
							SourceSpan: ast.SourceSpan{
								Filename:  s.Filename(),
								StartLine: keyStartLine,
								StartCol:  keyStart,
								EndLine:   endLine,
								EndCol:    endCol,
							},
						},
						SourceSpan: ast.SourceSpan{
							Filename:  s.Filename(),
							StartLine: keyStartLine,
							StartCol:  keyStart,
							EndLine:   endLine,
							EndCol:    endCol,
						},
					})
					continue
				}
			}

			// Empty value
			entries = append(entries, ast.MapEntry{
				Key: key,
				Value: &ast.StringLiteral{
					Value: "",
					SourceSpan: ast.SourceSpan{
						Filename:  s.Filename(),
						StartLine: keyStartLine,
						StartCol:  keyStart,
						EndLine:   keyStartLine,
						EndCol:    keyStart + len(key) + 1,
					},
				},
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: keyStartLine,
					StartCol:  keyStart,
					EndLine:   keyStartLine,
					EndCol:    keyStart + len(key) + 1,
				},
			})
			continue
		}

		// Parse scalar value
		valueStartLine, valueStartCol := s.Line(), s.Column()
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		entries = append(entries, ast.MapEntry{
			Key:   key,
			Value: valueExpr,
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: keyStartLine,
				StartCol:  keyStart,
				EndLine:   valueExpr.Span().EndLine,
				EndCol:    valueExpr.Span().EndCol,
			},
		})
		s.SkipToNextLine()
	}

	return entries, nil
}

// parseValueExpr parses a value expression, which can be either a string literal,
// an inline reference expression of the form @alias:dotted.path, or a list.
func (p *Parser) parseValueExpr(s *scanner.Scanner, startLine, startCol int) (ast.Expr, error) {
	// startLine and startCol point to where the value starts (after skipping whitespace)

	// Check for empty list syntax []
	if s.PeekChar() == '[' {
		s.Advance() // consume '['

		s.SkipWhitespace()
		if s.PeekChar() == ']' {
			// Empty list
			s.Advance() // consume ']'
			return &ast.ListExpr{
				Elements: []ast.Expr{},
				SourceSpan: ast.SourceSpan{
					Filename:  s.Filename(),
					StartLine: startLine,
					StartCol:  startCol,
					EndLine:   startLine,
					EndCol:    startCol + 1, // Points to ']'
				},
			}, nil
		}

		// Not an empty list - the '[' will be treated as part of the value
		// ReadValue() will handle it (it may be an error like unterminated string)
	}

	// Check for list item marker (dash + space)
	if p.isListItemMarker(s) {
		// This is a list - need to determine base indentation
		baseIndent := s.GetIndentLevel()
		return p.parseListExpr(s, baseIndent, 1, startLine, startCol)
	}

	// ReadValue() reads the value and trims quotes/whitespace
	valueText := s.ReadValue()

	// Check for backslash error marker from scanner (FR-014)
	if strings.HasPrefix(valueText, "\x00BACKSLASH_ERROR\x00") {
		// Remove marker to get actual value
		actualValue := strings.TrimPrefix(valueText, "\x00BACKSLASH_ERROR\x00")
		// Find the position of the backslash for accurate error reporting
		backslashPos := strings.IndexRune(actualValue, '\\')
		errorCol := startCol + backslashPos
		err := NewParseError(SyntaxError, s.Filename(), startLine, errorCol,
			"invalid syntax: unexpected character '\\'")
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, errorCol))
		return nil, err
	}

	// Validate that strings are properly terminated (ReadValue only strips matching quotes)
	if len(valueText) > 0 && (valueText[0] == '\'' || valueText[0] == '"') {
		quote := valueText[0]
		return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			fmt.Sprintf("invalid syntax: unterminated string (missing closing %c)", quote))
	}

	// Check if this is an inline reference expression
	if strings.HasPrefix(valueText, "@") {
		// Validate no whitespace in reference
		if strings.ContainsAny(valueText, " \t\n\r") {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: whitespace not allowed in @ reference")
		}

		// Validate not just "@" alone
		if len(valueText) == 1 {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: incomplete @ reference expression")
		}

		// Check for double @@
		if strings.HasPrefix(valueText, "@@") {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: double @ in reference expression")
		}

		// Parse reference expression: @alias:path
		// T033: Parse inline reference syntax @alias:path
		refText := valueText[1:] // Remove @

		// Split by ":" to get alias and path
		parts := strings.SplitN(refText, ":", 2)
		if len(parts) != 2 {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: @ reference must use format @alias:path")
		}

		aliasName := parts[0]
		pathStr := parts[1]

		// T035: Validate alias identifier
		if aliasName == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: alias cannot be empty (@alias:path)")
		}

		// Validate alias name pattern: ^[a-zA-Z_][a-zA-Z0-9_-]*$
		if !isValidAliasName(aliasName) {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: alias name must start with letter or underscore and contain only letters, numbers, underscores, or hyphens")
		}

		// Validate path exists
		if pathStr == "" {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: path cannot be empty; use '*' for root (@alias:*)")
		}

		if strings.Contains(pathStr, ":") {
			return nil, NewParseError(SyntaxError, s.Filename(), startLine, startCol,
				"invalid syntax: @ reference path must use '.' only (no additional ':')")
		}

		// Parse path segments (dot-separated segments with optional bracket notation)
		pathParts, err := p.parseInlineReferencePath(pathStr, s.Filename(), startLine, startCol)
		if err != nil {
			return nil, err
		}

		// Calculate the end column (1-indexed, inclusive)
		// startCol is the 1-indexed column where the value starts
		// EndCol should point to the last character of the value (inclusive)
		refEndCol := startCol + len(valueText) - 1

		return &ast.ReferenceExpr{
			Alias: aliasName,
			Path:  pathParts,
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

// parseInlineReferencePath splits an inline reference path into components.
// It supports dot-separated components with optional list index notation, e.g. "matrix[0][1]".
// Indexes are appended to the current component (e.g., "matrix[0][1]").
// Errors are returned for malformed bracket usage (missing closing bracket, empty or non-numeric index,
// stray ']' or '[') and include actionable messages.
func (p *Parser) parseInlineReferencePath(pathStr, filename string, line, col int) ([]string, error) {
	if pathStr == "." {
		return nil, NewParseError(SyntaxError, filename, line, col,
			"invalid syntax: root references must use '*' (e.g., @alias:*)")
	}
	if pathStr == "*" {
		return []string{"*"}, nil
	}
	if !strings.ContainsAny(pathStr, "[]") {
		parts := strings.Split(pathStr, ".")
		for i, part := range parts {
			if part == "" {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path has empty segment")
			}
			if part == "*" {
				if i != len(parts)-1 {
					return nil, NewParseError(SyntaxError, filename, line, col,
						"invalid syntax: wildcard '*' must be the final path segment")
				}
				continue
			}
			if strings.Contains(part, "*") {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: wildcard '*' must be a full path segment")
			}
		}
		return parts, nil
	}

	var components []string
	var current strings.Builder

	for i := 0; i < len(pathStr); i++ {
		switch pathStr[i] {
		case '.':
			if current.Len() == 0 {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path has empty segment")
			}
			components = append(components, current.String())
			current.Reset()
		case '[':
			if current.Len() == 0 {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path has stray '[' without a preceding path segment")
			}

			indexStart := i + 1
			if indexStart >= len(pathStr) {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path index is missing closing ']' (e.g., [0])")
			}
			if pathStr[indexStart] == ']' {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path index cannot be empty (e.g., [0])")
			}

			j := indexStart
			for ; j < len(pathStr) && pathStr[j] != ']'; j++ {
				if pathStr[j] < '0' || pathStr[j] > '9' {
					return nil, NewParseError(SyntaxError, filename, line, col,
						"invalid syntax: inline reference path index must be numeric (e.g., [0])")
				}
			}
			if j >= len(pathStr) {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: inline reference path index is missing closing ']' (e.g., [0])")
			}

			current.WriteString(pathStr[i : j+1])
			i = j
		case ']':
			return nil, NewParseError(SyntaxError, filename, line, col,
				"invalid syntax: inline reference path has stray ']' without a matching '['")
		default:
			current.WriteByte(pathStr[i])
		}
	}
	if current.Len() == 0 {
		return nil, NewParseError(SyntaxError, filename, line, col,
			"invalid syntax: inline reference path has empty segment")
	}

	components = append(components, current.String())
	for i, component := range components {
		if component == "*" {
			if i != len(components)-1 {
				return nil, NewParseError(SyntaxError, filename, line, col,
					"invalid syntax: wildcard '*' must be the final path segment")
			}
			continue
		}
		if strings.Contains(component, "*") {
			return nil, NewParseError(SyntaxError, filename, line, col,
				"invalid syntax: wildcard '*' must be a full path segment")
		}
	}
	return components, nil
}

// parseInlineReferencePathSegments splits a path on ':' and '.' (with bracket support)
// to produce the full list of path segments. Providers interpret these segments.

// parseListExpr parses a list expression with YAML-style block notation (dash markers).
// It enforces 2-space indentation, validates against empty items, supports nested lists,
// and enforces the maximum nesting depth.
//
// Parameters:
//   - s: Scanner positioned at the first list item marker or after the key that introduced the list
//   - baseIndent: Expected indentation level for list items (must be consistent)
//   - depth: Current nesting depth (1 for top-level list)
//   - startLine: Source line where the list started (for error reporting)
//   - startCol: Source column where the list started (for error reporting)
//
// Returns:
//   - *ast.ListExpr with parsed elements
//   - error if validation fails (empty items, inconsistent indentation, tabs, whitespace-only)
func (p *Parser) parseListExpr(s *scanner.Scanner, baseIndent int, depth int, startLine, startCol int) (*ast.ListExpr, error) {
	var elements []ast.Expr
	hasAnyNonCommentContent := false
	inlineFirstItemUsed := false

	if depth > scanner.MaxListNestingDepth {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			listDepthExceededErrorMessage(depth, scanner.MaxListNestingDepth))
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	for !s.IsEOF() {
		// Check for end of input or blank lines
		if s.IsEOF() || s.PeekChar() == '\n' {
			s.SkipToNextLine()
			if s.IsEOF() || !s.IsIndented() {
				break
			}
			continue
		}

		// Check if we're still at the expected indentation level
		if !s.IsIndented() {
			break
		}

		currentIndent, hasTab := p.peekIndentLevel(s)
		if hasTab {
			err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(), listTabIndentationErrorMessage())
			err.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
			return nil, err
		}
		if currentIndent < baseIndent {
			if depth == 1 {
				snapshot := s.Snapshot()
				s.SkipWhitespace()
				isListItem := p.isListItemMarker(s)
				s.Restore(snapshot)
				if isListItem {
					parseErr := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
						listInconsistentIndentErrorMessage(baseIndent, currentIndent))
					parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
					return nil, parseErr
				}
			}
			break
		}

		// Skip whitespace after indentation
		s.SkipWhitespace()

		// Skip empty lines after indentation
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()
			continue
		}

		// Skip comment lines
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}

		inlineFirstItemAllowed := !inlineFirstItemUsed && s.Line() == startLine && currentIndent == baseIndent-2

		// Enforce consistent indentation for list items
		if p.isListItemMarker(s) && currentIndent != baseIndent && !inlineFirstItemAllowed {
			parseErr := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				listInconsistentIndentErrorMessage(baseIndent, currentIndent))
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
			return nil, parseErr
		}

		// Check for list item marker
		if !p.isListItemMarker(s) {
			// Not a list item at this indentation level - end of list
			break
		}

		// Validate indentation consistency
		if currentIndent != baseIndent && !inlineFirstItemAllowed {
			parseErr := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
				listInconsistentIndentErrorMessage(baseIndent, currentIndent))
			parseErr.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
			return nil, parseErr
		}

		if inlineFirstItemAllowed {
			inlineFirstItemUsed = true
		}

		// Consume the dash and optional space
		itemLine, itemCol := s.Line(), s.Column()
		s.Advance() // consume '-'
		if s.PeekChar() == ' ' {
			s.Advance() // consume optional space
		}

		// Skip any additional whitespace after dash
		s.SkipWhitespace()

		// Check for comment after dash (also counts as empty item)
		if s.IsCommentStart() {
			err := NewParseError(SyntaxError, s.Filename(), itemLine, itemCol, listEmptyItemErrorMessage())
			err.SetSnippet(generateSnippetFromSource(p.sourceText, itemLine, itemCol))
			return nil, err
		}

		// Parse the value after the dash
		valueStartLine, valueStartCol := s.Line(), s.Column()
		ch := s.PeekChar()

		// Nested list on next line (dash with no inline value)
		if ch == '\n' || ch == '\r' || s.IsEOF() {
			s.SkipToNextLine()
			if s.IsEOF() || !s.IsIndented() {
				err := NewParseError(SyntaxError, s.Filename(), itemLine, itemCol, listEmptyItemErrorMessage())
				err.SetSnippet(generateSnippetFromSource(p.sourceText, itemLine, itemCol))
				return nil, err
			}

			nextIndent, hasTab := p.peekIndentLevel(s)
			if hasTab {
				err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(), listTabIndentationErrorMessage())
				err.SetSnippet(generateSnippetFromSource(p.sourceText, s.Line(), s.Column()))
				return nil, err
			}
			if nextIndent <= baseIndent {
				err := NewParseError(SyntaxError, s.Filename(), itemLine, itemCol, listEmptyItemErrorMessage())
				err.SetSnippet(generateSnippetFromSource(p.sourceText, itemLine, itemCol))
				return nil, err
			}

			s.SkipWhitespace()
			if !p.isListItemMarker(s) {
				err := NewParseError(SyntaxError, s.Filename(), itemLine, itemCol, listEmptyItemErrorMessage())
				err.SetSnippet(generateSnippetFromSource(p.sourceText, itemLine, itemCol))
				return nil, err
			}

			nestedStartLine, nestedStartCol := s.Line(), s.Column()
			nestedList, err := p.parseListExpr(s, nextIndent, depth+1, nestedStartLine, nestedStartCol)
			if err != nil {
				return nil, err
			}
			elements = append(elements, nestedList)
			hasAnyNonCommentContent = true
			continue
		}

		hasAnyNonCommentContent = true

		// Inline object list item (e.g., "- name: alice")
		mapSnapshot := s.Snapshot()
		mapKeyStartLine, mapKeyStartCol := s.Line(), s.Column()
		mapKey := s.ReadIdentifier()
		if mapKey != "" {
			s.SkipWhitespace()
			if s.PeekChar() == ':' {
				_ = s.Expect(':')
				if p.isInlineMapDelimiter(s.PeekChar()) {
					s.SkipWhitespace()
					keyIndent := mapKeyStartCol - 1

					valueExpr, valueConsumedLine, err := p.parseInlineMapValue(s, mapKey, mapKeyStartLine, mapKeyStartCol, keyIndent, depth+1)
					if err != nil {
						return nil, err
					}

					if !valueConsumedLine {
						s.SkipToNextLine()
					}

					entries := []ast.MapEntry{
						{
							Key:   mapKey,
							Value: valueExpr,
							SourceSpan: ast.SourceSpan{
								Filename:  s.Filename(),
								StartLine: mapKeyStartLine,
								StartCol:  mapKeyStartCol,
								EndLine:   valueExpr.Span().EndLine,
								EndCol:    valueExpr.Span().EndCol,
							},
						},
					}
					entries, err = p.parseInlineMapAdditionalEntries(s, keyIndent, entries, depth+1)
					if err != nil {
						return nil, err
					}

					endLine := s.Line()
					endCol := p.mapEndColumn(s)
					elements = append(elements, &ast.MapExpr{
						Entries: entries,
						SourceSpan: ast.SourceSpan{
							Filename:  s.Filename(),
							StartLine: mapKeyStartLine,
							StartCol:  mapKeyStartCol,
							EndLine:   endLine,
							EndCol:    endCol,
						},
					})
					hasAnyNonCommentContent = true
					continue
				}
			}
		}
		s.Restore(mapSnapshot)

		// Inline nested list (e.g., "- - 1")
		if p.isListItemMarker(s) {
			nestedList, err := p.parseListExpr(s, baseIndent+2, depth+1, valueStartLine, valueStartCol)
			if err != nil {
				return nil, err
			}
			elements = append(elements, nestedList)
			s.SkipToNextLine()
			continue
		}

		// Check if the value is a nested list (dash on next line with more indentation)
		if ch == '\n' || ch == '\r' {
			s.SkipToNextLine()
			if !s.IsEOF() && s.IsIndented() {
				nextIndent := s.GetIndentLevel()
				if nextIndent > baseIndent {
					s.SkipWhitespace()
					if p.isListItemMarker(s) {
						// Nested list
						nestedList, err := p.parseListExpr(s, nextIndent, depth+1, valueStartLine, valueStartCol)
						if err != nil {
							return nil, err
						}
						elements = append(elements, nestedList)
						continue
					}
				}
			}
			// Empty value after dash - this is an error
			err := NewParseError(SyntaxError, s.Filename(), itemLine, itemCol,
				listEmptyItemErrorMessage())
			err.SetSnippet(generateSnippetFromSource(p.sourceText, itemLine, itemCol))
			return nil, err
		}

		// Parse value expression (can be scalar, reference, or nested structure)
		valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
		if err != nil {
			return nil, err
		}

		elements = append(elements, valueExpr)

		// Move to next line
		s.SkipToNextLine()
	}

	// Check for whitespace-only list (no actual items found)
	if !hasAnyNonCommentContent && len(elements) == 0 {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, listWhitespaceOnlyErrorMessage())
		err.SetSnippet(generateSnippetFromSource(p.sourceText, startLine, startCol))
		return nil, err
	}

	endLine, endCol := s.Line(), s.Column()

	return &ast.ListExpr{
		Elements: elements,
		SourceSpan: ast.SourceSpan{
			Filename:  s.Filename(),
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
		},
	}, nil
}

// parseInlineMapValue parses the value for an inline map entry within a list item.
// It returns the parsed expression and whether the parser already consumed the next line.
func (p *Parser) parseInlineMapValue(
	s *scanner.Scanner,
	key string,
	keyStartLine int,
	keyStartCol int,
	keyIndent int,
	listDepth int,
) (ast.Expr, bool, error) {
	if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
		s.SkipToNextLine()

		foundContent := false
		for !s.IsEOF() && s.IsIndented() {
			s.SkipWhitespace()
			if s.IsCommentStart() {
				s.SkipComment()
				s.SkipToNextLine()
				continue
			}
			if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
				s.SkipToNextLine()
				continue
			}
			foundContent = true
			break
		}

		if foundContent {
			nextIndent := s.Column() - 1
			if nextIndent > keyIndent {
				if p.isListItemMarker(s) {
					listStartLine, listStartCol := s.Line(), s.Column()
					listExpr, err := p.parseListExpr(s, nextIndent, listDepth, listStartLine, listStartCol)
					return listExpr, true, err
				}

				nestedEntries, err := p.parseNestedMapWithListDepth(s, nextIndent, listDepth)
				if err != nil {
					return nil, true, err
				}

				endLine := s.Line()
				endCol := p.mapEndColumn(s)
				return &ast.MapExpr{
					Entries: nestedEntries,
					SourceSpan: ast.SourceSpan{
						Filename:  s.Filename(),
						StartLine: keyStartLine,
						StartCol:  keyStartCol,
						EndLine:   endLine,
						EndCol:    endCol,
					},
				}, true, nil
			}
		}

		return &ast.StringLiteral{
			Value: "",
			SourceSpan: ast.SourceSpan{
				Filename:  s.Filename(),
				StartLine: keyStartLine,
				StartCol:  keyStartCol,
				EndLine:   keyStartLine,
				EndCol:    keyStartCol + len(key) + 1,
			},
		}, true, nil
	}

	valueStartLine, valueStartCol := s.Line(), s.Column()
	valueExpr, err := p.parseValueExpr(s, valueStartLine, valueStartCol)
	if err != nil {
		return nil, false, err
	}

	return valueExpr, false, nil
}

// parseInlineMapAdditionalEntries parses additional entries for a list item map at the same indentation level.
func (p *Parser) parseInlineMapAdditionalEntries(s *scanner.Scanner, expectedIndent int, entries []ast.MapEntry, listDepth int) ([]ast.MapEntry, error) {
	snapshot := s.Snapshot()
	if p.seekToIndentedContent(s, expectedIndent) {
		currentIndent := s.Column() - 1
		if currentIndent == expectedIndent {
			s.Restore(snapshot)
			additionalEntries, err := p.parseNestedMapWithListDepth(s, expectedIndent, listDepth)
			if err != nil {
				return nil, err
			}
			entries = append(entries, additionalEntries...)
			return entries, nil
		}
	}

	s.Restore(snapshot)
	return entries, nil
}

// findBaseIndent scans forward to find the base indentation for a block.
// It skips empty lines and comments without consuming scanner state.
func (p *Parser) findBaseIndent(s *scanner.Scanner) (int, bool) {
	snapshot := s.Snapshot()
	defer s.Restore(snapshot)

	for !s.IsEOF() {
		if !s.IsIndented() {
			if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
				s.SkipToNextLine()
				continue
			}
			return 0, false
		}

		s.SkipWhitespace()
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()
			continue
		}

		return s.Column() - 1, true
	}

	return 0, false
}

// isWhitespaceOnlyIndentedBlock returns true if the upcoming indented block
// contains only blank lines or comments before dedenting or EOF.
func (p *Parser) isWhitespaceOnlyIndentedBlock(s *scanner.Scanner) bool {
	snapshot := s.Snapshot()
	defer s.Restore(snapshot)

	seenIndented := false
	for !s.IsEOF() {
		if !s.IsIndented() {
			if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
				s.SkipToNextLine()
				continue
			}
			return seenIndented
		}

		seenIndented = true
		s.SkipWhitespace()
		if s.IsEOF() {
			return seenIndented
		}
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()
			continue
		}
		return false
	}

	return seenIndented
}

// mapEndColumn normalizes map end columns to the start of the current line.
func (p *Parser) mapEndColumn(s *scanner.Scanner) int {
	if s.Column() > 1 {
		return 1
	}
	return s.Column()
}

// isListOnlyBlock determines whether a block contains only list items at baseIndent.
func (p *Parser) isListOnlyBlock(s *scanner.Scanner, baseIndent int) bool {
	snapshot := s.Snapshot()
	defer s.Restore(snapshot)

	seenList := false
	for !s.IsEOF() {
		if !s.IsIndented() {
			break
		}
		s.SkipWhitespace()
		currentIndent := s.Column() - 1
		if currentIndent < baseIndent {
			break
		}

		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()
			continue
		}

		if currentIndent == baseIndent {
			if p.isListItemMarker(s) {
				seenList = true
				s.SkipToNextLine()
				continue
			}
			return false
		}
		if p.isListItemMarker(s) {
			seenList = true
		}
		s.SkipToNextLine()
	}

	return seenList
}

// peekIndentLevel calculates indentation from the current line start without moving the scanner.
// It returns the indentation level (spaces) and whether a tab was found in the indentation.
func (p *Parser) peekIndentLevel(s *scanner.Scanner) (int, bool) {
	lineStart := s.Pos() - (s.Column() - 1)
	if lineStart < 0 {
		lineStart = 0
	}

	indent := 0
	for i := lineStart; i < len(p.sourceText); i++ {
		switch p.sourceText[i] {
		case ' ':
			indent++
		case '\t':
			return indent, true
		default:
			return indent, false
		}
	}

	return indent, false
}

// seekToIndentedContent advances the scanner to the next non-empty, non-comment line.
func (p *Parser) seekToIndentedContent(s *scanner.Scanner, baseIndent int) bool {
	for !s.IsEOF() && s.IsIndented() {
		s.SkipWhitespace()
		currentIndent := s.Column() - 1
		if currentIndent < baseIndent {
			return false
		}
		if s.IsCommentStart() {
			s.SkipComment()
			s.SkipToNextLine()
			continue
		}
		if s.PeekChar() == '\n' || s.PeekChar() == '\r' {
			s.SkipToNextLine()
			continue
		}
		return true
	}
	return false
}

// isListItemMarker returns true if the scanner is positioned at a list item marker (dash + space).
// This delegates to the scanner's implementation to keep list marker detection consistent.
func (p *Parser) isListItemMarker(s *scanner.Scanner) bool {
	return s.IsListItemMarker()
}

// isInlineMapDelimiter reports whether the rune following a colon indicates a map value boundary.
// This avoids treating scalar values containing colons (e.g., inline references or URLs) as map items.
func (p *Parser) isInlineMapDelimiter(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '#' || ch == 0
}

// generateSnippetFromSource is a convenience wrapper around generateSnippet from errors.go.
func generateSnippetFromSource(sourceText string, line, col int) string {
	return generateSnippet(sourceText, line, col)
}

// isValidAliasName checks if an alias name matches pattern [a-zA-Z_][a-zA-Z0-9_-]*
func isValidAliasName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// First character: letter or underscore
	first := name[0]
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}

	// Remaining characters: letter, digit, underscore, or hyphen
	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}

	return true
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
