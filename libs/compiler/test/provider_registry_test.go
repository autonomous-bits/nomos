package test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// mockProvider is a simple test double for testing provider registry.
// This avoids import cycles by not depending on test/fakes.
type mockProvider struct {
	alias       string
	version     string
	initError   error
	fetchError  error
	InitCount   int // Exported for tests
	FetchCount  int // Exported for tests
	fetchResult any
}

func newMockProvider(alias string) *mockProvider {
	return &mockProvider{
		alias:   alias,
		version: "test-v1.0.0",
	}
}

func (m *mockProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
	m.InitCount++
	if m.initError != nil {
		return m.initError
	}
	return nil
}

func (m *mockProvider) Fetch(ctx context.Context, path []string) (any, error) {
	m.FetchCount++
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	return m.fetchResult, nil
}

func (m *mockProvider) Info() (alias string, version string) {
	return m.alias, m.version
}

// TestProviderRegistry_Register tests provider constructor registration.
func TestProviderRegistry_Register(t *testing.T) {
	t.Run("register and retrieve provider", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()
		alias := "test-provider"

		constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
			return newMockProvider(alias), nil
		}

		// Act
		registry.Register(alias, constructor)

		// Assert - Get should instantiate the provider
		provider, err := registry.GetProvider(context.Background(), alias)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if provider == nil {
			t.Fatal("expected provider, got nil")
		}

		// Verify it's the right provider by checking Info
		if infoProvider, ok := provider.(compiler.ProviderWithInfo); ok {
			gotAlias, _ := infoProvider.Info()
			if gotAlias != alias {
				t.Errorf("expected alias %q, got %q", alias, gotAlias)
			}
		} else {
			t.Error("provider should implement compiler.ProviderWithInfo")
		}
	})

	t.Run("get unregistered provider returns error", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()

		// Act
		ctx := context.Background()
		provider, err := registry.GetProvider(ctx, "nonexistent")

		// Assert
		if err == nil {
			t.Fatal("expected error for unregistered provider, got nil")
		}

		if provider != nil {
			t.Errorf("expected nil provider, got %v", provider)
		}
	})

	t.Run("constructor error is propagated", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()
		alias := "failing-provider"
		expectedErr := errors.New("constructor failed")

		constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
			return nil, expectedErr
		}

		registry.Register(alias, constructor)

		// Act
		ctx := context.Background()
		provider, err := registry.GetProvider(ctx, alias)

		// Assert
		if err == nil {
			t.Fatal("expected error from constructor, got nil")
		}

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error to wrap constructor error, got %v", err)
		}

		if provider != nil {
			t.Errorf("expected nil provider on constructor error, got %v", provider)
		}
	})
}

// TestProviderRegistry_InstanceCaching tests that providers are cached per run.
func TestProviderRegistry_InstanceCaching(t *testing.T) {
	t.Run("same instance returned for same alias", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()
		alias := "cached-provider"

		constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
			return &mockProvider{alias: alias, version: "test-v1.0.0"}, nil
		}

		registry.Register(alias, constructor)

		// Act - Get the provider twice
		ctx := context.Background()
		provider1, err1 := registry.GetProvider(ctx, alias)
		provider2, err2 := registry.GetProvider(ctx, alias)

		// Assert
		if err1 != nil {
			t.Fatalf("first GetProvider failed: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("second GetProvider failed: %v", err2)
		}

		// Verify they are the same instance
		if provider1 != provider2 {
			t.Error("expected same provider instance on subsequent calls")
		}
	})

	t.Run("Init called only once", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()
		alias := "init-once-provider"
		mock := newMockProvider(alias)

		constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
			return mock, nil
		}

		registry.Register(alias, constructor)

		// Act - Get the provider multiple times
		for i := 0; i < 3; i++ {
			_, err := registry.GetProvider(context.Background(), alias)
			if err != nil {
				t.Fatalf("GetProvider call %d failed: %v", i+1, err)
			}
		}

		// Assert - Init should only be called once
		if mock.InitCount != 1 {
			t.Errorf("expected Init to be called once, got %d calls", mock.InitCount)
		}
	})
}

// TestProviderRegistry_ConcurrentRegistration tests thread-safety of provider registration.
func TestProviderRegistry_ConcurrentRegistration(t *testing.T) {
	t.Run("concurrent registration is safe", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()

		// Act - Register providers concurrently
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			i := i // Capture loop variable
			go func() {
				alias := fmt.Sprintf("provider-%d", i)
				constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
					return &mockProvider{alias: alias, version: "test-v1.0.0"}, nil
				}
				registry.Register(alias, constructor)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Assert - All providers should be retrievable
		for i := 0; i < 10; i++ {
			alias := fmt.Sprintf("provider-%d", i)
			provider, err := registry.GetProvider(context.Background(), alias)
			if err != nil {
				t.Errorf("failed to get provider %q: %v", alias, err)
			}
			if provider == nil {
				t.Errorf("expected provider %q, got nil", alias)
			}
		}
	})

	t.Run("concurrent Get is safe", func(t *testing.T) {
		// Arrange
		registry := compiler.NewProviderRegistry()
		alias := "concurrent-get-provider"
		mock := newMockProvider(alias)

		constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
			return mock, nil
		}

		registry.Register(alias, constructor)

		// Act - Get provider concurrently
		done := make(chan compiler.Provider, 10)
		for i := 0; i < 10; i++ {
			go func() {
				provider, err := registry.GetProvider(context.Background(), alias)
				if err != nil {
					t.Errorf("concurrent Get failed: %v", err)
				}
				done <- provider
			}()
		}

		// Collect results
		providers := make([]compiler.Provider, 10)
		for i := 0; i < 10; i++ {
			providers[i] = <-done
		}

		// Assert - All should be the same instance
		for i := 1; i < len(providers); i++ {
			if providers[i] != providers[0] {
				t.Error("concurrent Get calls returned different instances")
			}
		}

		// Init should still only be called once
		if mock.InitCount != 1 {
			t.Errorf("expected Init to be called once despite concurrent access, got %d calls", mock.InitCount)
		}
	})
}
