// Package parse provides parser integration helpers for the compiler.
package parse

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"
	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// ParseFile parses a Nomos configuration file from the filesystem.
// It returns the AST, any diagnostics generated during parsing, and a fatal error if parsing cannot proceed.
// Parse errors are captured as diagnostics with structured SourceSpan information.
func ParseFile(path string) (*ast.AST, []diagnostic.Diagnostic, error) {
	// Read source text for error formatting
	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		// I/O errors are returned as diagnostics
		diag := diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("failed to read file: %v", err),
			SourceSpan: ast.SourceSpan{
				Filename:  path,
				StartLine: 0,
				StartCol:  0,
				EndLine:   0,
				EndCol:    0,
			},
			FormattedMessage: fmt.Sprintf("%s: failed to read file: %v", path, err),
		}
		return nil, []diagnostic.Diagnostic{diag}, nil
	}
	sourceText := string(sourceBytes)

	// Call parser
	astNode, err := parser.ParseFile(path)
	if err != nil {
		// Transform parser error to diagnostic
		diags := transformParseError(err, sourceText)
		return nil, diags, nil
	}

	return astNode, nil, nil
}

// ParseReader parses Nomos configuration from an io.Reader.
// The filename parameter is used for error messages and source spans.
// It returns the AST, any diagnostics generated during parsing, and a fatal error if parsing cannot proceed.
func ParseReader(r io.Reader, filename string) (*ast.AST, []diagnostic.Diagnostic, error) {
	// Read source text for error formatting
	sourceBytes, err := io.ReadAll(r)
	if err != nil {
		// I/O errors are returned as diagnostics
		diag := diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Message:  fmt.Sprintf("failed to read input: %v", err),
			SourceSpan: ast.SourceSpan{
				Filename:  filename,
				StartLine: 0,
				StartCol:  0,
				EndLine:   0,
				EndCol:    0,
			},
			FormattedMessage: fmt.Sprintf("%s: failed to read input: %v", filename, err),
		}
		return nil, []diagnostic.Diagnostic{diag}, nil
	}
	sourceText := string(sourceBytes)

	// Parse using parser.Parse with the source text
	astNode, err := parser.Parse(strings.NewReader(sourceText), filename)
	if err != nil {
		// Transform parser error to diagnostic
		diags := transformParseError(err, sourceText)
		return nil, diags, nil
	}

	return astNode, nil, nil
}

// transformParseError converts a parser error to compiler diagnostics.
func transformParseError(err error, sourceText string) []diagnostic.Diagnostic {
	// Check if it's a ParseError
	if parseErr, ok := err.(*parser.ParseError); ok {
		// Format the error with snippet and caret using parser's formatter
		formattedMsg := parser.FormatParseError(parseErr, sourceText)

		diag := diagnostic.Diagnostic{
			Severity:         diagnostic.SeverityError,
			Message:          parseErr.Message(),
			SourceSpan:       parseErr.Span(),
			FormattedMessage: formattedMsg,
		}
		return []diagnostic.Diagnostic{diag}
	}

	// Fallback for unknown error types
	diag := diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Message:  err.Error(),
		SourceSpan: ast.SourceSpan{
			Filename:  "unknown",
			StartLine: 0,
			StartCol:  0,
			EndLine:   0,
			EndCol:    0,
		},
		FormattedMessage: err.Error(),
	}
	return []diagnostic.Diagnostic{diag}
}
