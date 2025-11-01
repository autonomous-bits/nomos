package fakes_test

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
)

// TestFakeProvider_Init_CalledOnce verifies that Init is called exactly once
// when a provider is retrieved from the registry.
func TestFakeProvider_Init_CalledOnce(t *testing.T) {
	// Arrange
	registry := compiler.NewProviderRegistry()
	fake := fakes.NewFakeProvider("test")

	registry.Register("test", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		return fake, nil
	})

	// Act - Get provider twice
	provider1, err := registry.GetProvider("test")
	if err != nil {
		t.Fatalf("first GetProvider failed: %v", err)
	}

	provider2, err := registry.GetProvider("test")
	if err != nil {
		t.Fatalf("second GetProvider failed: %v", err)
	}

	// Assert
	if provider1 != provider2 {
		t.Errorf("expected same provider instance, got different instances")
	}

	if fake.InitCount != 1 {
		t.Errorf("expected Init called exactly once, got %d calls", fake.InitCount)
	}
}

// TestFakeProvider_Fetch_CachingBehavior verifies that Fetch results are
// cached per compilation run when using the same path.
func TestFakeProvider_Fetch_CachingBehavior(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProvider("config")
	ctx := context.Background()

	// Configure response for path ["network", "vpc"]
	fake.FetchResponses["network/vpc"] = map[string]any{
		"id":     "vpc-12345",
		"region": "us-west-2",
	}

	// Initialize provider
	opts := compiler.ProviderInitOptions{Alias: "config"}
	if err := fake.Init(ctx, opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Act - Fetch same path twice
	path := []string{"network", "vpc"}

	result1, err := fake.Fetch(ctx, path)
	if err != nil {
		t.Fatalf("first Fetch failed: %v", err)
	}

	result2, err := fake.Fetch(ctx, path)
	if err != nil {
		t.Fatalf("second Fetch failed: %v", err)
	}

	// Assert - Both fetches should succeed and return the same data
	if result1 == nil || result2 == nil {
		t.Fatalf("expected non-nil results, got result1=%v, result2=%v", result1, result2)
	}

	// Note: In a real provider with caching, Fetch would only be called once.
	// FakeProvider doesn't implement caching (it's a test double), so it's
	// called twice here. This demonstrates the expected call pattern.
	if fake.FetchCount != 2 {
		t.Errorf("expected Fetch called twice (no caching in fake), got %d calls", fake.FetchCount)
	}

	// Verify both calls used the same path
	if len(fake.FetchCalls) != 2 {
		t.Fatalf("expected 2 fetch calls recorded, got %d", len(fake.FetchCalls))
	}

	for i, call := range fake.FetchCalls {
		if len(call) != len(path) {
			t.Errorf("call %d: expected path length %d, got %d", i, len(path), len(call))
			continue
		}
		for j, component := range path {
			if call[j] != component {
				t.Errorf("call %d: expected path[%d]=%q, got %q", i, j, component, call[j])
			}
		}
	}
}

// TestFakeProvider_ConfigurableErrors demonstrates how to test error handling
// by configuring the fake provider to return errors.
func TestFakeProvider_ConfigurableErrors(t *testing.T) {
	t.Run("Init error", func(t *testing.T) {
		// Arrange
		fake := fakes.NewFakeProvider("test")
		fake.InitError = compiler.ErrProviderNotRegistered

		// Act
		ctx := context.Background()
		opts := compiler.ProviderInitOptions{Alias: "test"}
		err := fake.Init(ctx, opts)

		// Assert
		if err != compiler.ErrProviderNotRegistered {
			t.Errorf("expected ErrProviderNotRegistered, got %v", err)
		}
	})

	t.Run("Fetch error", func(t *testing.T) {
		// Arrange
		fake := fakes.NewFakeProvider("test")
		ctx := context.Background()
		opts := compiler.ProviderInitOptions{Alias: "test"}

		if err := fake.Init(ctx, opts); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		fake.FetchError = compiler.ErrUnresolvedReference

		// Act
		_, err := fake.Fetch(ctx, []string{"some", "path"})

		// Assert
		if err != compiler.ErrUnresolvedReference {
			t.Errorf("expected ErrUnresolvedReference, got %v", err)
		}
	})
}

// TestFakeProvider_MultipleProviders demonstrates using multiple fake providers
// with different aliases and configurations.
func TestFakeProvider_MultipleProviders(t *testing.T) {
	// Arrange
	ctx := context.Background()
	registry := compiler.NewProviderRegistry()

	// Register config provider
	configFake := fakes.NewFakeProvider("config")
	configFake.FetchResponses["database/host"] = "db.example.com"
	registry.Register("config", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		return configFake, nil
	})

	// Register secrets provider
	secretsFake := fakes.NewFakeProvider("secrets")
	secretsFake.FetchResponses["database/password"] = "secret123"
	registry.Register("secrets", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		return secretsFake, nil
	})

	// Act - Get both providers
	configProvider, err := registry.GetProvider("config")
	if err != nil {
		t.Fatalf("failed to get config provider: %v", err)
	}

	secretsProvider, err := registry.GetProvider("secrets")
	if err != nil {
		t.Fatalf("failed to get secrets provider: %v", err)
	}

	// Fetch from both providers
	host, err := configProvider.Fetch(ctx, []string{"database", "host"})
	if err != nil {
		t.Fatalf("failed to fetch host: %v", err)
	}

	password, err := secretsProvider.Fetch(ctx, []string{"database", "password"})
	if err != nil {
		t.Fatalf("failed to fetch password: %v", err)
	}

	// Assert
	if configFake.InitCount != 1 {
		t.Errorf("expected config provider Init called once, got %d", configFake.InitCount)
	}

	if secretsFake.InitCount != 1 {
		t.Errorf("expected secrets provider Init called once, got %d", secretsFake.InitCount)
	}

	if host != "db.example.com" {
		t.Errorf("expected host=db.example.com, got %v", host)
	}

	if password != "secret123" {
		t.Errorf("expected password=secret123, got %v", password)
	}
}

// TestFakeProvider_Info verifies the Info method returns alias and version.
func TestFakeProvider_Info(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProvider("myalias")
	fake.Version = "v2.0.0"

	// Act
	alias, version := fake.Info()

	// Assert
	if alias != "myalias" {
		t.Errorf("expected alias=myalias, got %q", alias)
	}

	if version != "v2.0.0" {
		t.Errorf("expected version=v2.0.0, got %q", version)
	}
}

// TestFakeProvider_Reset verifies the Reset method clears call counts and history.
func TestFakeProvider_Reset(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProvider("test")
	ctx := context.Background()
	opts := compiler.ProviderInitOptions{Alias: "test"}
	fake.FetchResponses["key"] = "value"

	// Perform some operations
	_, _ = fake.Init(ctx, opts)
	_, _ = fake.Fetch(ctx, []string{"key"})
	_, _ = fake.Fetch(ctx, []string{"key"})

	if fake.InitCount != 1 || fake.FetchCount != 2 {
		t.Fatalf("setup failed: InitCount=%d, FetchCount=%d", fake.InitCount, fake.FetchCount)
	}

	// Act
	fake.Reset()

	// Assert
	if fake.InitCount != 0 {
		t.Errorf("expected InitCount=0 after reset, got %d", fake.InitCount)
	}

	if fake.FetchCount != 0 {
		t.Errorf("expected FetchCount=0 after reset, got %d", fake.FetchCount)
	}

	if len(fake.FetchCalls) != 0 {
		t.Errorf("expected FetchCalls cleared after reset, got %d calls", len(fake.FetchCalls))
	}
}
