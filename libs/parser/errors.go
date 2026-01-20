// Package parser provides error types and formatting for parse errors.
package parser

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// ParseErrorKind enumerates the types of parse errors.
type ParseErrorKind int

const (
	// LexError indicates a lexical/tokenization error.
	LexError ParseErrorKind = iota
	// SyntaxError indicates a grammar/syntax error.
	SyntaxError
	// IOError indicates a file I/O error.
	IOError
)

const (
	listEmptyItemErrorTitle          = "empty list item"
	listInconsistentIndentErrorTitle = "inconsistent list indentation"
	// listDepthExceededErrorTitle is used when list nesting exceeds the allowed limit.
	listDepthExceededErrorTitle  = "list nesting depth exceeded"
	listTabIndentationErrorTitle = "tab character in indentation"
	listWhitespaceOnlyErrorTitle = "list contains only whitespace"
)

func listEmptyItemErrorMessage() string {
	return fmt.Sprintf("%s\n\nList items cannot be empty. Provide a value after the dash.", listEmptyItemErrorTitle)
}

func listInconsistentIndentErrorMessage(expected, got int) string {
	return fmt.Sprintf("%s\n\nExpected %d spaces (matching previous list item), got %d spaces.", listInconsistentIndentErrorTitle, expected, got)
}

func listDepthExceededErrorMessage(currentDepth, maxDepth int) string {
	return fmt.Sprintf("%s\n\nMaximum list nesting depth is %d levels. Current depth: %d.\nSimplify your data structure or split into multiple sections.", listDepthExceededErrorTitle, maxDepth, currentDepth)
}

func listTabIndentationErrorMessage() string {
	return fmt.Sprintf("%s\n\nIndentation must use spaces only. Replace tabs with 2 spaces per level.", listTabIndentationErrorTitle)
}

func listWhitespaceOnlyErrorMessage() string {
	return fmt.Sprintf("%s\n\nList must have explicit items or use empty list syntax: []", listWhitespaceOnlyErrorTitle)
}

// String returns the string representation of the error kind.
func (k ParseErrorKind) String() string {
	switch k {
	case LexError:
		return "LexError"
	case SyntaxError:
		return "SyntaxError"
	case IOError:
		return "IOError"
	default:
		return "UnknownError"
	}
}

// ParseError represents a structured parse error with source location.
type ParseError struct {
	kind     ParseErrorKind
	filename string
	line     int
	col      int
	message  string
	snippet  string // Context lines showing the error location
}

// NewParseError creates a new ParseError.
func NewParseError(kind ParseErrorKind, filename string, line, col int, message string) *ParseError {
	return &ParseError{
		kind:     kind,
		filename: filename,
		line:     line,
		col:      col,
		message:  message,
	}
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.filename, e.line, e.col, e.message)
}

// Kind returns the error kind.
func (e *ParseError) Kind() ParseErrorKind {
	return e.kind
}

// Span returns the source location of the error.
func (e *ParseError) Span() ast.SourceSpan {
	return ast.SourceSpan{
		Filename:  e.filename,
		StartLine: e.line,
		StartCol:  e.col,
		EndLine:   e.line,
		EndCol:    e.col,
	}
}

// Filename returns the source filename.
func (e *ParseError) Filename() string {
	return e.filename
}

// Line returns the line number.
func (e *ParseError) Line() int {
	return e.line
}

// Column returns the column number.
func (e *ParseError) Column() int {
	return e.col
}

// Message returns the error message.
func (e *ParseError) Message() string {
	return e.message
}

// Snippet returns the context snippet with error location.
func (e *ParseError) Snippet() string {
	return e.snippet
}

// SetSnippet sets the context snippet for this error.
func (e *ParseError) SetSnippet(snippet string) {
	e.snippet = snippet
}

// FormatParseError formats a parse error with a snippet and caret marker.
// It returns a multi-line string with:
//   - file:line:col: message (machine-parseable prefix)
//   - 1-3 lines of context
//   - A caret (^) pointing to the error position
//
// If err is not a ParseError, it returns the error's string representation.
func FormatParseError(err error, sourceText string) string {
	parseErr, ok := err.(*ParseError)
	if !ok {
		// Try to extract from wrapped error
		for e := err; e != nil; {
			if pe, ok := e.(*ParseError); ok {
				parseErr = pe
				break
			}
			// Check if it's a standard wrapped error
			type unwrapper interface {
				Unwrap() error
			}
			if u, ok := e.(unwrapper); ok {
				e = u.Unwrap()
			} else {
				break
			}
		}
		if parseErr == nil {
			return err.Error()
		}
	}

	var b strings.Builder

	// Machine-parseable prefix: file:line:col: message
	b.WriteString(parseErr.Error())
	b.WriteString("\n")

	// Generate snippet if not already set and source text is provided
	if parseErr.snippet == "" && sourceText != "" {
		snippet := generateSnippet(sourceText, parseErr.line, parseErr.col)
		parseErr.SetSnippet(snippet)
	}

	// Add snippet if available
	if parseErr.snippet != "" {
		b.WriteString(parseErr.snippet)
	}

	return b.String()
}

// generateSnippet creates a context snippet with a caret pointing to the error.
// It shows 1-3 lines of context centered around the error line.
func generateSnippet(sourceText string, line, col int) string {
	lines := strings.Split(sourceText, "\n")
	if line < 1 || line > len(lines) {
		return ""
	}

	var b strings.Builder

	// Determine context lines to show (up to 1 before, error line, up to 1 after)
	startLine := line - 1
	if startLine < 1 {
		startLine = 1
	}
	endLine := line + 1
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// Show context lines
	for i := startLine; i <= endLine; i++ {
		if i >= 1 && i <= len(lines) {
			b.WriteString(fmt.Sprintf("%4d | %s\n", i, lines[i-1]))
		}
	}

	// Add caret line pointing to the error column
	if col > 0 {
		// Account for line number prefix (4 digits + " | ")
		prefix := "     | "
		// Count runes up to col-1 to handle multi-byte characters correctly
		lineText := lines[line-1]
		runeCount := 0
		byteCount := 0
		for byteCount < len(lineText) && runeCount < col-1 {
			_, size := utf8.DecodeRuneInString(lineText[byteCount:])
			byteCount += size
			runeCount++
		}

		spaces := strings.Repeat(" ", runeCount)
		b.WriteString(prefix)
		b.WriteString(spaces)
		b.WriteString("^\n")
	}

	return b.String()
}
