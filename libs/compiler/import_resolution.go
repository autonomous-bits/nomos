package compiler

import (
	"context"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/imports"
)

// resolveFileImports processes a single file's imports and returns merged data.
// If the file has no imports, returns the file's own data.
func resolveFileImports(ctx context.Context, filePath string, opts Options) (map[string]any, error) {
	// Check if we have a type registry for dynamic provider creation
	if opts.ProviderTypeRegistry == nil {
		// No type registry - can't process source declarations
		// Fall back to regular AST conversion without import resolution
		return nil, nil // Signal to use regular flow
	}

	// Create adapters
	regAdapter := &providerRegistryImportsAdapter{registry: opts.ProviderRegistry}
	typeRegAdapter := &providerTypeRegistryImportsAdapter{typeRegistry: opts.ProviderTypeRegistry}

	// Resolve imports
	data, err := imports.ResolveImports(ctx, filePath, regAdapter, typeRegAdapter)
	if err != nil {
		return nil, err
	}

	return data, nil
}
