package config

import (
	"errors"
	"fmt"
	"os"
)

// Resolver provides an API to query provider configuration from both
// lockfile and manifest. It implements the precedence rules:
// - Lockfile is authoritative for installed binaries (version, path, checksum)
// - Manifest provides source hints and default configuration
// - Versions MUST be specified in .csl source declarations
type Resolver struct {
	lockfile *Lockfile
	manifest *Manifest
}

// ResolvedProvider contains the resolved provider information combining
// data from lockfile and manifest according to precedence rules.
type ResolvedProvider struct {
	// Alias is the provider alias.
	Alias string

	// Type is the provider implementation type.
	Type string

	// Version is the provider version (from lockfile if present).
	Version string

	// OS and Arch are the binary platform (from lockfile if present).
	OS   string
	Arch string

	// Path is the installed binary path (from lockfile if present).
	Path string

	// Checksum is the binary checksum (from lockfile if present).
	Checksum string

	// Source describes where the provider was obtained (from lockfile or manifest).
	Source ProviderSource

	// ManifestSource provides source hints from manifest (if present).
	ManifestSource ManifestSource

	// Config provides default configuration from manifest (if present).
	Config map[string]any
}

// NewResolver creates a Resolver from lockfile and manifest paths.
// At least one of lockfile or manifest must exist.
// Missing files are tolerated (nil) but both missing returns an error.
func NewResolver(lockfilePath, manifestPath string) (*Resolver, error) {
	var lockfile *Lockfile
	var manifest *Manifest

	// Try to load lockfile
	if _, err := os.Stat(lockfilePath); err == nil {
		lf, err := LoadLockfile(lockfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load lockfile: %w", err)
		}
		lockfile = lf
	}

	// Try to load manifest
	if _, err := os.Stat(manifestPath); err == nil {
		mf, err := LoadManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load manifest: %w", err)
		}
		manifest = mf
	}

	// At least one must exist
	if lockfile == nil && manifest == nil {
		return nil, errors.New("neither lockfile nor manifest found; run 'nomos init'")
	}

	return &Resolver{
		lockfile: lockfile,
		manifest: manifest,
	}, nil
}

// ResolveProvider resolves a provider by alias, combining information from
// lockfile and manifest according to precedence rules.
func (r *Resolver) ResolveProvider(alias string) (*ResolvedProvider, error) {
	// Try lockfile first (authoritative for installed binaries)
	var lockfileProvider *Provider
	if r.lockfile != nil {
		for i := range r.lockfile.Providers {
			if r.lockfile.Providers[i].Alias == alias {
				lockfileProvider = &r.lockfile.Providers[i]
				break
			}
		}
	}

	// Try manifest (provides hints and config)
	var manifestProvider *ManifestProvider
	if r.manifest != nil {
		if mp, found := r.manifest.FindProvider(alias); found {
			manifestProvider = &mp
		}
	}

	// Must be found in at least one
	if lockfileProvider == nil && manifestProvider == nil {
		return nil, fmt.Errorf("provider %q not found in lockfile or manifest", alias)
	}

	// Build resolved provider
	resolved := &ResolvedProvider{
		Alias: alias,
	}

	// Lockfile data takes precedence for installed binary info
	if lockfileProvider != nil {
		resolved.Type = lockfileProvider.Type
		resolved.Version = lockfileProvider.Version
		resolved.OS = lockfileProvider.OS
		resolved.Arch = lockfileProvider.Arch
		resolved.Path = lockfileProvider.Path
		resolved.Checksum = lockfileProvider.Checksum
		resolved.Source = lockfileProvider.Source
	}

	// Manifest provides additional data
	if manifestProvider != nil {
		// Type can come from manifest if not in lockfile
		if resolved.Type == "" {
			resolved.Type = manifestProvider.Type
		}

		// Always include manifest source hints and config
		resolved.ManifestSource = manifestProvider.Source
		resolved.Config = manifestProvider.Config
	}

	return resolved, nil
}

// GetAllProviders returns all known providers from both lockfile and manifest,
// merged by alias according to precedence rules.
func (r *Resolver) GetAllProviders() []*ResolvedProvider {
	// Collect all unique aliases
	aliasSet := make(map[string]bool)

	if r.lockfile != nil {
		for _, p := range r.lockfile.Providers {
			aliasSet[p.Alias] = true
		}
	}

	if r.manifest != nil {
		for _, p := range r.manifest.Providers {
			aliasSet[p.Alias] = true
		}
	}

	// Resolve each alias
	var providers []*ResolvedProvider
	for alias := range aliasSet {
		if resolved, err := r.ResolveProvider(alias); err == nil {
			providers = append(providers, resolved)
		}
	}

	return providers
}
