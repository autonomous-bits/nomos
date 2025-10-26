// Package file provides a file system provider for the Nomos compiler.
//
// The file provider resolves data from a single local file, supporting
// JSON and YAML formats. Path components are used to navigate within
// the file's data structure (e.g., path ["database", "host"] navigates
// to result.database.host in the parsed file).
package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"gopkg.in/yaml.v3"
)

// FileProvider implements the compiler.Provider interface for local file access.
type FileProvider struct {
	filePath       string
	alias          string
	configFilePath string // file path from config, used during Init
}

// Init initializes the file provider with the given options.
// The file configuration is required and must point to an existing file.
func (p *FileProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	p.alias = opts.Alias

	// Use configFilePath if set (from RegisterFileProvider), otherwise extract from config
	var filePath string
	if p.configFilePath != "" {
		filePath = p.configFilePath
	} else {
		// Extract file from config
		fileVal, ok := opts.Config["file"]
		if !ok {
			return errors.New("file provider requires 'file' in config")
		}

		var ok2 bool
		filePath, ok2 = fileVal.(string)
		if !ok2 {
			return fmt.Errorf("file must be a string, got %T", fileVal)
		}
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file to absolute path: %w", err)
	}

	// Verify file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("file path points to a directory, not a file: %s", absPath)
	}

	// Clean and store the file path
	p.filePath = filepath.Clean(absPath)

	return nil
}

// Fetch retrieves and parses the configured file, then navigates to the requested path.
// The path components are used to navigate within the file's data structure.
// Supports JSON (.json) and YAML (.yaml, .yml) formats.
func (p *FileProvider) Fetch(ctx context.Context, path []string) (any, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check context again before I/O
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read the configured file
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", p.filePath)
		}
		return nil, fmt.Errorf("failed to read file %q: %w", p.filePath, err)
	}

	// Determine file format by extension
	ext := strings.ToLower(filepath.Ext(p.filePath))

	var result any

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON file %q: %w", p.filePath, err)
		}

	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse YAML file %q: %w", p.filePath, err)
		}

	default:
		return nil, fmt.Errorf("unsupported file format %q for file %q (supported: .json, .yaml, .yml)", ext, p.filePath)
	}

	// If no path specified, return entire file content
	if len(path) == 0 {
		return result, nil
	}

	// Navigate to the requested path within the data structure
	current := result
	for i, key := range path {
		// Current must be a map to navigate further
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate to path %v: element at index %d is not a map (type: %T)", path, i, current)
		}

		// Get the value at this key
		val, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("path element %q not found in file %s", key, p.filePath)
		}

		current = val
	}

	return current, nil
}

// Info returns the provider's alias and version for metadata tracking.
func (p *FileProvider) Info() (alias string, version string) {
	return p.alias, "v0.1.0"
}
