package test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
)

// TestProviderRegistry_ConcurrentAccess tests thread-safety of provider registry.
// This test uses the race detector to catch concurrent access issues.
// Run with: go test -race ./test
func TestProviderRegistry_ConcurrentAccess(t *testing.T) {
	// Create a provider registry with a fake provider
	provider := fakes.NewFakeProvider("test")
	provider.FetchResponses["config/key"] = "value"

	registry := &concurrencyTestRegistry{
		providers: make(map[string]compiler.Provider),
	}

	// Register provider from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			alias := fmt.Sprintf("provider-%d", id)
			p := fakes.NewFakeProvider(alias)
			registry.registerConcurrent(alias, p)
		}(i)
	}

	wg.Wait()

	// Verify all providers were registered
	aliases := registry.RegisteredAliases()
	if len(aliases) != numGoroutines {
		t.Errorf("expected %d providers, got %d", numGoroutines, len(aliases))
	}
}

// TestProviderRegistry_ConcurrentGetProvider tests concurrent provider retrieval.
func TestProviderRegistry_ConcurrentGetProvider(t *testing.T) {
	provider := fakes.NewFakeProvider("test")
	provider.FetchResponses["config/key"] = "value"

	registry := &concurrencyTestRegistry{
		providers: map[string]compiler.Provider{
			"test": provider,
		},
	}

	// Get provider from multiple goroutines concurrently
	var wg sync.WaitGroup
	numGoroutines := 100
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			p, err := registry.GetProvider(context.Background(), "test")
			if err != nil {
				errors <- err
				return
			}

			if p == nil {
				errors <- fmt.Errorf("got nil provider")
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent GetProvider error: %v", err)
	}
}

// TestProvider_ConcurrentFetch tests thread-safety of provider fetch operations.
// This validates that provider caching and fetch are safe for concurrent use.
func TestProvider_ConcurrentFetch(t *testing.T) {
	ctx := context.Background()
	provider := fakes.NewFakeProvider("test")

	// Configure responses for multiple paths
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("config/key%d", i)
		provider.FetchResponses[path] = fmt.Sprintf("value%d", i)
	}

	// Initialize provider
	if err := provider.Init(ctx, compiler.ProviderInitOptions{Alias: "test"}); err != nil {
		t.Fatalf("failed to initialize provider: %v", err)
	}

	// Fetch from multiple goroutines concurrently
	var wg sync.WaitGroup
	numGoroutines := 50
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine fetches multiple paths
			for j := 0; j < 10; j++ {
				path := []string{"config", fmt.Sprintf("key%d", j%10)}
				_, err := provider.Fetch(ctx, path)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: %w", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent fetch error: %v", err)
	}

	// Verify fetch count is correct
	// Should be 50 goroutines * 10 fetches = 500 total
	expectedFetchCount := numGoroutines * 10
	if provider.FetchCount != expectedFetchCount {
		t.Errorf("expected %d fetches, got %d", expectedFetchCount, provider.FetchCount)
	}
}

// TestProvider_ConcurrentInit tests that concurrent Init calls are safe.
func TestProvider_ConcurrentInit(t *testing.T) {
	ctx := context.Background()
	provider := fakes.NewFakeProvider("test")

	// Initialize from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 20
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			opts := compiler.ProviderInitOptions{
				Alias: fmt.Sprintf("test-%d", id),
			}

			if err := provider.Init(ctx, opts); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent init error: %v", err)
	}

	// Verify init was called the correct number of times
	if provider.InitCount != numGoroutines {
		t.Errorf("expected %d init calls, got %d", numGoroutines, provider.InitCount)
	}
}

// TestCompile_ConcurrentCalls tests that multiple concurrent Compile calls work correctly.
// This is a higher-level integration test for concurrency.
func TestCompile_ConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create provider registry
	provider := fakes.NewFakeProvider("test")
	provider.FetchResponses["config/db/host"] = "localhost"
	provider.FetchResponses["config/db/port"] = 5432

	// Run multiple compilations concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine gets its own registry instance
			registry := &concurrencyTestRegistry{
				providers: map[string]compiler.Provider{
					"test": provider,
				},
			}

			opts := compiler.Options{
				Path:             tmpDir,
				ProviderRegistry: registry,
			}

			_, err := compiler.Compile(ctx, opts)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent compile error: %v", err)
	}
}

// concurrencyTestRegistry is a thread-safe provider registry for testing.
type concurrencyTestRegistry struct {
	mu        sync.RWMutex
	providers map[string]compiler.Provider
}

func (r *concurrencyTestRegistry) Register(alias string, constructor compiler.ProviderConstructor) {
	// Not used in concurrency tests
}

func (r *concurrencyTestRegistry) registerConcurrent(alias string, provider compiler.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[alias] = provider
}

func (r *concurrencyTestRegistry) GetProvider(ctx context.Context, alias string) (compiler.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.providers[alias]; ok {
		return p, nil
	}
	return nil, compiler.ErrProviderNotRegistered
}

func (r *concurrencyTestRegistry) RegisteredAliases() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	aliases := make([]string, 0, len(r.providers))
	for alias := range r.providers {
		aliases = append(aliases, alias)
	}
	return aliases
}
