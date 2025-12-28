package compiler

import (
	"context"
	"errors"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/imports"
)

// ErrImportResolutionNotAvailable is returned when import resolution cannot proceed
// due to missing dependencies (e.g., no ProviderTypeRegistry).
var ErrImportResolutionNotAvailable = errors.New("import resolution not available: ProviderTypeRegistry required")

// resolveFileImports processes a single file's imports and returns merged data.
// Returns ErrImportResolutionNotAvailable if the file has no type registry
// for dynamic provider creation.
func resolveFileImports(ctx context.Context, filePath string, opts Options) (map[string]any, error) {
	// Check if we have a type registry for dynamic provider creation
	if opts.ProviderTypeRegistry == nil {
		// No type registry - can't process source declarations
		return nil, ErrImportResolutionNotAvailable
	}

	// Resolve imports directly - no adapters needed since all use core interfaces
	data, err := imports.ResolveImports(ctx, filePath, opts.ProviderRegistry, opts.ProviderTypeRegistry)
	if err != nil {
		return nil, err
	}

	return data, nil
}
