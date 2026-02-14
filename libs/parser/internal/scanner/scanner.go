// Package scanner provides a lexical scanner for Nomos configuration files.
package scanner //nolint:revive // internal package, no actual conflict with stdlib

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxListNestingDepth defines the maximum allowed nesting depth for lists.
// This limit prevents excessive recursion and maintains reasonable parsing performance.
// A depth of 20 levels should accommodate complex configuration structures while
// preventing pathological cases that could lead to stack overflow or performance issues.
//
// Example: A list nested 20 levels deep would have indentation of 40 spaces (20 * 2).
const MaxListNestingDepth = 20

// Scanner represents a lexical scanner for Nomos source text.
type Scanner struct {
	input     string // The input string
	filename  string // Source filename for error messages
	pos       int    // Current position in input
	line      int    // Current line number (1-indexed)
	col       int    // Current column number (1-indexed)
	lineStart int    // Position of start of current line
}

// Snapshot captures the current scanner position for later restoration.
// It is intended for internal parser lookahead operations.
type Snapshot struct {
	pos       int
	line      int
	col       int
	lineStart int
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

// Snapshot returns a snapshot of the current scanner position.
func (s *Scanner) Snapshot() Snapshot {
	return Snapshot{
		pos:       s.pos,
		line:      s.line,
		col:       s.col,
		lineStart: s.lineStart,
	}
}

// Restore resets the scanner to a previously captured snapshot.
func (s *Scanner) Restore(snapshot Snapshot) {
	s.pos = snapshot.pos
	s.line = snapshot.line
	s.col = snapshot.col
	s.lineStart = snapshot.lineStart
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

// IsCommentStart reports whether the scanner is positioned at a YAML-style comment delimiter (#).
// It checks the current character without advancing, allowing comment lookahead
// without state modification.
//
// Returns true if the scanner is at a '#' character, false otherwise (including at EOF).
func (s *Scanner) IsCommentStart() bool {
	return !s.IsEOF() && s.PeekChar() == '#'
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

// SkipComment advances the scanner past a YAML-style comment that begins with '#'.
// The scanner should be positioned at a '#' character when this method is called.
// It advances past the '#' delimiter and all subsequent characters on the line,
// stopping at either a newline character or EOF. The newline itself is NOT consumed;
// the caller is responsible for advancing past the line terminator.
//
// SkipComment correctly handles UTF-8 encoded comment content, advancing by full
// Unicode code points rather than individual bytes. Comments may contain any valid
// UTF-8 text including emoji, non-ASCII characters, and multi-byte sequences.
//
// Example:
//
//	# This is a comment with emoji 🎉
//	key: value  # inline comment
//
// After SkipComment(), the scanner is positioned at the newline or EOF.
func (s *Scanner) SkipComment() {
	// Skip the '#' character if we're at it
	if !s.IsEOF() && s.PeekChar() == '#' {
		s.Advance()
	}

	// Skip until newline or EOF
	for !s.IsEOF() && s.PeekChar() != '\n' {
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

// IsListItemMarker reports whether the scanner is positioned at a list item marker (dash + space).
// It checks for the pattern "- " (hyphen followed by exactly one space), which indicates
// the start of a list item in YAML-style block notation. This method does not advance the scanner,
// allowing lookahead without state modification.
//
// Returns true if at "- " pattern, false otherwise (including at EOF).
func (s *Scanner) IsListItemMarker() bool {
	if s.IsEOF() {
		return false
	}

	// Check if current position has '-'
	if s.PeekChar() != '-' {
		return false
	}

	// Check if next position has ' ' (space)
	if s.pos+1 >= len(s.input) {
		return false
	}

	return s.input[s.pos+1] == ' '
}

// GetIndentLevel returns the number of space characters from the start of the current line
// to the current scanner position. This method is used to validate list item indentation
// consistency. Tab characters are not counted as valid indentation (lists require spaces only).
//
// Returns the indentation level as a count of space characters. Returns 0 if at start of line
// or if any non-space characters (including tabs) appear before current position.
func (s *Scanner) GetIndentLevel() int {
	// Calculate distance from line start
	indentChars := s.pos - s.lineStart

	if indentChars <= 0 {
		return 0
	}

	// Verify all characters are spaces (no tabs)
	for i := s.lineStart; i < s.pos; i++ {
		if s.input[i] != ' ' {
			return 0 // Tab or other character found
		}
	}

	return indentChars
}

// ValidateListIndentation validates that the current line's indentation matches the expected level.
// This enforces the 2-space indentation requirement for lists. The method checks:
// - Indentation is exactly expectedIndent spaces (no more, no less)
// - No tab characters in the indentation
// - Consistent spacing across all list items
//
// Returns nil if indentation is valid, or an error with details about the violation.
func (s *Scanner) ValidateListIndentation(expectedIndent int) error {
	actualIndent := s.GetIndentLevel()

	if actualIndent != expectedIndent {
		return fmt.Errorf(
			"%s:%d:%d: invalid list indentation: expected %d spaces, found %d spaces",
			s.filename,
			s.line,
			s.col,
			expectedIndent,
			actualIndent,
		)
	}

	return nil
}

// PeekToken peeks at the next identifier token without consuming it.
// Optimized to scan forward without saving/restoring state.
func (s *Scanner) PeekToken() string {
	pos := s.pos

	// Find start of identifier (skip any leading whitespace is caller's responsibility)
	start := pos

	// Scan identifier characters
	for pos < len(s.input) {
		// Fast path for ASCII alphanumeric, dash, underscore
		ch := s.input[pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			pos++
			continue
		}

		// Check for multi-byte UTF-8 characters
		if ch >= 0x80 {
			r, size := utf8.DecodeRuneInString(s.input[pos:])
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				pos += size
				continue
			}
		}

		break
	}

	return s.input[start:pos]
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

// ReadValue reads a value from the current position until end of line or comment.
// The method implements context-aware comment handling: it stops reading at a '#' character
// when outside quoted strings (treating it as the start of a comment), but preserves '#'
// characters that appear inside single-quoted or double-quoted strings as literal content.
//
// The returned value has leading/trailing whitespace trimmed and surrounding quotes removed.
// Quote characters (' or ") are only stripped if they appear as matching pairs at the start
// and end of the value. The method tracks quote state to handle nested quotes correctly.
//
// Comment handling examples:
//
//	key: value # comment     → returns "value" (stops at #)
//	key: 'val#ue'            → returns "val#ue" (# inside quotes preserved)
//	key: "test # ok"         → returns "test # ok" (# inside quotes preserved)
//
// The method also detects and marks unquoted backslash characters as syntax errors
// (see FR-014 in the feature specification).
func (s *Scanner) ReadValue() string {
	start := s.pos
	end := s.pos

	inSingleQuote := false
	inDoubleQuote := false

	// Read until newline, or # (outside quotes)
	for !s.IsEOF() && s.PeekChar() != '\n' && s.PeekChar() != '\r' {
		ch := s.PeekChar()

		// Check for comment start outside quotes
		if ch == '#' && !inSingleQuote && !inDoubleQuote {
			break // Stop at comment
		}

		// Toggle quote states
		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		}
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}

		s.Advance()
		end = s.pos
	}

	value := strings.TrimSpace(s.input[start:end])

	// Check if value is quoted (before stripping quotes)
	valueWasQuoted := len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[0] == value[len(value)-1]

	// Remove surrounding quotes if present
	if valueWasQuoted {
		value = value[1 : len(value)-1]
	}

	// Detect backslash outside quotes (FR-014)
	// If the original value was quoted, backslashes inside are valid
	// If not quoted, backslashes are syntax errors - include a marker
	if !valueWasQuoted && strings.ContainsRune(value, '\\') {
		// Prepend a special marker that the parser will detect and report as error
		// Use a null byte as marker since it's not valid in normal text
		value = "\x00BACKSLASH_ERROR\x00" + value
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
