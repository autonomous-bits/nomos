package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ValidateChecksum verifies that a file's SHA256 checksum matches the expected value.
// The expected checksum should be in the format "sha256:hexdigest".
// Returns nil if the checksum matches, error otherwise.
func ValidateChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return fmt.Errorf("checksum is empty - cannot validate binary (security risk)")
	}

	// Parse expected checksum format
	if len(expectedChecksum) < 8 || expectedChecksum[:7] != "sha256:" {
		return fmt.Errorf("invalid checksum format %q: expected 'sha256:hexdigest'", expectedChecksum)
	}
	expectedHash := expectedChecksum[7:]

	// Open file
	f, err := os.Open(filePath) //nolint:gosec // G304: File path from lockfile/config, validated by caller
	if err != nil {
		return fmt.Errorf("failed to open file for checksum validation: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Compute SHA256
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s (file may be corrupted or tampered with)", expectedHash, actualHash)
	}

	return nil
}

// ComputeChecksum computes the SHA256 checksum of a file and returns it in the format "sha256:hexdigest".
// This is useful for generating checksums when creating lockfiles.
func ComputeChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath) //nolint:gosec // G304: File path from lockfile/config, validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return "sha256:" + hash, nil
}
