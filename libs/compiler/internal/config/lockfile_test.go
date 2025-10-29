package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/config"
)

// TestLockfileProvider_JSONMarshalUnmarshal tests JSON serialization of a Provider entry.
// RED: This test will fail until we implement the Lockfile and Provider structs.
func TestLockfileProvider_JSONMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		provider config.Provider
	}{
		{
			name: "complete provider entry",
			provider: config.Provider{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Source:   config.ProviderSource{GitHub: &config.GitHubSource{Owner: "autonomous-bits", Repo: "nomos-provider-file", Asset: "nomos-provider-file-0.2.0-darwin-arm64"}},
				Checksum: "sha256:abcdef1234567890",
				Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
			},
		},
		{
			name: "provider with local source",
			provider: config.Provider{
				Alias:    "http",
				Type:     "http",
				Version:  "1.0.0",
				OS:       "linux",
				Arch:     "amd64",
				Source:   config.ProviderSource{Local: &config.LocalSource{Path: "/usr/local/bin/nomos-provider-http"}},
				Checksum: "sha256:fedcba0987654321",
				Path:     ".nomos/providers/http/1.0.0/linux-amd64/provider",
			},
		},
		{
			name: "minimal provider entry",
			provider: config.Provider{
				Alias:   "minimal",
				Type:    "minimal",
				Version: "0.1.0",
				OS:      "darwin",
				Arch:    "amd64",
				Path:    ".nomos/providers/minimal/0.1.0/darwin-amd64/provider",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.provider)
			if err != nil {
				t.Fatalf("failed to marshal provider: %v", err)
			}

			// Unmarshal back
			var unmarshaled config.Provider
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal provider: %v", err)
			}

			// Compare key fields
			if unmarshaled.Alias != tt.provider.Alias {
				t.Errorf("alias: got %q, want %q", unmarshaled.Alias, tt.provider.Alias)
			}
			if unmarshaled.Type != tt.provider.Type {
				t.Errorf("type: got %q, want %q", unmarshaled.Type, tt.provider.Type)
			}
			if unmarshaled.Version != tt.provider.Version {
				t.Errorf("version: got %q, want %q", unmarshaled.Version, tt.provider.Version)
			}
			if unmarshaled.OS != tt.provider.OS {
				t.Errorf("os: got %q, want %q", unmarshaled.OS, tt.provider.OS)
			}
			if unmarshaled.Arch != tt.provider.Arch {
				t.Errorf("arch: got %q, want %q", unmarshaled.Arch, tt.provider.Arch)
			}
			if unmarshaled.Path != tt.provider.Path {
				t.Errorf("path: got %q, want %q", unmarshaled.Path, tt.provider.Path)
			}
		})
	}
}

// TestLockfile_JSONMarshalUnmarshal tests JSON serialization of a complete Lockfile.
// RED: This test will fail until we implement the Lockfile struct.
func TestLockfile_JSONMarshalUnmarshal(t *testing.T) {
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Source:   config.ProviderSource{GitHub: &config.GitHubSource{Owner: "autonomous-bits", Repo: "nomos-provider-file"}},
				Checksum: "sha256:abc123",
				Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
			},
		},
	}

	// Marshal
	data, err := json.MarshalIndent(lockfile, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal lockfile: %v", err)
	}

	// Unmarshal
	var unmarshaled config.Lockfile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal lockfile: %v", err)
	}

	// Validate
	if len(unmarshaled.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(unmarshaled.Providers))
	}

	provider := unmarshaled.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("alias: got %q, want %q", provider.Alias, "configs")
	}
	if provider.Version != "0.2.0" {
		t.Errorf("version: got %q, want %q", provider.Version, "0.2.0")
	}
}

// TestLockfile_Validate tests validation rules for lockfile entries.
// RED: This test will fail until we implement Validate() method.
func TestLockfile_Validate(t *testing.T) {
	tests := []struct {
		name      string
		lockfile  config.Lockfile
		wantError bool
	}{
		{
			name: "valid lockfile",
			lockfile: config.Lockfile{
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
			},
			wantError: false,
		},
		{
			name: "missing alias",
			lockfile: config.Lockfile{
				Providers: []config.Provider{
					{
						Type:    "file",
						Version: "0.2.0",
						OS:      "darwin",
						Arch:    "arm64",
						Path:    ".nomos/providers/file/0.2.0/darwin-arm64/provider",
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing version",
			lockfile: config.Lockfile{
				Providers: []config.Provider{
					{
						Alias: "configs",
						Type:  "file",
						OS:    "darwin",
						Arch:  "arm64",
						Path:  ".nomos/providers/file/0.2.0/darwin-arm64/provider",
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing path",
			lockfile: config.Lockfile{
				Providers: []config.Provider{
					{
						Alias:   "configs",
						Type:    "file",
						Version: "0.2.0",
						OS:      "darwin",
						Arch:    "arm64",
					},
				},
			},
			wantError: true,
		},
		{
			name: "duplicate aliases",
			lockfile: config.Lockfile{
				Providers: []config.Provider{
					{
						Alias:   "configs",
						Type:    "file",
						Version: "0.2.0",
						OS:      "darwin",
						Arch:    "arm64",
						Path:    ".nomos/providers/file/0.2.0/darwin-arm64/provider",
					},
					{
						Alias:   "configs",
						Type:    "http",
						Version: "1.0.0",
						OS:      "darwin",
						Arch:    "arm64",
						Path:    ".nomos/providers/http/1.0.0/darwin-arm64/provider",
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lockfile.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestLockfile_ReadWrite tests reading and writing lockfiles from disk.
// RED: This test will fail until we implement Load() and Save() methods.
func TestLockfile_ReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	lockfilePath := filepath.Join(tempDir, ".nomos", "providers.lock.json")

	// Create a lockfile
	lockfile := config.Lockfile{
		Providers: []config.Provider{
			{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Checksum: "sha256:test",
				Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
			},
		},
	}

	// Write lockfile
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		t.Fatalf("lockfile not created at %s", lockfilePath)
	}

	// Read lockfile back
	loaded, err := config.LoadLockfile(lockfilePath)
	if err != nil {
		t.Fatalf("failed to load lockfile: %v", err)
	}

	// Validate contents
	if len(loaded.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(loaded.Providers))
	}

	provider := loaded.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("alias: got %q, want %q", provider.Alias, "configs")
	}
	if provider.Version != "0.2.0" {
		t.Errorf("version: got %q, want %q", provider.Version, "0.2.0")
	}
}

// TestProvider_BinaryPath tests the path helper for constructing binary paths.
// RED: This test will fail until we implement BinaryPath() method.
func TestProvider_BinaryPath(t *testing.T) {
	tests := []struct {
		name     string
		provider config.Provider
		baseDir  string
		want     string
	}{
		{
			name: "standard path",
			provider: config.Provider{
				Type:    "file",
				Version: "0.2.0",
				OS:      "darwin",
				Arch:    "arm64",
			},
			baseDir: ".nomos/providers",
			want:    ".nomos/providers/file/0.2.0/darwin-arm64/provider",
		},
		{
			name: "linux amd64",
			provider: config.Provider{
				Type:    "http",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			baseDir: "/tmp/providers",
			want:    "/tmp/providers/http/1.0.0/linux-amd64/provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.BinaryPath(tt.baseDir)
			if got != tt.want {
				t.Errorf("BinaryPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
