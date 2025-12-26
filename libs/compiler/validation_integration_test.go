//go:build integration
// +build integration

package compiler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/validator"
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
	_, err := Compile(ctx, opts)

	// Assert: Expect error
	if err == nil {
		t.Fatal("expected error for unresolved reference, got nil")
	}

	// Check if it's an unresolved reference error
	var unresolvedErr *validator.ErrUnresolvedReference
	if !stderrors.As(err, &unresolvedErr) {
		t.Fatalf("expected *validator.ErrUnresolvedReference, got %T: %v", err, err)
	}

	// Verify error details
	if unresolvedErr.Alias != "nonexistent" {
		t.Errorf("expected alias %q, got %q", "nonexistent", unresolvedErr.Alias)
	}

	// Check that error message is user-friendly
	errMsg := err.Error()
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
  value: reference:fil:some/path
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
	_, err := Compile(ctx, opts)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var unresolvedErr *validator.ErrUnresolvedReference
	if !stderrors.As(err, &unresolvedErr) {
		t.Fatalf("expected *validator.ErrUnresolvedReference, got %T", err)
	}

	// Check suggestions
	if len(unresolvedErr.Suggestions) == 0 {
		t.Error("expected suggestions for typo, got none")
	}

	// Error message should include suggestion
	errMsg := err.Error()
	if !contains(errMsg, "did you mean") {
		t.Errorf("error message should include suggestion hint, got: %s", errMsg)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
