package test

import (
	"context"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
)

// fakeResolver implements compiler.ProviderResolver for testing.
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

// TestProviderTypeRegistry_RemoteProviderIntegration tests the full flow of
// creating a remote provider via resolver + manager.
func TestProviderTypeRegistry_RemoteProviderIntegration(t *testing.T) {
	t.Run("creates and uses remote provider through type registry", func(t *testing.T) {
		resolver := &fakeResolver{
			entries: map[string]string{
				"testprovider": "/fake/binary/path",
			},
		}

		manager := compiler.NewManager()
		defer func() {
			_ = manager.Shutdown(context.Background())
		}()

		typeRegistry := compiler.NewProviderTypeRegistryWithResolver(resolver, manager)

		config := map[string]any{"test": "config"}
		_, err := typeRegistry.CreateProvider(context.Background(), "testprovider", config)

		if err == nil {
			t.Fatal("expected error for non-existent binary, got nil")
		}

		if !strings.Contains(err.Error(), "failed to start remote provider") {
			t.Errorf("expected error containing 'failed to start remote provider', got: %v", err)
		}
	})

	t.Run("prefers in-process constructor over remote provider", func(t *testing.T) {
		resolver := &fakeResolver{
			entries: map[string]string{
				"testprovider": "/fake/binary/path",
			},
		}

		manager := compiler.NewManager()
		defer func() {
			_ = manager.Shutdown(context.Background())
		}()

		typeRegistry := compiler.NewProviderTypeRegistryWithResolver(resolver, manager)

		inProcessCalled := false
		typeRegistry.RegisterType("testprovider", func(_ map[string]any) (compiler.Provider, error) {
			inProcessCalled = true
			return fakes.NewFakeProvider("testprovider"), nil
		})

		config := map[string]any{"test": "config"}
		provider, err := typeRegistry.CreateProvider(context.Background(), "testprovider", config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !inProcessCalled {
			t.Error("expected in-process constructor to be called")
		}

		if provider == nil {
			t.Fatal("expected provider to be non-nil")
		}

		err = provider.Init(context.Background(), compiler.ProviderInitOptions{
			Alias:  "testprovider",
			Config: config,
		})
		if err != nil {
			t.Fatalf("provider init failed: %v", err)
		}
	})

	t.Run("resolver not found returns appropriate error", func(t *testing.T) {
		resolver := &fakeResolver{
			entries: map[string]string{},
		}

		manager := compiler.NewManager()
		defer func() {
			_ = manager.Shutdown(context.Background())
		}()

		typeRegistry := compiler.NewProviderTypeRegistryWithResolver(resolver, manager)

		_, err := typeRegistry.CreateProvider(context.Background(), "missing", map[string]any{})

		if err == nil {
			t.Fatal("expected error for missing provider, got nil")
		}

		hasExpectedError := strings.Contains(err.Error(), "failed to resolve") ||
			strings.Contains(err.Error(), "not registered")
		if !hasExpectedError {
			t.Errorf("expected 'failed to resolve' or 'not registered' in error, got: %v", err)
		}
	})
}
