// Package diagnostics provides utilities for formatting compiler diagnostics
// into human-readable messages for CLI output.
//
// This package converts compiler.Metadata errors and warnings into formatted
// messages that include file, line, and column information when available,
// following the PRD requirements for diagnostic presentation.
package diagnostics

import (
	"fmt"
	"io"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// Formatter formats compiler diagnostics for CLI output.
type Formatter struct {
	// UseColor enables colored output for terminals.
	UseColor bool
}

// NewFormatter creates a new diagnostics formatter.
func NewFormatter(useColor bool) *Formatter {
	return &Formatter{
		UseColor: useColor,
	}
}

// FormatErrors formats error diagnostics from compiler metadata.
// Returns a slice of formatted error strings ready for output.
func (f *Formatter) FormatErrors(errors []string) []string {
	if len(errors) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(errors))
	for _, errMsg := range errors {
		formatted = append(formatted, f.formatMessage("error", errMsg))
	}
	return formatted
}

// FormatWarnings formats warning diagnostics from compiler metadata.
// Returns a slice of formatted warning strings ready for output.
func (f *Formatter) FormatWarnings(warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(warnings))
	for _, warnMsg := range warnings {
		formatted = append(formatted, f.formatMessage("warning", warnMsg))
	}
	return formatted
}

// formatMessage formats a single diagnostic message.
// The message is expected to already contain file:line:col information
// from the compiler's diagnostic formatting.
func (f *Formatter) formatMessage(severity, message string) string {
	// Messages from compiler already include file:line:col: severity: message format
	// We just need to optionally add color
	if f.UseColor {
		return f.colorize(severity, message)
	}
	return message
}

// colorize adds ANSI color codes to diagnostic messages.
func (f *Formatter) colorize(severity, message string) string {
	const (
		red    = "\033[31m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	switch severity {
	case "error":
		return red + message + reset
	case "warning":
		return yellow + message + reset
	default:
		return message
	}
}

// PrintErrors writes formatted errors to the provided writer.
func (f *Formatter) PrintErrors(w io.Writer, errors []string) {
	formatted := f.FormatErrors(errors)
	for _, msg := range formatted {
		_, _ = fmt.Fprintln(w, msg)
	}
}

// PrintWarnings writes formatted warnings to the provided writer.
func (f *Formatter) PrintWarnings(w io.Writer, warnings []string) {
	formatted := f.FormatWarnings(warnings)
	for _, msg := range formatted {
		_, _ = fmt.Fprintln(w, msg)
	}
}

// SummarizeDiagnostics creates a summary of compilation diagnostics.
func SummarizeDiagnostics(metadata compiler.Metadata) string {
	var parts []string

	errorCount := len(metadata.Errors)
	warningCount := len(metadata.Warnings)

	if errorCount > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errorCount))
	}

	if warningCount > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warningCount))
	}

	if len(parts) == 0 {
		return "compilation successful"
	}

	return "compilation completed with " + strings.Join(parts, " and ")
}
