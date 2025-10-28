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
	"github.com/autonomous-bits/nomos/libs/compiler/internal/converter"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/parse"
	"gopkg.in/yaml.v3"
)

// FileProvider implements the compiler.Provider interface for local file access.
// In v0.2.0+, it supports directory-based .csl file resolution with named imports.
type FileProvider struct {
	// v0.2.0+ fields for directory-based provider
	directory string            // absolute path to directory containing .csl files
	cslFiles  map[string]string // map of base name -> absolute file path

	// Legacy fields (kept for backward compatibility during migration)
	filePath       string
	alias          string
	configFilePath string // file path from config, used during Init
}

// Init initializes the file provider with the given options.
// In v0.2.0+, the provider operates in directory mode if directory and cslFiles
// are already set (via RegisterFileProvider). No additional init needed in this case.
func (p *FileProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	p.alias = opts.Alias

	// If directory and cslFiles are already set (from RegisterFileProvider),
	// provider is already initialized - no additional work needed
	if p.directory != "" && p.cslFiles != nil {
		return nil
	}

	// Legacy path: extract file/directory from config (for backward compatibility during transition)
	var configPath string
	if p.configFilePath != "" {
		configPath = p.configFilePath
	} else {
		// Try 'directory' config first (new v0.2.0 approach)
		if dirVal, ok := opts.Config["directory"]; ok {
			if dirStr, ok := dirVal.(string); ok {
				configPath = dirStr
			} else {
				return fmt.Errorf("directory must be a string, got %T", dirVal)
			}
		} else if fileVal, ok := opts.Config["file"]; ok {
			// Fall back to 'file' config (legacy)
			if fileStr, ok := fileVal.(string); ok {
				configPath = fileStr
			} else {
				return fmt.Errorf("file must be a string, got %T", fileVal)
			}
		} else {
			return errors.New("file provider requires 'directory' or 'file' in config")
		}
	}

	// Resolve to absolute path
	// If the path is relative and we have a source file path, resolve relative to the source file's directory
	var absPath string
	if !filepath.IsAbs(configPath) && opts.SourceFilePath != "" {
		// Resolve relative to the source file's directory
		sourceDir := filepath.Dir(opts.SourceFilePath)
		absPath = filepath.Join(sourceDir, configPath)
	} else {
		// Absolute path or no source file path - resolve from current directory
		var err error
		absPath, err = filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path to absolute: %w", err)
		}
	}

	// Check if it's a directory or file
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Directory mode - enumerate .csl files
		return p.initDirectoryMode(absPath)
	}

	// File mode - REJECT (no longer supported in v0.2.0+)
	return fmt.Errorf("file mode is no longer supported; provider must point to a directory containing .csl files (got file: %s)", absPath)
}

// initDirectoryMode initializes the provider in directory mode.
func (p *FileProvider) initDirectoryMode(absPath string) error {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", absPath, err)
	}

	// Build map of base names -> file paths
	cslFiles := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".csl") {
			continue
		}

		baseName := strings.TrimSuffix(fileName, ".csl")

		if _, exists := cslFiles[baseName]; exists {
			return fmt.Errorf("duplicate file base name %q found in directory %s", baseName, absPath)
		}

		cslFiles[baseName] = filepath.Join(absPath, fileName)
	}

	if len(cslFiles) == 0 {
		return fmt.Errorf("no .csl files found in directory: %s", absPath)
	}

	p.directory = absPath
	p.cslFiles = cslFiles

	return nil
}

// Fetch retrieves content from the provider based on the path.
// In v0.2.0+ directory mode, path[0] is the base name of the .csl file to fetch.
// Returns the raw file content as a string.
func (p *FileProvider) Fetch(ctx context.Context, path []string) (any, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Directory mode (v0.2.0+)
	if p.directory != "" && p.cslFiles != nil {
		return p.fetchFromDirectory(ctx, path)
	}

	// Legacy file mode - should not reach here in v0.2.0+
	// Kept for backward compatibility during migration
	return p.fetchLegacyFile(ctx, path)
}

// fetchFromDirectory fetches a .csl file by base name from the directory.
// For .csl files, it parses the file and returns structured data (map[string]any).
// If additional path components are provided, navigates to that path within the file's data.
func (p *FileProvider) fetchFromDirectory(ctx context.Context, path []string) (any, error) {
	// Path must have at least one element (the base name)
	if len(path) == 0 {
		return nil, fmt.Errorf("path cannot be empty; specify file base name to fetch")
	}

	// Check for context cancellation before I/O
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// First path component is the base name
	baseName := path[0]

	// Look up file path
	filePath, exists := p.cslFiles[baseName]
	if !exists {
		return nil, fmt.Errorf("import file %q not found in provider %q", baseName, p.alias)
	}

	// Parse the .csl file and return structured data
	tree, _, err := parse.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .csl file %q: %w", filePath, err)
	}

	// Convert AST to data
	data, err := converter.ASTToData(tree)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AST to data for %q: %w", filePath, err)
	}

	// If no additional path components, return entire file data
	if len(path) == 1 {
		return data, nil
	}

	// Navigate to the requested path within the data
	var current any = data
	for i, key := range path[1:] {
		// Current must be a map to navigate further
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate to path %v in file %q: element at index %d is not a map (type: %T)", path, baseName, i+1, current)
		}

		// Get the value at this key
		val, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("path element %q not found in file %q (full path: %v)", key, baseName, path)
		}

		current = val
	}

	return current, nil
}

// fetchLegacyFile handles the legacy file-based fetch (deprecated in v0.2.0).
func (p *FileProvider) fetchLegacyFile(ctx context.Context, path []string) (any, error) {
	// Check context
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
	return p.alias, "v0.2.0"
}
