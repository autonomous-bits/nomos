// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
	downloader "github.com/autonomous-bits/nomos/libs/provider-downloader"
)

// Options holds configuration for the init command.
type Options struct {
	// Paths are the input .csl file paths to scan
	Paths []string

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
	// Timestamp records when the lockfile was last written (RFC3339 format).
	Timestamp string          `json:"timestamp,omitempty"`
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
	Size     int64                  `json:"size,omitempty"`
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
// Returns InitResult with details of what was installed/skipped.
func Run(opts Options) (*InitResult, error) {
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
		return nil, err
	}

	result := &InitResult{
		DryRun:    opts.DryRun,
		Providers: make([]ProviderResult, 0, len(providers)),
	}

	if len(providers) == 0 {
		return result, nil
	}

	// Validate all providers have versions
	for _, p := range providers {
		if p.Version == "" {
			return nil, fmt.Errorf("provider %q (type: %s) missing required 'version' field in source declaration", p.Alias, p.Type)
		}
	}

	if opts.DryRun {
		// Populate result for dry-run
		for _, p := range providers {
			result.Providers = append(result.Providers, ProviderResult{
				Alias:   p.Alias,
				Type:    p.Type,
				Version: p.Version,
				OS:      opts.OS,
				Arch:    opts.Arch,
				Status:  ProviderStatusDryRun,
			})
		}
		return result, nil
	}

	// Read existing lockfile if present (unless --force)
	existingLock := readLockFile()

	// Install providers
	lockEntries := []ProviderEntry{}
	for _, p := range providers {
		// Check if provider already exists in lockfile with matching version
		if !opts.Force && existingLock != nil {
			if existing := findProviderInLock(existingLock, p.Alias, p.Type, p.Version, opts.OS, opts.Arch); existing != nil {
				// Provider already installed with matching version
				lockEntries = append(lockEntries, *existing)
				result.Skipped++
				result.Providers = append(result.Providers, ProviderResult{
					Alias:   p.Alias,
					Type:    p.Type,
					Version: p.Version,
					OS:      opts.OS,
					Arch:    opts.Arch,
					Status:  ProviderStatusSkipped,
					Path:    existing.Path,
				})
				continue
			}
		}

		entry, installErr := installProvider(p, opts)
		if installErr != nil {
			// Record failed installation
			result.Providers = append(result.Providers, ProviderResult{
				Alias:   p.Alias,
				Type:    p.Type,
				Version: p.Version,
				OS:      opts.OS,
				Arch:    opts.Arch,
				Status:  ProviderStatusFailed,
				Error:   installErr,
			})
			return result, fmt.Errorf("failed to install provider %q: %w", p.Alias, installErr)
		}
		lockEntries = append(lockEntries, entry)
		result.Installed++
		result.Providers = append(result.Providers, ProviderResult{
			Alias:   entry.Alias,
			Type:    entry.Type,
			Version: entry.Version,
			OS:      entry.OS,
			Arch:    entry.Arch,
			Status:  ProviderStatusInstalled,
			Size:    entry.Size,
			Path:    entry.Path,
		})
	}

	// Write lock file
	lockFile := LockFile{Providers: lockEntries}
	if err := writeLockFile(lockFile); err != nil {
		return result, fmt.Errorf("failed to write lock file: %w", err)
	}

	return result, nil
}

