package downloader

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDownloadAndInstall_TarGzExtraction tests successful extraction of tar.gz archives.
func TestDownloadAndInstall_TarGzExtraction(t *testing.T) {
	// Arrange: Create a tar.gz archive containing a provider binary
	providerContent := []byte("#!/bin/bash\necho 'fake provider'\n")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"provider": providerContent,
	})

	// Compute expected checksum for the archive itself
	expectedArchiveChecksum := computeSHA256(archiveBytes)

	// Setup httptest server that serves the tar.gz archive
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "test-provider-linux-amd64.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedArchiveChecksum,
	}

	// Act: Download and extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Verify success
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify extracted binary exists
	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify file exists and is executable
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("failed to stat extracted file: %v", err)
	}

	// Check permissions (0755)
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %o", info.Mode().Perm())
	}

	// Verify content matches
	extractedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extractedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, extractedContent)
	}
}

// TestDownloadAndInstall_TarGzWithNestedDirectories tests extraction with nested directories.
func TestDownloadAndInstall_TarGzWithNestedDirectories(t *testing.T) {
	// Arrange: Create a tar.gz with nested directory structure
	providerContent := []byte("provider-binary-content")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"bin/provider":        providerContent,
		"lib/libhelper.so":    []byte("library-content"),
		"docs/README.md":      []byte("# Provider Docs"),
		"nested/dir/file.txt": []byte("some file"),
	})

	expectedChecksum := computeSHA256(archiveBytes)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "test-provider.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: Download and extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should find provider binary in nested directory
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify extracted provider binary (should be flattened to destDir/provider)
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

	// Note: Other files in the archive are NOT extracted to destDir.
	// The downloader only extracts the provider binary itself.
	// This is correct behavior - we don't want to pollute destDir with
	// auxiliary files from the archive.
}

// TestDownloadAndInstall_TarGzCorrupted tests handling of corrupted archives.
func TestDownloadAndInstall_TarGzCorrupted(t *testing.T) {
	// Arrange: Create corrupted gzip data
	corruptedData := []byte("not-a-valid-gzip-archive-just-random-bytes")

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(corruptedData)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider.tar.gz",
		Name: "corrupted-provider.tar.gz",
		Size: int64(len(corruptedData)),
		// No checksum - let extraction fail
	}

	// Act: Attempt to download and extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should fail with extraction error
	if err == nil {
		t.Fatal("expected extraction error for corrupted archive, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}

	// Verify error message mentions extraction or gzip
	errMsg := err.Error()
	if !strings.Contains(errMsg, "extract") && !strings.Contains(errMsg, "gzip") {
		t.Errorf("expected error message to mention extraction or gzip, got: %s", errMsg)
	}
}

// TestDownloadAndInstall_TarGzBinaryNotFound tests when archive doesn't contain expected binary.
func TestDownloadAndInstall_TarGzBinaryNotFound(t *testing.T) {
	// Arrange: Create archive without "provider" binary
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"README.md":    []byte("# Documentation"),
		"config.yaml":  []byte("key: value"),
		"other-binary": []byte("not-the-provider"),
	})

	expectedChecksum := computeSHA256(archiveBytes)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "no-provider-binary.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: Attempt to extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should fail with "provider binary not found" error
	if err == nil {
		t.Fatal("expected error when provider binary not found, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}

	// Verify error message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "provider binary not found") {
		t.Errorf("expected error message about provider binary not found, got: %s", errMsg)
	}
}

// TestDownloadAndInstall_TarGzMultipleExecutables tests handling of multiple provider binaries.
func TestDownloadAndInstall_TarGzMultipleExecutables(t *testing.T) {
	// Arrange: Create archive with multiple potential provider binaries
	providerContent := []byte("correct-provider-binary")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"provider":             providerContent,                // Should be selected (exact match)
		"nomos-provider-file":  []byte("alternative-provider"), // Also valid name
		"bin/other-executable": []byte("other-binary"),
	})

	expectedChecksum := computeSHA256(archiveBytes)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tar.gz",
		Name:     "multi-binary.tar.gz",
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: Extract archive
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should prefer "provider" over other names
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the correct binary was selected
	extractedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extractedContent) != string(providerContent) {
		t.Errorf("expected 'provider' binary to be selected, but content doesn't match: got %q", extractedContent)
	}
}

// TestDownloadAndInstall_TgzExtension tests .tgz extension (shorthand for .tar.gz).
func TestDownloadAndInstall_TgzExtension(t *testing.T) {
	// Arrange: Create a .tgz archive
	providerContent := []byte("tgz-provider-content")
	archiveBytes := createTarGzArchive(t, map[string][]byte{
		"provider": providerContent,
	})

	expectedChecksum := computeSHA256(archiveBytes)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider.tgz",
		Name:     "test-provider.tgz", // .tgz extension
		Size:     int64(len(archiveBytes)),
		Checksum: expectedChecksum,
	}

	// Act: Download and extract
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should extract successfully
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify content
	extractedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extractedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, extractedContent)
	}
}

// TestDownloadAndInstall_NonArchiveFile tests that non-archive files are handled normally.
func TestDownloadAndInstall_NonArchiveFile(t *testing.T) {
	// Arrange: Plain binary file (not an archive)
	providerContent := []byte("plain-binary-content")
	expectedChecksum := computeSHA256(providerContent)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider-binary",
		Name:     "test-provider-linux-amd64", // No archive extension
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download (should NOT attempt extraction)
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should install as-is without extraction
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify content is unchanged
	installedContent, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}

	if string(installedContent) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, installedContent)
	}
}

// Helper: createTarGzArchive creates a tar.gz archive with the given files.
func createTarGzArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	// Create a buffer to hold the archive
	var buf []byte
	tmpFile, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// Create gzip writer
	gzw := gzip.NewWriter(tmpFile)
	defer func() { _ = gzw.Close() }()

	// Create tar writer
	tw := tar.NewWriter(gzw)
	defer func() { _ = tw.Close() }()

	// Add files to archive
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}

		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", name, err)
		}

		if _, err := tw.Write(content); err != nil {
			t.Fatalf("failed to write tar content for %s: %v", name, err)
		}
	}

	// Close writers to flush
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	// Read the archive bytes
	buf, err = os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read archive: %v", err)
	}

	return buf
}
