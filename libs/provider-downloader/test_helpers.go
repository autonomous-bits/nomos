package downloader

import (
	"crypto/sha256"
	"encoding/hex"
)

// computeSHA256 computes SHA256 checksum for test data.
// This is a shared test helper used across multiple test files.
// Returns checksum in the format "sha256:hexdigest" to match the expected format.
func computeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}
