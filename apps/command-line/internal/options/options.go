// Package options provides utilities for building compiler.Options from CLI flags.
package options

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
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
// BREAKING CHANGE: As of v0.3.0, only external providers are supported.
// In-process providers have been removed. Users must run 'nomos init' to install
// provider binaries before running 'nomos build'.
//
// This function requires a lockfile (.nomos/providers.lock.json) to be present.
// If the lockfile is missing or malformed, an empty registry is returned and
// compilation will fail with a clear error message instructing users to run
// 'nomos init'.
//
// Migration guide: https://github.com/autonomous-bits/nomos/blob/main/docs/guides/external-providers-migration.md
//
// Returns provider registry and provider type registry.
func NewProviderRegistries() (compiler.ProviderRegistry, compiler.ProviderTypeRegistry) {
	providerRegistry := compiler.NewProviderRegistry()

	// Check for lockfile in current directory
	lockfilePath := ".nomos/providers.lock.json"
	manifestPath := ".nomos/providers.yaml"

	// Check if lockfile exists
	if _, err := os.Stat(lockfilePath); err != nil {
		// BREAKING CHANGE: No fallback to in-process providers
		// Return empty registry - compiler will fail with clear error
		return providerRegistry, compiler.NewProviderTypeRegistry()
	}

	// Lockfile exists - use external providers via providerproc
	baseDirFunc := func() string {
		// Get absolute path to .nomos/providers directory
		wd, _ := os.Getwd()
		return filepath.Join(wd, ".nomos", "providers")
	}

	// Create lockfile-based resolver
	resolver, err := compiler.NewLockfileProviderResolver(lockfilePath, manifestPath, baseDirFunc)
	if err != nil {
		// BREAKING CHANGE: No fallback to in-process providers
		// Return empty registry - compiler will fail with clear error about malformed lockfile
		return providerRegistry, compiler.NewProviderTypeRegistry()
	}

	// Create provider type registry with lockfile resolver
	// The registry will internally create and manage the provider process manager
	// Provider subprocesses will be cleaned up by the OS when the CLI process exits
	providerTypeRegistry := compiler.NewProviderTypeRegistryWithLockfile(resolver)

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
