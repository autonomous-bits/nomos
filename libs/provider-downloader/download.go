package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autonomous-bits/nomos/libs/provider-downloader/internal/archive"
)

// downloadAndInstall downloads a provider binary from the given AssetInfo
// and installs it atomically to the destination directory.
//
// The process:
//  1. Checks cache if caching is enabled
//  2. Creates a temporary file
//  3. Streams the HTTP response body to the temp file while computing SHA256
//  4. Verifies checksum if provided in AssetInfo
//  5. Extracts archive if needed
//  6. Sets executable permissions (0755)
//  7. Atomically renames to final destination
//  8. Saves to cache if caching is enabled
//
// Returns InstallResult with path, checksum, and size on success.
// Returns ChecksumMismatchError if checksums don't match.
// Returns error for network or filesystem failures.
func (c *Client) downloadAndInstall(ctx context.Context, asset *AssetInfo, destDir string) (*InstallResult, error) {
	// Create destination directory if it doesn't exist
	//nolint:gosec // G301: Standard directory permissions (0755) are appropriate for provider installation directories
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Check cache first if caching is enabled
	if c.cacheDir != "" && asset.Checksum != "" {
		cachedPath := c.getCachePath(asset.Checksum)
		if _, err := os.Stat(cachedPath); err == nil {
			c.debugf("Cache hit: %s", cachedPath)
			// Copy from cache to destination
			return c.installFromCache(cachedPath, destDir, asset.Checksum)
		}
		c.debugf("Cache miss: %s", cachedPath)
	}

	// Create temporary directory for download
	tmpDir := filepath.Join(filepath.Dir(destDir), ".nomos-tmp")
	//nolint:gosec // G301: Standard directory permissions (0755) are appropriate for temporary directories
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(tmpDir, "provider-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()    // Ignore close error in defer
		_ = os.Remove(tmpPath) // Best effort cleanup, ignore error
	}()

	// Download asset with retry logic
	actualChecksum, size, err := c.downloadWithRetry(ctx, asset.URL, tmpFile)
	if err != nil {
		return nil, err
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Verify checksum if provided
	if asset.Checksum != "" && asset.Checksum != actualChecksum {
		return nil, &ChecksumMismatchError{
			Expected: asset.Checksum,
			Actual:   actualChecksum,
		}
	}

	// If the asset is an archive (tar.gz, zip), extract it
	if needsExtraction(asset.Name) {
		// Create temporary extraction directory
		extractDir, err := os.MkdirTemp(tmpDir, "extract-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction directory: %w", err)
		}
		defer func() { _ = os.RemoveAll(extractDir) }() // Best effort cleanup

		// Get appropriate extractor
		extractor, err := archive.GetExtractor(asset.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get extractor: %w", err)
		}

		// Extract archive
		extractedPath, err := extractor.Extract(tmpPath, extractDir)
		if err != nil {
			return nil, fmt.Errorf("failed to extract archive: %w", err)
		}
		// Update tmpPath to point to the extracted binary
		tmpPath = extractedPath
	}

	// Set executable permissions
	//nolint:gosec // G302: Executable permissions (0755) required for provider binaries
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename to destination
	finalPath := filepath.Join(destDir, "provider")
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return nil, fmt.Errorf("failed to install provider: %w", err)
	}

	// Save to cache if caching is enabled AND asset had a checksum
	// We only cache when the asset provided a checksum for verification
	if c.cacheDir != "" && asset.Checksum != "" {
		if err := c.saveToCache(finalPath, actualChecksum); err != nil {
			// Log error but don't fail the installation
			c.debugf("Failed to save to cache: %v", err)
		}
	}

	return &InstallResult{
		Path:     finalPath,
		Checksum: actualChecksum,
		Size:     size,
	}, nil
}

