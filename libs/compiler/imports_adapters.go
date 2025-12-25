package compiler

import (
	"context"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/imports"
)

// providerRegistryImportsAdapter adapts ProviderRegistry to imports.ProviderRegistry.
type providerRegistryImportsAdapter struct {
	registry ProviderRegistry
}

func (a *providerRegistryImportsAdapter) GetProvider(ctx context.Context, alias string) (imports.Provider, error) {
	provider, err := a.registry.GetProvider(ctx, alias)
	if err != nil {
		return nil, err
	}
	return &providerImportsAdapter{provider: provider}, nil
}

func (a *providerRegistryImportsAdapter) Register(alias string, constructor func(imports.ProviderInitOptions) (imports.Provider, error)) {
	// Adapt the constructor
	a.registry.Register(alias, func(opts ProviderInitOptions) (Provider, error) {
		importOpts := imports.ProviderInitOptions{
			Alias:          opts.Alias,
			Config:         opts.Config,
			SourceFilePath: opts.SourceFilePath,
		}
		importProvider, err := constructor(importOpts)
		if err != nil {
			return nil, err
		}
		return &providerFromImportsAdapter{provider: importProvider}, nil
	})
}

// providerImportsAdapter adapts Provider to imports.Provider.
type providerImportsAdapter struct {
	provider Provider
}

func (a *providerImportsAdapter) Fetch(ctx context.Context, path []string) (any, error) {
	return a.provider.Fetch(ctx, path)
}

func (a *providerImportsAdapter) Init(ctx context.Context, opts imports.ProviderInitOptions) error {
	compilerOpts := ProviderInitOptions{
		Alias:          opts.Alias,
		Config:         opts.Config,
		SourceFilePath: opts.SourceFilePath,
	}
	return a.provider.Init(ctx, compilerOpts)
}

// providerFromImportsAdapter adapts imports.Provider to Provider.
type providerFromImportsAdapter struct {
	provider imports.Provider
}

func (a *providerFromImportsAdapter) Fetch(ctx context.Context, path []string) (any, error) {
	return a.provider.Fetch(ctx, path)
}

func (a *providerFromImportsAdapter) Init(ctx context.Context, opts ProviderInitOptions) error {
	importOpts := imports.ProviderInitOptions{
		Alias:          opts.Alias,
		Config:         opts.Config,
		SourceFilePath: opts.SourceFilePath,
	}
	return a.provider.Init(ctx, importOpts)
}

func (a *providerFromImportsAdapter) Info() (alias string, version string) {
	// Default implementation
	return "", "unknown"
}

// providerTypeRegistryImportsAdapter adapts ProviderTypeRegistry to imports.ProviderTypeRegistry.
type providerTypeRegistryImportsAdapter struct {
	typeRegistry ProviderTypeRegistry
}

func (a *providerTypeRegistryImportsAdapter) CreateProvider(ctx context.Context, typeName string, config map[string]any) (imports.Provider, error) {
	provider, err := a.typeRegistry.CreateProvider(ctx, typeName, config)
	if err != nil {
		return nil, err
	}
	return &providerImportsAdapter{provider: provider}, nil
}
