// Package archive provides utilities for extracting provider binaries from archives.
package archive

import (
	"fmt"
)

// Extractor defines the interface for archive extraction.
type Extractor interface {
	// Extract extracts a provider binary from an archive file to the destination directory.
	// Returns the path to the extracted provider binary.
	Extract(archivePath, destDir string) (string, error)
}

// GetExtractor returns an appropriate Extractor based on the archive filename.
// Returns an error if the archive format is not supported.
func GetExtractor(filename string) (Extractor, error) {
	if contains(filename, ".tar.gz") || contains(filename, ".tgz") {
		return &TarGzExtractor{}, nil
	}
	if contains(filename, ".zip") {
		return &ZipExtractor{}, nil
	}
	return nil, fmt.Errorf("unsupported archive format: %s", filename)
}

// contains checks if a string contains a substring.
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
