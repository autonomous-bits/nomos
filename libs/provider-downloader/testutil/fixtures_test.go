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
