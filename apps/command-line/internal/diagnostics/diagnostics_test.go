package diagnostics_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/diagnostics"
	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestFormatter_FormatErrors_EmptySlice tests that formatting empty errors returns nil.
func TestFormatter_FormatErrors_EmptySlice(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	var errors []string

	// Act
	result := formatter.FormatErrors(errors)

	// Assert
	if result != nil {
		t.Errorf("Expected nil for empty errors, got %v", result)
	}
}

// TestFormatter_FormatErrors_SingleError tests formatting a single error message.
func TestFormatter_FormatErrors_SingleError(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	errors := []string{"config.csl:3:9: error: unresolved reference to provider 'config'"}

	// Act
	result := formatter.FormatErrors(errors)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 formatted error, got %d", len(result))
	}

	if !strings.Contains(result[0], "config.csl:3:9:") {
		t.Errorf("Expected file:line:col in error, got: %s", result[0])
	}

	if !strings.Contains(result[0], "unresolved reference") {
		t.Errorf("Expected error message text, got: %s", result[0])
	}
}

// TestFormatter_FormatErrors_MultipleErrors tests formatting multiple error messages.
func TestFormatter_FormatErrors_MultipleErrors(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	errors := []string{
		"app.csl:5:12: error: duplicate key 'database'",
		"config.csl:10:3: error: invalid provider alias",
	}

	// Act
	result := formatter.FormatErrors(errors)

	// Assert
	if len(result) != 2 {
		t.Fatalf("Expected 2 formatted errors, got %d", len(result))
	}

	if !strings.Contains(result[0], "app.csl:5:12:") {
		t.Errorf("Expected first error to contain location, got: %s", result[0])
	}

	if !strings.Contains(result[1], "config.csl:10:3:") {
		t.Errorf("Expected second error to contain location, got: %s", result[1])
	}
}

// TestFormatter_FormatWarnings_EmptySlice tests that formatting empty warnings returns nil.
func TestFormatter_FormatWarnings_EmptySlice(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	var warnings []string

	// Act
	result := formatter.FormatWarnings(warnings)

	// Assert
	if result != nil {
		t.Errorf("Expected nil for empty warnings, got %v", result)
	}
}

// TestFormatter_FormatWarnings_SingleWarning tests formatting a single warning message.
func TestFormatter_FormatWarnings_SingleWarning(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	warnings := []string{"config.csl:7:5: warning: provider fetch failed, using nil value"}

	// Act
	result := formatter.FormatWarnings(warnings)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 formatted warning, got %d", len(result))
	}

	if !strings.Contains(result[0], "config.csl:7:5:") {
		t.Errorf("Expected file:line:col in warning, got: %s", result[0])
	}

	if !strings.Contains(result[0], "provider fetch failed") {
		t.Errorf("Expected warning message text, got: %s", result[0])
	}
}

// TestFormatter_PrintErrors tests printing errors to a writer.
func TestFormatter_PrintErrors(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	errors := []string{
		"app.csl:5:12: error: duplicate key 'database'",
		"config.csl:10:3: error: invalid provider alias",
	}
	var buf bytes.Buffer

	// Act
	formatter.PrintErrors(&buf, errors)

	// Assert
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Expect 3 lines: "Errors:" header + 2 error lines
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines of output (header + 2 errors), got %d", len(lines))
	}

	// First line should be "Errors:" header
	if !strings.Contains(lines[0], "Errors:") {
		t.Errorf("Expected first line to be header, got: %s", lines[0])
	}

	// Second line should be first error
	if !strings.Contains(lines[1], "app.csl:5:12:") {
		t.Errorf("Expected second line to contain error location, got: %s", lines[1])
	}

	// Third line should be second error
	if !strings.Contains(lines[2], "config.csl:10:3:") {
		t.Errorf("Expected third line to contain error location, got: %s", lines[2])
	}
}

// TestFormatter_PrintWarnings tests printing warnings to a writer.
func TestFormatter_PrintWarnings(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	warnings := []string{"config.csl:7:5: warning: provider fetch failed"}
	var buf bytes.Buffer

	// Act
	formatter.PrintWarnings(&buf, warnings)

	// Assert
	output := buf.String()
	if !strings.Contains(output, "config.csl:7:5:") {
		t.Errorf("Expected output to contain warning location, got: %s", output)
	}
}

