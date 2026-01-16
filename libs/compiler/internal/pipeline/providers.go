package pipeline

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// InitializeProvidersFromSources parses input files, extracts source declarations,
// and initializes providers in the registry. This ensures providers are available
// for inline reference resolution, even without import statements.
func InitializeProvidersFromSources(
	ctx context.Context,
	inputFiles []string,
	registry core.ProviderRegistry,
	typeRegistry core.ProviderTypeRegistry,
) error {
	for _, filePath := range inputFiles {
		// Parse the file
		tree, _, err := parse.ParseFile(filePath)
		if err != nil {
			// Skip files that can't be parsed - they'll fail in the main compilation flow
			continue
		}

		// Extract source declarations
		for _, stmt := range tree.Statements {
			sourceDecl, ok := stmt.(*ast.SourceDecl)
			if !ok {
				continue
			}

			// Check if provider is already registered
			if _, err := registry.GetProvider(ctx, sourceDecl.Alias); err == nil {
				// Already registered, skip
				continue
			}

			// Convert config expressions to values
			config := make(map[string]any)
			for k, expr := range sourceDecl.Config {
				config[k] = exprToConfigValue(expr)
			}

			// Create provider from type using the type registry
			provider, err := typeRegistry.CreateProvider(ctx, sourceDecl.Type, sourceDecl.Alias, config)
			if err != nil {
				return fmt.Errorf("failed to create provider %q of type %q: %w", sourceDecl.Alias, sourceDecl.Type, err)
			}

			// Initialize the provider
			initOpts := core.ProviderInitOptions{
				Alias:          sourceDecl.Alias,
				Config:         config,
				SourceFilePath: filePath,
			}

			if err := provider.Init(ctx, initOpts); err != nil {
				return fmt.Errorf("failed to initialize provider %q: %w", sourceDecl.Alias, err)
			}

			// Register the provider with a constructor function
			// Capture the provider in a closure
			capturedProvider := provider
			registry.Register(sourceDecl.Alias, func(_ core.ProviderInitOptions) (core.Provider, error) {
				return capturedProvider, nil
			})
		}
	}

	return nil
}

// exprToConfigValue converts an AST expression to a configuration value.
func exprToConfigValue(expr ast.Expr) any {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.ReferenceExpr:
		// References in config are kept as-is for now
		// Providers should handle them if needed
		return e
	default:
		return nil
	}
}
