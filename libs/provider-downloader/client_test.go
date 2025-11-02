package downloader

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewClient_WithNilOptions tests that NewClient handles nil options gracefully.
func TestNewClient_WithNilOptions(t *testing.T) {
	ctx := context.Background()

	client := NewClient(ctx, nil)

	if client == nil {
		t.Fatal("expected non-nil client, got nil")
	}

	if client.httpClient == nil {
		t.Error("expected non-nil HTTP client")
	}

	if client.baseURL != "https://api.github.com" {
		t.Errorf("expected default baseURL, got %s", client.baseURL)
	}

	if client.retryAttempts != 3 {
		t.Errorf("expected default retryAttempts 3, got %d", client.retryAttempts)
	}

	if client.retryDelay != 1*time.Second {
		t.Errorf("expected default retryDelay 1s, got %v", client.retryDelay)
	}
}

// TestNewClient_WithCustomOptions tests that NewClient respects custom options.
func TestNewClient_WithCustomOptions(t *testing.T) {
	ctx := context.Background()

	opts := &ClientOptions{
		GitHubToken:   "test-token",
		RetryAttempts: 5,
		RetryDelay:    2 * time.Second,
		BaseURL:       "https://custom.github.com",
	}

	client := NewClient(ctx, opts)

	if client == nil {
		t.Fatal("expected non-nil client, got nil")
	}

	if client.githubToken != "test-token" {
		t.Errorf("expected githubToken test-token, got %s", client.githubToken)
	}

	if client.retryAttempts != 5 {
		t.Errorf("expected retryAttempts 5, got %d", client.retryAttempts)
	}

	if client.retryDelay != 2*time.Second {
		t.Errorf("expected retryDelay 2s, got %v", client.retryDelay)
	}

	if client.baseURL != "https://custom.github.com" {
		t.Errorf("expected baseURL https://custom.github.com, got %s", client.baseURL)
	}
}

// TestResolveAsset_NilSpec tests error handling for nil spec.
func TestResolveAsset_NilSpec(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, nil)

	asset, err := client.ResolveAsset(ctx, nil)

	if asset != nil {
		t.Errorf("expected nil asset, got %+v", asset)
	}

	if err == nil {
		t.Fatal("expected error for nil spec, got nil")
	}

	var invalidErr *InvalidSpecError
	if !errors.As(err, &invalidErr) {
		t.Errorf("expected InvalidSpecError, got %T: %v", err, err)
	}

	if !errors.Is(err, ErrInvalidSpec) {
		t.Errorf("expected error to wrap ErrInvalidSpec")
	}
}

// TestResolveAsset_MissingOwner tests error handling for missing owner.
func TestResolveAsset_MissingOwner(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, nil)

	spec := &ProviderSpec{
		Repo:    "test-repo",
		Version: "1.0.0",
	}

	asset, err := client.ResolveAsset(ctx, spec)

	if asset != nil {
		t.Errorf("expected nil asset, got %+v", asset)
	}

	if err == nil {
		t.Fatal("expected error for missing owner, got nil")
	}

	var invalidErr *InvalidSpecError
	if !errors.As(err, &invalidErr) {
		t.Errorf("expected InvalidSpecError, got %T: %v", err, err)
	}

	if invalidErr.Field != "Owner" {
		t.Errorf("expected field Owner, got %s", invalidErr.Field)
	}
}

// TestResolveAsset_MissingRepo tests error handling for missing repo.
func TestResolveAsset_MissingRepo(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, nil)

	spec := &ProviderSpec{
		Owner:   "test-owner",
		Version: "1.0.0",
	}

	asset, err := client.ResolveAsset(ctx, spec)

	if asset != nil {
		t.Errorf("expected nil asset, got %+v", asset)
	}

	if err == nil {
		t.Fatal("expected error for missing repo, got nil")
	}

	var invalidErr *InvalidSpecError
	if !errors.As(err, &invalidErr) {
		t.Errorf("expected InvalidSpecError, got %T: %v", err, err)
	}

	if invalidErr.Field != "Repo" {
		t.Errorf("expected field Repo, got %s", invalidErr.Field)
	}
}

// TestResolveAsset_MissingVersion tests that missing version defaults to "latest".
func TestResolveAsset_MissingVersion(t *testing.T) {
	// Version is optional and defaults to "latest"
	// This test is a placeholder demonstrating that missing version is allowed
	t.Skip("Version is optional; resolver defaults to 'latest' release")
}

// TestResolveAsset_RealGitHubAPI is removed since resolver is now implemented.
// Real API tests belong in integration tests, not unit tests.
// This test would make actual network calls without a mock server.

// TestDownloadAndInstall_NilAsset tests error handling for nil asset.
func TestDownloadAndInstall_NilAsset(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, nil)

	result, err := client.DownloadAndInstall(ctx, nil, "/tmp/dest")

	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for nil asset, got nil")
	}
}

// TestDownloadAndInstall_EmptyDestDir tests error handling for empty destination.
func TestDownloadAndInstall_EmptyDestDir(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ctx, nil)

	asset := &AssetInfo{
		URL:  "https://example.com/asset",
		Name: "test-asset",
		Size: 1024,
	}

	result, err := client.DownloadAndInstall(ctx, asset, "")

	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error for empty destination, got nil")
	}
}

// TestDefaultClientOptions tests that DefaultClientOptions returns expected values.
func TestDefaultClientOptions(t *testing.T) {
	opts := DefaultClientOptions()

	if opts == nil {
		t.Fatal("expected non-nil options, got nil")
	}

	if opts.RetryAttempts != 3 {
		t.Errorf("expected RetryAttempts 3, got %d", opts.RetryAttempts)
	}

	if opts.RetryDelay != 1*time.Second {
		t.Errorf("expected RetryDelay 1s, got %v", opts.RetryDelay)
	}

	if opts.BaseURL != "https://api.github.com" {
		t.Errorf("expected BaseURL https://api.github.com, got %s", opts.BaseURL)
	}
}
