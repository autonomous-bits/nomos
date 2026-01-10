// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadLockFile reads the existing lockfile from .nomos/providers.lock.json.
// Returns the parsed lockfile or an error if the file doesn't exist or
// contains invalid JSON.
//
// This differs from the previous implementation which returned nil on error;
// it now returns explicit errors for better error handling.
func ReadLockFile() (*LockFile, error) {
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	//nolint:gosec // G304: Path is hardcoded to .nomos/providers.lock.json, safe
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile JSON: %w", err)
	}

	return &lock, nil
}

// WriteLockFile writes the lock file to .nomos/providers.lock.json atomically.
// Uses temp file + rename pattern for crash safety.
//
// The timestamp field is automatically set to the current time in RFC3339 format
// if not already set.
func WriteLockFile(lock LockFile) error {
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	// Ensure directory exists
	lockDir := filepath.Dir(lockPath)
	if err := os.MkdirAll(lockDir, 0750); err != nil {
		return fmt.Errorf("failed to create lockfile directory: %w", err)
	}

	// Set timestamp if not already set
	if lock.Timestamp == "" {
		lock.Timestamp = timeNowRFC3339()
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
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

// MergeLockFiles merges an existing lockfile with new provider entries.
// It preserves all existing entries and updates them with matching new entries
// based on alias, type, OS, and arch. New entries that don't match existing ones
// are appended.
//
// This function never removes entries automatically - manual cleanup must be
// performed separately if needed.
//
// The returned lockfile has a fresh timestamp set.
func MergeLockFiles(existing *LockFile, newEntries []ProviderEntry) LockFile {
	merged := LockFile{
		Timestamp: timeNowRFC3339(),
		Providers: []ProviderEntry{},
	}

	// Add all existing entries
	if existing != nil {
		merged.Providers = append(merged.Providers, existing.Providers...)
	}

	// Add or update with new entries
	for _, newEntry := range newEntries {
		found := false
		for i, existingEntry := range merged.Providers {
			if existingEntry.Alias == newEntry.Alias &&
				existingEntry.Type == newEntry.Type &&
				existingEntry.OS == newEntry.OS &&
				existingEntry.Arch == newEntry.Arch {
				// Update existing entry
				merged.Providers[i] = newEntry
				found = true
				break
			}
		}
		if !found {
			// Append new entry
			merged.Providers = append(merged.Providers, newEntry)
		}
	}

	return merged
}
