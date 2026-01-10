package providercmd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestEnsureProviders_EmptyPaths tests error handling for empty paths.
func TestEnsureProviders_EmptyPaths(t *testing.T) {
	opts := ProviderOptions{
		Paths: []string{},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	summary, err := EnsureProviders(opts)

	if err == nil {
		t.Fatal("expected error for empty paths, got nil")
	}
	if !contains(err.Error(), "no input paths") {
		t.Errorf("error = %q, want to contain 'no input paths'", err.Error())
	}
	if summary != nil {
		t.Errorf("summary should be nil on error, got %+v", summary)
	}
}

// TestEnsureProviders_NoProviders tests handling of files with no providers.
func TestEnsureProviders_NoProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.csl")

	// Create config with no source declarations
	configContent := `database:
  host: localhost
  port: 5432
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	opts := ProviderOptions{
		Paths: []string{configPath},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	summary, err := EnsureProviders(opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Fatal("summary should not be nil")
	}
	if summary.Total != 0 {
		t.Errorf("Total = %d, want 0", summary.Total)
	}
}

// TestEnsureProviders_MissingVersion tests validation of missing version.
func TestEnsureProviders_MissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.csl")

	// Create config with provider missing version
	configContent := `source:
  alias: 'test'
  type: 'owner/repo'
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	opts := ProviderOptions{
		Paths: []string{configPath},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	summary, err := EnsureProviders(opts)

	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
	if !errors.Is(err, ErrMissingVersion) {
		t.Errorf("error should be ErrMissingVersion, got %v", err)
	}
	if summary != nil {
		t.Errorf("summary should be nil on validation error, got %+v", summary)
	}
}

// TestEnsureProviders_VersionConflict tests detection of version conflicts.
func TestEnsureProviders_VersionConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two config files with conflicting versions
	config1Path := filepath.Join(tmpDir, "config1.csl")
	config1Content := `source:
  alias: 'aws'
  type: 'owner/repo'
  version: '1.0.0'
`
	if err := os.WriteFile(config1Path, []byte(config1Content), 0600); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}

	config2Path := filepath.Join(tmpDir, "config2.csl")
	config2Content := `source:
  alias: 'gcp'
  type: 'owner/repo'
  version: '2.0.0'
`
	if err := os.WriteFile(config2Path, []byte(config2Content), 0600); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	opts := ProviderOptions{
		Paths: []string{config1Path, config2Path},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	summary, err := EnsureProviders(opts)

	if err == nil {
		t.Fatal("expected error for version conflict, got nil")
	}
	if !errors.Is(err, ErrVersionConflict) {
		t.Errorf("error should be ErrVersionConflict, got %v", err)
	}
	if !contains(err.Error(), "1.0.0") || !contains(err.Error(), "2.0.0") {
		t.Errorf("error should mention conflicting versions, got %q", err.Error())
	}
	if summary != nil {
		t.Errorf("summary should be nil on validation error, got %+v", summary)
	}
}

