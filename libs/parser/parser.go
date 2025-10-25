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

	"github.com/autonomous-bits/nomos/libs/parser/internal/scanner"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Parser represents a parser instance. It can be reused for multiple parse operations.
type Parser struct {
	// Future: add configuration options here
}

// ParserOption is a function that configures a Parser.
type ParserOption func(*Parser)

// NewParser creates a new Parser with the given options.
func NewParser(opts ...ParserOption) *Parser {
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
	file, err := os.Open(path)
	if err != nil {
		return nil, NewParseError(IOError, path, 0, 0, fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

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
	sourceText := string(content)

	// Create scanner
	s := scanner.New(sourceText, filename)

	// Parse statements
	statements, err := p.parseStatements(s, sourceText)
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
func (p *Parser) parseStatements(s *scanner.Scanner, sourceText string) ([]ast.Stmt, error) {
	var statements []ast.Stmt

	for !s.IsEOF() {
		// Skip whitespace and empty lines
		s.SkipWhitespace()
		if s.IsEOF() {
			break
		}

		stmt, err := p.parseStatement(s, sourceText)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	return statements, nil
}

// parseStatement parses a single statement.
func (p *Parser) parseStatement(s *scanner.Scanner, sourceText string) (ast.Stmt, error) {
	startLine, startCol := s.Line(), s.Column()

	// Check for invalid characters first
	ch := s.PeekChar()
	if ch == '!' || ch == '@' || ch == '#' || ch == '$' || ch == '%' || ch == '^' || ch == '&' || ch == '*' || ch == '(' || ch == ')' {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: unexpected character '%c'", ch))
		err.SetSnippet(generateSnippetFromSource(sourceText, startLine, startCol))
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
		return p.parseSourceDecl(s, startLine, startCol, sourceText)
	case "import":
		return p.parseImportStmt(s, startLine, startCol, sourceText)
	case "reference":
		return p.parseReferenceStmt(s, startLine, startCol, sourceText)
	default:
		// Try to parse as a section declaration
		if ch != '\n' && ch != '\r' && !s.IsEOF() {
			return p.parseSectionDecl(s, startLine, startCol, sourceText)
		}
		// Skip unknown/empty lines
		s.SkipToNextLine()
		return nil, nil
	}
}

// parseSourceDecl parses a source declaration.
func (p *Parser) parseSourceDecl(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.SourceDecl, error) {
	s.ConsumeToken() // consume "source"

	// Validate colon after keyword
	if s.PeekChar() != ':' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'source' keyword must be followed by ':'")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}
	s.Expect(':')

	// Parse configuration block (indented key-value pairs)
	config, err := p.parseConfigBlock(s)
	if err != nil {
		return nil, err
	}

	// Validate alias is not empty
	alias, ok := config["alias"]
	if !ok || alias == "" {
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol,
			"invalid syntax: 'source' declaration requires a non-empty 'alias' field")
		err.SetSnippet(generateSnippetFromSource(sourceText, startLine, startCol))
		return nil, err
	}

	// Extract alias and type from config
	typeName := config["type"]

	endLine, endCol := s.Line(), s.Column()

	return &ast.SourceDecl{
		Alias:  alias,
		Type:   typeName,
		Config: config,
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
func (p *Parser) parseImportStmt(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.ImportStmt, error) {
	s.ConsumeToken() // consume "import"

	// Validate colon after keyword
	if s.PeekChar() != ':' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'import' keyword must be followed by ':'")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}
	s.Expect(':')

	alias := s.ReadIdentifier()

	// Validate alias is not empty
	if alias == "" {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'import' statement requires an alias")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
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
func (p *Parser) parseReferenceStmt(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.ReferenceStmt, error) {
	s.ConsumeToken() // consume "reference"

	// Validate colon after keyword
	if s.PeekChar() != ':' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'reference' keyword must be followed by ':'")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}
	s.Expect(':')

	alias := s.ReadIdentifier()

	// Validate alias is not empty
	if alias == "" {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'reference' statement requires an alias")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}

	// Validate second colon exists
	if s.PeekChar() != ':' {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: expected ':' after alias in 'reference' statement")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}
	s.Expect(':')

	path := s.ReadPath()

	// Validate path is not empty
	if path == "" {
		err := NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
			"invalid syntax: 'reference' statement requires a path")
		err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
		return nil, err
	}

	s.SkipToNextLine()
	endLine, endCol := s.Line(), s.Column()

	return &ast.ReferenceStmt{
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

// parseSectionDecl parses a configuration section.
func (p *Parser) parseSectionDecl(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.SectionDecl, error) {
	name := s.ReadIdentifier()
	if s.PeekChar() != ':' {
		// Not a valid section declaration - this is invalid syntax
		err := NewParseError(SyntaxError, s.Filename(), startLine, startCol, fmt.Sprintf("invalid syntax: expected ':' after identifier '%s'", name))
		err.SetSnippet(generateSnippetFromSource(sourceText, startLine, startCol))
		return nil, err
	}
	s.Expect(':')
	s.SkipToNextLine()

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
func (p *Parser) parseConfigBlock(s *scanner.Scanner) (map[string]string, error) {
	config := make(map[string]string)

	s.SkipToNextLine()

	// Parse indented key-value pairs
	for !s.IsEOF() {
		// Check if line is indented
		if !s.IsIndented() {
			break
		}

		s.SkipWhitespace()

		if s.IsEOF() || s.PeekChar() == '\n' {
			break
		}

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

		// TODO: Duplicate key detection needs to be scope-aware for nested structures
		// Currently disabled to allow nested sections with same key names
		// Check for duplicate keys
		//if _, exists := config[key]; exists {
		//	return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), keyStart,
		//		fmt.Sprintf("invalid syntax: duplicate key '%s'", key))
		//}

		s.Expect(':')
		s.SkipWhitespace()
		value := s.ReadValue()

		// Validate value is properly terminated (no unterminated strings)
		if strings.HasPrefix(value, "'") || strings.HasPrefix(value, "\"") {
			quote := value[0]
			if len(value) < 2 || value[len(value)-1] != quote {
				return nil, NewParseError(SyntaxError, s.Filename(), s.Line(), s.Column(),
					fmt.Sprintf("invalid syntax: unterminated string (missing closing %c)", quote))
			}
		}

		config[key] = value
		s.SkipToNextLine()
	}

	return config, nil
}

// generateSnippetFromSource is a convenience wrapper around generateSnippet from errors.go.
func generateSnippetFromSource(sourceText string, line, col int) string {
	return generateSnippet(sourceText, line, col)
}
