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
// Returns a slice of ProviderResult (one per input provider) and an error if
// any critical operation fails. Individual provider failures are recorded in
// the ProviderResult.Error field with ProviderStatusFailed.
func DownloadProviders(providers []DiscoveredProvider, opts ProviderOptions) ([]ProviderResult, error) {
	results := make([]ProviderResult, 0, len(providers))

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
		return results, nil
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

		// Check if provider should be downloaded
		shouldDownload := opts.Force
		var existingEntry *ProviderEntry

		if !shouldDownload && existingLock != nil {
			existingEntry = findProviderInLockfile(existingLock, p.Alias, p.Type, p.Version, opts.OS, opts.Arch)
			if existingEntry != nil {
				// Validate existing provider binary
				if validateErr := ValidateProvider(*existingEntry); validateErr == nil {
					// Provider exists and is valid - skip download
					result.Status = ProviderStatusSkipped
					result.Path = existingEntry.Path
					result.Size = existingEntry.Size
					results = append(results, result)
					continue
				}
				// Validation failed - need to re-download
				shouldDownload = true
			} else {
				// Not in lockfile - need to download
				shouldDownload = true
			}
		}

		if !shouldDownload {
			shouldDownload = true
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
				return results, result.Error
			}
			continue
		}

		// Download succeeded
		result.Status = ProviderStatusInstalled
		result.Size = entry.Size
		result.Path = entry.Path
		results = append(results, result)
	}

	return results, nil
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
