// Package archive provides utilities for extracting provider binaries from archives.
package archive

import (
	"fmt"
	"strings"
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
	if strings.Contains(filename, ".tar.gz") || strings.Contains(filename, ".tgz") {
		return &TarGzExtractor{}, nil
	}
	if strings.Contains(filename, ".zip") {
		return &ZipExtractor{}, nil
	}
	return nil, fmt.Errorf("unsupported archive format: %s", filename)
}
