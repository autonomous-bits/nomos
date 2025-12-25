package downloader

import (
	"crypto/sha256"
	"encoding/hex"
)

// computeSHA256 computes SHA256 checksum for test data.
// This is a shared test helper used across multiple test files.
func computeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
