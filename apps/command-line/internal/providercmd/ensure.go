// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
)

// EnsureProviders is the primary entry point that orchestrates the complete
// provider management workflow. It discovers providers from .csl files,
// validates versions, downloads missing providers, and updates the lockfile.
//
// Workflow:
//
//  1. Discover Phase:
//     - Extracts provider declarations from .csl files
//     - Validates all providers have versions (returns ErrMissingVersion if not)
//     - Detects version conflicts (returns ErrVersionConflict if found)
//     - Returns empty summary if no providers found
//
//  2. Download Phase:
//     - Checks lockfile for cached providers
//     - Downloads missing/invalid providers
//     - Respects opts.DryRun (preview mode without downloads)
//     - Respects opts.AllowMissing (continue vs. fail-fast on errors)
//
//  3. Lockfile Update Phase:
//     - Reads existing lockfile
//     - Merges newly installed providers
//     - Writes updated lockfile atomically
//     - Skipped in dry-run mode
//
//  4. Summary Phase:
//     - Returns aggregate statistics (total, cached, downloaded, failed)
//
// Returns ProviderSummary with operation statistics and an error if the
// workflow cannot complete. Individual provider failures are recorded in
// the summary's Failed count and only cause an error return if
// opts.AllowMissing is false.
func EnsureProviders(opts ProviderOptions) (*ProviderSummary, error) {
	// Print progress message to stderr
	fmt.Fprintln(os.Stderr, "Checking providers...")

	// Validate inputs
	if len(opts.Paths) == 0 {
		return nil, errors.New("no input paths provided")
	}

	// Phase 1: Discover providers from .csl files
	providers, err := DiscoverProviders(opts.Paths)
	if err != nil {
		return nil, fmt.Errorf("failed to discover providers: %w", err)
	}

	// Handle empty provider list (success case)
	if len(providers) == 0 {
		return &ProviderSummary{
			Total:      0,
			Cached:     0,
			Downloaded: 0,
			Failed:     0,
		}, nil
	}

	// Validate all providers have versions
	if err := validateProviderVersions(providers); err != nil {
		return nil, err
	}

	// Detect version conflicts across files
	if err := detectVersionConflicts(providers); err != nil {
		return nil, err
	}

	// Phase 2: Download providers
	results, downloadEntries, err := downloadProvidersWithEntries(providers, opts)
	if err != nil {
		// Download failed with AllowMissing=false
		// Return partial results in the summary
		summary := buildSummary(results)
		return summary, err
	}

	// Check if all providers were cached
	summary := buildSummary(results)
	if summary.Downloaded == 0 && summary.Failed == 0 {
		fmt.Fprintln(os.Stderr, "(all cached)")
	}

	// Phase 3: Update lockfile (skip in dry-run mode)
	if !opts.DryRun {
		if err := updateLockfile(downloadEntries); err != nil {
			// Providers were downloaded successfully but lockfile update failed
			// This is a critical error - return with partial summary
			return summary, fmt.Errorf("providers downloaded but lockfile update failed: %w", err)
		}
	}

	// Phase 4: Return summary
	return summary, nil
}

// validateProviderVersions checks that all discovered providers have a version field.
// Returns ErrMissingVersion if any provider is missing the version.
func validateProviderVersions(providers []DiscoveredProvider) error {
	for _, p := range providers {
		if p.Version == "" {
			return fmt.Errorf("%w: provider %q (type %q) missing version field",
				ErrMissingVersion, p.Alias, p.Type)
		}
	}
	return nil
}

// detectVersionConflicts checks if the same provider type has different versions
// across multiple .csl files. Returns ErrVersionConflict if conflicts are found.
func detectVersionConflicts(providers []DiscoveredProvider) error {
	// Track versions for each provider type
	versions := make(map[string]map[string]bool) // type → versions

	for _, p := range providers {
		if _, exists := versions[p.Type]; !exists {
			versions[p.Type] = make(map[string]bool)
		}
		versions[p.Type][p.Version] = true
	}

	// Check for conflicts
	for providerType, versionSet := range versions {
		if len(versionSet) > 1 {
			// Collect version list for error message (sorted for determinism)
			versionList := make([]string, 0, len(versionSet))
			for v := range versionSet {
				versionList = append(versionList, v)
			}
			sort.Strings(versionList)

			return fmt.Errorf("%w: provider %q has conflicting versions: %v",
				ErrVersionConflict, providerType, versionList)
		}
	}

	return nil
}

// downloadProvidersWithEntries wraps DownloadProviders and returns both
// results and entries. This is a simple pass-through function that maintains
// the interface expected by EnsureProviders.
//
// Returns:
//   - results: Status of each provider (cached, installed, failed)
//   - entries: Complete ProviderEntry objects for newly installed providers only
//   - error: Critical failure if AllowMissing=false and download fails
func downloadProvidersWithEntries(providers []DiscoveredProvider, opts ProviderOptions) ([]ProviderResult, []ProviderEntry, error) {
	// Call DownloadProviders which now returns both results and entries
	return DownloadProviders(providers, opts)
}

// updateLockfile reads the existing lockfile, merges newly installed provider
// entries, and writes the updated lockfile atomically.
//
// This function receives complete ProviderEntry objects (with Checksum and
// Source metadata) for newly installed providers only.
func updateLockfile(newEntries []ProviderEntry) error {
	// If no new installations, skip lockfile update
	if len(newEntries) == 0 {
		return nil
	}

	// Read existing lockfile
	existingLock, err := ReadLockFile()
	if err != nil {
		// If file doesn't exist, that's OK - treat as empty lockfile
		if !os.IsNotExist(err) {
			// Log warning but continue with merge
			fmt.Fprintf(os.Stderr, "Warning: failed to read existing lockfile: %v\n", err)
		}
		existingLock = nil
	}

	// Merge lockfiles
	merged := MergeLockFiles(existingLock, newEntries)

	// Write merged lockfile
	if err := WriteLockFile(merged); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}

// buildSummary constructs a ProviderSummary from ProviderResult slice.
func buildSummary(results []ProviderResult) *ProviderSummary {
	summary := &ProviderSummary{
		Total:      len(results),
		Cached:     0,
		Downloaded: 0,
		Failed:     0,
	}

	for _, result := range results {
		switch result.Status {
		case ProviderStatusSkipped:
			summary.Cached++
		case ProviderStatusInstalled:
			summary.Downloaded++
		case ProviderStatusFailed:
			summary.Failed++
			// ProviderStatusDryRun doesn't increment any counter
			// as it's just a preview
		}
	}

	return summary
}
