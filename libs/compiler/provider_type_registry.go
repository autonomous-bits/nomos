package compiler

import (
	"context"
	"fmt"
	"sync"
)

// ProviderTypeConstructor creates a Provider from configuration.
// Unlike ProviderConstructor which is called during GetProvider, this is called
// when processing source declarations in .csl files.
type ProviderTypeConstructor func(config map[string]any) (Provider, error)

// ProviderTypeRegistry manages provider type constructors.
// This allows creating providers dynamically from source declarations in .csl files.
// Provider types use owner/repo format for proper namespacing (e.g., "autonomous-bits/nomos-provider-file").
type ProviderTypeRegistry interface {
	// RegisterType registers a provider type constructor.
	// Example: RegisterType("autonomous-bits/nomos-provider-file", NewFileProvider)
	RegisterType(typeName string, constructor ProviderTypeConstructor)

	// CreateProvider creates a provider instance of the given type with the provided config.
	// Returns an error if the type is not registered.
	CreateProvider(typeName string, config map[string]any) (Provider, error)

	// IsTypeRegistered checks if a provider type is registered.
	IsTypeRegistered(typeName string) bool

	// RegisteredTypes returns all registered provider type names.
	RegisteredTypes() []string
}

// providerTypeRegistry is the default implementation of ProviderTypeRegistry.
type providerTypeRegistry struct {
	mu           sync.RWMutex
	constructors map[string]ProviderTypeConstructor
	resolver     ProviderResolver // optional: resolves types to binary paths
	manager      ProviderManager  // optional: manages provider subprocesses
}

// NewProviderTypeRegistry creates a new ProviderTypeRegistry.
func NewProviderTypeRegistry() ProviderTypeRegistry {
	return &providerTypeRegistry{
		constructors: make(map[string]ProviderTypeConstructor),
	}
}

// NewProviderTypeRegistryWithResolver creates a ProviderTypeRegistry that can resolve
// and instantiate remote (external) providers via a lockfile resolver and process manager.
// When a provider type is requested, the registry first checks for a registered in-process
// constructor. If none is found and a resolver+manager are available, it attempts to
// locate and start an external provider binary.
func NewProviderTypeRegistryWithResolver(resolver ProviderResolver, manager ProviderManager) ProviderTypeRegistry {
	return &providerTypeRegistry{
		constructors: make(map[string]ProviderTypeConstructor),
		resolver:     resolver,
		manager:      manager,
	}
}

// NewProviderTypeRegistryWithLockfile creates a ProviderTypeRegistry that uses
// a lockfile resolver to locate and start external provider subprocesses.
// This is a convenience function that creates the provider manager internally.
//
// The provider manager lifecycle is handled automatically - subprocesses will be
// cleaned up by the OS when the parent process exits.
func NewProviderTypeRegistryWithLockfile(resolver ProviderResolver) ProviderTypeRegistry {
	// Create the provider manager internally
	manager := NewManager()

	return &providerTypeRegistry{
		constructors: make(map[string]ProviderTypeConstructor),
		resolver:     resolver,
		manager:      manager,
	}
}

// RegisterType implements ProviderTypeRegistry.RegisterType.
func (r *providerTypeRegistry) RegisterType(typeName string, constructor ProviderTypeConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[typeName] = constructor
}

// CreateProvider implements ProviderTypeRegistry.CreateProvider.
func (r *providerTypeRegistry) CreateProvider(typeName string, config map[string]any) (Provider, error) {
	// First, check for in-process constructor
	r.mu.RLock()
	constructor, hasConstructor := r.constructors[typeName]
	hasResolver := r.resolver != nil && r.manager != nil
	r.mu.RUnlock()

	// Prefer in-process constructor if available
	if hasConstructor {
		provider, err := constructor(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider of type %q: %w", typeName, err)
		}
		return provider, nil
	}

	// Fall back to remote provider if resolver+manager available
	if hasResolver {
		ctx := context.Background() // TODO: Accept context from caller
		binaryPath, err := r.resolver.ResolveBinaryPath(ctx, typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve provider type %q: %w", typeName, err)
		}

		// Use the provider type as the alias for now (can be refined later)
		opts := ProviderInitOptions{
			Alias:  typeName,
			Config: config,
		}

		provider, err := r.manager.GetProvider(ctx, typeName, binaryPath, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to start remote provider %q: %w", typeName, err)
		}

		return provider, nil
	}

	// No constructor or resolver available
	// BREAKING CHANGE (v0.3.0): In-process providers have been removed.
	// Provide clear migration guidance to users.
	return nil, fmt.Errorf("provider type %q not found: external providers are required (in-process providers removed in v0.3.0). "+
		"Run 'nomos init' to install provider binaries. "+
		"See migration guide: https://github.com/autonomous-bits/nomos/blob/main/docs/guides/external-providers-migration.md", typeName)
}

// IsTypeRegistered implements ProviderTypeRegistry.IsTypeRegistered.
func (r *providerTypeRegistry) IsTypeRegistered(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.constructors[typeName]
	return ok
}

// RegisteredTypes implements ProviderTypeRegistry.RegisteredTypes.
func (r *providerTypeRegistry) RegisteredTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.constructors))
	for typeName := range r.constructors {
		types = append(types, typeName)
	}

	return types
}
