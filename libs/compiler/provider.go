package compiler

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// ProviderConstructor is a function that creates a new Provider instance.
type ProviderConstructor func(opts ProviderInitOptions) (Provider, error)

// ProviderRegistry manages provider instances for a compilation run.
type ProviderRegistry interface {
	// Register registers a provider constructor for the given alias.
	// The constructor will be called on-demand when GetProvider is first called
	// for the alias. Subsequent calls to GetProvider for the same alias return
	// the cached provider instance.
	Register(alias string, constructor ProviderConstructor)

	// GetProvider returns a provider for the given alias.
	// Providers are instantiated on demand and cached for the compilation run.
	// Returns ErrProviderNotRegistered if the alias is not registered.
	GetProvider(alias string) (Provider, error)

	// RegisteredAliases returns the list of all registered provider aliases.
	// Used by semantic validation to check for unresolved references.
	RegisteredAliases() []string
}

// providerRegistry is the default implementation of ProviderRegistry.
type providerRegistry struct {
	mu            sync.RWMutex
	constructors  map[string]ProviderConstructor
	instances     map[string]Provider
	instanceMutex sync.Mutex // Separate mutex for instance creation to avoid deadlock
}

// NewProviderRegistry creates a new ProviderRegistry.
func NewProviderRegistry() ProviderRegistry {
	return &providerRegistry{
		constructors: make(map[string]ProviderConstructor),
		instances:    make(map[string]Provider),
	}
}

// Register implements ProviderRegistry.Register.
func (r *providerRegistry) Register(alias string, constructor ProviderConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[alias] = constructor
}

// GetProvider implements ProviderRegistry.GetProvider.
func (r *providerRegistry) GetProvider(alias string) (Provider, error) {
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
	opts := ProviderInitOptions{
		Alias: alias,
	}

	provider, err := constructor(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to construct provider %q: %w", alias, err)
	}

	// Initialize provider
	ctx := context.Background() // TODO: Accept context from caller
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

// Provider defines the interface for external data source adapters.
//
// Providers are responsible for:
//   - Initializing connections/resources via Init
//   - Fetching data by path via Fetch
//   - Optionally exposing metadata via Info
//
// The compiler instantiates providers on demand and caches them for
// the duration of a single compilation run.
type Provider interface {
	// Init initializes the provider with the given options.
	// Called once per compilation run when the provider is first used.
	// Must be called before Fetch.
	Init(ctx context.Context, opts ProviderInitOptions) error

	// Fetch retrieves data from the provider at the specified path.
	// The path is a sequence of components (e.g., ["config", "network", "vpc"]).
	// Returns the resolved value or an error if the fetch fails.
	//
	// Fetch results are cached per compilation run. Subsequent calls with
	// the same path return the cached value without re-fetching.
	Fetch(ctx context.Context, path []string) (any, error)
}

// ProviderWithInfo is an optional interface providers can implement to expose metadata.
type ProviderWithInfo interface {
	Provider
	// Info returns the provider's alias and version for metadata tracking.
	Info() (alias string, version string)
}

// ProviderInitOptions configures a provider during initialization.
type ProviderInitOptions struct {
	// Alias is the provider's registered alias in the ProviderRegistry.
	Alias string

	// Config contains provider-specific configuration (from source declarations).
	Config map[string]any

	// SourceFilePath is the path to the .csl file containing the source declaration.
	// This allows providers to resolve relative paths from the source file's directory.
	SourceFilePath string
}
