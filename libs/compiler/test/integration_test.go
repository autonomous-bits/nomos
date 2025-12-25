package test

import (
	"context"
	"errors"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// fakeProviderRegistry is a minimal implementation for smoke tests.
type fakeProviderRegistry struct {
	aliases []string
}

func (f *fakeProviderRegistry) Register(alias string, _ compiler.ProviderConstructor) {
	// No-op for smoke tests
	f.aliases = append(f.aliases, alias)
}

func (f *fakeProviderRegistry) GetProvider(_ context.Context, _ string) (compiler.Provider, error) {
	return nil, errors.New("no providers registered")
}

func (f *fakeProviderRegistry) RegisteredAliases() []string {
	return f.aliases
}

// TestIntegration_SmokeTest verifies Compile works on an empty directory.
func TestIntegration_SmokeTest(t *testing.T) {
	// Create empty temp directory
	tmpDir := t.TempDir()

	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: &fakeProviderRegistry{},
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify snapshot structure
	if snapshot.Data == nil {
		t.Error("snapshot.Data should not be nil")
	}

	if snapshot.Metadata.InputFiles == nil {
		t.Error("snapshot.Metadata.InputFiles should not be nil")
	}

	if len(snapshot.Metadata.InputFiles) != 0 {
		t.Errorf("expected 0 input files for empty directory, got %d", len(snapshot.Metadata.InputFiles))
	}

	if snapshot.Metadata.StartTime.IsZero() {
		t.Error("snapshot.Metadata.StartTime should be set")
	}

	if snapshot.Metadata.EndTime.IsZero() {
		t.Error("snapshot.Metadata.EndTime should be set")
	}

	if snapshot.Metadata.PerKeyProvenance == nil {
		t.Error("snapshot.Metadata.PerKeyProvenance should not be nil")
	}
}
