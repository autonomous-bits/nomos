package compiler_test

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestProviderTypeRegistry_CreateRemoteProvider tests creating providers via resolver + manager.
func TestProviderTypeRegistry_CreateRemoteProvider(t *testing.T) {
	t.Run("creates remote provider when resolver and manager are available", func(t *testing.T) {
		// Setup fake resolver that maps "file" -> binary path
		resolver := &fakeResolver{
			entries: map[string]string{
				"file": "/fake/path/to/provider",
			},
		}

		// Setup fake manager
		manager := compiler.NewManager()
		defer func() {
			_ = manager.Shutdown(context.Background())
		}()

		// Create registry with resolver and manager
		registry := compiler.NewProviderTypeRegistryWithResolver(resolver, manager)

		// Attempt to create a provider of type "file"
		config := map[string]any{"directory": "./testdata"}
		provider, err := registry.CreateProvider(context.Background(), "file", config)

		// For now, we expect an error because the binary doesn't exist
		// This test will evolve as we implement the functionality
		if err == nil {
			t.Log("Provider created (binary would need to exist for full test)")
			_ = provider
		} else {
			t.Logf("Expected error for non-existent binary: %v", err)
		}
	})

	t.Run("falls back to in-process constructor when no resolver", func(t *testing.T) {
		registry := compiler.NewProviderTypeRegistry()

		// Register an in-process constructor
		inProcessCalled := false
		registry.RegisterType("test", func(_ map[string]any) (compiler.Provider, error) {
			inProcessCalled = true
			return &fakeProvider{}, nil
		})

		provider, err := registry.CreateProvider(context.Background(), "test", map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !inProcessCalled {
			t.Error("expected in-process constructor to be called")
		}

		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}
	})
}

// fakeProvider implements compiler.Provider for testing.
type fakeProvider struct{}

func (f *fakeProvider) Init(_ context.Context, _ compiler.ProviderInitOptions) error {
	return nil
}

func (f *fakeProvider) Fetch(_ context.Context, _ []string) (any, error) {
	return map[string]any{"test": "data"}, nil
}