// TestEnsureProviders_DryRun tests dry-run mode preview.
func TestEnsureProviders_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configPath := filepath.Join(tmpDir, "config.csl")
	configContent := `source:
  alias: 'test'
  type: 'owner/repo'
  version: '1.0.0'
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	opts := ProviderOptions{
		Paths:  []string{configPath},
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		DryRun: true,
	}

	summary, err := EnsureProviders(opts)

	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
	if summary == nil {
		t.Fatal("summary should not be nil")
	}

	// In dry-run, providers should be counted but not downloaded
	if summary.Total != 1 {
		t.Errorf("Total = %d, want 1", summary.Total)
	}

	// Verify no lockfile was created
	lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lockfile should not be created in dry-run mode")
	}
}

// TestValidateProviderVersions tests the version validation helper.
func TestValidateProviderVersions(t *testing.T) {
	tests := []struct {
		name      string
		providers []DiscoveredProvider
		wantErr   bool
		errIs     error
	}{
		{
			name: "all providers have versions",
			providers: []DiscoveredProvider{
				{Alias: "aws", Type: "owner/repo1", Version: "1.0.0"},
				{Alias: "gcp", Type: "owner/repo2", Version: "2.0.0"},
			},
			wantErr: false,
		},
		{
			name: "one provider missing version",
			providers: []DiscoveredProvider{
				{Alias: "aws", Type: "owner/repo1", Version: "1.0.0"},
				{Alias: "gcp", Type: "owner/repo2", Version: ""},
			},
			wantErr: true,
			errIs:   ErrMissingVersion,
		},
		{
			name:      "empty providers list",
			providers: []DiscoveredProvider{},
			wantErr:   false,
		},
		{
			name: "single provider with version",
			providers: []DiscoveredProvider{
				{Alias: "test", Type: "owner/repo", Version: "1.0.0"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProviderVersions(tt.providers)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateProviderVersions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("error should be %v, got %v", tt.errIs, err)
			}
		})
	}
}

// TestDetectVersionConflicts tests the version conflict detection helper.
func TestDetectVersionConflicts(t *testing.T) {
	tests := []struct {
		name      string
		providers []DiscoveredProvider
		wantErr   bool
		errIs     error
	}{
		{
			name: "no conflicts - different types",
			providers: []DiscoveredProvider{
				{Alias: "aws", Type: "owner/repo1", Version: "1.0.0"},
				{Alias: "gcp", Type: "owner/repo2", Version: "2.0.0"},
			},
			wantErr: false,
		},
		{
			name: "no conflicts - same type same version",
			providers: []DiscoveredProvider{
				{Alias: "aws1", Type: "owner/repo", Version: "1.0.0"},
				{Alias: "aws2", Type: "owner/repo", Version: "1.0.0"},
			},
			wantErr: false,
		},
		{
			name: "conflict - same type different versions",
			providers: []DiscoveredProvider{
				{Alias: "aws1", Type: "owner/repo", Version: "1.0.0"},
				{Alias: "aws2", Type: "owner/repo", Version: "2.0.0"},
			},
			wantErr: true,
			errIs:   ErrVersionConflict,
		},
		{
			name: "conflict - three different versions",
			providers: []DiscoveredProvider{
				{Alias: "p1", Type: "owner/repo", Version: "1.0.0"},
				{Alias: "p2", Type: "owner/repo", Version: "2.0.0"},
				{Alias: "p3", Type: "owner/repo", Version: "3.0.0"},
			},
			wantErr: true,
			errIs:   ErrVersionConflict,
		},
		{
			name:      "empty providers list",
			providers: []DiscoveredProvider{},
			wantErr:   false,
		},
		{
			name: "single provider",
			providers: []DiscoveredProvider{
				{Alias: "test", Type: "owner/repo", Version: "1.0.0"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detectVersionConflicts(tt.providers)

			if (err != nil) != tt.wantErr {
				t.Errorf("detectVersionConflicts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errIs != nil && !errors.Is(err, tt.errIs) {
				t.Errorf("error should be %v, got %v", tt.errIs, err)
			}
		})
	}
}

// TestBuildSummary tests the summary builder helper.
func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name    string
		results []ProviderResult
		want    ProviderSummary
	}{
		{
			name:    "empty results",
			results: []ProviderResult{},
			want: ProviderSummary{
				Total:      0,
				Cached:     0,
				Downloaded: 0,
				Failed:     0,
			},
		},
		{
			name: "all cached",
			results: []ProviderResult{
				{Status: ProviderStatusSkipped},
				{Status: ProviderStatusSkipped},
			},
			want: ProviderSummary{
				Total:      2,
				Cached:     2,
				Downloaded: 0,
				Failed:     0,
			},
		},
		{
			name: "all downloaded",
			results: []ProviderResult{
				{Status: ProviderStatusInstalled},
				{Status: ProviderStatusInstalled},
			},
			want: ProviderSummary{
				Total:      2,
				Cached:     0,
				Downloaded: 2,
				Failed:     0,
			},
		},
		{
			name: "all failed",
			results: []ProviderResult{
				{Status: ProviderStatusFailed},
				{Status: ProviderStatusFailed},
			},
			want: ProviderSummary{
				Total:      2,
				Cached:     0,
				Downloaded: 0,
				Failed:     2,
			},
		},
		{
			name: "mixed statuses",
			results: []ProviderResult{
				{Status: ProviderStatusSkipped},
				{Status: ProviderStatusInstalled},
				{Status: ProviderStatusFailed},
			},
			want: ProviderSummary{
				Total:      3,
				Cached:     1,
				Downloaded: 1,
				Failed:     1,
			},
		},
		{
			name: "dry-run results don't count",
			results: []ProviderResult{
				{Status: ProviderStatusDryRun},
				{Status: ProviderStatusDryRun},
			},
			want: ProviderSummary{
				Total:      2,
				Cached:     0,
				Downloaded: 0,
				Failed:     0,
			},
		},
		{
			name: "mixed with dry-run",
			results: []ProviderResult{
				{Status: ProviderStatusInstalled},
				{Status: ProviderStatusDryRun},
				{Status: ProviderStatusSkipped},
			},
			want: ProviderSummary{
				Total:      3,
				Cached:     1,
				Downloaded: 1,
				Failed:     0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSummary(tt.results)

			if got == nil {
				t.Fatal("buildSummary returned nil")
			}

			if got.Total != tt.want.Total {
				t.Errorf("Total = %d, want %d", got.Total, tt.want.Total)
			}
			if got.Cached != tt.want.Cached {
				t.Errorf("Cached = %d, want %d", got.Cached, tt.want.Cached)
			}
			if got.Downloaded != tt.want.Downloaded {
				t.Errorf("Downloaded = %d, want %d", got.Downloaded, tt.want.Downloaded)
			}
			if got.Failed != tt.want.Failed {
				t.Errorf("Failed = %d, want %d", got.Failed, tt.want.Failed)
			}
		})
	}
}