// downloadWithRetry downloads content from URL to file with retry logic.
// Implements exponential backoff with jitter for transient failures.
// Returns checksum, size, and error.
func (c *Client) downloadWithRetry(ctx context.Context, url string, f *os.File) (checksum string, size int64, err error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryAttempts; attempt++ {
		if attempt > 0 {
			// Reset file for retry
			if _, err := f.Seek(0, 0); err != nil {
				return "", 0, fmt.Errorf("failed to reset file for retry: %w", err)
			}
			if err := f.Truncate(0); err != nil {
				return "", 0, fmt.Errorf("failed to truncate file for retry: %w", err)
			}

			// Exponential backoff with jitter
			backoff := c.retryDelay * (1 << (attempt - 1))
			jitter := backoff / 10 // 10% jitter

			select {
			case <-ctx.Done():
				return "", 0, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		// Attempt download
		chk, sz, err := c.attemptDownload(ctx, url, f)
		if err == nil {
			return chk, sz, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return "", 0, lastErr
		}
	}

	return "", 0, fmt.Errorf("download failed after %d retries: %w", c.retryAttempts, lastErr)
}

// attemptDownload performs a single download attempt.
func (c *Client) attemptDownload(ctx context.Context, url string, w io.Writer) (checksum string, size int64, err error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if token is available
	if c.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.githubToken)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Get total size from Content-Length header (0 if not available)
	totalSize := resp.ContentLength

	// Stream response body to file while computing checksum
	hasher := sha256.New()
	multiWriter := io.MultiWriter(w, hasher)

	// Wrap writer with progress reporting if callback is set
	var written int64
	if c.progressCallback != nil {
		pw := &progressWriter{
			writer:   multiWriter,
			callback: c.progressCallback,
			total:    totalSize,
		}
		written, err = io.Copy(pw, resp.Body)
	} else {
		written, err = io.Copy(multiWriter, resp.Body)
	}
	if err != nil {
		return "", 0, fmt.Errorf("failed to stream download: %w", err)
	}

	checksumBytes := hasher.Sum(nil)
	checksumHex := hex.EncodeToString(checksumBytes)

	return checksumHex, written, nil
}

// isRetryable determines if an error is retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors and incomplete reads are retryable
	// HTTP 5xx errors are retryable, 4xx are not
	errStr := err.Error()

	// Retryable conditions:
	// - Connection errors
	// - Timeout errors
	// - Incomplete reads (io.ErrUnexpectedEOF)
	// - 5xx server errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "unexpected EOF") ||
		strings.Contains(errStr, "status 5") {
		return true
	}

	return false
}

// needsExtraction checks if the asset file needs to be extracted.
func needsExtraction(filename string) bool {
	return strings.Contains(filename, ".tar.gz") || strings.Contains(filename, ".tgz") || strings.Contains(filename, ".zip")
}

// getCachePath returns the cache path for a given checksum.
func (c *Client) getCachePath(checksum string) string {
	return filepath.Join(c.cacheDir, checksum)
}

// installFromCache copies a provider binary from the cache to the destination.
func (c *Client) installFromCache(cachedPath, destDir, checksum string) (*InstallResult, error) {
	// Read cached file
	//nolint:gosec // G304: cachedPath is from our controlled cache directory
	data, err := os.ReadFile(cachedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read from cache: %w", err)
	}

	// Set executable permissions
	finalPath := filepath.Join(destDir, "provider")
	//nolint:gosec // G306: Executable permissions (0755) required for provider binaries
	if err := os.WriteFile(finalPath, data, 0755); err != nil {
		return nil, fmt.Errorf("failed to install from cache: %w", err)
	}

	return &InstallResult{
		Path:     finalPath,
		Checksum: checksum,
		Size:     int64(len(data)),
	}, nil
}

// saveToCache saves a provider binary to the cache.
func (c *Client) saveToCache(providerPath, checksum string) error {
	// Create cache directory if needed
	//nolint:gosec // G301: Standard directory permissions (0755) are appropriate for cache directories
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Read provider binary
	//nolint:gosec // G304: providerPath is from our controlled temp directory
	data, err := os.ReadFile(providerPath)
	if err != nil {
		return fmt.Errorf("failed to read provider: %w", err)
	}

	// Write to cache
	cachePath := c.getCachePath(checksum)
	//nolint:gosec // G306: Cache files should be readable (0644) but not executable
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	c.debugf("Saved to cache: %s", cachePath)
	return nil
}

// progressWriter wraps an io.Writer and calls a callback function
// to report download progress.
type progressWriter struct {
	writer     io.Writer
	callback   ProgressCallback
	downloaded int64
	total      int64
}

// Write implements io.Writer and calls the progress callback.
func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.downloaded += int64(n)
	if pw.callback != nil {
		pw.callback(pw.downloaded, pw.total)
	}
	return n, err
}