// TestFormatter_ColorOutput tests that color mode adds ANSI codes.
func TestFormatter_ColorOutput(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(true)
	errors := []string{"app.csl:5:12: error: test error"}

	// Act
	result := formatter.FormatErrors(errors)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 formatted error, got %d", len(result))
	}

	// Should contain ANSI color codes (red for errors)
	// fatih/color uses different escape sequences
	if !strings.Contains(result[0], "\x1b[") && !strings.Contains(result[0], "\033[") {
		t.Errorf("Expected color codes in output, got: %s", result[0])
	}
}

// TestSummarizeDiagnostics_NoIssues tests summary with clean compilation.
func TestSummarizeDiagnostics_NoIssues(t *testing.T) {
	// Arrange
	metadata := compiler.Metadata{
		Errors:   []string{},
		Warnings: []string{},
	}

	// Act
	summary := diagnostics.SummarizeDiagnostics(metadata)

	// Assert
	if summary != "compilation successful" {
		t.Errorf("Expected 'compilation successful', got: %s", summary)
	}
}

// TestSummarizeDiagnostics_ErrorsOnly tests summary with errors only.
func TestSummarizeDiagnostics_ErrorsOnly(t *testing.T) {
	// Arrange
	metadata := compiler.Metadata{
		Errors:   []string{"error1", "error2"},
		Warnings: []string{},
	}

	// Act
	summary := diagnostics.SummarizeDiagnostics(metadata)

	// Assert
	if !strings.Contains(summary, "2 error(s)") {
		t.Errorf("Expected summary to contain '2 error(s)', got: %s", summary)
	}
}

// TestSummarizeDiagnostics_WarningsOnly tests summary with warnings only.
func TestSummarizeDiagnostics_WarningsOnly(t *testing.T) {
	// Arrange
	metadata := compiler.Metadata{
		Errors:   []string{},
		Warnings: []string{"warning1"},
	}

	// Act
	summary := diagnostics.SummarizeDiagnostics(metadata)

	// Assert
	if !strings.Contains(summary, "1 warning(s)") {
		t.Errorf("Expected summary to contain '1 warning(s)', got: %s", summary)
	}
}

// TestSummarizeDiagnostics_ErrorsAndWarnings tests summary with both errors and warnings.
func TestSummarizeDiagnostics_ErrorsAndWarnings(t *testing.T) {
	// Arrange
	metadata := compiler.Metadata{
		Errors:   []string{"error1"},
		Warnings: []string{"warning1", "warning2"},
	}

	// Act
	summary := diagnostics.SummarizeDiagnostics(metadata)

	// Assert
	if !strings.Contains(summary, "1 error(s)") {
		t.Errorf("Expected summary to contain '1 error(s)', got: %s", summary)
	}

	if !strings.Contains(summary, "2 warning(s)") {
		t.Errorf("Expected summary to contain '2 warning(s)', got: %s", summary)
	}

	if !strings.Contains(summary, "and") {
		t.Errorf("Expected summary to contain 'and', got: %s", summary)
	}
}

// TestFormatter_FormatErrors_PreservesOriginalFormat tests that the formatter
// preserves the file:line:col format from compiler diagnostics.
func TestFormatter_FormatErrors_PreservesOriginalFormat(t *testing.T) {
	// Arrange
	formatter := diagnostics.NewFormatter(false)
	// This format comes from compiler's diagnostic.FormatDiagnostic
	errors := []string{"test.csl:10:5: error: unresolved reference to provider 'db'\n   10 |   host: @db:host\n     |         ^"}

	// Act
	result := formatter.FormatErrors(errors)

	// Assert
	if len(result) != 1 {
		t.Fatalf("Expected 1 formatted error, got %d", len(result))
	}

	// Should preserve multi-line format with snippet
	if !strings.Contains(result[0], "test.csl:10:5:") {
		t.Errorf("Expected preserved location format, got: %s", result[0])
	}

	if !strings.Contains(result[0], "@db:host") {
		t.Errorf("Expected preserved snippet, got: %s", result[0])
	}

	if !strings.Contains(result[0], "^") {
		t.Errorf("Expected preserved caret marker, got: %s", result[0])
	}
}
