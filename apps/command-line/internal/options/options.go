// Package options provides utilities for building compiler.Options from CLI flags.
package options

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/providers/file"
)

// BuildParams holds the parameters for building compiler.Options.
type BuildParams struct {
	// Path specifies the input file or directory.
	Path string

	// Vars holds variable substitutions in key=value form.
	Vars []string

	// TimeoutPerProvider sets timeout for each provider fetch (duration string).
	TimeoutPerProvider string

	// MaxConcurrentProviders limits concurrent provider fetches.
	MaxConcurrentProviders int

	// AllowMissingProvider allows missing provider fetches.
	AllowMissingProvider bool

	// ProviderRegistry is the registry to use for providers.
	// If nil, NewProviderRegistries creates a default empty registry.
	ProviderRegistry compiler.ProviderRegistry

	// ProviderTypeRegistry is the registry to use for provider types.
	// If nil, NewProviderRegistries creates a default empty registry.
	ProviderTypeRegistry compiler.ProviderTypeRegistry
}

// NewProviderRegistries creates default provider and provider type registries.
// It checks for a lockfile (.nomos/providers.lock.json) and if found, wires up
// support for external provider subprocesses. Otherwise, it registers the
// in-process file provider for backward compatibility.
//
// When a lockfile is present:
//   - Creates a ProviderResolver that locates provider binaries
//   - Creates a ProviderTypeRegistry with resolver and manager support
//   - External provider processes will be managed by the compiler
//
// Returns provider registry and provider type registry.
func NewProviderRegistries() (compiler.ProviderRegistry, compiler.ProviderTypeRegistry) {
	providerRegistry := compiler.NewProviderRegistry()

	// Check for lockfile in current directory
	lockfilePath := ".nomos/providers.lock.json"
	manifestPath := ".nomos/providers.yaml"

	// Check if lockfile exists
	if _, err := os.Stat(lockfilePath); err == nil {
		// Lockfile exists - use external providers via providerproc
		baseDirFunc := func() string {
			// Get absolute path to .nomos/providers directory
			wd, _ := os.Getwd()
			return filepath.Join(wd, ".nomos", "providers")
		}

		// Create lockfile-based resolver
		resolver, err := compiler.NewLockfileProviderResolver(lockfilePath, manifestPath, baseDirFunc)
		if err != nil {
			// If resolver creation fails, fall back to in-process providers
			// This allows build to work even with a malformed lockfile
			// (compile will fail later with clear errors if providers are needed)
			providerTypeRegistry := compiler.NewProviderTypeRegistry()
			providerTypeRegistry.RegisterType("file", file.NewFileProviderFromConfig)
			return providerRegistry, providerTypeRegistry
		}

		// Create provider type registry with lockfile resolver
		// The registry will internally create and manage the provider process manager
		// Provider subprocesses will be cleaned up by the OS when the CLI process exits
		providerTypeRegistry := compiler.NewProviderTypeRegistryWithLockfile(resolver)

		return providerRegistry, providerTypeRegistry
	}

	// No lockfile - use in-process providers (backward compatibility)
	providerTypeRegistry := compiler.NewProviderTypeRegistry()
	providerTypeRegistry.RegisterType("file", file.NewFileProviderFromConfig)

	return providerRegistry, providerTypeRegistry
}

// BuildOptions constructs compiler.Options from BuildParams.
// This function handles:
// - Variable parsing and validation
// - Timeout duration parsing
// - Provider registry wiring
// - All field mapping from CLI flags to compiler.Options
func BuildOptions(params BuildParams) (compiler.Options, error) {
	opts := compiler.Options{
		Path:                 params.Path,
		AllowMissingProvider: params.AllowMissingProvider,
		Vars:                 make(map[string]any),
		ProviderRegistry:     params.ProviderRegistry,
		ProviderTypeRegistry: params.ProviderTypeRegistry,
	}

	// Parse and validate vars
	for _, v := range params.Vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return compiler.Options{}, fmt.Errorf("invalid var format %q (expected key=value)", v)
		}
		key := parts[0]
		if key == "" {
			return compiler.Options{}, fmt.Errorf("invalid var format %q (key cannot be empty)", v)
		}
		opts.Vars[key] = parts[1]
	}

	// Parse timeout duration if provided
	if params.TimeoutPerProvider != "" {
		duration, err := time.ParseDuration(params.TimeoutPerProvider)
		if err != nil {
			return compiler.Options{}, fmt.Errorf("invalid timeout-per-provider: %w", err)
		}
		opts.Timeouts.PerProviderFetch = duration
	}

	// Set max concurrent providers
	opts.Timeouts.MaxConcurrentProviders = params.MaxConcurrentProviders

	return opts, nil
}
