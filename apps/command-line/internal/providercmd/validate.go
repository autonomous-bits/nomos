// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ValidateProvider performs checksum verification on an installed provider binary.
// It checks that the binary exists, computes its SHA256 checksum, and compares it
// with the expected checksum from the lockfile entry.
//
// On Unix systems, it also verifies that the binary has the executable permission bit set.
//
// If the checksum doesn't match, the corrupted binary is deleted and ErrChecksumMismatch
// is returned. The caller is responsible for re-downloading the provider.
//
// Returns nil if validation succeeds, or an error describing the validation failure.
func ValidateProvider(entry ProviderEntry) error {
	// Build full path to provider binary
	fullPath := filepath.Join(".nomos", "providers", entry.Path)

	// Check if binary exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("provider binary not found at %s: %w", fullPath, err)
		}
		return fmt.Errorf("failed to stat provider binary %s: %w", fullPath, err)
	}

	// Compute SHA256 checksum
	//nolint:gosec // G304: Path is constructed from validated lockfile entry
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read provider binary for checksum: %w", err)
	}

	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	// Normalize expected checksum (handle both "sha256:..." and plain hex formats)
	expectedChecksum := entry.Checksum
	if len(expectedChecksum) > 7 && expectedChecksum[:7] == "sha256:" {
		expectedChecksum = expectedChecksum[7:]
	}

	// Compare checksums
	if actualChecksum != expectedChecksum {
		// Delete corrupted binary
		if removeErr := os.Remove(fullPath); removeErr != nil {
			// Log but don't fail - return the checksum error
			fmt.Fprintf(os.Stderr, "Warning: failed to delete corrupted binary %s: %v\n", fullPath, removeErr)
		}
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, entry.Checksum, actualChecksum)
	}

	// On Unix systems, verify executable bit is set
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("provider binary %s is not executable", fullPath)
		}
	}

	return nil
}
