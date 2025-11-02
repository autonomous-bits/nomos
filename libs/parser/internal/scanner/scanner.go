// Package scanner provides a lexical scanner for Nomos configuration files.
package scanner

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner represents a lexical scanner for Nomos source text.
type Scanner struct {
	input     string // The input string
	filename  string // Source filename for error messages
	pos       int    // Current position in input
	line      int    // Current line number (1-indexed)
	col       int    // Current column number (1-indexed)
	lineStart int    // Position of start of current line
}

// New creates a new Scanner for the given input.
func New(input, filename string) *Scanner {
	return &Scanner{
		input:     input,
		filename:  filename,
		pos:       0,
		line:      1,
		col:       1,
		lineStart: 0,
	}
}

// Filename returns the source filename.
func (s *Scanner) Filename() string {
	return s.filename
}

// Line returns the current line number.
func (s *Scanner) Line() int {
	return s.line
}

// Column returns the current column number.
func (s *Scanner) Column() int {
	return s.col
}

// Pos returns the current byte position in the input.
func (s *Scanner) Pos() int {
	return s.pos
}

// IsEOF returns true if the scanner has reached the end of input.
func (s *Scanner) IsEOF() bool {
	return s.pos >= len(s.input)
}

// PeekChar returns the current character without consuming it.
func (s *Scanner) PeekChar() rune {
	if s.IsEOF() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[s.pos:])
	return r
}

// Advance moves to the next character.
func (s *Scanner) Advance() {
	if s.IsEOF() {
		return
	}

	if s.input[s.pos] == '\n' {
		s.line++
		s.col = 1
		s.lineStart = s.pos + 1
		s.pos++
	} else {
		_, size := utf8.DecodeRuneInString(s.input[s.pos:])
		s.col++
		s.pos += size
	}
}

// SkipWhitespace skips whitespace characters except newlines.
func (s *Scanner) SkipWhitespace() {
	for !s.IsEOF() {
		ch := s.PeekChar()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			s.Advance()
		} else {
			break
		}
	}
}

// SkipToNextLine skips to the start of the next line.
func (s *Scanner) SkipToNextLine() {
	for !s.IsEOF() {
		if s.PeekChar() == '\n' {
			s.Advance()
			break
		}
		s.Advance()
	}
}

// IsIndented returns true if the current line starts with whitespace.
func (s *Scanner) IsIndented() bool {
	// Check if we're at the start of a line
	if s.pos == 0 || (s.pos > 0 && s.input[s.pos-1] == '\n') {
		return !s.IsEOF() && (s.PeekChar() == ' ' || s.PeekChar() == '\t')
	}
	// Already past indentation
	return s.col > 1 && s.pos > s.lineStart
}

// GetIndentLevel returns the indentation level (number of leading spaces/tabs) of the current line.
// Must be called at the start of a line. Tabs count as 1 indent level each.
func (s *Scanner) GetIndentLevel() int {
	savedPos := s.pos
	savedLine := s.line
	savedCol := s.col

	level := 0
	for !s.IsEOF() {
		ch := s.PeekChar()
		if ch == ' ' || ch == '\t' {
			level++
			s.Advance()
		} else {
			break
		}
	}

	// Restore position
	s.pos = savedPos
	s.line = savedLine
	s.col = savedCol

	return level
}

// PeekToken peeks at the next identifier token without consuming it.
func (s *Scanner) PeekToken() string {
	savedPos := s.pos
	savedLine := s.line
	savedCol := s.col

	token := s.ReadIdentifier()

	// Restore position
	s.pos = savedPos
	s.line = savedLine
	s.col = savedCol

	return token
}

// ConsumeToken consumes an identifier token.
func (s *Scanner) ConsumeToken() string {
	return s.ReadIdentifier()
}

// ReadIdentifier reads an identifier (alphanumeric + dash).
func (s *Scanner) ReadIdentifier() string {
	start := s.pos
	for !s.IsEOF() {
		ch := s.PeekChar()
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_' {
			s.Advance()
		} else {
			break
		}
	}
	if start == s.pos {
		// No valid identifier characters found
		return ""
	}
	return s.input[start:s.pos]
}

// ReadPath reads a dotted path (e.g., config.key.value).
func (s *Scanner) ReadPath() string {
	start := s.pos
	for !s.IsEOF() {
		ch := s.PeekChar()
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '.' || ch == '-' || ch == '_' {
			s.Advance()
		} else {
			break
		}
	}
	return s.input[start:s.pos]
}

// ReadValue reads a value (everything until end of line, trimming quotes and whitespace).
func (s *Scanner) ReadValue() string {
	start := s.pos
	end := s.pos

	// Read until newline
	for !s.IsEOF() && s.PeekChar() != '\n' && s.PeekChar() != '\r' {
		s.Advance()
		end = s.pos
	}

	value := strings.TrimSpace(s.input[start:end])

	// Remove surrounding quotes if present
	if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
		if value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}
	}

	return value
}

// Expect consumes the expected character or returns an error.
func (s *Scanner) Expect(expected rune) error {
	actual := s.PeekChar()
	if actual != expected {
		return fmt.Errorf("%s:%d:%d: expected '%c', got '%c'", s.filename, s.line, s.col, expected, actual)
	}
	s.Advance()
	return nil
}
