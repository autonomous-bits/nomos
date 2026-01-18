// Package scanner_test contains unit tests for the scanner package.
package scanner_test

import (
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/internal/scanner"
)

// TestScanner_New_InitializesCorrectly tests scanner initialization.
func TestScanner_New_InitializesCorrectly(t *testing.T) {
	// Arrange
	input := "test input"
	filename := "test.csl"

	// Act
	s := scanner.New(input, filename)

	// Assert
	if s.Filename() != filename {
		t.Errorf("expected filename %s, got %s", filename, s.Filename())
	}
	if s.Line() != 1 {
		t.Errorf("expected line 1, got %d", s.Line())
	}
	if s.Column() != 1 {
		t.Errorf("expected column 1, got %d", s.Column())
	}
	if s.IsEOF() {
		t.Error("expected not EOF for non-empty input")
	}
}

// TestScanner_PeekChar_ReturnsCurrentCharacter tests character peeking.
func TestScanner_PeekChar_ReturnsCurrentCharacter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{"letter", "abc", 'a'},
		{"number", "123", '1'},
		{"symbol", ":test", ':'},
		{"unicode", "こんにちは", 'こ'},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.PeekChar()
			if result != tt.expected {
				t.Errorf("expected '%c' (%d), got '%c' (%d)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

// TestScanner_Advance_MovesPosition tests position advancement.
func TestScanner_Advance_MovesPosition(t *testing.T) {
	// Arrange
	input := "ab\ncd"
	s := scanner.New(input, "test.csl")

	// Act & Assert - first character 'a'
	if s.PeekChar() != 'a' {
		t.Errorf("expected 'a', got '%c'", s.PeekChar())
	}
	if s.Line() != 1 || s.Column() != 1 {
		t.Errorf("expected position 1:1, got %d:%d", s.Line(), s.Column())
	}

	s.Advance() // move to 'b'
	if s.PeekChar() != 'b' {
		t.Errorf("expected 'b', got '%c'", s.PeekChar())
	}
	if s.Line() != 1 || s.Column() != 2 {
		t.Errorf("expected position 1:2, got %d:%d", s.Line(), s.Column())
	}

	s.Advance() // move to '\n'
	if s.PeekChar() != '\n' {
		t.Errorf("expected newline, got '%c'", s.PeekChar())
	}

	s.Advance() // move to 'c' (new line)
	if s.Line() != 2 || s.Column() != 1 {
		t.Errorf("expected position 2:1, got %d:%d", s.Line(), s.Column())
	}
	if s.PeekChar() != 'c' {
		t.Errorf("expected 'c', got '%c'", s.PeekChar())
	}
}

// TestScanner_ReadIdentifier_ReadsValidIdentifiers tests identifier reading.
func TestScanner_ReadIdentifier_ReadsValidIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "test", "test"},
		{"with dash", "test-name", "test-name"},
		{"with underscore", "test_name", "test_name"},
		{"with numbers", "test123", "test123"},
		{"keyword source", "source", "source"},
		{"keyword import", "import", "import"},
		{"keyword reference", "reference", "reference"},
		{"unicode", "設定", "設定"},
		{"mixed", "config-2_test", "config-2_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadIdentifier()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestScanner_ReadIdentifier_StopsAtDelimiters tests identifier boundaries.
func TestScanner_ReadIdentifier_StopsAtDelimiters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		remaining rune
	}{
		{"colon", "test:", "test", ':'},
		{"space", "test ", "test", ' '},
		{"newline", "test\n", "test", '\n'},
		{"dot", "test.name", "test", '.'},
		{"tab", "test\t", "test", '\t'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadIdentifier()
			if result != tt.expected {
				t.Errorf("expected identifier '%s', got '%s'", tt.expected, result)
			}
			if s.PeekChar() != tt.remaining {
				t.Errorf("expected remaining char '%c', got '%c'", tt.remaining, s.PeekChar())
			}
		})
	}
}

