package testutil

import (
	"context"
	"errors"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// FakeProviderRegistry is a thread-safe test double for compiler.ProviderRegistry.
// It allows registering providers and tracks which providers have been requested.
type FakeProviderRegistry struct {
	mu           sync.RWMutex
	providers    map[string]compiler.Provider
	constructors map[string]compiler.ProviderConstructor
	getError     error // Error to return from GetProvider
}

// NewFakeProviderRegistry creates a new empty FakeProviderRegistry.
func NewFakeProviderRegistry() *FakeProviderRegistry {
	return &FakeProviderRegistry{
		providers:    make(map[string]compiler.Provider),
		constructors: make(map[string]compiler.ProviderConstructor),
	}
}

// Register implements compiler.ProviderRegistry.Register.
func (r *FakeProviderRegistry) Register(alias string, constructor compiler.ProviderConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[alias] = constructor
}

// GetProvider implements compiler.ProviderRegistry.GetProvider.
// It initializes the provider on first access and caches it.
func (r *FakeProviderRegistry) GetProvider(ctx context.Context, alias string) (compiler.Provider, error) {
	r.mu.RLock()
	if r.getError != nil {
		r.mu.RUnlock()
		return nil, r.getError
	}

	// Check if provider is already initialized
	if provider, ok := r.providers[alias]; ok {
		r.mu.RUnlock()
		return provider, nil
	}
	r.mu.RUnlock()

	// Initialize provider
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if provider, ok := r.providers[alias]; ok {
		return provider, nil
	}

	// Look up constructor
	constructor, ok := r.constructors[alias]
	if !ok {
		return nil, errors.New("provider not registered: " + alias)
	}

	// Create provider with empty opts
	opts := compiler.ProviderInitOptions{Config: make(map[string]any)}
	provider, err := constructor(opts)
	if err != nil {
		return nil, err
	}

	// Initialize provider
	if err := provider.Init(ctx, opts); err != nil {
		return nil, err
	}

	// Cache provider
	r.providers[alias] = provider
	return provider, nil
}

// RegisteredAliases implements compiler.ProviderRegistry.RegisteredAliases.
func (r *FakeProviderRegistry) RegisteredAliases() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	aliases := make([]string, 0, len(r.constructors)+len(r.providers))
	for alias := range r.constructors {
		aliases = append(aliases, alias)
	}
	for alias := range r.providers {
		if _, ok := r.constructors[alias]; !ok {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

// AddProvider directly adds a provider to the registry without using a constructor.
// This is useful for tests that want to provide pre-configured fake providers.
func (r *FakeProviderRegistry) AddProvider(alias string, provider compiler.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[alias] = provider
}

// SetGetProviderError configures GetProvider to return the given error.
// This is useful for testing error handling paths.
func (r *FakeProviderRegistry) SetGetProviderError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.getError = err
}

// ProviderCount returns the number of providers currently registered or added.
func (r *FakeProviderRegistry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers) + len(r.constructors)
}