// discoverProviders scans .csl files and extracts provider requirements.
func discoverProviders(paths []string) ([]DiscoveredProvider, error) {
	providers := []DiscoveredProvider{}
	seen := make(map[string]bool)

	for _, path := range paths {
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

// installProvider installs a provider binary from GitHub Releases and returns its lock entry.
func installProvider(p DiscoveredProvider, opts Options) (ProviderEntry, error) {
	// Parse owner/repo from type (e.g., "autonomous-bits/nomos-provider-file")
	owner, repo, err := parseOwnerRepo(p.Type)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("provider %q: %w", p.Alias, err)
	}

	// Create context for download operations
	ctx := context.Background()

	// Create downloader client
	// Check for GitHub token in environment for higher rate limits
	githubToken := os.Getenv("GITHUB_TOKEN")
	client := downloader.NewClient(&downloader.ClientOptions{
		GitHubToken: githubToken,
	})

	// Build ProviderSpec
	spec := &downloader.ProviderSpec{
		Owner:   owner,
		Repo:    repo,
		Version: p.Version,
		OS:      opts.OS,
		Arch:    opts.Arch,
	}

	// Resolve asset from GitHub Releases
	asset, err := client.ResolveAsset(ctx, spec)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to resolve provider %q from GitHub: %w", p.Alias, err)
	}

	// Determine installation directory
	// Pattern: .nomos/providers/{owner}/{repo}/{version}/{os-arch}/
	destDir := filepath.Join(".nomos", "providers", owner, repo, p.Version, fmt.Sprintf("%s-%s", opts.OS, opts.Arch))

	// Download and install binary
	result, err := client.DownloadAndInstall(ctx, asset, destDir)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to download provider %q: %w", p.Alias, err)
	}

	// Normalize version format (add 'v' prefix if resolver normalized it)
	releaseTag := p.Version
	if len(p.Version) > 0 && p.Version[0] != 'v' {
		releaseTag = "v" + p.Version
	}

	// Build ProviderEntry with GitHub metadata
	// Store path relative to destDir for portability
	// The resolver will join this with the base directory at runtime
	relativePath := filepath.Join(owner, repo, p.Version, fmt.Sprintf("%s-%s", opts.OS, opts.Arch), "provider")

	entry := ProviderEntry{
		Alias:    p.Alias,
		Type:     p.Type,
		Version:  p.Version,
		OS:       opts.OS,
		Arch:     opts.Arch,
		Checksum: result.Checksum,
		Size:     result.Size,
		Path:     relativePath,
		Source: map[string]interface{}{
			"github": map[string]interface{}{
				"owner":       owner,
				"repo":        repo,
				"release_tag": releaseTag,
				"asset":       asset.Name,
			},
		},
	}

	return entry, nil
}

// parseOwnerRepo splits a provider type string in "owner/repo" format.
func parseOwnerRepo(providerType string) (owner, repo string, err error) {
	slashIdx := -1
	for i, c := range providerType {
		if c == '/' {
			if slashIdx != -1 {
				return "", "", fmt.Errorf("type %q contains multiple slashes", providerType)
			}
			slashIdx = i
		}
	}

	if slashIdx == -1 {
		return "", "", fmt.Errorf("type %q must be in 'owner/repo' format", providerType)
	}

	owner = providerType[:slashIdx]
	repo = providerType[slashIdx+1:]

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("type %q must be in 'owner/repo' format with non-empty owner and repo", providerType)
	}

	return owner, repo, nil
}

// writeLockFile writes the lock file to .nomos/providers.lock.json atomically.
// Uses temp file + rename pattern for crash safety.
func writeLockFile(lock LockFile) error {
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	// Ensure directory exists
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0750); err != nil {
		return err
	}

	// Set timestamp if not already set
	if lock.Timestamp == "" {
		lock.Timestamp = timeNowRFC3339()
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(lockDir, ".providers.lock.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tmpFile = nil // Prevent cleanup in defer

	// Atomic rename
	if err := os.Rename(tmpPath, lockPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up on rename failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// readLockFile reads the existing lockfile, returns nil if not found or invalid.
func readLockFile() *LockFile {
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	//nolint:gosec // G304: Path is hardcoded to .nomos/providers.lock.json, safe
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil // File doesn't exist or can't be read
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil // Invalid JSON
	}

	return &lock
}

// findProviderInLock searches for a provider in the lockfile that matches
// the given criteria. Returns nil if not found.
func findProviderInLock(lock *LockFile, alias, providerType, version, os, arch string) *ProviderEntry {
	for _, entry := range lock.Providers {
		if entry.Alias == alias &&
			entry.Type == providerType &&
			entry.Version == version &&
			entry.OS == os &&
			entry.Arch == arch {
			return &entry
		}
	}
	return nil
}

// timeNowRFC3339 returns the current time in RFC3339 format.
// Can be overridden in tests via NOMOS_TEST_TIMESTAMP env var.
func timeNowRFC3339() string {
	if testTime := os.Getenv("NOMOS_TEST_TIMESTAMP"); testTime != "" {
		return testTime
	}
	return time.Now().UTC().Format(time.RFC3339)
}
