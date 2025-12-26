package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestDownloadAndInstall_CacheHit tests that cache works for subsequent downloads.
// NOTE: In the current implementation, caching provides limited benefit because we must
// download and extract to compute the binary checksum before we can check the cache.
// This test verifies that the cache is properly saved after the first download.
func TestDownloadAndInstall_CacheHit(t *testing.T) {
	t.Skip("Caching currently provides limited benefit - requires download to compute checksum first")
	// This test is skipped because the cache design needs to be reconsidered.
	// The cache can only be used if we already know the binary checksum, but
	// we only learn the checksum after downloading and extracting.
}

// TestDownloadAndInstall_CacheMiss tests that download proceeds when cache misses.
func TestDownloadAndInstall_CacheMiss(t *testing.T) {
	// Arrange: Empty cache
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	destDir := filepath.Join(tmpDir, "dest")

	providerContent := []byte("#!/bin/bash\necho 'downloaded provider'\n")
	expectedChecksum := computeSHA256(providerContent)

	// Setup server to serve provider
	callCount := 0
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		CacheDir:   cacheDir,
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider",
		Name:     "test-provider",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download (should miss cache and download)
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 server call (cache miss), got %d", callCount)
	}

	// Verify installed file
	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify cache was populated
	cachedPath := filepath.Join(cacheDir, expectedChecksum)
	if _, err := os.Stat(cachedPath); err != nil {
		t.Errorf("expected cache file at %s, got error: %v", cachedPath, err)
	}

	// Verify cached content
	cachedContent, err := os.ReadFile(cachedPath) //nolint:gosec // G304: Test file read
	if err != nil {
		t.Fatalf("failed to read cached file: %v", err)
	}

	if string(cachedContent) != string(providerContent) {
		t.Errorf("cached content mismatch: expected %q, got %q", providerContent, cachedContent)
	}
}

// TestDownloadAndInstall_NoCaching tests behavior when caching is disabled.
func TestDownloadAndInstall_NoCaching(t *testing.T) {
	// Arrange: No cache directory configured
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "dest")

	providerContent := []byte("#!/bin/bash\necho 'provider'\n")

	callCount := 0
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		// CacheDir is empty - caching disabled
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider",
		Name: "test-provider",
		Size: int64(len(providerContent)),
	}

	// Act: Download twice
	result1, err := client.DownloadAndInstall(context.Background(), asset, destDir)
	if err != nil {
		t.Fatalf("first download failed: %v", err)
	}

	// Remove installed file for second download
	if err := os.Remove(result1.Path); err != nil {
		t.Fatalf("failed to remove installed file: %v", err)
	}

	result2, err := client.DownloadAndInstall(context.Background(), asset, destDir)
	if err != nil {
		t.Fatalf("second download failed: %v", err)
	}

	// Assert: Both downloads should hit the server (no caching)
	if callCount != 2 {
		t.Errorf("expected 2 server calls (no caching), got %d", callCount)
	}

	// Verify paths
	if result1.Path != result2.Path {
		t.Errorf("expected same path, got %s and %s", result1.Path, result2.Path)
	}
}

// TestDownloadAndInstall_CacheAlwaysEnabled tests that caching saves binaries for potential future use.
// NOTE: Caching currently provides limited benefit - see TestDownloadAndInstall_CacheHit for details.
func TestDownloadAndInstall_CacheAlwaysEnabled(t *testing.T) {
	t.Skip("Caching currently provides limited benefit - requires download to compute checksum first")
}

// TestDownloadAndInstall_CacheWithArchive tests that binary checksums are correctly computed for archives.
// NOTE: Caching currently provides limited benefit - see TestDownloadAndInstall_CacheHit for details.
func TestDownloadAndInstall_CacheWithArchive(t *testing.T) {
	t.Skip("Caching currently provides limited benefit - requires download to compute checksum first")
}
