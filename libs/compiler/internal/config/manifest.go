package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest represents the .nomos/providers.yaml structure.
// It provides declarative provider configuration with source hints.
// Versions MUST be specified in .csl source declarations; the manifest
// only provides hints for where to obtain provider binaries.
type Manifest struct {
	// Providers is the list of provider configurations in the manifest.
	Providers []ManifestProvider `yaml:"providers"`
}

// ManifestProvider represents a single provider entry in the manifest.
// This provides source hints and optional default configuration, but
// versions MUST come from .csl source declarations.
type ManifestProvider struct {
	// Alias is the provider alias used in .csl source declarations.
	Alias string `yaml:"alias"`

	// Type is the provider type in owner/repo format (e.g., "autonomous-bits/nomos-provider-file").
	// This provides proper namespacing and avoids conflicts between providers with similar names.
	Type string `yaml:"type"`

	// Source provides hints for where to obtain the provider binary.
	Source ManifestSource `yaml:"source,omitempty"`

	// Config provides optional default configuration passed to provider Init.
	// Keys are provider-specific (e.g., "directory" for file provider).
	Config map[string]any `yaml:"config,omitempty"`
}

// ManifestSource provides hints for where to obtain a provider binary.
// Either GitHub or Local should be set, but not both.
type ManifestSource struct {
	// GitHub indicates the provider should be downloaded from a GitHub Release.
	GitHub *ManifestGitHubSource `yaml:"github,omitempty"`

	// Local indicates the provider should be copied from a local filesystem path.
	Local *ManifestLocalSource `yaml:"local,omitempty"`
}

// ManifestGitHubSource provides GitHub Release source hints.
type ManifestGitHubSource struct {
	// Owner is the GitHub repository owner (user or organization).
	Owner string `yaml:"owner"`

	// Repo is the GitHub repository name.
	Repo string `yaml:"repo"`

	// Asset is an optional template for the release asset filename.
	// Supports {version}, {os}, {arch} placeholders.
	// Example: "nomos-provider-file-{version}-{os}-{arch}"
	Asset string `yaml:"asset,omitempty"`
}

// ManifestLocalSource provides local filesystem source hints.
type ManifestLocalSource struct {
	// Path is the local filesystem path to the provider binary.
	Path string `yaml:"path"`
}

// Validate checks if the manifest is valid according to Nomos requirements.
// It returns an error if any validation rule is violated.
func (m *Manifest) Validate() error {
	if len(m.Providers) == 0 {
		return errors.New("manifest must contain at least one provider")
	}

	// Check for duplicate aliases and validate each provider
	seen := make(map[string]bool)
	for i, provider := range m.Providers {
		if provider.Alias == "" {
			return fmt.Errorf("provider at index %d: alias is required", i)
		}
		if provider.Type == "" {
			return fmt.Errorf("provider %q: type is required", provider.Alias)
		}

		// Validate source: cannot have both GitHub and Local
		if provider.Source.GitHub != nil && provider.Source.Local != nil {
			return fmt.Errorf("provider %q: cannot specify both GitHub and Local sources", provider.Alias)
		}

		// Validate GitHub source if present
		if gh := provider.Source.GitHub; gh != nil {
			if gh.Owner == "" {
				return fmt.Errorf("provider %q: GitHub source owner is required", provider.Alias)
			}
			if gh.Repo == "" {
				return fmt.Errorf("provider %q: GitHub source repo is required", provider.Alias)
			}
		}

		// Validate Local source if present
		if local := provider.Source.Local; local != nil {
			if local.Path == "" {
				return fmt.Errorf("provider %q: Local source path is required", provider.Alias)
			}
		}

		if seen[provider.Alias] {
			return fmt.Errorf("duplicate provider alias: %q", provider.Alias)
		}
		seen[provider.Alias] = true
	}

	return nil
}

// Save writes the manifest to the specified path as formatted YAML.
// It creates parent directories if they don't exist.
func (m *Manifest) Save(path string) error {
	// Validate before saving
	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// LoadManifest reads and parses a manifest from the specified path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate after loading
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// FindProvider searches for a provider by alias in the manifest.
// Returns the provider and true if found, or zero-value and false if not found.
func (m *Manifest) FindProvider(alias string) (ManifestProvider, bool) {
	for _, provider := range m.Providers {
		if provider.Alias == alias {
			return provider, true
		}
	}
	return ManifestProvider{}, false
}
