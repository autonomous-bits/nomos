package diagnostic_test

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"
	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestFormatDiagnostic_ParseError tests formatting of parse errors using parser.FormatParseError.
func TestFormatDiagnostic_ParseError(t *testing.T) {
	// Arrange
	sourceText := "source:\n\talias: 'test'\n\ttype: 'folder'"
	parseErr := parser.NewParseError(parser.SyntaxError, "test.csl", 1, 1, "invalid syntax")

	diag := &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Message:  "invalid syntax",
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 1,
			StartCol:  1,
			EndLine:   1,
			EndCol:    1,
		},
	}

	// Act
	formatted := diagnostic.FormatDiagnostic(diag, sourceText, parseErr)

	// Assert
	// Should contain machine-parseable prefix
	if !strings.Contains(formatted, "test.csl:1:1:") {
		t.Errorf("Formatted error should contain file:line:col prefix, got:\n%s", formatted)
	}

	// Should contain message
	if !strings.Contains(formatted, "invalid syntax") {
		t.Errorf("Formatted error should contain message, got:\n%s", formatted)
	}

	// Should contain snippet with line number
	if !strings.Contains(formatted, "   1 |") {
		t.Errorf("Formatted error should contain line number, got:\n%s", formatted)
	}

	// Should contain caret marker
	if !strings.Contains(formatted, "^") {
		t.Errorf("Formatted error should contain caret marker, got:\n%s", formatted)
	}
}

// TestFormatDiagnostic_SemanticError tests formatting of semantic errors with SourceSpan.
func TestFormatDiagnostic_SemanticError(t *testing.T) {
	// Arrange
	sourceText := "app:\n  name: 'myapp'\n  port: @config:port"

	diag := &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Message:  "unresolved reference to provider 'config'",
		SourceSpan: ast.SourceSpan{
			Filename:  "app.csl",
			StartLine: 3,
			StartCol:  9,
			EndLine:   3,
			EndCol:    28,
		},
	}

	// Act
	formatted := diagnostic.FormatDiagnostic(diag, sourceText, nil)

	// Assert
	// Should contain machine-parseable prefix
	if !strings.Contains(formatted, "app.csl:3:9:") {
		t.Errorf("Expected file:line:col prefix, got:\n%s", formatted)
	}

	// Should contain error severity
	if !strings.Contains(formatted, "error") {
		t.Errorf("Expected 'error' severity label, got:\n%s", formatted)
	}

	// Should contain message
	if !strings.Contains(formatted, "unresolved reference to provider 'config'") {
		t.Errorf("Expected error message, got:\n%s", formatted)
	}

	// Should contain line 3 context
	if !strings.Contains(formatted, "   3 |") {
		t.Errorf("Expected line 3 context, got:\n%s", formatted)
	}

	// Should contain caret pointing to the reference
	if !strings.Contains(formatted, "^") {
		t.Errorf("Expected caret marker, got:\n%s", formatted)
	}
}

// TestFormatDiagnostic_Warning tests formatting of warnings.
func TestFormatDiagnostic_Warning(t *testing.T) {
	// Arrange
	sourceText := "database:\n  timeout: 30"

	diag := &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Message:  "provider fetch failed, using nil value",
		SourceSpan: ast.SourceSpan{
			Filename:  "config.csl",
			StartLine: 2,
			StartCol:  3,
			EndLine:   2,
			EndCol:    14,
		},
	}

	// Act
	formatted := diagnostic.FormatDiagnostic(diag, sourceText, nil)

	// Assert
	// Should contain warning severity
	if !strings.Contains(formatted, "warning") {
		t.Errorf("Expected 'warning' severity label, got:\n%s", formatted)
	}

	// Should contain message
	if !strings.Contains(formatted, "provider fetch failed") {
		t.Errorf("Expected warning message, got:\n%s", formatted)
	}

	// Should contain context
	if !strings.Contains(formatted, "   2 |") {
		t.Errorf("Expected line 2 context, got:\n%s", formatted)
	}
}

// TestFormatDiagnostic_NoSourceText tests behavior when source text is not available.
func TestFormatDiagnostic_NoSourceText(t *testing.T) {
	// Arrange
	diag := &diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Message:  "compilation failed",
		SourceSpan: ast.SourceSpan{
			Filename:  "missing.csl",
			StartLine: 10,
			StartCol:  5,
			EndLine:   10,
			EndCol:    15,
		},
	}

	// Act
	formatted := diagnostic.FormatDiagnostic(diag, "", nil)

	// Assert
	// Should still contain basic location info
	if !strings.Contains(formatted, "missing.csl:10:5:") {
		t.Errorf("Expected file:line:col prefix, got:\n%s", formatted)
	}

	// Should contain message
	if !strings.Contains(formatted, "compilation failed") {
		t.Errorf("Expected error message, got:\n%s", formatted)
	}

	// No snippet should be present
	if strings.Contains(formatted, "|") {
		t.Errorf("Should not contain snippet when source text is empty, got:\n%s", formatted)
	}
}
