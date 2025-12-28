package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCreateBinaryFixture_GeneratesConsistentContent(t *testing.T) {
	fixture1 := CreateBinaryFixture(t, 1024, "test-provider")
	fixture2 := CreateBinaryFixture(t, 1024, "test-provider")

	if fixture1.Size != fixture2.Size {
		t.Errorf("size mismatch: %d != %d", fixture1.Size, fixture2.Size)
	}

	if fixture1.Checksum != fixture2.Checksum {
		t.Errorf("checksum mismatch: %s != %s", fixture1.Checksum, fixture2.Checksum)
	}

	if string(fixture1.Content) != string(fixture2.Content) {
		t.Error("content mismatch: fixtures not deterministic")
	}
}

func TestCreateBinaryFixture_CorrectChecksum(t *testing.T) {
	fixture := CreateBinaryFixture(t, 512, "my-provider")

	hash := sha256.Sum256(fixture.Content)
	expectedChecksum := hex.EncodeToString(hash[:])

	if fixture.Checksum != expectedChecksum {
		t.Errorf("checksum incorrect: got %s, want %s", fixture.Checksum, expectedChecksum)
	}
}

func TestCreateCorruptedFixture_ProducesMismatch(t *testing.T) {
	corrupted := CreateCorruptedFixture(t, 1024, "test")

	hash := sha256.Sum256(corrupted.Content)
	actualChecksum := hex.EncodeToString(hash[:])

	if corrupted.Checksum == actualChecksum {
		t.Error("expected checksum mismatch for corrupted fixture")
	}
}

// TestBinaryFixture_String tests the String method.
func TestBinaryFixture_String(t *testing.T) {
	fixture := CreateBinaryFixture(t, 1024, "test")
	result := fixture.String()

	if !stringContains(result, "BinaryFixture") {
		t.Errorf("expected String to contain 'BinaryFixture', got %q", result)
	}
	if !stringContains(result, "Size: 1024") {
		t.Errorf("expected String to contain 'Size: 1024', got %q", result)
	}
	if !stringContains(result, "Checksum:") {
		t.Errorf("expected String to contain 'Checksum:', got %q", result)
	}
}

// stringContains checks if a string contains a substring.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
