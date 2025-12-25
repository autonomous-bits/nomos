package downloader

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// downloadAndInstall downloads a provider binary from the given AssetInfo
// and installs it atomically to the destination directory.
//
// The process:
//  1. Creates a temporary file
//  2. Streams the HTTP response body to the temp file while computing SHA256
//  3. Verifies checksum if provided in AssetInfo
//  4. Sets executable permissions (0755)
//  5. Atomically renames to final destination
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

		extractedPath, err := extractArchive(tmpPath, extractDir, asset.Name)
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

	// Stream response body to file while computing checksum
	hasher := sha256.New()
	multiWriter := io.MultiWriter(w, hasher)

	written, err := io.Copy(multiWriter, resp.Body)
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
	if contains(errStr, "connection") ||
		contains(errStr, "timeout") ||
		contains(errStr, "unexpected EOF") ||
		contains(errStr, "status 5") {
		return true
	}

	return false
}

// contains is a simple string contains helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// needsExtraction checks if the asset file needs to be extracted.
func needsExtraction(filename string) bool {
	return contains(filename, ".tar.gz") || contains(filename, ".tgz") || contains(filename, ".zip")
}

// extractArchive extracts an archive file and returns the path to the extracted binary.
// For tar.gz files, it assumes the binary is named "provider" or matches the base name without extension.
func extractArchive(archivePath, destDir, assetName string) (string, error) {
	if contains(assetName, ".tar.gz") || contains(assetName, ".tgz") {
		return extractTarGz(archivePath, destDir)
	}
	// TODO: Add zip extraction if needed
	return "", fmt.Errorf("unsupported archive format: %s", assetName)
}

// extractTarGz extracts a tar.gz archive and returns the path to the provider binary.
func extractTarGz(archivePath, destDir string) (string, error) {
	// Open the archive file
	//nolint:gosec // G304: archivePath is from our own download, not user input
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Create gzip reader
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Track all extracted files and find the provider binary
	var providerPath string
	var fallbackPath string // For "nomos-provider-*" names

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Extract file (flattened to destDir)
		baseName := filepath.Base(header.Name)
		target := filepath.Join(destDir, baseName)
		//nolint:gosec // G304: target is constructed from our controlled temp directory and sanitized basename
		//nolint:gosec // G110: Archive from trusted GitHub releases, size limited by download timeout
		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to create file %s: %w", target, err)
		}

		//nolint:gosec // G110: Archive from trusted GitHub releases, size limited by download timeout
		if _, err := io.Copy(outFile, tr); err != nil {
			_ = outFile.Close()
			return "", fmt.Errorf("failed to extract file %s: %w", target, err)
		}
		if err := outFile.Close(); err != nil {
			return "", fmt.Errorf("failed to close extracted file %s: %w", target, err)
		}

		// Check if this is a provider binary
		// Priority: exact "provider" match > "nomos-provider-*" match
		if baseName == "provider" {
			providerPath = target
		} else if providerPath == "" && contains(baseName, "nomos-provider-") {
			fallbackPath = target
		}
	}

	// Prefer exact "provider" match, fall back to "nomos-provider-*"
	if providerPath != "" {
		return providerPath, nil
	}
	if fallbackPath != "" {
		return fallbackPath, nil
	}

	return "", fmt.Errorf("provider binary not found in archive")
}
