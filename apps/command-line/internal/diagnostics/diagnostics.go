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
	"github.com/fatih/color"
)

// Formatter formats compiler diagnostics for CLI output.
type Formatter struct {
	// UseColor enables colored output for terminals.
	UseColor bool

	// Color functions for styling
	errorColor   *color.Color
	warningColor *color.Color
	boldColor    *color.Color
}

// NewFormatter creates a new diagnostics formatter.
func NewFormatter(useColor bool) *Formatter {
	f := &Formatter{
		UseColor:     useColor,
		errorColor:   color.New(color.FgRed, color.Bold),
		warningColor: color.New(color.FgYellow, color.Bold),
		boldColor:    color.New(color.Bold),
	}

	// Configure color output based on useColor flag
	// Note: We don't set color.NoColor globally as it affects all instances
	// Instead, we control color output through formatMessage
	return f
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
	if !f.UseColor {
		return message
	}

	// Temporarily enable color for this specific formatting
	oldNoColor := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = oldNoColor }()

	// Apply color based on severity
	switch severity {
	case "error":
		return f.errorColor.Sprint(message)
	case "warning":
		return f.warningColor.Sprint(message)
	default:
		return message
	}
}

// PrintErrors writes formatted errors to the provided writer.
func (f *Formatter) PrintErrors(w io.Writer, errors []string) {
	if len(errors) == 0 {
		return
	}

	// Temporarily enable color if requested
	if f.UseColor {
		oldNoColor := color.NoColor
		color.NoColor = false
		defer func() { color.NoColor = oldNoColor }()
	}

	// Print header
	if f.UseColor {
		_, _ = f.errorColor.Fprintf(w, "Errors:\n") // Ignore write errors
	} else {
		_, _ = fmt.Fprintf(w, "Errors:\n") // Ignore write errors
	}

	formatted := f.FormatErrors(errors)
	for _, msg := range formatted {
		_, _ = fmt.Fprintln(w, msg)
	}
}

// PrintWarnings writes formatted warnings to the provided writer.
func (f *Formatter) PrintWarnings(w io.Writer, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	// Temporarily enable color if requested
	if f.UseColor {
		oldNoColor := color.NoColor
		color.NoColor = false
		defer func() { color.NoColor = oldNoColor }()
	}

	// Print header
	if f.UseColor {
		_, _ = f.warningColor.Fprintf(w, "Warnings:\n") // Ignore write errors
	} else {
		_, _ = fmt.Fprintf(w, "Warnings:\n") // Ignore write errors
	}

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
