// Package traverse provides deterministic file discovery for Nomos CLI.
//
// This package implements UTF-8 lexicographic ordering of .csl files,
// supports recursive directory traversal, and handles edge cases like
// symlinks, empty directories, and unreadable files.
package traverse

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverFiles discovers all .csl files from the given path.
//
// If path is a file, it returns that file (must have .csl extension).
// If path is a directory, it recursively discovers all .csl files
// and returns them in UTF-8 lexicographic order.
//
// Returns an error if:
//   - path does not exist
//   - path is a file without .csl extension
//   - path is a directory with no .csl files
//   - files cannot be read due to permissions
func DiscoverFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path %q: %w", path, err)
	}

	// Handle single file case
	if !info.IsDir() {
		return handleSingleFile(path)
	}

	// Handle directory case
	return discoverFilesInDirectory(path)
}

// handleSingleFile validates and returns a single .csl file.
func handleSingleFile(path string) ([]string, error) {
	if !strings.HasSuffix(path, ".csl") {
		return nil, fmt.Errorf("file %q is not a .csl file", path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	return []string{absPath}, nil
}

// discoverFilesInDirectory recursively discovers all .csl files in a directory.
func discoverFilesInDirectory(path string) ([]string, error) {
	var files []string
	visited := make(map[string]bool) // Track visited directories to prevent symlink loops

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access %q: %w", p, err)
		}

		// Detect and prevent symlink loops
		if d.IsDir() {
			if shouldSkipDirectory(p, visited) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .csl files
		if !strings.HasSuffix(d.Name(), ".csl") {
			return nil
		}

		absPath, err := filepath.Abs(p)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %q: %w", p, err)
		}

		files = append(files, absPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("no .csl files found")
	}

	// Sort for deterministic output (UTF-8 lexicographic order)
	sort.Strings(files)

	return files, nil
}

// shouldSkipDirectory checks if a directory should be skipped due to symlink loops.
// Updates the visited map and returns true if the directory should be skipped.
func shouldSkipDirectory(path string, visited map[string]bool) bool {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve the symlink, skip it
		return true
	}

	if visited[realPath] {
		// Symlink loop detected
		return true
	}

	visited[realPath] = true
	return false
}
