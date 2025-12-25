// Package compiler provides compilation functionality for Nomos configuration scripts.
//
// The compiler transforms Nomos .csl source files into deterministic, serializable
// configuration snapshots. It integrates with the parser library for syntax analysis,
// supports pluggable provider adapters for external data sources, and enforces
// deterministic composition semantics with deep-merge and last-wins behavior.
package compiler

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/resolver"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/validator"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
) // Options configures a compilation run.
type Options struct {
	// Path specifies the input file or directory to compile.
	Path string

	// ProviderRegistry provides access to external data sources.
	ProviderRegistry ProviderRegistry

	// ProviderTypeRegistry provides provider type constructors for dynamic creation.
	// Required for processing source declarations in .csl files.
	ProviderTypeRegistry ProviderTypeRegistry

	// Vars provides variable substitutions available during compilation.
	Vars map[string]any

	// Timeouts configures timeout behavior for compilation operations.
	Timeouts OptionsTimeouts

	// AllowMissingProvider controls behavior when a provider fetch fails.
	AllowMissingProvider bool
}

// OptionsTimeouts configures timeout behavior for compilation operations.
type OptionsTimeouts struct {
	// PerProviderFetch sets the default timeout for each provider Fetch call.
	PerProviderFetch time.Duration

	// MaxConcurrentProviders limits concurrent provider fetch operations.
	MaxConcurrentProviders int
}

// Snapshot represents a compiled configuration snapshot.
type Snapshot struct {
	// Data contains the compiled configuration.
	Data map[string]any `json:"data"`

	// Metadata provides provenance and diagnostic information.
	Metadata Metadata `json:"metadata"`
}

// Metadata contains provenance and diagnostic information for a compilation run.
type Metadata struct {
	// InputFiles lists all .csl source files processed during compilation.
	InputFiles []string `json:"input_files"`

	// ProviderAliases lists the aliases of all providers used.
	ProviderAliases []string `json:"provider_aliases"`

	// StartTime records when compilation began.
	StartTime time.Time `json:"start_time"`

	// EndTime records when compilation completed.
	EndTime time.Time `json:"end_time"`

	// Errors contains fatal parse or compilation errors.
	Errors []string `json:"errors"`

	// Warnings contains non-fatal issues encountered during compilation.
	Warnings []string `json:"warnings"`

	// PerKeyProvenance maps each top-level configuration key to its origin.
	PerKeyProvenance map[string]Provenance `json:"per_key_provenance"`
}

// Provenance records the origin of a configuration value.
type Provenance struct {
	// Source identifies the .csl file that contributed this value.
	Source string `json:"source"`

	// ProviderAlias identifies the provider that resolved this value.
	ProviderAlias string `json:"provider_alias"`
}

