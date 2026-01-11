// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// discoverCslFiles expands a path into a list of .csl files.
// If path is a file with .csl extension, returns that file.
// If path is a directory, returns all .csl files in that directory (non-recursive).
// Results are sorted lexicographically for determinism.
//
// Returns an error if:
// - Path does not exist
// - Path is a file without .csl extension
// - Failed to read directory contents
func discoverCslFiles(path string) ([]string, error) {
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

// DiscoverProviders scans .csl files and extracts provider requirements.
// It parses each file and extracts SourceDecl nodes, converting them to
// DiscoveredProvider structs. Duplicate provider aliases are automatically
// deduplicated (first occurrence wins).
//
// Paths can be individual .csl files or directories. Directories are expanded
// to include all .csl files in lexicographic order (non-recursive).
//
// Returns a slice of discovered providers and any parsing errors encountered.
func DiscoverProviders(paths []string) ([]DiscoveredProvider, error) {
	providers := []DiscoveredProvider{}
	seen := make(map[string]bool)

	// Expand paths to include all .csl files
	allFiles := []string{}
	for _, path := range paths {
		files, err := discoverCslFiles(path)
		if err != nil {
			return nil, fmt.Errorf("failed to discover files at %s: %w", path, err)
		}
		allFiles = append(allFiles, files...)
	}

	for _, path := range allFiles {
		//nolint:gosec // G304: Path comes from user CLI input, intentional file inclusion
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer func() { _ = file.Close() }()

		tree, err := parser.Parse(file, path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Extract source declarations from AST
		for _, stmt := range tree.Statements {
			srcDecl, ok := stmt.(*ast.SourceDecl)
			if !ok {
				continue
			}

			// Skip duplicates
			if seen[srcDecl.Alias] {
				continue
			}
			seen[srcDecl.Alias] = true

			// Convert config expressions to values
			config := make(map[string]any)
			for k, expr := range srcDecl.Config {
				config[k] = exprToValue(expr)
			}

			// Use the Version field from SourceDecl (not from Config map)
			version := srcDecl.Version

			providers = append(providers, DiscoveredProvider{
				Alias:   srcDecl.Alias,
				Type:    srcDecl.Type,
				Version: version,
				Config:  config,
			})
		}
	}

	return providers, nil
}
