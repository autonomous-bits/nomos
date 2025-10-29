package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/config"
	"gopkg.in/yaml.v3"
)

// TestManifestProvider_YAMLMarshalUnmarshal tests YAML serialization of a manifest provider entry.
// RED: This test will fail until we implement the Manifest and ManifestProvider structs.
func TestManifestProvider_YAMLMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		provider config.ManifestProvider
	}{
		{
			name: "provider with GitHub source",
			provider: config.ManifestProvider{
				Alias: "configs",
				Type:  "file",
				Source: config.ManifestSource{
					GitHub: &config.ManifestGitHubSource{
						Owner: "autonomous-bits",
						Repo:  "nomos-provider-file",
						Asset: "nomos-provider-file-{version}-{os}-{arch}",
					},
				},
				Config: map[string]any{
					"directory": "./apps/command-line/testdata/configs",
				},
			},
		},
		{
			name: "provider with local source",
			provider: config.ManifestProvider{
				Alias: "http",
				Type:  "http",
				Source: config.ManifestSource{
					Local: &config.ManifestLocalSource{
						Path: "/usr/local/bin/nomos-provider-http",
					},
				},
			},
		},
		{
			name: "minimal provider (no source or config)",
			provider: config.ManifestProvider{
				Alias: "minimal",
				Type:  "minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to YAML
			data, err := yaml.Marshal(tt.provider)
			if err != nil {
				t.Fatalf("failed to marshal provider: %v", err)
			}

			// Unmarshal back
			var unmarshaled config.ManifestProvider
			if err := yaml.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal provider: %v", err)
			}

			// Compare key fields
			if unmarshaled.Alias != tt.provider.Alias {
				t.Errorf("alias: got %q, want %q", unmarshaled.Alias, tt.provider.Alias)
			}
			if unmarshaled.Type != tt.provider.Type {
				t.Errorf("type: got %q, want %q", unmarshaled.Type, tt.provider.Type)
			}
		})
	}
}

// TestManifest_YAMLMarshalUnmarshal tests YAML serialization of a complete Manifest.
// RED: This test will fail until we implement the Manifest struct.
func TestManifest_YAMLMarshalUnmarshal(t *testing.T) {
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
				Config: map[string]any{
					"directory": "./testdata/configs",
				},
			},
			{
				Alias: "http",
				Type:  "http",
				Source: config.ManifestSource{
					Local: &config.ManifestLocalSource{
						Path: "/usr/local/bin/nomos-provider-http",
					},
				},
			},
		},
	}

	// Marshal
	data, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	// Unmarshal
	var unmarshaled config.Manifest
	if err := yaml.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Validate
	if len(unmarshaled.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(unmarshaled.Providers))
	}

	// Check first provider
	provider := unmarshaled.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("alias: got %q, want %q", provider.Alias, "configs")
	}
	if provider.Source.GitHub == nil {
		t.Error("expected GitHub source, got nil")
	}
}

// TestManifest_Validate tests validation rules for manifest entries.
// RED: This test will fail until we implement Validate() method.
func TestManifest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		manifest  config.Manifest
		wantError bool
	}{
		{
			name: "valid manifest",
			manifest: config.Manifest{
				Providers: []config.ManifestProvider{
					{
						Alias: "configs",
						Type:  "file",
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing alias",
			manifest: config.Manifest{
				Providers: []config.ManifestProvider{
					{
						Type: "file",
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing type",
			manifest: config.Manifest{
				Providers: []config.ManifestProvider{
					{
						Alias: "configs",
					},
				},
			},
			wantError: true,
		},
		{
			name: "duplicate aliases",
			manifest: config.Manifest{
				Providers: []config.ManifestProvider{
					{
						Alias: "configs",
						Type:  "file",
					},
					{
						Alias: "configs",
						Type:  "http",
					},
				},
			},
			wantError: true,
		},
		{
			name: "both GitHub and local sources",
			manifest: config.Manifest{
				Providers: []config.ManifestProvider{
					{
						Alias: "configs",
						Type:  "file",
						Source: config.ManifestSource{
							GitHub: &config.ManifestGitHubSource{Owner: "owner", Repo: "repo"},
							Local:  &config.ManifestLocalSource{Path: "/path"},
						},
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestManifest_ReadWrite tests reading and writing manifests from disk.
// RED: This test will fail until we implement Load() and Save() methods.
func TestManifest_ReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, ".nomos", "providers.yaml")

	// Create a manifest
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
		},
	}

	// Write manifest
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatalf("manifest not created at %s", manifestPath)
	}

	// Read manifest back
	loaded, err := config.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	// Validate contents
	if len(loaded.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(loaded.Providers))
	}

	provider := loaded.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("alias: got %q, want %q", provider.Alias, "configs")
	}
	if provider.Type != "file" {
		t.Errorf("type: got %q, want %q", provider.Type, "file")
	}
}

// TestManifest_FindProvider tests finding a provider by alias.
// RED: This test will fail until we implement FindProvider() method.
func TestManifest_FindProvider(t *testing.T) {
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

	tests := []struct {
		name      string
		alias     string
		wantFound bool
		wantType  string
	}{
		{
			name:      "existing provider",
			alias:     "configs",
			wantFound: true,
			wantType:  "file",
		},
		{
			name:      "another existing provider",
			alias:     "http",
			wantFound: true,
			wantType:  "http",
		},
		{
			name:      "non-existent provider",
			alias:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, found := manifest.FindProvider(tt.alias)
			if found != tt.wantFound {
				t.Errorf("FindProvider() found = %v, want %v", found, tt.wantFound)
			}
			if found && provider.Type != tt.wantType {
				t.Errorf("FindProvider() type = %q, want %q", provider.Type, tt.wantType)
			}
		})
	}
}
