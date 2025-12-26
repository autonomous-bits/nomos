package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TarGzExtractor extracts provider binaries from tar.gz archives.
type TarGzExtractor struct{}

// Extract extracts a provider binary from a tar.gz archive.
// It searches for a file named "provider" or matching "nomos-provider-*" pattern.
// Files are flattened to the destination directory (directory structure is not preserved).
func (e *TarGzExtractor) Extract(archivePath, destDir string) (string, error) {
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

	// Track extracted files and find the provider binary
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

	return "", fmt.Errorf("provider binary not found in archive")
}
