package providercmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestValidateProvider tests checksum verification and validation logic.
func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (tmpDir string, entry ProviderEntry)
		wantErr     bool
		errContains string
		errIs       error
	}{
		{
			name: "valid provider with matching checksum",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				tmpDir := t.TempDir()

				// Create provider binary
				binContent := []byte("fake provider binary content")
				binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
				if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
					t.Fatalf("failed to create provider dir: %v", err)
				}
				if err := os.WriteFile(binPath, binContent, 0755); err != nil {
					t.Fatalf("failed to write provider binary: %v", err)
				}

				// Calculate actual checksum
				hash := sha256.Sum256(binContent)
				checksum := hex.EncodeToString(hash[:])

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: checksum,
					Size:     int64(len(binContent)),
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr: false,
		},
		{
			name: "binary doesn't exist",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				tmpDir := t.TempDir()

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: "abc123",
					Size:     1024,
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr:     true,
			errContains: "provider binary not found",
		},
		{
			name: "checksum mismatch",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				tmpDir := t.TempDir()

				// Create provider binary with content
				binContent := []byte("fake provider binary content")
				binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
				if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
					t.Fatalf("failed to create provider dir: %v", err)
				}
				if err := os.WriteFile(binPath, binContent, 0755); err != nil {
					t.Fatalf("failed to write provider binary: %v", err)
				}

				// Use wrong checksum
				wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: wrongChecksum,
					Size:     int64(len(binContent)),
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr:     true,
			errIs:       ErrChecksumMismatch,
			errContains: "expected 0000000000",
		},
		{
			name: "checksum mismatch deletes corrupted binary",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				tmpDir := t.TempDir()

				// Create provider binary
				binContent := []byte("corrupted binary")
				binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
				if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
					t.Fatalf("failed to create provider dir: %v", err)
				}
				if err := os.WriteFile(binPath, binContent, 0755); err != nil {
					t.Fatalf("failed to write provider binary: %v", err)
				}

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: "wrongchecksum",
					Size:     int64(len(binContent)),
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr: true,
			errIs:   ErrChecksumMismatch,
		},
		{
			name: "non-executable on unix",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("skipping unix-specific test on windows")
				}

				tmpDir := t.TempDir()

				// Create provider binary without executable bit
				binContent := []byte("fake provider binary")
				binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
				if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
					t.Fatalf("failed to create provider dir: %v", err)
				}
				// Write with 0644 (no execute permission)
				if err := os.WriteFile(binPath, binContent, 0644); err != nil {
					t.Fatalf("failed to write provider binary: %v", err)
				}

				// Calculate checksum
				hash := sha256.Sum256(binContent)
				checksum := hex.EncodeToString(hash[:])

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: checksum,
					Size:     int64(len(binContent)),
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr:     true,
			errContains: "not executable",
		},
		{
			name: "empty file",
			setupFunc: func(t *testing.T) (string, ProviderEntry) {
				t.Helper()
				tmpDir := t.TempDir()

				// Create empty provider binary
				binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
				if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
					t.Fatalf("failed to create provider dir: %v", err)
				}
				if err := os.WriteFile(binPath, []byte{}, 0755); err != nil {
					t.Fatalf("failed to write provider binary: %v", err)
				}

				// Calculate checksum of empty file
				hash := sha256.Sum256([]byte{})
				checksum := hex.EncodeToString(hash[:])

				entry := ProviderEntry{
					Alias:    "test",
					Type:     "owner/repo",
					Version:  "1.0.0",
					OS:       "linux",
					Arch:     "amd64",
					Checksum: checksum,
					Size:     0,
					Path:     "owner/repo/1.0.0/linux-amd64/provider",
				}

				return tmpDir, entry
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, entry := tt.setupFunc(t)
			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to chdir: %v", err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			err := ValidateProvider(entry)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error = %v, want errors.Is(%v)", err, tt.errIs)
				}
			}
		})
	}
}

// TestValidateProvider_DeletesCorruptedBinary verifies corrupted files are deleted.
func TestValidateProvider_DeletesCorruptedBinary(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create corrupted provider binary
	binPath := filepath.Join(tmpDir, ".nomos", "providers", "owner", "repo", "1.0.0", "linux-amd64", "provider")
	if err := os.MkdirAll(filepath.Dir(binPath), 0750); err != nil {
		t.Fatalf("failed to create provider dir: %v", err)
	}
	if err := os.WriteFile(binPath, []byte("corrupted"), 0755); err != nil {
		t.Fatalf("failed to write provider binary: %v", err)
	}

	entry := ProviderEntry{
		Alias:    "test",
		Type:     "owner/repo",
		Version:  "1.0.0",
		OS:       "linux",
		Arch:     "amd64",
		Checksum: "wrongchecksum",
		Path:     "owner/repo/1.0.0/linux-amd64/provider",
	}

	// Validate (should fail and delete file)
	err := ValidateProvider(entry)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}

	// Verify file was deleted
	if _, statErr := os.Stat(binPath); !os.IsNotExist(statErr) {
		t.Error("corrupted binary should have been deleted")
	}
}
