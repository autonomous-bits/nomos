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
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/pipeline"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/validator"
)

// Options configures a compilation run.
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

// CompilationResult wraps a Snapshot and provides convenience methods
// for checking compilation status.
type CompilationResult struct {
	// Snapshot contains the compilation output and metadata.
	Snapshot Snapshot
}

// HasErrors returns true if the compilation encountered any errors.
func (r CompilationResult) HasErrors() bool {
	return len(r.Snapshot.Metadata.Errors) > 0
}

// HasWarnings returns true if the compilation encountered any warnings.
func (r CompilationResult) HasWarnings() bool {
	return len(r.Snapshot.Metadata.Warnings) > 0
}

// Errors returns all compilation errors.
func (r CompilationResult) Errors() []string {
	return r.Snapshot.Metadata.Errors
}

// Warnings returns all compilation warnings.
func (r CompilationResult) Warnings() []string {
	return r.Snapshot.Metadata.Warnings
}

// Error implements the error interface, returning a combined error message
// if any errors occurred, or nil if compilation succeeded.
func (r CompilationResult) Error() error {
	if !r.HasErrors() {
		return nil
	}
	if len(r.Snapshot.Metadata.Errors) == 1 {
		return stderrors.New(r.Snapshot.Metadata.Errors[0])
	}
	return fmt.Errorf("compilation failed with %d errors: %v",
		len(r.Snapshot.Metadata.Errors),
		r.Snapshot.Metadata.Errors)
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
// Returns a CompilationResult containing the snapshot and all collected errors/warnings.
// The compilation process attempts to continue through recoverable errors to collect
// as many issues as possible in a single run.
func Compile(ctx context.Context, opts Options) CompilationResult {
	// Create result with empty snapshot
	result := CompilationResult{
		Snapshot: Snapshot{
			Data: make(map[string]any),
			Metadata: Metadata{
				InputFiles:       []string{},
				ProviderAliases:  []string{},
				StartTime:        time.Now(),
				Errors:           []string{},
				Warnings:         []string{},
				PerKeyProvenance: make(map[string]Provenance),
			},
		},
	}

	// Validate context
	if ctx == nil {
		result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors, "context must not be nil")
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}

	// Validate options
	if opts.Path == "" {
		result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors, "options.Path must not be empty")
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}

	if opts.ProviderRegistry == nil {
		result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors, "options.ProviderRegistry must not be nil")
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}

	// Register "var" provider for variable access
	opts.ProviderRegistry.Register("var", func(_ ProviderInitOptions) (Provider, error) {
		return &varProvider{vars: opts.Vars}, nil
	})

	// Discover input files
	inputFiles, err := pipeline.DiscoverInputFiles(opts.Path)
	if err != nil {
		result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors,
			fmt.Sprintf("failed to discover input files: %v", err))
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}
	result.Snapshot.Metadata.InputFiles = inputFiles

	// Initialize error and warning slices (already initialized in result)
	errors := &result.Snapshot.Metadata.Errors
	warnings := &result.Snapshot.Metadata.Warnings

	// Special case: If compiling a single file and type registry is provided,
	// check for imports and resolve them first
	var data map[string]any
	var provenance map[string]Provenance

	if len(inputFiles) == 1 && opts.ProviderTypeRegistry != nil {
		// Try to resolve imports for this file
		importData, err := resolveFileImports(ctx, inputFiles[0], opts)
		if err != nil && !stderrors.Is(err, ErrImportResolutionNotAvailable) {
			*errors = append(*errors, fmt.Sprintf("failed to resolve imports: %v", err))
			result.Snapshot.Metadata.EndTime = time.Now()
			return result
		}

		if err == nil {
			// Successfully resolved with imports
			data = importData
			provenance = make(map[string]Provenance)
			// TODO: Track provenance for imported data
			// Currently all keys are attributed to the root file, but they may originate from:
			// - Import statements (deprecated) -> should track source file of the import if still supported
			// - Inline references (@alias:path) -> should track provider alias
			// Requires internal/imports to return provenance metadata alongside merged data.
			// See GitHub issue for detailed design: provenance should flow through deep-merge.
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
		parseErrors := false

		for _, filePath := range inputFiles {
			_, diags, err := parse.ParseFile(filePath)
			if err != nil {
				*errors = append(*errors, fmt.Sprintf("fatal parse error for %q: %v", filePath, err))
				parseErrors = true
				continue // Continue parsing other files to collect all errors
			}

			// Collect diagnostics
			allDiags = append(allDiags, diags...)
		}

		// If we had fatal parse errors, stop here
		if parseErrors {
			result.Snapshot.Metadata.EndTime = time.Now()
			return result
		}

		// Separate errors and warnings from diagnostics
		for _, diag := range allDiags {
			if diag.IsError() {
				*errors = append(*errors, diag.FormattedMessage)
			} else if diag.IsWarning() {
				*warnings = append(*warnings, diag.FormattedMessage)
			}
		}

		// If we have parse errors from diagnostics, we can continue but note them
		// Convert ASTs to data maps and merge them following deterministic order
		data = make(map[string]any)
		provenance = make(map[string]Provenance)

		for _, filePath := range inputFiles {
			// Parse file again to get AST for conversion
			ast, _, err := parse.ParseFile(filePath)
			if err != nil {
				// Already collected this error above, skip
				continue
			}
			if ast == nil {
				continue
			}

			// Convert AST to data
			fileData, err := converter.ASTToData(ast)
			if err != nil {
				*errors = append(*errors, fmt.Sprintf("failed to convert AST for %q: %v", filePath, err))
				continue // Continue with other files
			}

			// Merge using DeepMergeWithProvenance
			data = DeepMergeWithProvenance(data, "", fileData, filePath, provenance)
		}

		// Initialize providers from source declarations in all input files
		if opts.ProviderTypeRegistry != nil {
			// Convert ProviderTypeRegistry to core.ProviderTypeRegistry interface
			// This works because ProviderTypeRegistry is an alias for core.ProviderTypeRegistry
			if err := pipeline.InitializeProvidersFromSources(ctx, inputFiles, opts.ProviderRegistry, opts.ProviderTypeRegistry); err != nil {
				*errors = append(*errors, fmt.Sprintf("failed to initialize providers: %v", err))
				// Continue - some validation may still be useful
			}
		}
	}

	// Store the data and provenance
	result.Snapshot.Data = data
	result.Snapshot.Metadata.PerKeyProvenance = provenance
	result.Snapshot.Metadata.ProviderAliases = opts.ProviderRegistry.RegisteredAliases()

	// Perform semantic validation before reference resolution
	validatorInst := validator.New(validator.Options{
		RegisteredProviderAliases: opts.ProviderRegistry.RegisteredAliases(),
	})

	if err := validatorInst.Validate(ctx, data); err != nil {
		// Check if it's an unresolved reference error
		var unresolvedErr *validator.ErrUnresolvedReference
		if stderrors.As(err, &unresolvedErr) {
			*errors = append(*errors, unresolvedErr.Error())
			result.Snapshot.Metadata.EndTime = time.Now()
			return result
		}

		// Check if it's a cycle detection error
		var cycleErr *validator.ErrCycleDetected
		if stderrors.As(err, &cycleErr) {
			*errors = append(*errors, cycleErr.Error())
			result.Snapshot.Metadata.EndTime = time.Now()
			return result
		}

		// Unknown validation error
		*errors = append(*errors, fmt.Sprintf("semantic validation failed: %v", err))
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}

	// Resolve references in the data using the resolver
	resolvedData, resolveErr := pipeline.ResolveReferences(ctx, data, pipeline.ResolveOptions{
		ProviderRegistry:     opts.ProviderRegistry,
		AllowMissingProvider: opts.AllowMissingProvider,
		OnWarning: func(warning string) {
			*warnings = append(*warnings, warning)
		},
	})
	if resolveErr != nil {
		*errors = append(*errors, fmt.Sprintf("reference resolution failed: %v", resolveErr))
		result.Snapshot.Metadata.EndTime = time.Now()
		return result
	}

	// Update with resolved data
	result.Snapshot.Data = resolvedData
	result.Snapshot.Metadata.EndTime = time.Now()

	return result
}
