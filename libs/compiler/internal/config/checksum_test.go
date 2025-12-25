package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/config"
)

// TestValidateChecksum_ValidChecksum tests successful checksum validation.
func TestValidateChecksum_ValidChecksum(t *testing.T) {
	// Create temp file with known content
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")
	content := []byte("test content for checksum validation")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Compute the expected checksum
	expectedChecksum, err := config.ComputeChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	// Validate - should succeed
	if err := config.ValidateChecksum(filePath, expectedChecksum); err != nil {
		t.Errorf("expected checksum validation to succeed, got error: %v", err)
	}
}

// TestValidateChecksum_MismatchChecksum tests checksum mismatch detection.
func TestValidateChecksum_MismatchChecksum(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")
	content := []byte("original content")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Compute checksum
	originalChecksum, err := config.ComputeChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	// Modify file content
	modifiedContent := []byte("modified content - different from original")
	if err := os.WriteFile(filePath, modifiedContent, 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Validate with original checksum - should fail
	err = config.ValidateChecksum(filePath, originalChecksum)
	if err == nil {
		t.Fatal("expected checksum validation to fail for modified file, got nil error")
	}

	// Error should mention mismatch
	errMsg := err.Error()
	if !contains(errMsg, "mismatch") {
		t.Errorf("expected error to mention 'mismatch', got: %v", err)
	}
}

// TestValidateChecksum_EmptyChecksum tests error when checksum is empty.
func TestValidateChecksum_EmptyChecksum(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")

	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Validate with empty checksum - should fail
	err := config.ValidateChecksum(filePath, "")
	if err == nil {
		t.Fatal("expected error for empty checksum, got nil")
	}

	// Error should mention security risk
	errMsg := err.Error()
	if !contains(errMsg, "empty") && !contains(errMsg, "security") {
		t.Errorf("expected error to mention security risk, got: %v", err)
	}
}

// TestValidateChecksum_InvalidFormat tests error when checksum format is invalid.
func TestValidateChecksum_InvalidFormat(t *testing.T) {
	testCases := []struct {
		name     string
		checksum string
	}{
		{"no prefix", "abc123def456"},
		{"wrong prefix", "md5:abc123def456"},
		{"short prefix", "sha25:abc123"},
	}

	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")

	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := config.ValidateChecksum(filePath, tc.checksum)
			if err == nil {
				t.Fatalf("expected error for invalid checksum format %q, got nil", tc.checksum)
			}

			// Error should mention format
			errMsg := err.Error()
			if !contains(errMsg, "format") {
				t.Errorf("expected error to mention format, got: %v", err)
			}
		})
	}
}

// TestValidateChecksum_FileNotFound tests error when file doesn't exist.
func TestValidateChecksum_FileNotFound(t *testing.T) {
	nonexistentPath := "/nonexistent/path/to/binary"
	checksum := "sha256:abc123def456"

	err := config.ValidateChecksum(nonexistentPath, checksum)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}

	// Error should be about file not found
	errMsg := err.Error()
	if !contains(errMsg, "open") && !contains(errMsg, "no such file") {
		t.Errorf("expected error about file not found, got: %v", err)
	}
}

// TestComputeChecksum_ValidFile tests successful checksum computation.
func TestComputeChecksum_ValidFile(t *testing.T) {
	// Create temp file with known content
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")
	content := []byte("test content")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Compute checksum
	checksum, err := config.ComputeChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	// Verify format
	if len(checksum) < 8 || checksum[:7] != "sha256:" {
		t.Errorf("expected checksum to start with 'sha256:', got: %s", checksum)
	}

	// Verify checksum is deterministic
	checksum2, err := config.ComputeChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to compute checksum again: %v", err)
	}

	if checksum != checksum2 {
		t.Errorf("expected deterministic checksum, got %s and %s", checksum, checksum2)
	}
}

// TestComputeChecksum_FileNotFound tests error when file doesn't exist.
func TestComputeChecksum_FileNotFound(t *testing.T) {
	nonexistentPath := "/nonexistent/path/to/binary"

	_, err := config.ComputeChecksum(nonexistentPath)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// TestComputeChecksum_DifferentContentDifferentChecksum verifies different content produces different checksums.
func TestComputeChecksum_DifferentContentDifferentChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first file
	file1 := filepath.Join(tmpDir, "file1")
	if err := os.WriteFile(file1, []byte("content A"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	// Create second file with different content
	file2 := filepath.Join(tmpDir, "file2")
	if err := os.WriteFile(file2, []byte("content B"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Compute checksums
	checksum1, err := config.ComputeChecksum(file1)
	if err != nil {
		t.Fatalf("failed to compute checksum for file1: %v", err)
	}

	checksum2, err := config.ComputeChecksum(file2)
	if err != nil {
		t.Fatalf("failed to compute checksum for file2: %v", err)
	}

	// Checksums should be different
	if checksum1 == checksum2 {
		t.Errorf("expected different checksums for different content, got same: %s", checksum1)
	}
}

// TestRoundTrip_ComputeAndValidate tests that computed checksums can be validated.
func TestRoundTrip_ComputeAndValidate(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-binary")
	content := []byte("binary content for round-trip test")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Compute checksum
	checksum, err := config.ComputeChecksum(filePath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	// Validate with computed checksum - should succeed
	if err := config.ValidateChecksum(filePath, checksum); err != nil {
		t.Errorf("failed to validate file with its own checksum: %v", err)
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
