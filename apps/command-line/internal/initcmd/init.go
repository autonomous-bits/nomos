// Package initcmd implements the nomos init command functionality.
package initcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Options holds configuration for the init command.
type Options struct {
	// Paths are the input .csl file paths to scan
	Paths []string

	// FromPaths maps provider alias to local binary path
	FromPaths map[string]string

	// DryRun previews actions without executing
	DryRun bool

	// Force overwrites existing providers/lockfile
	Force bool

	// OS overrides the target OS (default: runtime.GOOS)
	OS string

	// Arch overrides the target architecture (default: runtime.GOARCH)
	Arch string

	// Upgrade forces upgrade to latest versions
	Upgrade bool
}

// LockFile represents the .nomos/providers.lock.json structure.
type LockFile struct {
	Providers []ProviderEntry `json:"providers"`
}

// ProviderEntry represents a single provider in the lock file.
type ProviderEntry struct {
	Alias    string                 `json:"alias"`
	Type     string                 `json:"type"`
	Version  string                 `json:"version"`
	OS       string                 `json:"os"`
	Arch     string                 `json:"arch"`
	Source   map[string]interface{} `json:"source"`
	Checksum string                 `json:"checksum,omitempty"`
	Path     string                 `json:"path"`
}

// DiscoveredProvider represents a provider discovered from .csl files.
type DiscoveredProvider struct {
	Alias   string
	Type    string
	Version string
	Config  map[string]any
}

// Run executes the init command with the given options.
func Run(opts Options) error {
	// Set defaults for OS/Arch
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}

	// Discover providers from .csl files
	providers, err := discoverProviders(opts.Paths)
	if err != nil {
		return err
	}

	if len(providers) == 0 {
		fmt.Println("No providers found in source files.")
		return nil
	}

	// Validate all providers have versions
	for _, p := range providers {
		if p.Version == "" {
			return fmt.Errorf("provider %q (type: %s) missing required 'version' field in source declaration", p.Alias, p.Type)
		}
	}

	if opts.DryRun {
		fmt.Println("Dry run mode - would install:")
		for _, p := range providers {
			fmt.Printf("  - %s (type: %s, version: %s)\n", p.Alias, p.Type, p.Version)
		}
		return nil
	}

	// Install providers
	lockEntries := []ProviderEntry{}
	for _, p := range providers {
		entry, err := installProvider(p, opts)
		if err != nil {
			return fmt.Errorf("failed to install provider %q: %w", p.Alias, err)
		}
		lockEntries = append(lockEntries, entry)
	}

	// Write lock file
	lockFile := LockFile{Providers: lockEntries}
	if err := writeLockFile(lockFile); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	fmt.Printf("Successfully installed %d provider(s)\n", len(lockEntries))
	return nil
}

// discoverProviders scans .csl files and extracts provider requirements.
func discoverProviders(paths []string) ([]DiscoveredProvider, error) {
	providers := []DiscoveredProvider{}
	seen := make(map[string]bool)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer file.Close()

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

			// Extract version from config
			version := ""
			if v, ok := config["version"]; ok {
				if vStr, ok := v.(string); ok {
					version = vStr
				}
			}

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

// exprToValue converts an AST expression to a Go value.
func exprToValue(expr ast.Expr) any {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.ReferenceExpr:
		// For now, just return nil for references
		// They would need to be resolved against providers
		return nil
	default:
		return nil
	}
}

// installProvider installs a provider binary and returns its lock entry.
func installProvider(p DiscoveredProvider, opts Options) (ProviderEntry, error) {
	// Check if we have a local --from override
	if localPath, ok := opts.FromPaths[p.Alias]; ok {
		return installFromLocal(p, localPath, opts)
	}

	// For now, without network support, we can only handle --from paths
	return ProviderEntry{}, fmt.Errorf("provider %q requires --from flag (network download not yet implemented)", p.Alias)
}

// installFromLocal copies a provider binary from a local path.
func installFromLocal(p DiscoveredProvider, sourcePath string, opts Options) (ProviderEntry, error) {
	// Validate source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("source path does not exist: %s", sourcePath)
	}
	if info.IsDir() {
		return ProviderEntry{}, fmt.Errorf("source path must be a file, not a directory: %s", sourcePath)
	}

	// Determine installation path
	targetDir := filepath.Join(".nomos", "providers", p.Type, p.Version, fmt.Sprintf("%s-%s", opts.OS, opts.Arch))
	targetPath := filepath.Join(targetDir, "provider")

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Copy binary
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to read source: %w", err)
	}

	if err := os.WriteFile(targetPath, sourceData, 0755); err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to write target: %w", err)
	}

	// Create lock entry
	entry := ProviderEntry{
		Alias:   p.Alias,
		Type:    p.Type,
		Version: p.Version,
		OS:      opts.OS,
		Arch:    opts.Arch,
		Source: map[string]interface{}{
			"local": map[string]string{
				"path": sourcePath,
			},
		},
		Path: targetPath,
	}

	return entry, nil
}

// writeLockFile writes the lock file to .nomos/providers.lock.json.
func writeLockFile(lock LockFile) error {
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}

	// Write file
	return os.WriteFile(lockPath, data, 0644)
}
