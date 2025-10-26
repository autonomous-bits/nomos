// Package options provides utilities for building compiler.Options from CLI flags.
package options

import (
	"fmt"
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
// Both registries are empty by default (no-network behavior).
func NewProviderRegistries() (compiler.ProviderRegistry, compiler.ProviderTypeRegistry) {
	return compiler.NewProviderRegistry(), compiler.NewProviderTypeRegistry()
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
