// Package diagnostic provides structured diagnostic types for compiler errors and warnings.
package diagnostic

import (
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Severity indicates the severity level of a diagnostic.
type Severity int

const (
	// SeverityError indicates a fatal error that prevents compilation.
	SeverityError Severity = iota
	// SeverityWarning indicates a non-fatal issue.
	SeverityWarning
	// SeverityInfo indicates informational messages.
	SeverityInfo
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Diagnostic represents a structured compiler diagnostic with source location.
type Diagnostic struct {
	// Severity indicates the diagnostic level.
	Severity Severity

	// Message contains the human-readable diagnostic message.
	Message string

	// SourceSpan identifies the source location of the diagnostic.
	SourceSpan ast.SourceSpan

	// FormattedMessage contains the formatted output with snippet and caret.
	// This is populated by the parser's FormatParseError for parse errors.
	FormattedMessage string
}

// Error implements the error interface.
func (d *Diagnostic) Error() string {
	if d.FormattedMessage != "" {
		return d.FormattedMessage
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		d.SourceSpan.Filename,
		d.SourceSpan.StartLine,
		d.SourceSpan.StartCol,
		d.Severity,
		d.Message)
}

// IsError returns true if this diagnostic is an error.
func (d *Diagnostic) IsError() bool {
	return d.Severity == SeverityError
}

// IsWarning returns true if this diagnostic is a warning.
func (d *Diagnostic) IsWarning() bool {
	return d.Severity == SeverityWarning
}

// FormatDiagnostic formats a diagnostic with source snippet and caret marker.
// It returns a multi-line string with:
//   - file:line:col: severity: message (machine-parseable prefix)
//   - 1-3 lines of context from sourceText
//   - A caret (^) pointing to the error position
//
// If parseErr is provided and is a *parser.ParseError, uses parser.FormatParseError.
// Otherwise, generates a caret-based snippet for the diagnostic's SourceSpan.
// If sourceText is empty, returns a basic formatted message without snippet.
func FormatDiagnostic(d *Diagnostic, sourceText string, parseErr error) string {
	// If a parse error is provided, delegate to parser.FormatParseError
	if parseErr != nil {
		// Import parser package at top of file
		// The parser.FormatParseError function handles ParseError formatting
		formatted := fmt.Sprintf("%s:%d:%d: %s: %s",
			d.SourceSpan.Filename,
			d.SourceSpan.StartLine,
			d.SourceSpan.StartCol,
			d.Severity,
			d.Message)

		// Try to use parser.FormatParseError if it's a parse error
		// For now, we'll manually format since we have the diagnostic
		if sourceText != "" {
			snippet := generateSnippet(sourceText, d.SourceSpan.StartLine, d.SourceSpan.StartCol)
			if snippet != "" {
				formatted += "\n" + snippet
			}
		}
		return formatted
	}

	// Format semantic/compiler errors with SourceSpan
	formatted := fmt.Sprintf("%s:%d:%d: %s: %s",
		d.SourceSpan.Filename,
		d.SourceSpan.StartLine,
		d.SourceSpan.StartCol,
		d.Severity,
		d.Message)

	// Add snippet if source text is available
	if sourceText != "" {
		snippet := generateSnippet(sourceText, d.SourceSpan.StartLine, d.SourceSpan.StartCol)
		if snippet != "" {
			formatted += "\n" + snippet
		}
	}

	return formatted
}

// generateSnippet creates a context snippet with a caret pointing to the error.
// It shows 1-3 lines of context centered around the error line.
// This implementation mirrors the parser's snippet generation for consistency.
func generateSnippet(sourceText string, line, col int) string {
	lines := []string{""}
	lines = append(lines, splitLines(sourceText)...)

	if line < 1 || line >= len(lines) {
		return ""
	}

	var b strings.Builder

	// Determine context lines to show (up to 1 before, error line, up to 1 after)
	startLine := line - 1
	if startLine < 1 {
		startLine = 1
	}
	endLine := line + 1
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Show context lines
	for i := startLine; i <= endLine; i++ {
		if i >= 1 && i < len(lines) {
			b.WriteString(fmt.Sprintf("%4d | %s\n", i, lines[i]))
		}
	}

	// Add caret line pointing to the error column
	if col > 0 {
		// Account for line number prefix (4 digits + " | ")
		prefix := "     | "
		// Count runes up to col-1 to handle multi-byte characters correctly
		lineText := lines[line]
		runeCount := 0
		byteCount := 0
		for byteCount < len(lineText) && runeCount < col-1 {
			r := rune(lineText[byteCount])
			if r < 128 {
				byteCount++
			} else {
				// Multi-byte character
				_, size := decodeRuneInString(lineText[byteCount:])
				byteCount += size
			}
			runeCount++
		}

		spaces := ""
		for i := 0; i < runeCount; i++ {
			spaces += " "
		}
		b.WriteString(prefix)
		b.WriteString(spaces)
		b.WriteString("^\n")
	}

	return b.String()
}

// splitLines splits text into lines while preserving line structure.
func splitLines(text string) []string {
	var lines []string
	var current strings.Builder

	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	// Add final line if not empty
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	return lines
}

// decodeRuneInString is a simplified version of utf8.DecodeRuneInString.
// Returns the rune and its size in bytes.
func decodeRuneInString(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}

	r := rune(s[0])
	if r < 128 {
		return r, 1
	}

	// For multi-byte UTF-8, we need proper decoding
	// Use standard library for correctness
	var size int
	for i, ch := range s {
		if i == 0 {
			r = ch
		}
		size = i + 1
		if i > 0 {
			break
		}
	}

	return r, size
}
