package downloader

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Client handles resolving, downloading, and installing provider binaries
// from GitHub Releases.
type Client struct {
	httpClient    *http.Client
	githubToken   string
	baseURL       string
	retryAttempts int
	retryDelay    time.Duration
	logger        Logger
}

// NewClient creates a new downloader client with the given options.
// If opts is nil, default options are used.
//
// Example:
//
//	client := downloader.NewClient(ctx, &downloader.ClientOptions{
//		GitHubToken: os.Getenv("GITHUB_TOKEN"),
//	})
func NewClient(ctx context.Context, opts *ClientOptions) *Client {
	if opts == nil {
		opts = DefaultClientOptions()
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	retryAttempts := opts.RetryAttempts
	if retryAttempts <= 0 {
		retryAttempts = 3
	}

	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 1 * time.Second
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	return &Client{
		httpClient:    httpClient,
		githubToken:   opts.GitHubToken,
		baseURL:       baseURL,
		retryAttempts: retryAttempts,
		retryDelay:    retryDelay,
		logger:        opts.Logger,
	}
}

// ResolveAsset resolves a GitHub Release asset based on the given ProviderSpec.
// It queries the GitHub Releases API to find a matching asset for the specified
// OS and architecture.
//
// If OS or Arch are empty in the spec, they are auto-detected from runtime.GOOS
// and runtime.GOARCH respectively.
//
// Returns ErrAssetNotFound if no matching asset is found.
// Returns ErrInvalidSpec if the spec is missing required fields.
// Returns ErrRateLimitExceeded if the GitHub API rate limit is exceeded.
//
// Example:
//
//	spec := &downloader.ProviderSpec{
//		Owner:   "autonomous-bits",
//		Repo:    "nomos-provider-file",
//		Version: "1.0.0",
//	}
//	asset, err := client.ResolveAsset(ctx, spec)
func (c *Client) ResolveAsset(ctx context.Context, spec *ProviderSpec) (*AssetInfo, error) {
	if spec == nil {
		return nil, &InvalidSpecError{
			Field:   "spec",
			Message: "spec cannot be nil",
		}
	}

	if spec.Owner == "" {
		return nil, &InvalidSpecError{
			Field:   "Owner",
			Message: "owner is required",
		}
	}

	if spec.Repo == "" {
		return nil, &InvalidSpecError{
			Field:   "Repo",
			Message: "repo is required",
		}
	}

	// Version is optional (defaults to "latest")
	// Delegate to resolver
	return c.resolveAssetFromGitHub(ctx, spec)
}

// DownloadAndInstall downloads a provider binary from the given AssetInfo
// and installs it atomically to the destination path.
//
// The download process:
//  1. Downloads asset to a temporary file
//  2. Verifies SHA256 checksum if provided in AssetInfo
//  3. Sets executable permissions (0755)
//  4. Atomically moves to destination path
//
// The destination path should be the final installation directory
// (e.g., ".nomos/providers/owner/repo/version"). The binary will be
// installed with the name "provider" in that directory.
//
// Returns ErrChecksumMismatch if the downloaded file's checksum doesn't match.
// Returns ErrNetworkFailure if download fails after all retry attempts.
//
// Example:
//
//	result, err := client.DownloadAndInstall(ctx, asset, ".nomos/providers/owner/repo/1.0.0")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Installed at: %s (checksum: %s)\n", result.Path, result.Checksum)
func (c *Client) DownloadAndInstall(ctx context.Context, asset *AssetInfo, destDir string) (*InstallResult, error) {
	if asset == nil {
		return nil, fmt.Errorf("asset cannot be nil")
	}

	if destDir == "" {
		return nil, fmt.Errorf("destination directory is required")
	}

	return c.downloadAndInstall(ctx, asset, destDir)
}

// debugf logs a debug message if a logger is configured.
func (c *Client) debugf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Debugf(format, args...)
	}
}
