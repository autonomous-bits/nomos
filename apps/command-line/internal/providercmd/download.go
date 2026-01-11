// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	downloader "github.com/autonomous-bits/nomos/libs/provider-downloader"
)

// DownloadProviders downloads provider binaries for the given discovered providers.
// It checks the existing lockfile and only downloads providers that are missing
// or have mismatched versions/checksums.
//
// The function performs the following steps:
//  1. Reads the existing lockfile (if present)
//  2. For each provider:
//     - If opts.Force is true: downloads regardless of lockfile
//     - Otherwise: checks if provider exists in lockfile with matching version/OS/arch
//     - If found in lockfile with matching metadata: skips download
//     - If not found or mismatched: downloads the provider binary
//  3. Returns an array of ProviderResult for each provider
//
// In dry-run mode (opts.DryRun), the function returns preview results without
// performing actual downloads.
//
// Returns:
//   - results: Status of each provider (cached, installed, failed)
//   - entries: Complete ProviderEntry objects for newly installed providers only
//   - error: Critical failure if any operation fails
//
// Individual provider failures are recorded in the ProviderResult.Error field
// with ProviderStatusFailed.
func DownloadProviders(providers []DiscoveredProvider, opts ProviderOptions) ([]ProviderResult, []ProviderEntry, error) {
	results := make([]ProviderResult, 0, len(providers))
	entries := make([]ProviderEntry, 0)

	// Handle dry-run mode early
	if opts.DryRun {
		for _, p := range providers {
			results = append(results, ProviderResult{
				Alias:   p.Alias,
				Type:    p.Type,
				Version: p.Version,
				OS:      opts.OS,
				Arch:    opts.Arch,
				Status:  ProviderStatusDryRun,
			})
		}
		return results, entries, nil
	}

	// Read existing lockfile (returns error if file exists but is invalid)
	existingLock, err := ReadLockFile()
	if err != nil {
		// If file doesn't exist, that's OK - treat as empty lockfile
		if !os.IsNotExist(err) {
			// Log warning but continue - don't fail the build
			fmt.Fprintf(os.Stderr, "Warning: failed to read lockfile: %v\n", err)
		}
		existingLock = nil
	}

	// Process each provider
	for _, p := range providers {
		result := ProviderResult{
			Alias:   p.Alias,
			Type:    p.Type,
			Version: p.Version,
			OS:      opts.OS,
			Arch:    opts.Arch,
		}

		// T043: When force flag is set, skip lockfile check and force re-download
		if opts.Force {
			// T044: Delete existing cached binary before re-download
			if existingLock != nil {
				if existingEntry := findProviderInLockfile(existingLock, p.Alias, p.Type, p.Version, opts.OS, opts.Arch); existingEntry != nil {
					deleteProviderBinary(*existingEntry)
				}
			}
		} else {
			// Normal flow: Check lockfile for existing valid provider
			if existingLock != nil {
				if existingEntry := findProviderInLockfile(existingLock, p.Alias, p.Type, p.Version, opts.OS, opts.Arch); existingEntry != nil {
					// Validate existing provider binary
					if validateErr := ValidateProvider(*existingEntry); validateErr == nil {
						// Provider exists and is valid - skip download
						result.Status = ProviderStatusSkipped
						result.Path = existingEntry.Path
						result.Size = existingEntry.Size
						results = append(results, result)
						continue
					}
					// Validation failed - will re-download below
				}
			}
		}

		// Download provider
		entry, downloadErr := downloadProvider(p, opts)
		if downloadErr != nil {
			result.Status = ProviderStatusFailed
			result.Error = fmt.Errorf("failed to download provider %q: %w", p.Alias, downloadErr)
			results = append(results, result)

			// Continue to next provider (don't fail fast unless user wants strict behavior)
			if !opts.AllowMissing {
				// Return early with accumulated results so far
				return results, entries, result.Error
			}
			continue
		}

		// Download succeeded
		result.Status = ProviderStatusInstalled
		result.Size = entry.Size
		result.Path = entry.Path
		results = append(results, result)

		// Collect entry for lockfile update
		entries = append(entries, entry)
	}

	return results, entries, nil
}

// downloadProvider downloads and installs a single provider binary.
// This is extracted from the existing installProvider() logic in init.go
// for reuse across different command contexts.
func downloadProvider(p DiscoveredProvider, opts ProviderOptions) (ProviderEntry, error) {
	// Parse owner/repo from provider type
	owner, repo, err := parseOwnerRepo(p.Type)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("invalid provider type: %w", err)
	}

	// Print download progress message to stderr
	fmt.Fprintf(os.Stderr, "Downloading %s/%s@%s for %s-%s...\n",
		owner, repo, p.Version, opts.OS, opts.Arch)

	// Create context for download operations
	ctx := context.Background()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create downloader client with optional GitHub token
	client := downloader.NewClient(&downloader.ClientOptions{
		GitHubToken: opts.GitHubToken,
	})

	// Build ProviderSpec for downloader
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
		return ProviderEntry{}, fmt.Errorf("failed to resolve provider from GitHub: %w", err)
	}

	// Determine installation directory
	// Pattern: .nomos/providers/{owner}/{repo}/{version}/{os-arch}/
	destDir := filepath.Join(".nomos", "providers", owner, repo, p.Version, fmt.Sprintf("%s-%s", opts.OS, opts.Arch))

	// Download and install binary
	result, err := client.DownloadAndInstall(ctx, asset, destDir)
	if err != nil {
		return ProviderEntry{}, fmt.Errorf("failed to download provider binary: %w", err)
	}

	// Normalize version format (add 'v' prefix if missing)
	releaseTag := p.Version
	if len(p.Version) > 0 && p.Version[0] != 'v' {
		releaseTag = "v" + p.Version
	}

	// Build relative path for lockfile entry
	// Path is relative to .nomos/providers/ for portability
	relativePath := filepath.Join(owner, repo, p.Version, fmt.Sprintf("%s-%s", opts.OS, opts.Arch), "provider")

	// Construct ProviderEntry with GitHub metadata
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

// findProviderInLockfile searches for a provider in the lockfile that matches
// the given criteria. Returns nil if not found.
func findProviderInLockfile(lock *LockFile, alias, providerType, version, os, arch string) *ProviderEntry {
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

// deleteProviderBinary removes an existing provider binary from the cache.
// It constructs the full path from the lockfile entry and removes the file.
// If the file doesn't exist, no error is returned. Deletion failures are
// logged as warnings but don't fail the operation - the download will
// overwrite the file anyway.
func deleteProviderBinary(entry ProviderEntry) {
	if entry.Path == "" {
		return
	}

	// Build full path to provider binary
	fullPath := filepath.Join(".nomos", "providers", entry.Path)

	// Attempt to remove the file
	if err := os.Remove(fullPath); err != nil {
		// Only log if it's not a "file doesn't exist" error
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete cached binary %s: %v\n", fullPath, err)
		}
		// Continue regardless - download will overwrite
	}
}
