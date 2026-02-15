// Package imports handles import statement resolution for the compiler.
package imports

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Provider is an alias to core.Provider for backward compatibility.
type Provider = core.Provider

// ProviderInitOptions is an alias to core.ProviderInitOptions for backward compatibility.
type ProviderInitOptions = core.ProviderInitOptions

// ProviderConstructor is an alias to core.ProviderConstructor for backward compatibility.
type ProviderConstructor = core.ProviderConstructor

// ProviderRegistry manages provider instances.
// This is the same interface as core.ProviderRegistry, kept here for clarity.
type ProviderRegistry = core.ProviderRegistry

// ProviderTypeRegistry manages provider type constructors.
type ProviderTypeRegistry = core.ProviderTypeRegistry

// ExtractedData represents parsed file data with source declarations extracted.
type ExtractedData struct {
	// Sources are the source provider declarations from the file
	Sources []SourceDecl

	// Data is the converted configuration data (sections only, no source statements)
	Data map[string]any
}

// SourceDecl represents a source provider declaration extracted from AST.
type SourceDecl struct {
	Alias  string
	Type   string
	Config map[string]any
}

// ExtractImports extracts source declarations and data from a parsed AST.
// Note: Import statements are no longer supported and have been removed from the language.
func ExtractImports(tree *ast.AST) (ExtractedData, error) {
	var sources []SourceDecl

	// Handle nil tree
	if tree == nil {
		return ExtractedData{
			Sources: sources,
			Data:    make(map[string]any),
		}, nil
	}

	for _, stmt := range tree.Statements {
		if s, ok := stmt.(*ast.SourceDecl); ok {
			// Extract source declaration
			config := make(map[string]any)
			for k, expr := range s.Config {
				value, err := exprToValue(expr)
				if err != nil {
					return ExtractedData{}, fmt.Errorf("failed to convert source %q config %q: %w", s.Alias, k, err)
				}
				config[k] = value
			}
			sources = append(sources, SourceDecl{
				Alias:  s.Alias,
				Type:   s.Type,
				Config: config,
			})
		}
	}

	// Convert remaining sections to data
	data, err := converter.ASTToData(tree)
	if err != nil {
		return ExtractedData{}, fmt.Errorf("failed to convert AST data: %w", err)
	}

	return ExtractedData{
		Sources: sources,
		Data:    data,
	}, nil
}

// exprToValue converts an AST expression to a Go value.
func exprToValue(expr ast.Expr) (any, error) {
	if expr == nil {
		return nil, fmt.Errorf("nil expression")
	}

	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value, nil
	case *ast.ReferenceExpr:
		// References are not resolved here - kept as ReferenceExpr
		return e, nil
	case *ast.MapExpr:
		result := make(map[string]any, len(e.Entries))
		for _, entry := range e.Entries {
			if entry.Spread {
				return nil, fmt.Errorf("spread entries are not supported in source config")
			}
			if entry.Key == "" {
				return nil, fmt.Errorf("map entry key cannot be empty")
			}
			val, err := exprToValue(entry.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map key %q: %w", entry.Key, err)
			}
			result[entry.Key] = val
		}
		return result, nil
	case *ast.ListExpr:
		result := make([]any, 0, len(e.Elements))
		for idx, element := range e.Elements {
			val, err := exprToValue(element)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list element %d: %w", idx, err)
			}
			result = append(result, val)
		}
		return result, nil
	case *ast.PathExpr:
		return pathExprToString(e), nil
	case *ast.IdentExpr:
		return e.Name, nil
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func pathExprToString(p *ast.PathExpr) string {
	if len(p.Components) == 0 {
		return ""
	}
	result := p.Components[0]
	for i := 1; i < len(p.Components); i++ {
		result += "." + p.Components[i]
	}
	return result
}

// ResolveImports initializes providers from source declarations and returns the file's data.
// Note: Import statements are no longer supported. References (@alias:path) are now
// used for cross-file dependencies and are resolved separately during compilation.
func ResolveImports(ctx context.Context, filePath string, registry ProviderRegistry, typeRegistry ProviderTypeRegistry) (map[string]any, error) {
	// Parse the file
	tree, diags, err := parse.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", filePath, err)
	}

	// Check for parse errors in diagnostics
	if len(diags) > 0 {
		// Return first error diagnostic as the error
		for _, d := range diags {
			if d.Severity == diagnostic.SeverityError {
				return nil, fmt.Errorf("parse error in %q: %s", filePath, d.Message)
			}
		}
	}

	// Check for nil tree (can happen with parse errors)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse %q: no AST returned", filePath)
	}

	// Extract declarations
	extracted, err := ExtractImports(tree)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data for %q: %w", filePath, err)
	}

	// Initialize providers from source declarations
	for _, src := range extracted.Sources {
		if err := initializeProvider(ctx, src, filePath, registry, typeRegistry); err != nil {
			return nil, fmt.Errorf("failed to initialize provider %q: %w", src.Alias, err)
		}
	}

	// Return the file's data (references will be resolved separately)
	return extracted.Data, nil
}

// initializeProvider initializes a provider from a source declaration.
func initializeProvider(ctx context.Context, src SourceDecl, sourceFilePath string, registry ProviderRegistry, typeRegistry ProviderTypeRegistry) error {
	// Check if provider already exists
	if _, err := registry.GetProvider(ctx, src.Alias); err == nil {
		// Already initialized
		return nil
	}

	// Check if type registry is available
	if typeRegistry == nil {
		return fmt.Errorf("cannot create provider %q: type registry not provided", src.Alias)
	}

	// Create and initialize the provider once
	provider, err := typeRegistry.CreateProvider(ctx, src.Type, src.Alias, src.Config)
	if err != nil {
		return fmt.Errorf("failed to create provider %q of type %q: %w", src.Alias, src.Type, err)
	}

	// Initialize the provider with the config from source declaration
	initOpts := ProviderInitOptions{
		Alias:          src.Alias,
		Config:         src.Config,
		SourceFilePath: sourceFilePath,
	}

	if err := provider.Init(ctx, initOpts); err != nil {
		return fmt.Errorf("failed to initialize provider %q: %w", src.Alias, err)
	}

	// Register a constructor that returns the already-initialized provider instance
	// This avoids double-initialization when the registry's GetProvider is called
	registry.Register(src.Alias, func(_ ProviderInitOptions) (Provider, error) {
		// Return the cached, already-initialized provider
		// The registry will call Init on it, but we'll wrap it to make Init a no-op
		return &alreadyInitializedProvider{provider: provider}, nil
	})

	return nil
}

// alreadyInitializedProvider wraps a provider that's already initialized
// and makes the Init method a no-op to prevent double-initialization errors.
type alreadyInitializedProvider struct {
	provider Provider
}

func (p *alreadyInitializedProvider) Init(_ context.Context, _ ProviderInitOptions) error {
	// No-op: provider is already initialized
	return nil
}

func (p *alreadyInitializedProvider) Fetch(ctx context.Context, path []string) (any, error) {
	return p.provider.Fetch(ctx, path)
}
