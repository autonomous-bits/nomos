// Package imports handles import statement resolution for the compiler.
package imports

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Provider is a minimal interface for fetching data.
type Provider interface {
	Fetch(ctx context.Context, path []string) (any, error)
	Init(ctx context.Context, opts ProviderInitOptions) error
}

// ProviderInitOptions contains initialization parameters for a provider.
type ProviderInitOptions struct {
	Alias  string
	Config map[string]any
}

// ProviderRegistry manages provider instances.
type ProviderRegistry interface {
	GetProvider(alias string) (Provider, error)
	Register(alias string, constructor func(ProviderInitOptions) (Provider, error))
}

// ProviderTypeRegistry manages provider type constructors.
type ProviderTypeRegistry interface {
	CreateProvider(typeName string, config map[string]any) (Provider, error)
}

// ExtractedData represents parsed file data with imports extracted.
type ExtractedData struct {
	// Sources are the source provider declarations from the file
	Sources []SourceDecl

	// Imports are the import statements from the file
	Imports []ImportDecl

	// Data is the converted configuration data (sections only, no source/import statements)
	Data map[string]any
}

// SourceDecl represents a source provider declaration extracted from AST.
type SourceDecl struct {
	Alias  string
	Type   string
	Config map[string]any
}

// ImportDecl represents an import statement extracted from AST.
type ImportDecl struct {
	Alias string   // Provider alias to import from
	Path  []string // Optional nested path (empty means import entire source)
}

// ExtractImports extracts source declarations, imports, and data from a parsed AST.
func ExtractImports(tree *ast.AST) ExtractedData {
	var sources []SourceDecl
	var imports []ImportDecl

	// Handle nil tree
	if tree == nil {
		return ExtractedData{
			Sources: sources,
			Imports: imports,
			Data:    make(map[string]any),
		}
	}

	for _, stmt := range tree.Statements {
		switch s := stmt.(type) {
		case *ast.SourceDecl:
			// Extract source declaration
			config := make(map[string]any)
			for k, expr := range s.Config {
				config[k] = exprToValue(expr)
			}
			sources = append(sources, SourceDecl{
				Alias:  s.Alias,
				Type:   s.Type,
				Config: config,
			})

		case *ast.ImportStmt:
			// Extract import statement
			// Format: import:alias or import:alias:path
			// The path can be a file path (e.g., "base.csl") or a map path (e.g., "config.database")
			// For file provider: path is the filename
			// For other providers: path might be a dotted map key path
			// We keep the path as a single element for now - providers will interpret it
			imp := ImportDecl{
				Alias: s.Alias,
			}
			if s.Path != "" {
				// Store path as single element - don't split on dots here
				// The provider will decide how to interpret the path
				imp.Path = []string{s.Path}
			}
			imports = append(imports, imp)
		}
	}

	// Convert remaining sections to data
	data, _ := converter.ASTToData(tree)

	return ExtractedData{
		Sources: sources,
		Imports: imports,
		Data:    data,
	}
}

// exprToValue converts an AST expression to a Go value.
func exprToValue(expr ast.Expr) any {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.ReferenceExpr:
		// References are not resolved here - kept as ReferenceExpr
		return e
	default:
		return nil
	}
}

// ResolveImports resolves imports using the provider registry and type registry.
// Returns merged data from all imports in dependency order.
func ResolveImports(ctx context.Context, filePath string, registry ProviderRegistry, typeRegistry ProviderTypeRegistry) (map[string]any, error) {
	// Parse the file
	tree, _, err := parse.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", filePath, err)
	}

	// Extract declarations
	extracted := ExtractImports(tree)

	// Initialize providers from source declarations
	for _, src := range extracted.Sources {
		if err := initializeProvider(ctx, src, registry, typeRegistry); err != nil {
			return nil, fmt.Errorf("failed to initialize provider %q: %w", src.Alias, err)
		}
	}

	// Resolve each import
	result := make(map[string]any)
	for _, imp := range extracted.Imports {
		importedData, err := resolveImport(ctx, imp, registry)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve import %q: %w", imp.Alias, err)
		}

		// Merge imported data (first imports win for now - we'll improve this)
		result = deepMerge(result, importedData)
	}

	// Merge main file data (last-wins)
	result = deepMerge(result, extracted.Data)

	return result, nil
}

// initializeProvider initializes a provider from a source declaration.
func initializeProvider(ctx context.Context, src SourceDecl, registry ProviderRegistry, typeRegistry ProviderTypeRegistry) error {
	// Check if provider already exists
	if _, err := registry.GetProvider(src.Alias); err == nil {
		// Already initialized
		return nil
	}

	// Check if type registry is available
	if typeRegistry == nil {
		return fmt.Errorf("cannot create provider %q: type registry not provided", src.Alias)
	}

	// Create provider from type
	provider, err := typeRegistry.CreateProvider(src.Type, src.Config)
	if err != nil {
		return fmt.Errorf("failed to create provider %q of type %q: %w", src.Alias, src.Type, err)
	}

	// Initialize the provider
	opts := ProviderInitOptions{
		Alias:  src.Alias,
		Config: src.Config,
	}

	if err := provider.Init(ctx, opts); err != nil {
		return fmt.Errorf("failed to initialize provider %q: %w", src.Alias, err)
	}

	// Register the initialized provider instance
	// We need to wrap it in a constructor for the registry
	registry.Register(src.Alias, func(opts ProviderInitOptions) (Provider, error) {
		return provider, nil
	})

	return nil
}

// resolveImport resolves a single import using the provider.
func resolveImport(ctx context.Context, imp ImportDecl, registry ProviderRegistry) (map[string]any, error) {
	// Get the provider
	provider, err := registry.GetProvider(imp.Alias)
	if err != nil {
		return nil, fmt.Errorf("provider %q not found: %w", imp.Alias, err)
	}

	// Determine fetch path
	var fetchPath []string
	if len(imp.Path) == 0 {
		// No path specified - import:alias form
		// This is ambiguous for most providers
		// For now, return error - providers could implement a "default" path in the future
		return nil, fmt.Errorf("import %q: path required (use import:alias:path syntax)", imp.Alias)
	}

	fetchPath = imp.Path

	// Fetch data
	data, err := provider.Fetch(ctx, fetchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %q path %v: %w", imp.Alias, fetchPath, err)
	}

	// Convert to map
	result, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("import %q returned non-map data: %T", imp.Alias, data)
	}

	return result, nil
}

// deepMerge merges two maps with last-wins semantics for scalars and deep-merge for nested maps.
func deepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Apply overrides
	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			// Both exist - check if both are maps
			if baseMap, ok := baseVal.(map[string]any); ok {
				if overrideMap, ok := v.(map[string]any); ok {
					// Deep merge maps
					result[k] = deepMerge(baseMap, overrideMap)
					continue
				}
			}
		}
		// Last-wins for scalars, arrays, or type mismatches
		result[k] = v
	}

	return result
}
