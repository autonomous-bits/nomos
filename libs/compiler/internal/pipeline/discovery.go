// Package pipeline provides compilation pipeline stages for the Nomos compiler.
//
// This package contains functions that implement individual stages of the
// compilation process, such as file discovery, provider initialization, and
// reference resolution. These stages are orchestrated by the main Compile
// function in the compiler package.
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverInputFiles finds all .csl files at the given path.
// If path is a file, returns that file.
// If path is a directory, returns all .csl files in lexicographic order.
func DiscoverInputFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}

	// Single file
	if !info.IsDir() {
		if !strings.HasSuffix(path, ".csl") {
			return nil, fmt.Errorf("file %q is not a .csl file", path)
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		return []string{absPath}, nil
	}

	// Directory: list entries and filter for .csl files
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", path, err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".csl") {
			continue
		}
		fullPath := filepath.Join(path, entry.Name())
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %q: %w", fullPath, err)
		}
		files = append(files, absPath)
	}

	// Sort lexicographically for determinism
	sort.Strings(files)

	return files, nil
}
