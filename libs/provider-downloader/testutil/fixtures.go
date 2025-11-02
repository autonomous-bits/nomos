package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

// BinaryFixture represents a generated binary fixture for testing.
type BinaryFixture struct {
	Content  []byte
	Checksum string
	Size     int64
}

// CreateBinaryFixture generates a fake binary with specified size and optional prefix.
func CreateBinaryFixture(t *testing.T, size int, prefix string) *BinaryFixture {
	t.Helper()

	if size <= 0 {
		t.Fatalf("size must be positive, got %d", size)
	}

	content := make([]byte, size)
	prefixBytes := []byte(prefix)
	copy(content, prefixBytes)

	for i := len(prefixBytes); i < size; i++ {
		content[i] = byte((i * 7) % 256)
	}

	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	return &BinaryFixture{
		Content:  content,
		Checksum: checksum,
		Size:     int64(size),
	}
}

// CreateCorruptedFixture creates a fixture where the content doesn't match the checksum.
func CreateCorruptedFixture(t *testing.T, size int, prefix string) *BinaryFixture {
	t.Helper()

	fixture := CreateBinaryFixture(t, size, prefix)

	if len(fixture.Content) > 0 {
		fixture.Content[0] ^= 0xFF
	}

	return fixture
}

// String provides a human-readable representation for debugging.
func (f *BinaryFixture) String() string {
	return fmt.Sprintf("BinaryFixture{Size: %d, Checksum: %s...}", f.Size, f.Checksum[:8])
}
