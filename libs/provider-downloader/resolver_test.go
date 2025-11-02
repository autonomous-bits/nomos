package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

// TestResolveAsset_ExactPatternMatch tests that the resolver correctly selects
// an asset when it matches exact naming patterns.
func TestResolveAsset_ExactPatternMatch(t *testing.T) {
	tests := []struct {
		name          string
		spec          *ProviderSpec
		releaseAssets []string
		expectedAsset string
	}{
		{
			name: "repo-os-arch pattern",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "test-provider",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			releaseAssets: []string{
				"test-provider-linux-amd64",
				"test-provider-darwin-amd64",
				"test-provider-windows-amd64",
			},
			expectedAsset: "test-provider-linux-amd64",
		},
		{
			name: "nomos-provider-os-arch pattern",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "nomos-provider-file",
				Version: "1.0.0",
				OS:      "darwin",
				Arch:    "arm64",
			},
			releaseAssets: []string{
				"nomos-provider-file-linux-amd64",
				"nomos-provider-file-darwin-arm64",
				"nomos-provider-file-windows-amd64",
			},
			expectedAsset: "nomos-provider-file-darwin-arm64",
		},
		{
			name: "repo-os pattern without arch",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "test-provider",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			releaseAssets: []string{
				"test-provider-linux",
				"test-provider-darwin",
			},
			expectedAsset: "test-provider-linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create httptest server that returns a mocked release
			server := newMockGitHubServer(t, tt.spec.Owner, tt.spec.Repo, tt.spec.Version, tt.releaseAssets)
			defer server.Close()

			// Create client with test server
			client := NewClient(context.Background(), &ClientOptions{
				BaseURL: server.URL,
			})

			// Act: resolve the asset
			asset, err := client.ResolveAsset(context.Background(), tt.spec)

			// Assert: no error and correct asset selected
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if asset.Name != tt.expectedAsset {
				t.Errorf("expected asset %q, got %q", tt.expectedAsset, asset.Name)
			}
		})
	}
}

// TestResolveAsset_AutoDetectOSArch tests that OS and Arch are auto-detected
// from runtime when not specified in the spec.
func TestResolveAsset_AutoDetectOSArch(t *testing.T) {
	spec := &ProviderSpec{
		Owner:   "test-owner",
		Repo:    "test-provider",
		Version: "1.0.0",
		// OS and Arch are empty, should be auto-detected
	}

	expectedOS := runtime.GOOS
	expectedArch := runtime.GOARCH
	expectedAsset := "test-provider-" + expectedOS + "-" + expectedArch

	releaseAssets := []string{
		"test-provider-linux-amd64",
		"test-provider-darwin-amd64",
		"test-provider-darwin-arm64",
		"test-provider-windows-amd64",
		expectedAsset, // Include the expected runtime asset
	}

	server := newMockGitHubServer(t, spec.Owner, spec.Repo, spec.Version, releaseAssets)
	defer server.Close()

	client := NewClient(context.Background(), &ClientOptions{
		BaseURL: server.URL,
	})

	asset, err := client.ResolveAsset(context.Background(), spec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if asset.Name != expectedAsset {
		t.Errorf("expected asset %q, got %q", expectedAsset, asset.Name)
	}
}

// TestResolveAsset_SubstringFallback tests that the resolver falls back to
// substring matching when exact patterns don't match.
func TestResolveAsset_SubstringFallback(t *testing.T) {
	tests := []struct {
		name          string
		spec          *ProviderSpec
		releaseAssets []string
		expectedAsset string
	}{
		{
			name: "finds asset with os and arch substrings",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "custom-provider",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			releaseAssets: []string{
				"provider_binary_linux_x86_64",
				"provider_binary_darwin_x86_64",
			},
			expectedAsset: "provider_binary_linux_x86_64",
		},
		{
			name: "case insensitive matching",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "custom-provider",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
			},
			releaseAssets: []string{
				"Provider-Linux-AMD64",
				"Provider-Darwin-AMD64",
			},
			expectedAsset: "Provider-Linux-AMD64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockGitHubServer(t, tt.spec.Owner, tt.spec.Repo, tt.spec.Version, tt.releaseAssets)
			defer server.Close()

			client := NewClient(context.Background(), &ClientOptions{
				BaseURL: server.URL,
			})

			asset, err := client.ResolveAsset(context.Background(), tt.spec)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if asset.Name != tt.expectedAsset {
				t.Errorf("expected asset %q, got %q", tt.expectedAsset, asset.Name)
			}
		})
	}
}

// TestResolveAsset_NotFound tests that appropriate errors are returned
// when no matching asset is found.
func TestResolveAsset_NotFound(t *testing.T) {
	spec := &ProviderSpec{
		Owner:   "test-owner",
		Repo:    "test-provider",
		Version: "1.0.0",
		OS:      "linux",
		Arch:    "amd64",
	}

	// Release has no matching assets
	releaseAssets := []string{
		"test-provider-darwin-arm64",
		"test-provider-windows-amd64",
	}

	server := newMockGitHubServer(t, spec.Owner, spec.Repo, spec.Version, releaseAssets)
	defer server.Close()

	client := NewClient(context.Background(), &ClientOptions{
		BaseURL: server.URL,
	})

	asset, err := client.ResolveAsset(context.Background(), spec)

	// Assert: should return AssetNotFoundError
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFoundErr *AssetNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected AssetNotFoundError, got %T: %v", err, err)
	}

	if asset != nil {
		t.Errorf("expected nil asset, got %+v", asset)
	}
}

