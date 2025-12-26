//go:build integration
// +build integration

package test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/testutil"
)

// TestParserIntegration_ValidFile tests compiling a valid .csl file.
func TestParserIntegration_ValidFile(t *testing.T) {
	// Arrange
	goodPath := filepath.Join("..", "testdata", "parser_integration", "good.csl")

	// Create fake provider registry
	fakeRegistry := testutil.NewFakeProviderRegistry()

	opts := compiler.Options{
		Path:             goodPath,
		ProviderRegistry: fakeRegistry,
	}

	// Act
	result := compiler.Compile(context.Background(), opts)

	// Assert
	if result.HasErrors() {
		t.Fatalf("expected no error, got %v", result.Error())
	}

	snapshot := result.Snapshot

	// Should have no errors in metadata
	if len(snapshot.Metadata.Errors) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(snapshot.Metadata.Errors), snapshot.Metadata.Errors)
	}

	// Should have 1 input file
	if len(snapshot.Metadata.InputFiles) != 1 {
		t.Errorf("expected 1 input file, got %d", len(snapshot.Metadata.InputFiles))
	}
}

// TestParserIntegration_InvalidFile tests compiling a file with syntax errors.
func TestParserIntegration_InvalidFile(t *testing.T) {
	// Arrange
	badPath := filepath.Join("..", "testdata", "parser_integration", "bad.csl")

	// Create fake provider registry
	fakeRegistry := testutil.NewFakeProviderRegistry()

	opts := compiler.Options{
		Path:             badPath,
		ProviderRegistry: fakeRegistry,
	}

	// Act
	result := compiler.Compile(context.Background(), opts)

	// Assert - can have errors collected in result without fatal error
	snapshot := result.Snapshot

	// Should have errors in metadata
	if len(snapshot.Metadata.Errors) == 0 {
		t.Error("expected errors in Metadata.Errors")
	}

	// Verify formatted error contains caret snippet
	if len(snapshot.Metadata.Errors) > 0 {
		errorMsg := snapshot.Metadata.Errors[0]
		if !strings.Contains(errorMsg, "|") {
			t.Errorf("expected error to contain line prefix '|', got: %s", errorMsg)
		}
		if !strings.Contains(errorMsg, "^") {
			t.Errorf("expected error to contain caret '^', got: %s", errorMsg)
		}
		if !strings.Contains(errorMsg, "bad.csl") {
			t.Errorf("expected error to reference bad.csl, got: %s", errorMsg)
		}
	}

	// Should still have 1 input file
	if len(snapshot.Metadata.InputFiles) != 1 {
		t.Errorf("expected 1 input file, got %d", len(snapshot.Metadata.InputFiles))
	}
}

// TestParserIntegration_MultipleFiles tests compiling multiple files with mixed validity.
func TestParserIntegration_MultipleFiles(t *testing.T) {
	// Arrange - use the directory containing both good and bad files
	dirPath := filepath.Join("..", "testdata", "parser_integration")

	// Create fake provider registry
	fakeRegistry := testutil.NewFakeProviderRegistry()

	opts := compiler.Options{
		Path:             dirPath,
		ProviderRegistry: fakeRegistry,
	}

	// Act
	result := compiler.Compile(context.Background(), opts)

	// Assert - can have errors collected in result without fatal error
	snapshot := result.Snapshot

	// Should have 2 input files
	if len(snapshot.Metadata.InputFiles) != 2 {
		t.Errorf("expected 2 input files, got %d", len(snapshot.Metadata.InputFiles))
	}

	// Should have at least one error from bad.csl
	if len(snapshot.Metadata.Errors) == 0 {
		t.Error("expected at least one error from bad.csl")
	}

	// Verify files are in lexicographic order
	if len(snapshot.Metadata.InputFiles) == 2 {
		file1 := filepath.Base(snapshot.Metadata.InputFiles[0])
		file2 := filepath.Base(snapshot.Metadata.InputFiles[1])
		if file1 > file2 {
			t.Errorf("expected files in lexicographic order, got %s before %s", file1, file2)
		}
	}
}
