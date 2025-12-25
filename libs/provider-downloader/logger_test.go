package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testLogger is a simple logger implementation for testing.
type testLogger struct {
	messages []string
}

func (l *testLogger) Debugf(format string, _ ...interface{}) {
	l.messages = append(l.messages, format)
}

// TestClient_WithLogger verifies that the logger is called when configured.
func TestClient_WithLogger(t *testing.T) {
	// Arrange: Setup a logger and test server
	logger := &testLogger{}

	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"tag_name": "v1.0.0",
			"assets": [
				{
					"name": "test-provider-linux-amd64",
					"browser_download_url": "https://example.com/provider",
					"size": 1024,
					"content_type": "application/octet-stream"
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		Logger:     logger,
	})

	spec := &ProviderSpec{
		Owner:   "test-owner",
		Repo:    "test-provider",
		Version: "1.0.0",
		OS:      "linux",
		Arch:    "amd64",
	}

	// Act: Resolve asset (which should trigger debug logging)
	asset, err := client.ResolveAsset(context.Background(), spec)

	// Assert: Verify logging occurred
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if asset == nil {
		t.Fatal("expected asset, got nil")
	}

	// Verify logger was called
	if len(logger.messages) == 0 {
		t.Fatal("expected logger to be called, but no messages were logged")
	}

	// Verify some expected log messages
	expectedPatterns := []string{
		"GitHub API request",
		"GitHub API response",
		"Searching for asset matching",
		"Matched asset",
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, msg := range logger.messages {
			if strings.Contains(msg, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected log message containing %q, but not found in: %v", pattern, logger.messages)
		}
	}
}

// TestClient_WithoutLogger verifies that no panics occur when logger is nil.
func TestClient_WithoutLogger(t *testing.T) {
	// Arrange: Create client without logger
	//nolint:revive // unused parameter 'r' required by http.HandlerFunc signature
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"tag_name": "v1.0.0",
			"assets": [
				{
					"name": "test-provider-darwin-arm64",
					"browser_download_url": "https://example.com/provider",
					"size": 2048,
					"content_type": "application/octet-stream"
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(&ClientOptions{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		// Logger is nil (default)
	})

	spec := &ProviderSpec{
		Owner:   "test-owner",
		Repo:    "test-provider",
		Version: "1.0.0",
		OS:      "darwin",
		Arch:    "arm64",
	}

	// Act: Resolve asset (should not panic with nil logger)
	asset, err := client.ResolveAsset(context.Background(), spec)

	// Assert: Verify success
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if asset == nil {
		t.Fatal("expected asset, got nil")
	}

	if asset.Name != "test-provider-darwin-arm64" {
		t.Errorf("expected asset name 'test-provider-darwin-arm64', got %s", asset.Name)
	}
}
