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
		return nil, fmt.Errorf("failed to open file: %w", err)
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
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Create scanner
	s := scanner.New(string(content), filename)

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

// parseStatement parses a single statement.
func (p *Parser) parseStatement(s *scanner.Scanner) (ast.Stmt, error) {
	startLine, startCol := s.Line(), s.Column()

	// Check for invalid characters first
	ch := s.PeekChar()
	if ch == '!' || ch == '@' || ch == '#' || ch == '$' || ch == '%' || ch == '^' || ch == '&' || ch == '*' || ch == '(' || ch == ')' {
		return nil, fmt.Errorf("%s:%d:%d: invalid syntax: unexpected character '%c'", s.Filename(), startLine, startCol, ch)
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
		return p.parseReferenceStmt(s, startLine, startCol)
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
	s.Expect(':')

	// Parse configuration block (indented key-value pairs)
	config, err := p.parseConfigBlock(s)
	if err != nil {
		return nil, err
	}

	// Extract alias and type from config
	alias := config["alias"]
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
func (p *Parser) parseImportStmt(s *scanner.Scanner, startLine, startCol int) (*ast.ImportStmt, error) {
	s.ConsumeToken() // consume "import"
	s.Expect(':')

	alias := s.ReadIdentifier()
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
func (p *Parser) parseReferenceStmt(s *scanner.Scanner, startLine, startCol int) (*ast.ReferenceStmt, error) {
	s.ConsumeToken() // consume "reference"
	s.Expect(':')

	alias := s.ReadIdentifier()
	s.Expect(':')
	path := s.ReadPath()

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
func (p *Parser) parseSectionDecl(s *scanner.Scanner, startLine, startCol int) (*ast.SectionDecl, error) {
	name := s.ReadIdentifier()
	if s.PeekChar() != ':' {
		// Not a valid section declaration - this is invalid syntax
		return nil, fmt.Errorf("%s:%d:%d: invalid syntax: expected ':' after identifier '%s'", s.Filename(), startLine, startCol, name)
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

		key := s.ReadIdentifier()
		s.Expect(':')
		s.SkipWhitespace()
		value := s.ReadValue()

		config[key] = value
		s.SkipToNextLine()
	}

	return config, nil
}
