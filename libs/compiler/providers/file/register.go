package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// RegisterFileProvider registers a file provider with the given registry.
//
// The provider now requires a directory path containing .csl files. This is
// a breaking change from v0.1.x which accepted a single file path.
//
// Parameters:
//   - registry: The provider registry to register with
//   - alias: The alias to register the provider under (e.g., "configs")
//   - directory: The directory path containing .csl files
//
// Example:
//
//	registry := compiler.NewProviderRegistry()
//	if err := RegisterFileProvider(registry, "configs", "./config-files"); err != nil {
//	    log.Fatal(err)
//	}
//
// The directory is validated immediately during registration:
//   - Must exist
//   - Must be a directory (not a file)
//   - Must contain at least one .csl file
//   - Must not have duplicate base names
func RegisterFileProvider(registry compiler.ProviderRegistry, alias string, directory string) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	if directory == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to resolve directory to absolute path: %w", err)
	}

	// Validate directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Validate it's a directory, not a file
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Enumerate .csl files in the directory
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", absPath, err)
	}

	// Build map of base names -> file paths
	cslFiles := make(map[string]string)
	for _, entry := range entries {
		// Skip directories and non-.csl files
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".csl") {
			continue
		}

		// Extract base name (without .csl extension)
		baseName := strings.TrimSuffix(fileName, ".csl")

		// Check for duplicates
		if _, exists := cslFiles[baseName]; exists {
			return fmt.Errorf("duplicate file base name %q found in directory %s", baseName, absPath)
		}

		// Store the full path
		cslFiles[baseName] = filepath.Join(absPath, fileName)
	}

	// Validate at least one .csl file exists
	if len(cslFiles) == 0 {
		return fmt.Errorf("no .csl files found in directory: %s", absPath)
	}

	// Create a constructor function that captures the directory and file map
	constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		provider := &FileProvider{
			directory: absPath,
			cslFiles:  cslFiles,
		}

		// Return the provider without additional initialization
		return provider, nil
	}

	registry.Register(alias, constructor)
	return nil
}
