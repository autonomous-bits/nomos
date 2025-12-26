package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipExtractor extracts provider binaries from zip archives.
type ZipExtractor struct{}

// Extract extracts a provider binary from a zip archive.
// It searches for a file named "provider" or matching "nomos-provider-*" pattern.
// Files are flattened to the destination directory (directory structure is not preserved).
func (e *ZipExtractor) Extract(archivePath, destDir string) (string, error) {
	// Open the zip archive
	//nolint:gosec // G304: archivePath is from our own download, not user input
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer func() { _ = reader.Close() }()

	// Track extracted files and find the provider binary
	var providerPath string
	var fallbackPath string // For "nomos-provider-*" names

	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Extract file (flattened to destDir)
		baseName := filepath.Base(file.Name)
		target := filepath.Join(destDir, baseName)

		// Open source file from zip
		srcFile, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open file %s in zip: %w", file.Name, err)
		}

		// Create destination file
		//nolint:gosec // G304: target is constructed from our controlled temp directory and sanitized basename
		//nolint:gosec // G110: Archive from trusted GitHub releases, size limited by download timeout
		dstFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			_ = srcFile.Close()
			return "", fmt.Errorf("failed to create file %s: %w", target, err)
		}

		// Copy content
		//nolint:gosec // G110: Archive from trusted GitHub releases, size limited by download timeout
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = srcFile.Close()
			_ = dstFile.Close()
			return "", fmt.Errorf("failed to extract file %s: %w", target, err)
		}

		_ = srcFile.Close()
		if err := dstFile.Close(); err != nil {
			return "", fmt.Errorf("failed to close extracted file %s: %w", target, err)
		}

		// Check if this is a provider binary
		// Priority: exact "provider" match > "nomos-provider-*" match
		if baseName == "provider" {
			providerPath = target
		} else if providerPath == "" && strings.Contains(baseName, "nomos-provider-") {
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

	return "", fmt.Errorf("provider binary not found in zip archive")
}
