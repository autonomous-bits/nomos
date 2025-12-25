package compiler_test

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestProviderResolver_ResolveBinaryPath tests the resolver interface contract.
func TestProviderResolver_ResolveBinaryPath(t *testing.T) {
	t.Run("resolves provider type to binary path", func(t *testing.T) {
		resolver := &fakeResolver{
			entries: map[string]string{
				"file": "/path/to/providers/file/0.1.0/darwin-arm64/provider",
			},
		}

		path, err := resolver.ResolveBinaryPath(context.Background(), "file")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "/path/to/providers/file/0.1.0/darwin-arm64/provider"
		if path != expected {
			t.Errorf("expected path %s, got %s", expected, path)
		}
	})

	t.Run("returns error for unknown provider type", func(t *testing.T) {
		resolver := &fakeResolver{
			entries: map[string]string{},
		}

		_, err := resolver.ResolveBinaryPath(context.Background(), "unknown")
		if err == nil {
			t.Fatal("expected error for unknown provider, got nil")
		}
	})
}

// fakeResolver implements ProviderResolver for testing.
type fakeResolver struct {
	entries map[string]string
}

func (f *fakeResolver) ResolveBinaryPath(_ context.Context, providerType string) (string, error) {
	path, ok := f.entries[providerType]
	if !ok {
		return "", compiler.ErrProviderNotRegistered
	}
	return path, nil
}
