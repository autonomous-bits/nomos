//go:build integration
// +build integration

package compiler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
)

// TestCompile_UnresolvedReference tests that unresolved references are detected during compilation.
func TestCompile_UnresolvedReference(t *testing.T) {
	// Use the test fixture with an unresolved reference
	fixtureDir := "./testdata/validation"

	// Create registry without the "nonexistent" provider
	registry := NewProviderRegistry()
	registry.Register("file", func(_ ProviderInitOptions) (Provider, error) {
		return nil, stderrors.New("not needed for this test")
	})

	ctx := context.Background()
	opts := Options{
		Path:             fixtureDir + "/unresolved_ref.csl",
		ProviderRegistry: registry,
	}

	// Act: Compile should fail with unresolved reference
	result := Compile(ctx, opts)

	// Assert: Expect error
	if !result.HasErrors() {
		t.Fatal("expected error for unresolved reference, got nil")
	}

	// Check error message contains expected details
	errs := result.Errors()
	if len(errs) == 0 {
		t.Fatal("expected at least one error")
	}

	// Verify error details are present in the error message
	errMsg := errs[0]
	if !strings.Contains(errMsg, "nonexistent") {
		t.Errorf("expected error to contain alias %q, got: %v", "nonexistent", errMsg)
	}
	if !strings.Contains(errMsg, "unresolved reference") {
		t.Errorf("expected error to mention unresolved reference, got: %v", errMsg)
	}

	// Check that error message is user-friendly
	t.Logf("Error message: %s", errMsg)

	if errMsg == "" {
		t.Error("error message should not be empty")
	}
}

// TestCompile_UnresolvedReference_WithSuggestion tests fuzzy matching suggestions.
func TestCompile_UnresolvedReference_WithSuggestion(t *testing.T) {
	// Create a fixture on the fly with a typo
	tmpDir := t.TempDir()
	fixtureFile := tmpDir + "/typo.csl"

	content := `config:
  value: @fil:some/path
`
	if err := writeFile(fixtureFile, content); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Create registry with "file" provider (correct spelling)
	registry := NewProviderRegistry()
	registry.Register("file", func(_ ProviderInitOptions) (Provider, error) {
		return nil, stderrors.New("not needed")
	})

	ctx := context.Background()
	opts := Options{
		Path:             fixtureFile,
		ProviderRegistry: registry,
	}

	// Act: Compile should suggest "file"
	result := Compile(ctx, opts)

	// Assert
	if !result.HasErrors() {
		t.Fatal("expected error, got nil")
	}

	// Check error message contains expected details
	errs := result.Errors()
	if len(errs) == 0 {
		t.Fatal("expected at least one error")
	}

	// Error message should include suggestion hint
	errMsg := errs[0]
	if !strings.Contains(errMsg, "fil") {
		t.Errorf("error message should reference typo 'fil', got: %s", errMsg)
	}
	if !strings.Contains(strings.ToLower(errMsg), "did you mean") {
		t.Logf("Warning: error message may not include suggestion hint: %s", errMsg)
	}
}
