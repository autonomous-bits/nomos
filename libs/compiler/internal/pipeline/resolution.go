package pipeline

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/resolver"
)

// ResolveOptions contains options for reference resolution.
type ResolveOptions struct {
	ProviderRegistry     core.ProviderRegistry
	AllowMissingProvider bool
	OnWarning            func(string)
}

// ResolveReferences resolves all ReferenceExpr nodes in the data using the resolver.
// Warnings generated during resolution are passed to the OnWarning callback if provided.
func ResolveReferences(ctx context.Context, data map[string]any, opts ResolveOptions) (map[string]any, error) {
	// Create adapter for ProviderRegistry with context
	// The resolver expects a synchronous GetProvider without context,
	// so we capture the context here and use it in the adapter.
	registryAdapter := &providerRegistryAdapter{
		registry: opts.ProviderRegistry,
		ctx:      ctx,
	}

	// Create resolver with options
	resolverOpts := resolver.ResolverOptions{
		ProviderRegistry:     registryAdapter,
		AllowMissingProvider: opts.AllowMissingProvider,
		OnWarning:            opts.OnWarning,
	}

	r := resolver.New(resolverOpts)

	// Resolve the entire data map
	resolved, err := r.ResolveValue(ctx, data)
	if err != nil {
		return nil, err
	}

	// Type assert back to map
	resolvedMap, ok := resolved.(map[string]any)
	if !ok {
		// This should never happen since we passed in a map
		return nil, fmt.Errorf("internal error: resolved value is not a map")
	}

	return resolvedMap, nil
}

// providerRegistryAdapter adapts core.ProviderRegistry to resolver.ProviderRegistry.
// Since resolver expects a context-free GetProvider, we capture the context and apply it.
type providerRegistryAdapter struct {
	registry core.ProviderRegistry
	ctx      context.Context
}

func (a *providerRegistryAdapter) GetProvider(alias string) (core.Provider, error) {
	// Delegate to the core registry with the captured context
	return a.registry.GetProvider(a.ctx, alias)
}