// Compile compiles Nomos source files into a deterministic configuration snapshot.
func Compile(ctx context.Context, opts Options) (Snapshot, error) {
	// Validate context
	if ctx == nil {
		return Snapshot{}, stderrors.New("context must not be nil")
	}

	// Validate options
	if opts.Path == "" {
		return Snapshot{}, stderrors.New("options.Path must not be empty")
	}

	if opts.ProviderRegistry == nil {
		return Snapshot{}, stderrors.New("options.ProviderRegistry must not be nil")
	}

	startTime := time.Now()

	// Discover input files
	inputFiles, err := discoverInputFiles(opts.Path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("failed to discover input files: %w", err)
	}

	// Initialize error and warning slices early
	errors := make([]string, 0)
	warnings := make([]string, 0)

	// Special case: If compiling a single file and type registry is provided,
	// check for imports and resolve them first
	var data map[string]any
	var provenance map[string]Provenance

	if len(inputFiles) == 1 && opts.ProviderTypeRegistry != nil {
		// Try to resolve imports for this file
		importData, err := resolveFileImports(ctx, inputFiles[0], opts)
		if err != nil {
			return Snapshot{}, fmt.Errorf("failed to resolve imports: %w", err)
		}

		if importData != nil {
			// Successfully resolved with imports
			data = importData
			provenance = make(map[string]Provenance)
			// TODO: Track provenance for imported data
			for key := range data {
				provenance[key] = Provenance{
					Source: inputFiles[0],
				}
			}
		}
	}

	// If we didn't resolve via imports, use regular flow
	if data == nil {
		// Parse files and collect diagnostics
		var allDiags []diagnostic.Diagnostic

		for _, filePath := range inputFiles {
			_, diags, err := parse.ParseFile(filePath)
			if err != nil {
				return Snapshot{}, fmt.Errorf("fatal parse error for %q: %w", filePath, err)
			}

			// Collect diagnostics
			allDiags = append(allDiags, diags...)
		}

		// Separate errors and warnings from diagnostics
		for _, diag := range allDiags {
			if diag.IsError() {
				errors = append(errors, diag.FormattedMessage)
			} else if diag.IsWarning() {
				warnings = append(warnings, diag.FormattedMessage)
			}
		}

		// Convert ASTs to data maps and merge them following deterministic order
		// This implements composition semantics: deep-merge maps, replace arrays, last-wins for scalars
		data = make(map[string]any)
		provenance = make(map[string]Provenance)

		for _, filePath := range inputFiles {
			// Parse file again to get AST for conversion
			ast, _, err := parse.ParseFile(filePath)
			if err != nil {
				return Snapshot{}, fmt.Errorf("failed to parse file for conversion %q: %w", filePath, err)
			}
			if ast == nil {
				continue
			}

			// Convert AST to data
			fileData, err := converter.ASTToData(ast)
			if err != nil {
				return Snapshot{}, fmt.Errorf("failed to convert AST for %q: %w", filePath, err)
			}

			// Merge using DeepMergeWithProvenance
			// Files are processed in lexicographic order (already sorted by discoverInputFiles)
			// so last-wins behavior is deterministic
			data = DeepMergeWithProvenance(data, "", fileData, filePath, provenance)
		}

		// Initialize providers from source declarations in all input files
		// This is necessary for inline references to work, even without import statements
		// Only do this if we didn't already resolve via imports path
		if opts.ProviderTypeRegistry != nil {
			if err := initializeProvidersFromSources(ctx, inputFiles, opts); err != nil {
				return Snapshot{}, fmt.Errorf("failed to initialize providers: %w", err)
			}
		}
	}

	// Perform semantic validation before reference resolution
	// This catches unresolved provider aliases early
	validatorInst := validator.New(validator.Options{
		RegisteredProviderAliases: opts.ProviderRegistry.RegisteredAliases(),
	})

	if err := validatorInst.Validate(ctx, data); err != nil {
		// Check if it's an unresolved reference error
		var unresolvedErr *validator.ErrUnresolvedReference
		if stderrors.As(err, &unresolvedErr) {
			// Add to errors with formatted message
			errors = append(errors, unresolvedErr.Error())
			return Snapshot{
				Data: data,
				Metadata: Metadata{
					InputFiles:       inputFiles,
					ProviderAliases:  opts.ProviderRegistry.RegisteredAliases(),
					StartTime:        startTime,
					EndTime:          time.Now(),
					Errors:           errors,
					Warnings:         warnings,
					PerKeyProvenance: provenance,
				},
			}, err
		}

		// Check if it's a cycle detection error
		var cycleErr *validator.ErrCycleDetected
		if stderrors.As(err, &cycleErr) {
			errors = append(errors, cycleErr.Error())
			return Snapshot{
				Data: data,
				Metadata: Metadata{
					InputFiles:       inputFiles,
					ProviderAliases:  opts.ProviderRegistry.RegisteredAliases(),
					StartTime:        startTime,
					EndTime:          time.Now(),
					Errors:           errors,
					Warnings:         warnings,
					PerKeyProvenance: provenance,
				},
			}, err
		}

		// Unknown validation error
		return Snapshot{}, fmt.Errorf("semantic validation failed: %w", err)
	}

	// Resolve references in the data using the resolver
	resolvedData, resolveErr := resolveReferences(ctx, data, opts, &warnings)
	if resolveErr != nil {
		return Snapshot{}, fmt.Errorf("reference resolution failed: %w", resolveErr)
	}

	// Return snapshot
	return Snapshot{
		Data: resolvedData,
		Metadata: Metadata{
			InputFiles:       inputFiles,
			ProviderAliases:  opts.ProviderRegistry.RegisteredAliases(),
			StartTime:        startTime,
			EndTime:          time.Now(),
			Errors:           errors,
			Warnings:         warnings,
			PerKeyProvenance: provenance,
		},
	}, nil
}

