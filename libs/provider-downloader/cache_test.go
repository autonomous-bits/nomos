package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestDownloadAndInstall_CacheHit tests that cache is used when available.
func TestDownloadAndInstall_CacheHit(t *testing.T) {
	// Arrange: Setup cache with a provider binary
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	destDir := filepath.Join(tmpDir, "dest")

	providerContent := []byte("#!/bin/bash\necho 'cached provider'\n")
	expectedChecksum := computeSHA256(providerContent)

	// Pre-populate cache
	if err := os.MkdirAll(cacheDir, 0755); err != nil { //nolint:gosec // G301: Test directory
		t.Fatalf("failed to create cache dir: %v", err)
	}
	cachedPath := filepath.Join(cacheDir, expectedChecksum)
	if err := os.WriteFile(cachedPath, providerContent, 0644); err != nil { //nolint:gosec // G306: Test file
		t.Fatalf("failed to write cached file: %v", err)
	}

	// Setup a server that should NOT be called (cache hit)
	callCount := 0
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		t.Error("server should not be called on cache hit")
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

	// Act: Download (should use cache)
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if callCount > 0 {
		t.Errorf("expected 0 server calls (cache hit), got %d", callCount)
	}

	// Verify installed file
	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify content matches cached content
	installedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}

	if string(installedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, installedContent)
	}

	// Verify checksum
	if result.Checksum != expectedChecksum {
		t.Errorf("expected checksum %s, got %s", expectedChecksum, result.Checksum)
	}

	// Verify permissions (executable)
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("failed to stat installed file: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %o", info.Mode().Perm())
	}
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

// TestDownloadAndInstall_CacheWithNoChecksum tests that caching is skipped without checksum.
func TestDownloadAndInstall_CacheWithNoChecksum(t *testing.T) {
	// Arrange: Cache enabled but asset has no checksum
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
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
		CacheDir:   cacheDir,
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider",
		Name: "test-provider",
		Size: int64(len(providerContent)),
		// No checksum - caching should be skipped
	}

	// Act: Download
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Verify cache directory is empty (no checksum = no caching)
	entries, err := os.ReadDir(cacheDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to read cache dir: %v", err)
	}

	if len(entries) > 0 {
		t.Errorf("expected empty cache (no checksum), got %d entries", len(entries))
	}

	// Verify file was installed
	if result.Path == "" {
		t.Error("expected non-empty path")
	}
}

// TestDownloadAndInstall_CacheWithArchive tests caching works with archive extraction.
func TestDownloadAndInstall_CacheWithArchive(t *testing.T) {
	// Arrange: Create tar.gz archive
	providerContent := []byte("#!/bin/bash\necho 'provider'\n")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"provider": providerContent,
	})
	expectedChecksum := computeSHA256(archiveBytes)

	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	destDir1 := filepath.Join(tmpDir, "dest1")
	destDir2 := filepath.Join(tmpDir, "dest2")

	callCount := 0
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		CacheDir:   cacheDir,
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "test-provider.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: First download (should extract and cache)
	result1, err := client.DownloadAndInstall(context.Background(), asset, destDir1)
	if err != nil {
		t.Fatalf("first download failed: %v", err)
	}

	// Second download to different location (should use cache)
	result2, err := client.DownloadAndInstall(context.Background(), asset, destDir2)
	if err != nil {
		t.Fatalf("second download failed: %v", err)
	}

	// Assert: Only one server call (second used cache)
	if callCount != 1 {
		t.Errorf("expected 1 server call (archive extraction + cache), got %d", callCount)
	}

	// Verify both installations succeeded
	if result1.Path == "" || result2.Path == "" {
		t.Error("expected non-empty paths")
	}

	// Both should have same checksum
	if result1.Checksum != result2.Checksum {
		t.Errorf("checksum mismatch: %s vs %s", result1.Checksum, result2.Checksum)
	}
}