// TestScanner_ReadPath_ReadsDottedPaths tests path expression reading.
func TestScanner_ReadPath_ReadsDottedPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "config", "config"},
		{"dotted", "config.key", "config.key"},
		{"deep", "a.b.c.d", "a.b.c.d"},
		{"with dash", "config-key.value", "config-key.value"},
		{"with underscore", "config_key.value_name", "config_key.value_name"},
		{"with numbers", "config1.key2.value3", "config1.key2.value3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadPath()
			if result != tt.expected {
				t.Errorf("expected path '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestScanner_ReadValue_ReadsAndTrimsValues tests value reading.
func TestScanner_ReadValue_ReadsAndTrimsValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", "value", "value"},
		{"with spaces", "  value  ", "value"},
		{"single quotes", "'value'", "value"},
		{"double quotes", `"value"`, "value"},
		{"quoted with spaces", "  'value'  ", "value"},
		{"path", "../config/file", "../config/file"},
		{"with special chars", "value-123_test", "value-123_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadValue()
			if result != tt.expected {
				t.Errorf("expected value '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestScanner_SkipWhitespace_SkipsSpacesAndTabs tests whitespace skipping.
func TestScanner_SkipWhitespace_SkipsSpacesAndTabs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{"spaces", "   a", 'a'},
		{"tabs", "\t\ta", 'a'},
		{"mixed", " \t \ta", 'a'},
		{"no whitespace", "a", 'a'},
		{"preserves newline", "  \na", '\n'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			s.SkipWhitespace()
			result := s.PeekChar()
			if result != tt.expected {
				t.Errorf("expected '%c', got '%c'", tt.expected, result)
			}
		})
	}
}

// TestScanner_IsIndented_DetectsIndentation tests indentation detection.
func TestScanner_IsIndented_DetectsIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		advance  int
		expected bool
	}{
		{"indented spaces", "  key", 0, true},
		{"indented tab", "\tkey", 0, true},
		{"not indented", "key", 0, false},
		{"after newline indented", "a\n  key", 2, true},
		{"after newline not indented", "a\nkey", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			for i := 0; i < tt.advance; i++ {
				s.Advance()
			}
			result := s.IsIndented()
			if result != tt.expected {
				t.Errorf("expected IsIndented=%v, got %v", tt.expected, result)
			}
		})
	}
}

// TestScanner_Expect_ConsumesExpectedCharacter tests character expectation.
func TestScanner_Expect_ConsumesExpectedCharacter(t *testing.T) {
	// Arrange
	s := scanner.New(":test", "test.csl")

	// Act
	err := s.Expect(':')

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if s.PeekChar() != 't' {
		t.Errorf("expected 't' after consuming ':', got '%c'", s.PeekChar())
	}
}

// TestScanner_Expect_ReturnsErrorForUnexpectedCharacter tests expectation errors.
func TestScanner_Expect_ReturnsErrorForUnexpectedCharacter(t *testing.T) {
	// Arrange
	s := scanner.New("test", "test.csl")

	// Act
	err := s.Expect(':')

	// Assert
	if err == nil {
		t.Error("expected error for unexpected character, got nil")
	}
}

// TestScanner_IsEOF_DetectsEndOfInput tests EOF detection.
func TestScanner_IsEOF_DetectsEndOfInput(t *testing.T) {
	// Arrange
	s := scanner.New("ab", "test.csl")

	// Assert - not EOF initially
	if s.IsEOF() {
		t.Error("expected not EOF at start")
	}

	// Act - advance through input
	s.Advance() // 'a'
	if s.IsEOF() {
		t.Error("expected not EOF after first char")
	}

	s.Advance() // 'b'
	if !s.IsEOF() {
		t.Error("expected EOF after consuming all input")
	}
}

// TestScanner_PeekToken_DoesNotConsumeInput tests token peeking.
func TestScanner_PeekToken_DoesNotConsumeInput(t *testing.T) {
	// Arrange
	s := scanner.New("source:test", "test.csl")

	// Act
	token := s.PeekToken()

	// Assert
	if token != "source" {
		t.Errorf("expected token 'source', got '%s'", token)
	}
	// Verify position hasn't changed
	if s.PeekChar() != 's' {
		t.Errorf("expected scanner still at 's', got '%c'", s.PeekChar())
	}
}

// TestScanner_PositionTracking_MultipleLines tests position tracking across lines.
func TestScanner_PositionTracking_MultipleLines(t *testing.T) {
	// Arrange
	input := "line1\nline2\nline3"
	s := scanner.New(input, "test.csl")

	// Line 1
	if s.Line() != 1 || s.Column() != 1 {
		t.Errorf("expected 1:1, got %d:%d", s.Line(), s.Column())
	}

	// Advance to end of line 1
	for i := 0; i < 5; i++ {
		s.Advance()
	}
	if s.Line() != 1 || s.Column() != 6 {
		t.Errorf("expected 1:6, got %d:%d", s.Line(), s.Column())
	}

	// Advance past newline to line 2
	s.Advance()
	if s.Line() != 2 || s.Column() != 1 {
		t.Errorf("expected 2:1, got %d:%d", s.Line(), s.Column())
	}
}

// TestScanner_UnicodeSupport_HandlesMultibyteCharacters tests unicode handling.
func TestScanner_UnicodeSupport_HandlesMultibyteCharacters(t *testing.T) {
	// Arrange
	input := "日本語"
	s := scanner.New(input, "test.csl")

	// Act - read identifier
	result := s.ReadIdentifier()

	// Assert
	if result != "日本語" {
		t.Errorf("expected '日本語', got '%s'", result)
	}
}

// ============================================================================
// Comment Support Tests (Feature: 003-yaml-comments)
// ============================================================================

