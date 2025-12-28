package compiler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
)

// Sentinel errors for provider and compilation failures.
var (
	// ErrUnresolvedReference indicates a reference could not be resolved.
	ErrUnresolvedReference = errors.New("unresolved reference")

	// ErrCycleDetected indicates a cycle was detected in imports or references.
	ErrCycleDetected = errors.New("cycle detected")

	// ErrProviderNotRegistered indicates a provider alias is not registered.
	ErrProviderNotRegistered = errors.New("provider not registered")
)

// Type aliases for backward compatibility with existing code.
// These aliases allow external code to continue using compiler.Provider, etc.
// while internally using the core package's definitions.
type (
	// Provider is the main provider interface.
	Provider = core.Provider
	// ProviderWithInfo extends Provider with metadata.
	ProviderWithInfo = core.ProviderWithInfo
	// ProviderInitOptions configures provider initialization.
	ProviderInitOptions = core.ProviderInitOptions
	// ProviderConstructor creates provider instances.
	ProviderConstructor = core.ProviderConstructor
	// ProviderRegistry manages provider instances.
	ProviderRegistry = core.ProviderRegistry
	// ProviderTypeRegistry manages provider type constructors.
	ProviderTypeRegistry = core.ProviderTypeRegistry
)

// providerRegistry is the default implementation of ProviderRegistry.
type providerRegistry struct {
	mu            sync.RWMutex
	constructors  map[string]core.ProviderConstructor
	instances     map[string]core.Provider
	instanceMutex sync.Mutex // Separate mutex for instance creation to avoid deadlock
}

// NewProviderRegistry creates a new ProviderRegistry.
func NewProviderRegistry() core.ProviderRegistry {
	return &providerRegistry{
		constructors: make(map[string]core.ProviderConstructor),
		instances:    make(map[string]core.Provider),
	}
}

// Register implements ProviderRegistry.Register.
func (r *providerRegistry) Register(alias string, constructor core.ProviderConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[alias] = constructor
}

// GetProvider implements ProviderRegistry.GetProvider.
func (r *providerRegistry) GetProvider(ctx context.Context, alias string) (core.Provider, error) {
	// Check if instance already exists (fast path with read lock)
	r.instanceMutex.Lock()
	if instance, ok := r.instances[alias]; ok {
		r.instanceMutex.Unlock()
		return instance, nil
	}
	r.instanceMutex.Unlock()

	// Get constructor (read lock for constructors)
	r.mu.RLock()
	constructor, ok := r.constructors[alias]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, alias)
	}

	// Create instance (hold instance lock to ensure single initialization)
	r.instanceMutex.Lock()
	defer r.instanceMutex.Unlock()

	// Double-check instance doesn't exist (another goroutine may have created it)
	if instance, ok := r.instances[alias]; ok {
		return instance, nil
	}

	// Construct provider
	opts := core.ProviderInitOptions{
		Alias: alias,
	}

	provider, err := constructor(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to construct provider %q: %w", alias, err)
	}

	// Initialize provider with context from caller
	if err := provider.Init(ctx, opts); err != nil {
		return nil, fmt.Errorf("failed to initialize provider %q: %w", alias, err)
	}

	// Cache instance
	r.instances[alias] = provider

	return provider, nil
}

// RegisteredAliases implements ProviderRegistry.RegisteredAliases.
func (r *providerRegistry) RegisteredAliases() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	aliases := make([]string, 0, len(r.constructors))
	for alias := range r.constructors {
		aliases = append(aliases, alias)
	}

	return aliases
}
