package compiler

import (
	"context"
	"fmt"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
)

// ProviderTypeConstructor is an alias for core.ProviderTypeConstructor for backward compatibility.
//
// Deprecated: Use core.ProviderTypeConstructor directly.
type ProviderTypeConstructor = core.ProviderTypeConstructor

// providerTypeRegistry is the default implementation of ProviderTypeRegistry.
type providerTypeRegistry struct {
	mu           sync.RWMutex
	constructors map[string]core.ProviderTypeConstructor
	resolver     ProviderResolver // optional: resolves types to binary paths
	manager      ProviderManager  // optional: manages provider subprocesses
}

// NewProviderTypeRegistry creates a new ProviderTypeRegistry.
func NewProviderTypeRegistry() core.ProviderTypeRegistry {
	return &providerTypeRegistry{
		constructors: make(map[string]ProviderTypeConstructor),
	}
}

// NewProviderTypeRegistryWithResolver creates a ProviderTypeRegistry that can resolve
// and instantiate remote (external) providers via a lockfile resolver and process manager.
// When a provider type is requested, the registry first checks for a registered in-process
// constructor. If none is found and a resolver+manager are available, it attempts to
// locate and start an external provider binary.
func NewProviderTypeRegistryWithResolver(resolver ProviderResolver, manager ProviderManager) core.ProviderTypeRegistry {
	return &providerTypeRegistry{
		constructors: make(map[string]core.ProviderTypeConstructor),
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
func NewProviderTypeRegistryWithLockfile(resolver ProviderResolver) core.ProviderTypeRegistry {
	// Create the provider manager internally
	manager := NewManager()

	return &providerTypeRegistry{
		constructors: make(map[string]core.ProviderTypeConstructor),
		resolver:     resolver,
		manager:      manager,
	}
}

// RegisterType implements ProviderTypeRegistry.RegisterType.
func (r *providerTypeRegistry) RegisterType(typeName string, constructor core.ProviderTypeConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[typeName] = constructor
}

// CreateProvider implements ProviderTypeRegistry.CreateProvider.
func (r *providerTypeRegistry) CreateProvider(ctx context.Context, typeName string, alias string, config map[string]any) (core.Provider, error) {
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
		binaryPath, err := r.resolver.ResolveBinaryPath(ctx, typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve provider type %q: %w", typeName, err)
		}

		// Use the actual provider alias for proper instance management
		opts := core.ProviderInitOptions{
			Alias:  alias,
			Config: config,
		}

		provider, err := r.manager.GetProvider(ctx, alias, binaryPath, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to start remote provider %q (alias %q): %w", typeName, alias, err)
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