// TestScanner_IsCommentStart_DetectsHashCharacter tests comment detection.
func TestScanner_IsCommentStart_DetectsHashCharacter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		advance  int
		expected bool
	}{
		{"at hash", "#comment", 0, true},
		{"not at hash - letter", "key", 0, false},
		{"not at hash - colon", ":", 0, false},
		{"at hash after text", "value #comment", 6, true},
		{"at EOF", "", 0, false},
		{"after EOF", "a", 2, false},
		{"hash mid-line", "key: value # comment", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			for i := 0; i < tt.advance; i++ {
				s.Advance()
			}
			result := s.IsCommentStart()
			if result != tt.expected {
				t.Errorf("expected IsCommentStart=%v, got %v at position %d (char '%c')",
					tt.expected, result, s.Pos(), s.PeekChar())
			}
		})
	}
}

// TestScanner_SkipComment_BasicBehavior tests basic comment skipping.
func TestScanner_SkipComment_BasicBehavior(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedAtChar rune
		expectedEOF    bool
	}{
		{"simple comment", "# comment\nkey", '\n', false},
		{"empty comment", "#\nkey", '\n', false},
		{"comment with spaces", "#   comment text  \nkey", '\n', false},
		{"comment with special chars", "# comment: with 'special' chars\nkey", '\n', false},
		{"comment at EOF no newline", "# comment", 0, true},
		{"comment with unicode", "# コメント\nkey", '\n', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")

			// Skip to # if not already there
			if s.PeekChar() != '#' {
				t.Fatal("test setup error: should start at #")
			}

			s.SkipComment()

			// Check final position
			if tt.expectedEOF {
				if !s.IsEOF() {
					t.Errorf("expected EOF, but not at EOF (char: '%c')", s.PeekChar())
				}
			} else {
				if s.PeekChar() != tt.expectedAtChar {
					t.Errorf("expected to stop at '%c', got '%c'",
						tt.expectedAtChar, s.PeekChar())
				}
			}
		})
	}
}

// TestScanner_SkipComment_PositionTracking tests position updates during skip.
func TestScanner_SkipComment_PositionTracking(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedLine int
		expectedCol  int
	}{
		{"short comment", "# abc\n", 1, 6},
		{"longer comment", "# this is a longer comment\n", 1, 27},
		{"comment with unicode", "# 日本語\n", 1, 6}, // # + space + 3 Unicode chars = 5 chars advanced, at pos 6
		{"empty comment", "#\n", 1, 2},
		{"comment at EOF", "# comment", 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			s.SkipComment()

			if s.Line() != tt.expectedLine {
				t.Errorf("expected line %d, got %d", tt.expectedLine, s.Line())
			}
			if s.Column() != tt.expectedCol {
				t.Errorf("expected column %d, got %d", tt.expectedCol, s.Column())
			}
		})
	}
}

// TestScanner_SkipComment_AtEOF tests comment skipping at end of file.
func TestScanner_SkipComment_AtEOF(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"comment at EOF", "# comment"},
		{"empty comment at EOF", "#"},
		{"comment with text at EOF", "# some text here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			s.SkipComment()

			// Should be at EOF
			if !s.IsEOF() {
				t.Errorf("expected EOF, but scanner not at EOF (char: '%c', pos: %d)",
					s.PeekChar(), s.Pos())
			}
		})
	}
}

// TestScanner_ReadValue_StopsAtComment tests value reading with comments.
func TestScanner_ReadValue_StopsAtComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"value with trailing comment", "value # comment", "value"},
		{"value with immediate comment", "value# comment", "value"},
		{"value with whitespace before comment", "value   # comment", "value"},
		{"empty value with comment", "# comment", ""},
		{"value at EOF no comment", "value", "value"},
		{"quoted value with comment after", "'value' # comment", "value"},
		{"double quoted with comment after", `"value" # comment`, "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadValue()

			if result != tt.expected {
				t.Errorf("expected value '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestScanner_ReadValue_PreservesHashInQuotedStrings tests # preservation in quotes.
func TestScanner_ReadValue_PreservesHashInQuotedStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hash in single quotes", "'value#with#hash'", "value#with#hash"},
		{"hash in double quotes", `"value#with#hash"`, "value#with#hash"},
		{"hash at start in quotes", "'#value'", "#value"},
		{"hash at end in quotes", "'value#'", "value#"},
		{"multiple hashes in quotes", "'##value##'", "##value##"},
		{"comment-like text in quotes", "'# this looks like comment'", "# this looks like comment"},
		{"hash in single quotes with trailing comment", "'value#hash' # actual comment", "value#hash"},
		{"hash in double quotes with trailing comment", `"value#hash" # actual comment`, "value#hash"},
		{"empty quoted with hash", "'#'", "#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanner.New(tt.input, "test.csl")
			result := s.ReadValue()

			if result != tt.expected {
				t.Errorf("expected value '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
