# Nomos Provider Downloader Library

The downloader library provides reusable functionality for resolving, downloading, and installing provider binaries from GitHub Releases.

## Overview

The Nomos provider downloader library enables consistent, testable downloads of provider binaries from GitHub Releases. It handles:

- **Asset Resolution**: Infer correct GitHub Release asset based on OS/architecture and provider spec
- **Download Management**: Streaming downloads with retries and SHA256 verification
- **Installation**: Atomic install into destination with proper permissions

This library is consumed by `nomos init` and other tools that manage provider lifecycles.

## Installation

```bash
go get github.com/autonomous-bits/nomos/libs/provider/downloader
```

## Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/autonomous-bits/nomos/libs/provider/downloader"
)

func main() {
	ctx := context.Background()
	
	// Create client with optional GitHub token
	client := downloader.NewClient(ctx, &downloader.ClientOptions{
		GitHubToken: "", // Optional for private repos
	})
	
	// Define provider spec
	spec := &downloader.ProviderSpec{
		Owner:   "autonomous-bits",
		Repo:    "nomos-provider-file",
		Version: "1.0.0",
		OS:      "linux",   // Auto-detected if empty
		Arch:    "amd64",   // Auto-detected if empty
	}
	
	// Resolve asset from GitHub Releases
	asset, err := client.ResolveAsset(ctx, spec)
	if err != nil {
		log.Fatalf("failed to resolve asset: %v", err)
	}
	
	// Download and install provider binary
	result, err := client.DownloadAndInstall(ctx, asset, ".nomos/providers/autonomous-bits/nomos-provider-file/1.0.0")
	if err != nil {
		log.Fatalf("failed to install: %v", err)
	}
	
	log.Printf("Installed provider at: %s (checksum: %s)", result.Path, result.Checksum)
}
```

## API Overview

### Client Creation

```go
client := downloader.NewClient(&downloader.ClientOptions{
	GitHubToken:   os.Getenv("GITHUB_TOKEN"), // Optional
	HTTPClient:    customHTTPClient,          // Optional
	CacheDir:      ".nomos/cache",            // Optional caching
	RetryAttempts: 3,                         // Optional retry config
	RetryDelay:    time.Second,               // Optional retry delay
})
```

### Caching

Enable caching to avoid redundant downloads of the same provider binary:

```go
client := downloader.NewClient(&downloader.ClientOptions{
	CacheDir: filepath.Join(os.UserHomeDir(), ".nomos", "cache"),
})

// First download - fetches from GitHub
result1, _ := client.DownloadAndInstall(ctx, asset, destDir1)

// Second download with same checksum - uses cache (no network call)
result2, _ := client.DownloadAndInstall(ctx, asset, destDir2)
```

**Cache behavior:**
- Cache key is the asset's SHA256 checksum
- Only caches when `AssetInfo.Checksum` is provided
- Cache hit avoids network calls entirely
- Cache directory is created automatically if it doesn't exist

### Asset Resolution

```go
// Resolve asset from provider spec
asset, err := client.ResolveAsset(ctx, &downloader.ProviderSpec{
	Owner:   "autonomous-bits",
	Repo:    "nomos-provider-file",
	Version: "1.0.0",
})
```

### Download and Install

```go
// Download and atomically install provider binary
result, err := client.DownloadAndInstall(ctx, asset, installPath)
```

## Download Process

The downloader implements a robust, multi-step process for safe binary installation:

### 1. Streaming Download

- Downloads asset to a temporary file (under `.nomos-tmp` or system temp)
- Computes SHA256 checksum incrementally during download (no full buffering in memory)
- Supports context cancellation at any stage

### 2. Checksum Verification

- If `AssetInfo.Checksum` is provided, verifies downloaded content matches expected hash
- Returns clear `ChecksumMismatchError` with both expected and actual values
- Prevents installation of corrupted or tampered binaries

### 3. Archive Extraction

The downloader automatically extracts provider binaries from archives:

- **Supported formats**: `.tar.gz`, `.tgz`, `.zip`
- **Binary detection**: Searches for `provider` or `nomos-provider-*` in the archive
- **Flat extraction**: Files are extracted and flattened to the destination directory
- **Automatic format detection**: Based on file extension

```go
// Works automatically for archive assets
asset := &AssetInfo{
	URL:  "https://github.com/owner/repo/releases/download/v1.0.0/provider.tar.gz",
	Name: "nomos-provider-file-linux-amd64.tar.gz",
}

// Downloader will:
// 1. Download the tar.gz file
// 2. Extract it to a temp directory
// 3. Find the provider binary
// 4. Install it to the destination
result, _ := client.DownloadAndInstall(ctx, asset, destDir)
```

### 4. Retry Logic

The downloader automatically retries transient failures:

- **Retryable errors**: 5xx server errors, timeouts, connection issues
- **Non-retryable errors**: 4xx client errors, invalid specs, checksum mismatches
- **Exponential backoff**: Delay doubles with each retry (1s, 2s, 4s, ...)
- **Jitter**: 10% randomization to prevent thundering herd
- **Configurable**: Set `RetryAttempts` and `RetryDelay` in `ClientOptions`
- **File reset**: Temp file is truncated and reset between retries for clean attempts

### 5. Atomic Installation

- Sets executable permissions (0755) on downloaded binary
- Atomically renames temp file to final destination
- Handles cross-filesystem renames (falls back to copy+remove if needed)
- Final path: `{destDir}/provider`

### Example with Retry Configuration

```go
client := downloader.NewClient(&downloader.ClientOptions{
	RetryAttempts: 5,                  // Retry up to 5 times
	RetryDelay:    2 * time.Second,    // Start with 2s delay
})

