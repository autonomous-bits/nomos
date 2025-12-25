// Package fakes provides test doubles (fakes, mocks) for the compiler package.
// These test implementations are used throughout the compiler test suite.
package fakes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
)

// FakeFileProvider is a test double for a file provider that reads .csl files.
// It implements the compiler.Provider interface.
type FakeFileProvider struct {
	BaseDirectory string
	InitCount     int
	FetchCount    int
}

// NewFakeFileProvider creates a new FakeFileProvider for the given directory.
func NewFakeFileProvider(baseDir string) *FakeFileProvider {
	return &FakeFileProvider{
		BaseDirectory: baseDir,
	}
}

// Init implements compiler.Provider.Init.
func (f *FakeFileProvider) Init(_ context.Context, opts compiler.ProviderInitOptions) error {
	f.InitCount++

	// Override base directory if provided in config
	if dir, ok := opts.Config["directory"].(string); ok && dir != "" {
		f.BaseDirectory = dir
	}

	// Check that base directory exists
	if _, err := os.Stat(f.BaseDirectory); err != nil {
		return fmt.Errorf("base directory does not exist: %w", err)
	}

	return nil
}

// Fetch implements compiler.Provider.Fetch.
// For file provider, path should be a single element: the filename.
func (f *FakeFileProvider) Fetch(_ context.Context, path []string) (any, error) {
	f.FetchCount++

	if len(path) == 0 {
		return nil, fmt.Errorf("path is required")
	}

	// First element is the filename
	filename := path[0]
	filePath := filepath.Join(f.BaseDirectory, filename)

	// Read and parse the file
	tree, _, err := parse.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Convert AST to data (excluding source and import declarations)
	data, err := converter.ASTToData(tree)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AST to data: %w", err)
	}

	// If additional path components were provided, navigate to nested value
	if len(path) > 1 {
		result := data
		for i := 1; i < len(path); i++ {
			key := path[i]
			val, ok := result[key]
			if !ok {
				return nil, fmt.Errorf("key %q not found at path %v", key, path[:i+1])
			}

			// Check if we can continue navigating
			if i < len(path)-1 {
				m, ok := val.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("cannot navigate into non-map value at %v", path[:i+1])
				}
				result = m
			} else {
				// Last component - return the value
				return val, nil
			}
		}
		return result, nil
	}

	return data, nil
}
