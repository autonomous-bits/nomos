// Package config provides data structures and utilities for managing
// Nomos provider configuration, including lockfiles and manifests.
//
// The lockfile (.nomos/providers.lock.json) records exact provider binaries
// with their versions, sources, and checksums for reproducible builds.
//
// The manifest (.nomos/providers.yaml) provides declarative provider configuration
// with source hints; versions must be specified in .csl source declarations.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Lockfile represents the .nomos/providers.lock.json structure.
// It records the exact provider binaries used for a project with their
// versions, sources, checksums, and installation paths.
type Lockfile struct {
	// Providers is the list of provider entries in the lockfile.
	Providers []Provider `json:"providers"`
}

// Provider represents a single provider entry in the lockfile.
// Each provider has a unique alias and records all metadata needed
// to locate and verify the provider binary.
type Provider struct {
	// Alias is the provider alias used in .csl source declarations.
	// Must be unique within a lockfile.
	Alias string `json:"alias"`

	// Type is the provider implementation type (e.g., "file", "http").
	Type string `json:"type"`

	// Version is the semantic version of the provider binary.
	Version string `json:"version"`

	// OS is the operating system the binary was built for (e.g., "darwin", "linux").
	OS string `json:"os"`

	// Arch is the architecture the binary was built for (e.g., "amd64", "arm64").
	Arch string `json:"arch"`

	// Source describes where the provider binary was obtained from.
	Source ProviderSource `json:"source,omitempty"`

	// Checksum is the SHA256 checksum of the provider binary for verification.
	Checksum string `json:"checksum,omitempty"`

	// Path is the relative path to the installed provider binary.
	Path string `json:"path"`
}

// ProviderSource describes where a provider binary was obtained from.
// Either GitHub or Local should be set, but not both.
type ProviderSource struct {
	// GitHub indicates the provider was downloaded from a GitHub Release.
	GitHub *GitHubSource `json:"github,omitempty"`

	// Local indicates the provider was copied from a local filesystem path.
	Local *LocalSource `json:"local,omitempty"`
}

// GitHubSource describes a provider obtained from a GitHub Release.
type GitHubSource struct {
	// Owner is the GitHub repository owner (user or organization).
	Owner string `json:"owner"`

	// Repo is the GitHub repository name.
	Repo string `json:"repo"`

	// Asset is the release asset filename that was downloaded.
	Asset string `json:"asset,omitempty"`
}

// LocalSource describes a provider copied from a local filesystem path.
type LocalSource struct {
	// Path is the original local filesystem path the provider was copied from.
	Path string `json:"path"`
}

// Validate checks if the lockfile is valid according to Nomos requirements.
// It returns an error if any validation rule is violated.
func (l *Lockfile) Validate() error {
	if len(l.Providers) == 0 {
		return errors.New("lockfile must contain at least one provider")
	}

	// Check for duplicate aliases
	seen := make(map[string]bool)
	for i, provider := range l.Providers {
		if provider.Alias == "" {
			return fmt.Errorf("provider at index %d: alias is required", i)
		}
		if provider.Type == "" {
			return fmt.Errorf("provider %q: type is required", provider.Alias)
		}
		if provider.Version == "" {
			return fmt.Errorf("provider %q: version is required", provider.Alias)
		}
		if provider.Path == "" {
			return fmt.Errorf("provider %q: path is required", provider.Alias)
		}

		if seen[provider.Alias] {
			return fmt.Errorf("duplicate provider alias: %q", provider.Alias)
		}
		seen[provider.Alias] = true
	}

	return nil
}

// Save writes the lockfile to the specified path as formatted JSON.
// It creates parent directories if they don't exist.
func (l *Lockfile) Save(path string) error {
	// Validate before saving
	if err := l.Validate(); err != nil {
		return fmt.Errorf("invalid lockfile: %w", err)
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}

// LoadLockfile reads and parses a lockfile from the specified path.
func LoadLockfile(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lockfile Lockfile
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	// Validate after loading
	if err := lockfile.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lockfile: %w", err)
	}

	return &lockfile, nil
}

// BinaryPath constructs the standard installation path for a provider binary
// given a base directory. The path follows the convention:
// {baseDir}/{type}/{version}/{os}-{arch}/provider
func (p *Provider) BinaryPath(baseDir string) string {
	return filepath.Join(baseDir, p.Type, p.Version, fmt.Sprintf("%s-%s", p.OS, p.Arch), "provider")
}