// discoverInputFiles finds all .csl files at the given path.
// If path is a file, returns that file.
// If path is a directory, returns all .csl files in lexicographic order.
func discoverInputFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}

	// Single file
	if !info.IsDir() {
		if !strings.HasSuffix(path, ".csl") {
			return nil, fmt.Errorf("file %q is not a .csl file", path)
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		return []string{absPath}, nil
	}

	// Directory: list entries and filter for .csl files
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", path, err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".csl") {
			continue
		}
		fullPath := filepath.Join(path, entry.Name())
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %q: %w", fullPath, err)
		}
		files = append(files, absPath)
	}

	// Sort lexicographically for determinism
	sort.Strings(files)

	return files, nil
}

// initializeProvidersFromSources parses input files, extracts source declarations,
// and initializes providers in the registry. This ensures providers are available
// for inline reference resolution, even without import statements.
func initializeProvidersFromSources(ctx context.Context, inputFiles []string, opts Options) error {
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
			if _, err := opts.ProviderRegistry.GetProvider(ctx, sourceDecl.Alias); err == nil {
				// Already registered, skip
				continue
			}

			// Convert config expressions to values
			config := make(map[string]any)
			for k, expr := range sourceDecl.Config {
				config[k] = exprToConfigValue(expr)
			}

			// Create provider from type using the type registry
			provider, err := opts.ProviderTypeRegistry.CreateProvider(ctx, sourceDecl.Type, config)
			if err != nil {
				return fmt.Errorf("failed to create provider %q of type %q: %w", sourceDecl.Alias, sourceDecl.Type, err)
			}

			// Initialize the provider
			initOpts := ProviderInitOptions{
				Alias:          sourceDecl.Alias,
				Config:         config,
				SourceFilePath: filePath,
			}

			if err := provider.Init(ctx, initOpts); err != nil {
				return fmt.Errorf("failed to initialize provider %q: %w", sourceDecl.Alias, err)
			}

			// Register the provider
			opts.ProviderRegistry.Register(sourceDecl.Alias, func(opts ProviderInitOptions) (Provider, error) {
				return provider, nil
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

// resolveReferences resolves all ReferenceExpr nodes in the data using the resolver.
// Warnings generated during resolution are appended to the warnings slice.
func resolveReferences(ctx context.Context, data map[string]any, opts Options, warnings *[]string) (map[string]any, error) {
	// Create adapter for ProviderRegistry with context
	registryAdapter := &providerRegistryAdapter{
		registry: opts.ProviderRegistry,
		ctx:      ctx,
	}

	// Create resolver with options
	resolverOpts := resolver.ResolverOptions{
		ProviderRegistry:     registryAdapter,
		AllowMissingProvider: opts.AllowMissingProvider,
		OnWarning: func(warning string) {
			*warnings = append(*warnings, warning)
		},
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

// providerRegistryAdapter adapts compiler.ProviderRegistry to resolver.ProviderRegistry.
type providerRegistryAdapter struct {
	registry ProviderRegistry
	ctx      context.Context
}

func (a *providerRegistryAdapter) GetProvider(alias string) (resolver.Provider, error) {
	provider, err := a.registry.GetProvider(a.ctx, alias)
	if err != nil {
		return nil, err
	}
	return &providerAdapter{provider: provider}, nil
}

// providerAdapter adapts compiler.Provider to resolver.Provider.
type providerAdapter struct {
	provider Provider
}

func (a *providerAdapter) Fetch(ctx context.Context, path []string) (any, error) {
	return a.provider.Fetch(ctx, path)
}
