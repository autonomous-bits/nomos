package config_test

import (
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/config"
)

// TestResolver_ResolveProvider tests the core resolver functionality.
// RED: This test will fail until we implement Resolver and ResolveProvider().
func TestResolver_ResolveProvider(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Create a lockfile
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Checksum: "sha256:abc123",
				Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
				Source: config.ProviderSource{
					GitHub: &config.GitHubSource{
						Owner: "autonomous-bits",
						Repo:  "nomos-provider-file",
					},
				},
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create a manifest with an additional provider not in lockfile
	manifest := config.Manifest{
		Providers: []config.ManifestProvider{
			{
				Alias: "configs",
				Type:  "file",
				Source: config.ManifestSource{
					GitHub: &config.ManifestGitHubSource{
						Owner: "autonomous-bits",
						Repo:  "nomos-provider-file",
					},
				},
			},
			{
				Alias: "http",
				Type:  "http",
				Source: config.ManifestSource{
					GitHub: &config.ManifestGitHubSource{
						Owner: "autonomous-bits",
						Repo:  "nomos-provider-http",
					},
				},
			},
		},
	}
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Create resolver
	resolver, err := config.NewResolver(lockfilePath, manifestPath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	tests := []struct {
		name      string
		alias     string
		wantFound bool
		wantType  string
		wantPath  string
	}{
		{
			name:      "provider in lockfile",
			alias:     "configs",
			wantFound: true,
			wantType:  "file",
			wantPath:  ".nomos/providers/file/0.2.0/darwin-arm64/provider",
		},
		{
			name:      "provider only in manifest",
			alias:     "http",
			wantFound: true,
			wantType:  "http",
			wantPath:  "", // No path yet since not in lockfile
		},
		{
			name:      "non-existent provider",
			alias:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveProvider(tt.alias)
			if !tt.wantFound {
				if err == nil {
					t.Errorf("ResolveProvider() expected error for %q, got nil", tt.alias)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveProvider() unexpected error: %v", err)
			}

			if result.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", result.Type, tt.wantType)
			}

			if tt.wantPath != "" && result.Path != tt.wantPath {
				t.Errorf("path: got %q, want %q", result.Path, tt.wantPath)
			}
		})
	}
}

// TestResolver_Precedence tests resolution precedence rules.
// RED: This test will fail until we implement proper precedence handling.
func TestResolver_Precedence(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Create lockfile with one version
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
				Checksum: "sha256:lockfile-checksum",
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create manifest with different config
	manifest := config.Manifest{
		Providers: []config.ManifestProvider{
			{
				Alias: "configs",
				Type:  "file",
				Config: map[string]any{
					"directory": "./testdata",
				},
			},
		},
	}
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	resolver, err := config.NewResolver(lockfilePath, manifestPath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Resolve provider
	result, err := resolver.ResolveProvider("configs")
	if err != nil {
		t.Fatalf("ResolveProvider() failed: %v", err)
	}

	// Lockfile should take precedence for version and path
	if result.Version != "0.2.0" {
		t.Errorf("version: got %q, want %q (from lockfile)", result.Version, "0.2.0")
	}
	if result.Path != ".nomos/providers/file/0.2.0/darwin-arm64/provider" {
		t.Errorf("path: got %q, want path from lockfile", result.Path)
	}

	// Manifest config should be available
	if result.Config == nil {
		t.Error("expected config from manifest, got nil")
	} else if dir, ok := result.Config["directory"].(string); !ok || dir != "./testdata" {
		t.Errorf("config directory: got %v, want ./testdata", result.Config["directory"])
	}
}

// TestResolver_LockfileOnly tests resolver with only lockfile (no manifest).
// RED: This test will fail until we implement handling of missing manifest.
func TestResolver_LockfileOnly(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml") // Doesn't exist

	// Create lockfile
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:   "configs",
				Type:    "file",
				Version: "0.2.0",
				OS:      "darwin",
				Arch:    "arm64",
				Path:    ".nomos/providers/file/0.2.0/darwin-arm64/provider",
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create resolver (manifest doesn't exist, should be OK)
	resolver, err := config.NewResolver(lockfilePath, manifestPath)
	if err != nil {
		t.Fatalf("failed to create resolver with missing manifest: %v", err)
	}

	// Should resolve from lockfile
	result, err := resolver.ResolveProvider("configs")
	if err != nil {
		t.Fatalf("ResolveProvider() failed: %v", err)
	}

	if result.Type != "file" {
		t.Errorf("type: got %q, want %q", result.Type, "file")
	}
}

// TestResolver_ManifestOnly tests resolver with only manifest (no lockfile).
// RED: This test will fail until we implement handling of missing lockfile.
func TestResolver_ManifestOnly(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json") // Doesn't exist
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Create manifest
	manifest := config.Manifest{
		Providers: []config.ManifestProvider{
			{
				Alias: "configs",
				Type:  "file",
			},
		},
	}
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Create resolver (lockfile doesn't exist, should be OK)
	resolver, err := config.NewResolver(lockfilePath, manifestPath)
	if err != nil {
		t.Fatalf("failed to create resolver with missing lockfile: %v", err)
	}

	// Should resolve from manifest
	result, err := resolver.ResolveProvider("configs")
	if err != nil {
		t.Fatalf("ResolveProvider() failed: %v", err)
	}

	if result.Type != "file" {
		t.Errorf("type: got %q, want %q", result.Type, "file")
	}

	// Path should be empty (no lockfile)
	if result.Path != "" {
		t.Errorf("path: got %q, want empty (no lockfile)", result.Path)
	}
}

// TestResolver_GetAllProviders tests listing all known providers.
// RED: This test will fail until we implement GetAllProviders().
func TestResolver_GetAllProviders(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Create lockfile
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:   "configs",
				Type:    "file",
				Version: "0.2.0",
				OS:      "darwin",
				Arch:    "arm64",
				Path:    ".nomos/providers/file/0.2.0/darwin-arm64/provider",
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create manifest with additional provider
	manifest := config.Manifest{
		Providers: []config.ManifestProvider{
			{
				Alias: "configs",
				Type:  "file",
			},
			{
				Alias: "http",
				Type:  "http",
			},
		},
	}
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	resolver, err := config.NewResolver(lockfilePath, manifestPath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Get all providers
	providers := resolver.GetAllProviders()

	// Should have both providers (configs from lockfile/manifest, http from manifest)
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	// Check aliases
	aliases := make(map[string]bool)
	for _, p := range providers {
		aliases[p.Alias] = true
	}

	if !aliases["configs"] {
		t.Error("expected 'configs' provider in result")
	}
	if !aliases["http"] {
		t.Error("expected 'http' provider in result")
	}
}

// TestNewResolver_MissingBoth tests error when both files are missing.
// RED: This test will fail until we implement proper error handling.
func TestNewResolver_MissingBoth(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Neither file exists
	_, err := config.NewResolver(lockfilePath, manifestPath)
	if err == nil {
		t.Error("expected error when both lockfile and manifest are missing, got nil")
	}
}
