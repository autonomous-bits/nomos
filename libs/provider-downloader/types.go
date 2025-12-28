package downloader

import (
	"net/http"
	"time"
)

// ProgressCallback is called periodically during download to report progress.
// downloaded is the number of bytes downloaded so far.
// total is the total number of bytes to download (0 if unknown).
type ProgressCallback func(downloaded, total int64)

// ProviderSpec describes a provider binary to download from GitHub Releases.
// It contains the repository information, version, and target platform.
type ProviderSpec struct {
	// Owner is the GitHub repository owner (e.g., "autonomous-bits").
	Owner string

	// Repo is the GitHub repository name (e.g., "nomos-provider-file").
	Repo string

	// Version is the semantic version or release tag (e.g., "1.0.0" or "v1.0.0").
	// The resolver will normalize version formats automatically.
	Version string

	// OS is the target operating system (e.g., "linux", "darwin", "windows").
	// If empty, runtime.GOOS is used for auto-detection.
	OS string

	// Arch is the target architecture (e.g., "amd64", "arm64").
	// If empty, runtime.GOARCH is used for auto-detection.
	Arch string
}

// AssetInfo describes a resolved GitHub Release asset.
// It contains download URL, metadata, and optional checksum information.
type AssetInfo struct {
	// URL is the download URL for the asset.
	URL string

	// Name is the asset filename (e.g., "nomos-provider-file-linux-amd64").
	Name string

	// Size is the asset size in bytes.
	Size int64

	// Checksum is the SHA256 checksum if available from release notes.
	// May be empty if checksum is not provided by the release.
	Checksum string

	// ContentType is the MIME type of the asset.
	ContentType string
}

// InstallResult contains the result of a successful installation.
type InstallResult struct {
	// Path is the absolute path to the installed binary.
	Path string

	// Checksum is the SHA256 checksum of the downloaded binary.
	Checksum string

	// Size is the size of the installed binary in bytes.
	Size int64
}

// Logger is an optional interface for debug logging.
// Implement this interface to receive debug logs from the downloader.
type Logger interface {
	// Debugf logs a debug message with format string and arguments.
	Debugf(format string, args ...interface{})
}

// ClientOptions configures the downloader client.
type ClientOptions struct {
	// GitHubToken is an optional GitHub personal access token for authenticated requests.
	// This increases rate limits and enables access to private repositories.
	// If empty, unauthenticated requests are used.
	GitHubToken string

	// HTTPClient is an optional custom HTTP client.
	// If nil, a default client with reasonable timeouts will be created.
	HTTPClient *http.Client

	// RetryAttempts is the number of retry attempts for failed downloads.
	// Default: 3
	RetryAttempts int

	// RetryDelay is the initial delay between retry attempts.
	// Exponential backoff is applied on subsequent retries.
	// Default: 1 second
	RetryDelay time.Duration

	// BaseURL is the GitHub API base URL.
	// Default: "https://api.github.com"
	// Can be overridden for testing or GitHub Enterprise.
	BaseURL string

	// Logger is an optional logger for debug output.
	// If nil, no debug logging is performed.
	Logger Logger

	// CacheDir is an optional directory for caching downloaded provider binaries.
	// If empty, caching is disabled and providers are always downloaded.
	// Cache key is based on the asset checksum.
	CacheDir string

	// HTTPTimeout is the timeout for HTTP requests.
	// Default: 30 seconds
	HTTPTimeout time.Duration

	// ProgressCallback is an optional callback for download progress updates.
	// Called periodically during download with bytes downloaded and total size.
	// If nil, no progress reporting is performed.
	ProgressCallback ProgressCallback
}

// DefaultClientOptions returns ClientOptions with sensible defaults.
func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		BaseURL:       "https://api.github.com",
		HTTPTimeout:   30 * time.Second,
	}
}
