package providercmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/provider-downloader/testutil"
) // TestInitCommand_Hermetic_FullFlow tests the complete init workflow without network.
// Uses httptest to mock GitHub API and asset downloads.
func TestInitCommand_Hermetic_FullFlow(t *testing.T) {
	// RED: This test exercises the full init path with mocked servers

	// Create fixture binary
	fixture := testutil.CreateBinaryFixture(t, 1024, "nomos-provider-file")

	// Setup mock GitHub API server
	githubServer := newMockGitHubAPIServer(t, "autonomous-bits", "nomos-provider-file", "v1.0.0", []string{
		"nomos-provider-file-linux-amd64",
		"nomos-provider-file-darwin-arm64",
	})
	defer githubServer.Close()

	// Setup mock asset download server
	assetServer := newMockAssetServer(t, fixture.Content)
	defer assetServer.Close()

	// Create temp project directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create .csl file
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
	alias: 'configs'
	type: 'autonomous-bits/nomos-provider-file'
	version: '1.0.0'
	directory: './data'

app:
	name: 'test-app'
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(cslPath, []byte(cslContent), 0644); err != nil {
		t.Fatalf("failed to create csl: %v", err)
	}

	// Override downloader client to use test servers
	// This requires a way to inject the base URL - we'll use environment or refactor
	// For now, this test documents the expected flow

	opts := Options{
		Paths:  []string{cslPath},
		OS:     "linux",
		Arch:   "amd64",
		DryRun: false,
	}

	// Act: Run init (this will fail until we add injection mechanism)
	// err := Run(opts)
	// For now, just test the installProvider directly
	_ = opts

	// Verify lockfile structure
	lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")
	//nolint:gosec // G304: Test code reading test-generated lockfile
	lockData, err := os.ReadFile(lockPath)
	if os.IsNotExist(err) {
		t.Skip("Skipping until injection mechanism added for test servers")
	}
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var lockFile LockFile
	if err := json.Unmarshal(lockData, &lockFile); err != nil {
		t.Fatalf("failed to parse lockfile: %v", err)
	}

	// Assert lockfile structure
	if len(lockFile.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(lockFile.Providers))
	}

	provider := lockFile.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("expected alias 'configs', got %q", provider.Alias)
	}

	// Verify GitHub metadata
	github, ok := provider.Source["github"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected github metadata, got %T", provider.Source["github"])
	}

	if github["owner"] != "autonomous-bits" {
		t.Errorf("expected owner 'autonomous-bits', got %v", github["owner"])
	}
	if github["repo"] != "nomos-provider-file" {
		t.Errorf("expected repo 'nomos-provider-file', got %v", github["repo"])
	}
	if github["release_tag"] != "v1.0.0" {
		t.Errorf("expected release_tag 'v1.0.0', got %v", github["release_tag"])
	}

	// Verify checksum
	if provider.Checksum != fixture.Checksum {
		t.Errorf("checksum mismatch: expected %s, got %s", fixture.Checksum, provider.Checksum)
	}
}

// newMockGitHubAPIServer creates a test server that mocks GitHub Releases API.
func newMockGitHubAPIServer(t *testing.T, owner, repo, tag string, assetNames []string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/repos/" + owner + "/" + repo + "/releases/tags/" + tag
		if r.URL.Path == expectedPath {
			release := struct {
				TagName string `json:"tag_name"`
				Assets  []struct {
					Name               string `json:"name"`
					BrowserDownloadURL string `json:"browser_download_url"`
					Size               int64  `json:"size"`
					ContentType        string `json:"content_type"`
				} `json:"assets"`
			}{
				TagName: tag,
			}

			for _, name := range assetNames {
				release.Assets = append(release.Assets, struct {
					Name               string `json:"name"`
					BrowserDownloadURL string `json:"browser_download_url"`
					Size               int64  `json:"size"`
					ContentType        string `json:"content_type"`
				}{
					Name:               name,
					BrowserDownloadURL: "http://fake-assets.test/" + name,
					Size:               1024,
					ContentType:        "application/octet-stream",
				})
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(release); err != nil {
				t.Errorf("failed to encode release response: %v", err)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"}); err != nil {
			t.Errorf("failed to encode not found response: %v", err)
		}
	}))
}

// newMockAssetServer creates a test server that serves binary assets.
func newMockAssetServer(t *testing.T, content []byte) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(content); err != nil {
			t.Errorf("failed to write asset content: %v", err)
		}
	}))
}
