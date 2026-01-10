package providercmd

import (
	"runtime"
	"testing"
)

// TestDownloadProviders_DryRun tests dry-run mode returns preview without downloads.
func TestDownloadProviders_DryRun(t *testing.T) {
	providers := []DiscoveredProvider{
		{Alias: "aws", Type: "owner/repo", Version: "1.0.0"},
		{Alias: "gcp", Type: "owner/repo2", Version: "2.0.0"},
	}

	opts := ProviderOptions{
		DryRun: true,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	}

	results, err := DownloadProviders(providers, opts)

	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	for _, result := range results {
		if result.Status != ProviderStatusDryRun {
			t.Errorf("status = %q, want %q", result.Status, ProviderStatusDryRun)
		}
	}
}

// TestDownloadProviders_EmptyList tests handling of empty provider list.
func TestDownloadProviders_EmptyList(t *testing.T) {
	providers := []DiscoveredProvider{}

	opts := ProviderOptions{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	results, err := DownloadProviders(providers, opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

// TestFindProviderInLockfile tests the lockfile search helper.
func TestFindProviderInLockfile(t *testing.T) {
	lock := &LockFile{
		Providers: []ProviderEntry{
			{
				Alias:   "aws",
				Type:    "owner/repo",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			{
				Alias:   "gcp",
				Type:    "owner/repo2",
				Version: "2.0.0",
				OS:      "darwin",
				Arch:    "arm64",
			},
			{
				Alias:   "aws",
				Type:    "owner/repo",
				Version: "1.0.0",
				OS:      "darwin",
				Arch:    "arm64",
			},
		},
	}

	tests := []struct {
		name     string
		alias    string
		provType string
		version  string
		os       string
		arch     string
		wantNil  bool
	}{
		{
			name:     "exact match",
			alias:    "aws",
			provType: "owner/repo",
			version:  "1.0.0",
			os:       "linux",
			arch:     "amd64",
			wantNil:  false,
		},
		{
			name:     "different os/arch",
			alias:    "aws",
			provType: "owner/repo",
			version:  "1.0.0",
			os:       "darwin",
			arch:     "arm64",
			wantNil:  false,
		},
		{
			name:     "alias not found",
			alias:    "notfound",
			provType: "owner/repo",
			version:  "1.0.0",
			os:       "linux",
			arch:     "amd64",
			wantNil:  true,
		},
		{
			name:     "version not found",
			alias:    "aws",
			provType: "owner/repo",
			version:  "9.9.9",
			os:       "linux",
			arch:     "amd64",
			wantNil:  true,
		},
		{
			name:     "os not found",
			alias:    "aws",
			provType: "owner/repo",
			version:  "1.0.0",
			os:       "windows",
			arch:     "amd64",
			wantNil:  true,
		},
		{
			name:     "arch not found",
			alias:    "aws",
			provType: "owner/repo",
			version:  "1.0.0",
			os:       "linux",
			arch:     "arm64",
			wantNil:  true,
		},
		{
			name:     "type not found",
			alias:    "aws",
			provType: "other/repo",
			version:  "1.0.0",
			os:       "linux",
			arch:     "amd64",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findProviderInLockfile(lock, tt.alias, tt.provType, tt.version, tt.os, tt.arch)

			if (result == nil) != tt.wantNil {
				t.Errorf("findProviderInLockfile() = %v, wantNil %v", result, tt.wantNil)
			}

			if !tt.wantNil && result != nil {
				// Verify the returned entry matches
				if result.Alias != tt.alias {
					t.Errorf("alias = %q, want %q", result.Alias, tt.alias)
				}
				if result.Version != tt.version {
					t.Errorf("version = %q, want %q", result.Version, tt.version)
				}
			}
		})
	}
}

// TestFindProviderInLockfile_NilLockfile tests handling of nil lockfile.
func TestFindProviderInLockfile_NilLockfile(t *testing.T) {
	// This test is more about documenting behavior - the function will panic
	// if lockfile is nil, which is expected since callers should check for nil first
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil lockfile, got none")
		}
	}()

	findProviderInLockfile(nil, "aws", "owner/repo", "1.0.0", "linux", "amd64")
}

// TestFindProviderInLockfile_EmptyLockfile tests searching in empty lockfile.
func TestFindProviderInLockfile_EmptyLockfile(t *testing.T) {
	lock := &LockFile{
		Providers: []ProviderEntry{},
	}

	result := findProviderInLockfile(lock, "aws", "owner/repo", "1.0.0", "linux", "amd64")

	if result != nil {
		t.Errorf("expected nil for empty lockfile, got %+v", result)
	}
}

// Note: Testing actual download functionality requires integration tests with GitHub API.
// The downloadProvider function is tested in integration tests with the --integration build tag.
// Unit tests for DownloadProviders focus on:
// 1. Dry-run behavior (tested above)
// 2. Lockfile checking logic (tested via findProviderInLockfile)
// 3. Force mode logic (requires mocking, deferred to integration tests)
// 4. Result aggregation (tested indirectly through ensure_test.go)
