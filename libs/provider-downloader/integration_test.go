//go:build integration
// +build integration

package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestIntegration_FullDownloadFlow tests the complete resolve → download → install flow.
func TestIntegration_FullDownloadFlow(t *testing.T) {
	// Arrange: Create a mock GitHub API server
	providerContent := []byte("#!/bin/bash\necho 'provider'\n")
	expectedChecksum := computeSHA256(providerContent)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve provider binary
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "providers", "test-owner", "test-repo", "1.0.0")

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider",
		Name:     "test-provider-linux-amd64",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download and install
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	if result.Checksum != expectedChecksum {
		t.Errorf("expected checksum %s, got %s", expectedChecksum, result.Checksum)
	}

	// Verify file exists and is executable
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("failed to stat installed file: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %o", info.Mode().Perm())
	}

	// Verify content
	installedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}

	if string(installedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, installedContent)
	}
}

// TestIntegration_MultipleProviders tests downloading multiple providers in sequence.
func TestIntegration_MultipleProviders(t *testing.T) {
	// Arrange: Create mock servers for multiple providers
	provider1Content := []byte("provider-1-content")
	provider2Content := []byte("provider-2-content")
	provider3Content := []byte("provider-3-content")

	checksum1 := computeSHA256(provider1Content)
	checksum2 := computeSHA256(provider2Content)
	checksum3 := computeSHA256(provider3Content)

	// Single server that handles all providers
	//nolint:revive // unused parameter is required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/provider1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(provider1Content)
		case "/provider2":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(provider2Content)
		case "/provider3":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(provider3Content)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	// Act: Download all providers
	assets := []*AssetInfo{
		{
			URL:      server.URL + "/provider1",
			Name:     "provider1",
			Checksum: checksum1,
		},
		{
			URL:      server.URL + "/provider2",
			Name:     "provider2",
			Checksum: checksum2,
		},
		{
			URL:      server.URL + "/provider3",
			Name:     "provider3",
			Checksum: checksum3,
		},
	}

	results := make([]*InstallResult, 0, len(assets))
	for i, asset := range assets {
		destDir := filepath.Join(tmpDir, "provider", string(rune('1'+i)))
		result, err := client.DownloadAndInstall(context.Background(), asset, destDir)
		if err != nil {
			t.Fatalf("provider %d download failed: %v", i+1, err)
		}
		results = append(results, result)
	}

	// Assert: All providers installed successfully
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify each provider
	expectedContents := [][]byte{provider1Content, provider2Content, provider3Content}
	for i, result := range results {
		content, err := os.ReadFile(result.Path)
		if err != nil {
			t.Fatalf("failed to read provider %d: %v", i+1, err)
		}
		if string(content) != string(expectedContents[i]) {
			t.Errorf("provider %d content mismatch", i+1)
		}
	}
}

// TestIntegration_ConcurrentDownloads tests concurrent downloads with race detection.
func TestIntegration_ConcurrentDownloads(t *testing.T) {
	// This test should be run with -race flag: go test -race

	// Arrange: Create mock server
	providerContent := []byte("concurrent-provider")
	expectedChecksum := computeSHA256(providerContent)

	requestCount := 0
	var mu sync.Mutex

	//nolint:revive // unused parameter is required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	// Act: Download concurrently to different destinations
	const concurrency = 5
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			destDir := filepath.Join(tmpDir, "provider", string(rune('0'+index)))
			asset := &AssetInfo{
				URL:      server.URL + "/provider",
				Name:     "concurrent-provider",
				Checksum: expectedChecksum,
			}

			_, err := client.DownloadAndInstall(context.Background(), asset, destDir)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Assert: All downloads succeeded
	for err := range errors {
		t.Errorf("concurrent download failed: %v", err)
	}

	// Verify all providers were downloaded
	mu.Lock()
	actualCount := requestCount
	mu.Unlock()

	if actualCount != concurrency {
		t.Errorf("expected %d requests, got %d", concurrency, actualCount)
	}

	// Verify all files exist
	for i := 0; i < concurrency; i++ {
		providerPath := filepath.Join(tmpDir, "provider", string(rune('0'+i)), "provider")
		if _, err := os.Stat(providerPath); err != nil {
			t.Errorf("provider %d not found: %v", i, err)
		}
	}
}

// TestIntegration_ArchiveExtraction tests end-to-end archive download and extraction.
func TestIntegration_ArchiveExtraction(t *testing.T) {
	// Arrange: Create tar.gz archive
	providerContent := []byte("#!/bin/bash\necho 'archived provider'\n")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"bin/provider":  providerContent,
		"README.md":     []byte("# Provider"),
		"lib/helper.so": []byte("library"),
	})
	expectedChecksum := computeSHA256(archiveBytes)

	//nolint:revive // unused parameter is required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "provider")

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "test-provider-linux-amd64.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: Download and extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify extracted binary
	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify content
	extractedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extractedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, extractedContent)
	}

	// Verify permissions
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %o", info.Mode().Perm())
	}
}

// TestIntegration_CacheEfficiency tests that caching reduces server load.
func TestIntegration_CacheEfficiency(t *testing.T) {
	// Arrange: Setup server with request tracking
	providerContent := []byte("cached-provider-content")
	expectedChecksum := computeSHA256(providerContent)

	requestCount := 0
	var mu sync.Mutex

	//nolint:revive // unused parameter is required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		CacheDir:   cacheDir,
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider",
		Name:     "cached-provider",
		Checksum: expectedChecksum,
	}

	// Act: Download same provider 5 times to different locations
	const downloadCount = 5
	for i := 0; i < downloadCount; i++ {
		destDir := filepath.Join(tmpDir, "dest", string(rune('0'+i)))
		_, err := client.DownloadAndInstall(context.Background(), asset, destDir)
		if err != nil {
			t.Fatalf("download %d failed: %v", i+1, err)
		}
	}

	// Assert: Only 1 server request (rest from cache)
	mu.Lock()
	actualCount := requestCount
	mu.Unlock()

	if actualCount != 1 {
		t.Errorf("expected 1 server request (with caching), got %d", actualCount)
	}

	// Verify cache file exists
	cachedPath := filepath.Join(cacheDir, expectedChecksum)
	if _, err := os.Stat(cachedPath); err != nil {
		t.Errorf("expected cache file at %s, got error: %v", cachedPath, err)
	}

	// Verify all installations succeeded
	for i := 0; i < downloadCount; i++ {
		providerPath := filepath.Join(tmpDir, "dest", string(rune('0'+i)), "provider")
		if _, err := os.Stat(providerPath); err != nil {
			t.Errorf("installation %d not found: %v", i, err)
		}
	}
}

// TestIntegration_ContextCancellation tests that downloads respect context cancellation.
func TestIntegration_ContextCancellation(t *testing.T) {
	// Arrange: Create server with artificial delay
	//nolint:revive // unused parameter is required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow download
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("slow-provider"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "provider")

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider",
		Name: "slow-provider",
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Act: Attempt download with cancelled context
	_, err := client.DownloadAndInstall(ctx, asset, destDir)

	// Assert: Should fail with context error
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	// Should be a context deadline exceeded error
	if !strings.Contains(err.Error(), "context") {
		t.Logf("Warning: expected context error, got: %v", err)
	}
}
