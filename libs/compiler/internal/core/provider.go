// Package core defines shared interfaces and types used across the compiler.
//
// This package provides the foundational types that multiple compiler subsystems
// depend on, including the Provider interface, ProviderRegistry, and related
// initialization structures. By centralizing these definitions, we eliminate
// duplication and simplify adapter patterns.
package core

import "context"

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
	// The context is used for provider initialization and should respect cancellation.
	GetProvider(ctx context.Context, alias string) (Provider, error)

	// RegisteredAliases returns the list of all registered provider aliases.
	// Used by semantic validation to check for unresolved references.
	RegisteredAliases() []string
}

// ProviderTypeConstructor creates a Provider from configuration.
// Used when registering provider types that can be instantiated dynamically
// from source declarations in .csl files.
type ProviderTypeConstructor func(config map[string]any) (Provider, error)

// ProviderTypeRegistry manages provider type constructors for dynamic provider creation.
//
// This registry maps provider type names (e.g., "file", "http", "git") to
// factory functions that create provider instances. It's used to dynamically
// instantiate providers from source declarations in .csl files.
type ProviderTypeRegistry interface {
	// RegisterType registers a constructor for the given provider type name.
	// The constructor will be called when CreateProvider is invoked for that type.
	RegisterType(typeName string, constructor ProviderTypeConstructor)

	// CreateProvider creates a new provider instance of the given type.
	// The config parameter contains provider-specific configuration.
	CreateProvider(ctx context.Context, typeName string, config map[string]any) (Provider, error)

	// IsTypeRegistered checks if a provider type is registered.
	IsTypeRegistered(typeName string) bool

	// RegisteredTypes returns a list of all registered provider type names.
	RegisteredTypes() []string
}