// TestResolveAsset_VersionNormalization tests that version formats are
// normalized correctly (e.g., "1.0.0" vs "v1.0.0").
func TestResolveAsset_VersionNormalization(t *testing.T) {
	tests := []struct {
		name           string
		specVersion    string
		releaseTagName string
		shouldMatch    bool
	}{
		{
			name:           "v-prefix added",
			specVersion:    "1.0.0",
			releaseTagName: "v1.0.0",
			shouldMatch:    true,
		},
		{
			name:           "v-prefix removed",
			specVersion:    "v1.0.0",
			releaseTagName: "1.0.0",
			shouldMatch:    true,
		},
		{
			name:           "exact match",
			specVersion:    "v1.0.0",
			releaseTagName: "v1.0.0",
			shouldMatch:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ProviderSpec{
				Owner:   "test-owner",
				Repo:    "test-provider",
				Version: tt.specVersion,
				OS:      "linux",
				Arch:    "amd64",
			}

			releaseAssets := []string{"test-provider-linux-amd64"}

			server := newMockGitHubServerWithTag(t, spec.Owner, spec.Repo, tt.releaseTagName, releaseAssets)
			defer server.Close()

			client := NewClient(context.Background(), &ClientOptions{
				BaseURL: server.URL,
			})

			asset, err := client.ResolveAsset(context.Background(), spec)

			if tt.shouldMatch {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if asset == nil {
					t.Fatal("expected asset, got nil")
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			}
		})
	}
}

// TestResolveAsset_InvalidSpec tests validation of ProviderSpec.
func TestResolveAsset_InvalidSpec(t *testing.T) {
	tests := []struct {
		name string
		spec *ProviderSpec
	}{
		{
			name: "nil spec",
			spec: nil,
		},
		{
			name: "missing owner",
			spec: &ProviderSpec{
				Repo:    "test-provider",
				Version: "1.0.0",
			},
		},
		{
			name: "missing repo",
			spec: &ProviderSpec{
				Owner:   "test-owner",
				Version: "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(context.Background(), nil)

			asset, err := client.ResolveAsset(context.Background(), tt.spec)

			// Assert: should return InvalidSpecError
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var invalidErr *InvalidSpecError
			if !errors.As(err, &invalidErr) {
				t.Errorf("expected InvalidSpecError, got %T: %v", err, err)
			}

			if asset != nil {
				t.Errorf("expected nil asset, got %+v", asset)
			}
		})
	}
}

// Helper: newMockGitHubServer creates an httptest server that mocks GitHub API responses.
func newMockGitHubServer(t *testing.T, owner, repo, version string, assetNames []string) *httptest.Server {
	t.Helper()
	return newMockGitHubServerWithTag(t, owner, repo, version, assetNames)
}

// Helper: newMockGitHubServerWithTag creates an httptest server with explicit tag name.
func newMockGitHubServerWithTag(t *testing.T, owner, repo, tag string, assetNames []string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock GET /repos/:owner/:repo/releases/tags/:tag
		expectedPath := "/repos/" + owner + "/" + repo + "/releases/tags/" + tag
		if r.URL.Path == expectedPath {
			release := mockRelease{
				TagName: tag,
				Assets:  make([]mockAsset, 0, len(assetNames)),
			}

			for i, name := range assetNames {
				release.Assets = append(release.Assets, mockAsset{
					Name:               name,
					BrowserDownloadURL: "https://github.com/" + owner + "/" + repo + "/releases/download/" + tag + "/" + name,
					Size:               int64(1024 * (i + 1)),
					ContentType:        "application/octet-stream",
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(release)
			return
		}

		// Mock GET /repos/:owner/:repo/releases/latest (for when version is empty)
		latestPath := "/repos/" + owner + "/" + repo + "/releases/latest"
		if r.URL.Path == latestPath {
			release := mockRelease{
				TagName: tag,
				Assets:  make([]mockAsset, 0, len(assetNames)),
			}

			for i, name := range assetNames {
				release.Assets = append(release.Assets, mockAsset{
					Name:               name,
					BrowserDownloadURL: "https://github.com/" + owner + "/" + repo + "/releases/download/" + tag + "/" + name,
					Size:               int64(1024 * (i + 1)),
					ContentType:        "application/octet-stream",
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(release)
			return
		}

		// Not found
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	}))
}

// mockRelease represents a GitHub release response.
type mockRelease struct {
	TagName string      `json:"tag_name"`
	Assets  []mockAsset `json:"assets"`
}

// mockAsset represents a GitHub release asset.
type mockAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}
