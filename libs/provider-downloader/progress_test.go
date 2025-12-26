package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestProgressCallback_Success tests that progress callback is called during download.
func TestProgressCallback_Success(t *testing.T) {
	// Arrange: Create fake provider binary content
	providerContent := []byte("fake-provider-binary-content-for-progress-test")
	expectedChecksum := computeSHA256(providerContent)

	// Setup httptest server that serves the fake binary
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", string(rune(len(providerContent))))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	// Create test destination directory
	destDir := t.TempDir()

	// Track progress updates
	var mu sync.Mutex
	var progressUpdates []struct {
		downloaded int64
		total      int64
	}

	progressCallback := func(downloaded, total int64) {
		mu.Lock()
		defer mu.Unlock()
		progressUpdates = append(progressUpdates, struct {
			downloaded int64
			total      int64
		}{downloaded, total})
	}

	// Create client with progress callback
	client := NewClient(&ClientOptions{
		HTTPClient:       server.Client(),
		BaseURL:          server.URL,
		ProgressCallback: progressCallback,
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
		t.Fatal("expected non-nil result, got nil")
	}

	// Verify progress updates were received
	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) == 0 {
		t.Error("expected progress updates, got none")
	}

	// Verify final progress update has correct total bytes
	if len(progressUpdates) > 0 {
		lastUpdate := progressUpdates[len(progressUpdates)-1]
		if lastUpdate.downloaded != int64(len(providerContent)) {
			t.Errorf("expected final downloaded bytes %d, got %d", len(providerContent), lastUpdate.downloaded)
		}
	}
}

// TestProgressCallback_NoCallback tests that download works without progress callback.
func TestProgressCallback_NoCallback(t *testing.T) {
	// Arrange: Create fake provider binary content
	providerContent := []byte("fake-provider-binary-content-no-callback")
	expectedChecksum := computeSHA256(providerContent)

	// Setup httptest server
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(providerContent)
	}))
	defer server.Close()

	// Create test destination directory
	destDir := t.TempDir()

	// Create client WITHOUT progress callback
	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		// No ProgressCallback set
	})

	asset := &AssetInfo{
		URL:      server.URL + "/provider-binary",
		Name:     "test-provider-linux-amd64",
		Size:     int64(len(providerContent)),
		Checksum: expectedChecksum,
	}

	// Act: Download and install
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert: Verify success (should work without callback)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result, got nil")
	}
}

// TestHTTPTimeout_CustomValue tests that custom HTTP timeout is respected.
func TestHTTPTimeout_CustomValue(t *testing.T) {
	customTimeout := 5 * time.Second

	client := NewClient(&ClientOptions{
		HTTPTimeout: customTimeout,
	})

	if client == nil {
		t.Fatal("expected non-nil client, got nil")
	}

	if client.httpClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}

	if client.httpClient.Timeout != customTimeout {
		t.Errorf("expected HTTP timeout %v, got %v", customTimeout, client.httpClient.Timeout)
	}
}

// TestHTTPTimeout_DefaultValue tests that default HTTP timeout is used.
func TestHTTPTimeout_DefaultValue(t *testing.T) {
	client := NewClient(&ClientOptions{
		// No HTTPTimeout specified
	})

	if client == nil {
		t.Fatal("expected non-nil client, got nil")
	}

	if client.httpClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}

	expectedTimeout := 30 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("expected default HTTP timeout %v, got %v", expectedTimeout, client.httpClient.Timeout)
	}
}

// TestHTTPTimeout_ZeroValue tests that zero timeout falls back to default.
func TestHTTPTimeout_ZeroValue(t *testing.T) {
	client := NewClient(&ClientOptions{
		HTTPTimeout: 0, // Explicitly set to zero
	})

	if client == nil {
		t.Fatal("expected non-nil client, got nil")
	}

	if client.httpClient == nil {
		t.Fatal("expected non-nil HTTP client")
	}

	expectedTimeout := 30 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("expected default HTTP timeout %v for zero value, got %v", expectedTimeout, client.httpClient.Timeout)
	}
}

// TestDefaultClientOptions_IncludesHTTPTimeout tests that default options include HTTPTimeout.
func TestDefaultClientOptions_IncludesHTTPTimeout(t *testing.T) {
	opts := DefaultClientOptions()

	if opts == nil {
		t.Fatal("expected non-nil options, got nil")
	}

	expectedTimeout := 30 * time.Second
	if opts.HTTPTimeout != expectedTimeout {
		t.Errorf("expected default HTTPTimeout %v, got %v", expectedTimeout, opts.HTTPTimeout)
	}
}

// TestProgressCallback_LargeDownload tests progress callback with larger downloads.
func TestProgressCallback_LargeDownload(t *testing.T) {
	// Create larger content to trigger multiple progress updates
	largeContent := make([]byte, 1024*1024) // 1 MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	expectedChecksum := computeSHA256(largeContent)

	// Setup httptest server
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(largeContent)
	}))
	defer server.Close()

	destDir := t.TempDir()

	var mu sync.Mutex
	var progressUpdates []int64

	progressCallback := func(downloaded, total int64) {
		mu.Lock()
		defer mu.Unlock()
		progressUpdates = append(progressUpdates, downloaded)
	}

	client := NewClient(&ClientOptions{
		HTTPClient:       server.Client(),
		BaseURL:          server.URL,
		ProgressCallback: progressCallback,
	})

	asset := &AssetInfo{
		URL:      server.URL + "/large-provider",
		Name:     "large-provider-linux-amd64",
		Size:     int64(len(largeContent)),
		Checksum: expectedChecksum,
	}

	// Act
	result, err := client.DownloadAndInstall(context.Background(), asset, destDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result, got nil")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) == 0 {
		t.Error("expected progress updates for large download, got none")
	}

	// Verify progress is monotonically increasing
	for i := 1; i < len(progressUpdates); i++ {
		if progressUpdates[i] < progressUpdates[i-1] {
			t.Errorf("progress should be monotonically increasing, got %d after %d",
				progressUpdates[i], progressUpdates[i-1])
		}
	}

	// Verify final progress equals total size
	if len(progressUpdates) > 0 {
		finalProgress := progressUpdates[len(progressUpdates)-1]
		if finalProgress != int64(len(largeContent)) {
			t.Errorf("expected final progress %d, got %d", len(largeContent), finalProgress)
		}
	}
}
