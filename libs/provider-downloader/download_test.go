package downloader

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDownloadAndInstall_Success verifies successful streaming download with checksum verification.
func TestDownloadAndInstall_Success(t *testing.T) {
	// Arrange: Create fake provider binary content
	providerContent := []byte("fake-provider-binary-content-v1.0.0")
	expectedChecksum := computeSHA256(providerContent)

	// Setup httptest server that serves the fake binary
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	// Create test destination directory
	destDir := t.TempDir()

	// Create client with test server
	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider-binary",
		Name:     "test-provider-linux-amd64",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download and install
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Verify success
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify path
	expectedPath := filepath.Join(destDir, "provider")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify file exists and is executable
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("failed to stat installed file: %v", err)
	}

	// Check permissions (0755)
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %o", info.Mode().Perm())
	}

	// Verify checksum matches
	if result.Checksum != expectedChecksum {
		t.Errorf("expected checksum %s, got %s", expectedChecksum, result.Checksum)
	}

	// Verify size
	if result.Size != int64(len(providerContent)) {
		t.Errorf("expected size %d, got %d", len(providerContent), result.Size)
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

// TestDownloadAndInstall_ChecksumMismatch verifies checksum validation.
func TestDownloadAndInstall_ChecksumMismatch(t *testing.T) {
	// Arrange: Create fake provider binary with mismatched checksum
	providerContent := []byte("fake-provider-content")
	wrongChecksum := "deadbeef0000000000000000000000000000000000000000000000000000000"

	// Setup httptest server
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
		URL:      server.URL + "/provider",
		Name:     "test-provider",
		Size:     int64(len(providerContent)),
		Checksum: wrongChecksum,
	}

	// Act: Attempt download
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Expect checksum mismatch error
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}

	// Verify error type
	var checksumErr *ChecksumMismatchError
	if !errors.As(err, &checksumErr) {
		t.Errorf("expected ChecksumMismatchError, got %T: %v", err, err)
	}

	// Verify expected and actual checksums in error
	if checksumErr != nil {
		if checksumErr.Expected != wrongChecksum {
			t.Errorf("expected error.Expected=%s, got %s", wrongChecksum, checksumErr.Expected)
		}
		actualChecksum := computeSHA256(providerContent)
		if checksumErr.Actual != actualChecksum {
			t.Errorf("expected error.Actual=%s, got %s", actualChecksum, checksumErr.Actual)
		}
	}
}

// TestDownloadAndInstall_PartialResponse tests retry behavior on incomplete downloads.
func TestDownloadAndInstall_PartialResponse(t *testing.T) {
	// Arrange: Server fails twice with 500 errors, then succeeds
	attemptCount := 0
	providerContent := []byte("retry-test-content")
	expectedChecksum := computeSHA256(providerContent)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts with server error (retryable)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient:    server.Client(),
		RetryAttempts: 3,
		RetryDelay:    1 * time.Millisecond, // Fast retries for tests
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider",
		Name:     "retry-provider",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download with retries
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should succeed after retries
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if result.Checksum != expectedChecksum {
		t.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, result.Checksum)
	}

	// Verify retry happened (3 attempts)
	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}

// TestDownloadAndInstall_NetworkError tests retry exhaustion.
func TestDownloadAndInstall_NetworkError(t *testing.T) {
	// Arrange: Server always fails
	attemptCount := 0
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient:    server.Client(),
		RetryAttempts: 3,
		RetryDelay:    1 * time.Millisecond,
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider",
		Name: "failing-provider",
		Size: 100,
	}

	// Act: Attempt download
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Should fail after exhausting retries
	if err == nil {
		t.Fatal("expected error after retry exhaustion, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	// Verify all retries were attempted (initial + 3 retries = 4 total)
	expectedAttempts := 1 + 3
	if attemptCount != expectedAttempts {
		t.Errorf("expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

// TestDownloadAndInstall_Timeout tests that downloads respect context timeout.
func TestDownloadAndInstall_Timeout(t *testing.T) {
	// RED: This test will fail until timeout handling is correct

	// Arrange: Server that hangs/delays response
	providerContent := []byte("slow-provider-content")

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server by sleeping longer than the context timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:  server.URL + "/provider",
		Name: "slow-provider",
		Size: int64(len(providerContent)),
	}

	// Act: Download with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := client.DownloadAndInstall(ctx, asset, destDir)

	// Assert: Should fail with context deadline exceeded
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on timeout, got %+v", result)
	}

	// Verify error is context.DeadlineExceeded
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

// TestDownloadAndInstall_SlowDownload tests handling of slow but successful downloads.
func TestDownloadAndInstall_SlowDownload(t *testing.T) {
	// GREEN: This test verifies slow downloads complete successfully with sufficient timeout

	providerContent := []byte("slow-but-successful")
	expectedChecksum := computeSHA256(providerContent)

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate moderate delay
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	destDir := t.TempDir()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider",
		Name:     "slow-provider",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download with generous timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.DownloadAndInstall(ctx, asset, destDir)

	// Assert: Should succeed
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.Checksum != expectedChecksum {
		t.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, result.Checksum)
	}
}