result, err := client.DownloadAndInstall(ctx, asset, destDir)
if err != nil {
	// Download failed after all retries
	log.Fatalf("installation failed: %v", err)
}

log.Printf("Provider installed: %s (checksum: %s, size: %d bytes)",
	result.Path, result.Checksum, result.Size)
```

## Asset Resolution Strategy

The resolver uses an ordered matching strategy to find the correct binary for your platform:

### 1. Exact Pattern Matching (Priority Order)

1. `{repo}-{os}-{arch}` (e.g., `nomos-provider-file-linux-amd64`)
2. `nomos-provider-{os}-{arch}` (e.g., `nomos-provider-darwin-arm64`)
3. `{repo}-{os}` (e.g., `test-provider-linux`)

### 2. Substring Matching (Fallback)

If no exact match is found, the resolver falls back to case-insensitive substring matching:

- Asset name must contain both the OS and architecture
- Handles common variations:
  - `amd64`, `x86_64`, `x86-64` (all match for amd64)
  - `arm64`, `aarch64` (all match for arm64)

### 3. Auto-Detection

- If `OS` is empty in the spec, uses `runtime.GOOS`
- If `Arch` is empty in the spec, uses `runtime.GOARCH`

### 4. Version Normalization

- Automatically tries both `v1.0.0` and `1.0.0` formats
- If version is empty or "latest", fetches the latest release

### Examples

```go
// Example 1: Explicit platform
spec := &downloader.ProviderSpec{
	Owner:   "autonomous-bits",
	Repo:    "nomos-provider-file",
	Version: "1.0.0",
	OS:      "linux",
	Arch:    "amd64",
}
// Matches: nomos-provider-file-linux-amd64

// Example 2: Auto-detect current platform
spec := &downloader.ProviderSpec{
	Owner:   "autonomous-bits",
	Repo:    "nomos-provider-file",
	Version: "1.0.0",
	// OS and Arch auto-detected from runtime
}

// Example 3: Latest version
spec := &downloader.ProviderSpec{
	Owner: "autonomous-bits",
	Repo:  "nomos-provider-file",
	// Version empty = latest release
}

// Example 4: Custom naming pattern (fallback matching)
spec := &downloader.ProviderSpec{
	Owner:   "custom-org",
	Repo:    "special-provider",
	Version: "2.5.0",
	OS:      "darwin",
	Arch:    "arm64",
}
// Can match assets like: "Provider_Darwin_ARM64", "provider-macos-aarch64"
```

## Configuration

### ClientOptions

- `GitHubToken`: Optional GitHub token for private repositories or higher rate limits
- `HTTPClient`: Optional custom HTTP client for testing or proxy configuration
- `RetryAttempts`: Number of retry attempts for failed downloads (default: 3)
- `RetryDelay`: Delay between retry attempts (default: 1s)

### ProviderSpec

- `Owner`: GitHub repository owner (e.g., "autonomous-bits")
- `Repo`: GitHub repository name (e.g., "nomos-provider-file")
- `Version`: Semantic version or release tag (e.g., "1.0.0")
- `OS`: Target operating system (auto-detected if empty)
- `Arch`: Target architecture (auto-detected if empty)

### AssetInfo

- `URL`: Download URL for the asset
- `Name`: Asset filename
- `Size`: Size in bytes
- `Checksum`: SHA256 checksum (if available in release notes)

### InstallResult

- `Path`: Absolute path to installed binary
- `Checksum`: SHA256 checksum of downloaded binary
- `Size`: Size in bytes

## Error Handling

The library provides typed errors for common failure modes:

- `ErrAssetNotFound`: No matching asset found for the given spec
- `ErrChecksumMismatch`: Downloaded file checksum doesn't match expected
- `ErrInvalidSpec`: Provider spec is missing required fields
- `ErrRateLimitExceeded`: GitHub API rate limit exceeded
- `ErrNetworkFailure`: Network error during download

Example:

```go
asset, err := client.ResolveAsset(ctx, spec)
if err != nil {
	if errors.Is(err, downloader.ErrAssetNotFound) {
		log.Printf("No matching asset found for %s/%s@%s", 
			spec.Owner, spec.Repo, spec.Version)
	}
	return err
}
```

## Testing

Run unit tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Design Principles

1. **No network calls in unit tests**: All tests use httptest for hermetic testing
2. **Atomic installation**: Downloads to temp file, verifies checksum, then renames into place
3. **Automatic OS/Arch detection**: Uses runtime.GOOS and runtime.GOARCH when not specified
4. **Retry with exponential backoff**: Handles transient network failures gracefully
5. **GitHub API best practices**: Respects rate limits and uses conditional requests

## Dependencies

- Standard library only for core functionality
- No external dependencies for basic operations
- Optional: `golang.org/x/time/rate` for rate limiting (future enhancement)

## Contributing

See the main repository [CONTRIBUTING.md](../../../CONTRIBUTING.md) for contribution guidelines.

## License

See [LICENSE](../../../LICENSE) in the repository root.
