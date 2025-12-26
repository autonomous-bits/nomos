package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLockfileProviderResolver_ResolveBinaryPath(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create the provider binary first (so we can compute its checksum)
	providerDir := filepath.Join(tmpDir, "providers", "file", "0.2.0", "darwin-arm64")
	if err := os.MkdirAll(providerDir, 0755); err != nil { //nolint:gosec // G301: Test fixture directory
		t.Fatalf("failed to create provider directory: %v", err)
	}
	providerPath := filepath.Join(providerDir, "provider")
	providerContent := []byte("test provider binary")
	if err := os.WriteFile(providerPath, providerContent, 0755); err != nil { //nolint:gosec // G306: Test provider binary requires executable permissions
		t.Fatalf("failed to create provider binary: %v", err)
	}

	// Compute checksum for the provider binary
	checksum, err := ComputeChecksum(providerPath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	// Create lockfile with checksum
	lockfilePath := filepath.Join(tmpDir, "providers.lock.json")
	lockfile := &Lockfile{
		Providers: []Provider{
			{
				Alias:    "configs",
				Type:     "file",
				Version:  "0.2.0",
				OS:       "darwin",
				Arch:     "arm64",
				Path:     "file/0.2.0/darwin-arm64/provider",
				Checksum: checksum,
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create resolver
	baseDirFunc := func() string {
		return filepath.Join(tmpDir, "providers")
	}
	resolver, err := NewLockfileProviderResolver(lockfilePath, "", baseDirFunc)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	t.Run("resolves existing provider type", func(t *testing.T) {
		ctx := context.Background()
		binaryPath, err := resolver.ResolveBinaryPath(ctx, "file")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := providerPath
		if binaryPath != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, binaryPath)
		}
	})

	t.Run("returns error for non-existent provider type", func(t *testing.T) {
		ctx := context.Background()
		_, err := resolver.ResolveBinaryPath(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent provider type")
		}

		expectedMsg := "provider type \"nonexistent\" not found in lockfile"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("expected error message starting with %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("returns error when binary file does not exist", func(t *testing.T) {
		// Create a lockfile with a provider whose binary doesn't exist
		lockfilePath2 := filepath.Join(tmpDir, "providers2.lock.json")
		lockfile2 := &Lockfile{
			Providers: []Provider{
				{
					Alias:   "missing",
					Type:    "missing",
					Version: "1.0.0",
					OS:      "darwin",
					Arch:    "arm64",
					Path:    "missing/1.0.0/darwin-arm64/provider",
				},
			},
		}
		if err := lockfile2.Save(lockfilePath2); err != nil {
			t.Fatalf("failed to save lockfile2: %v", err)
		}

		resolver2, err := NewLockfileProviderResolver(lockfilePath2, "", baseDirFunc)
		if err != nil {
			t.Fatalf("failed to create resolver2: %v", err)
		}

		ctx := context.Background()
		_, err = resolver2.ResolveBinaryPath(ctx, "missing")
		if err == nil {
			t.Fatal("expected error when binary file does not exist")
		}

		expectedMsg := "provider binary not found"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("expected error message starting with %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestLockfileProviderResolver_ChecksumValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a provider binary with known content
	providerDir := filepath.Join(tmpDir, "providers", "file", "0.2.0", "darwin-arm64")
	if err := os.MkdirAll(providerDir, 0755); err != nil { //nolint:gosec // G301: Test fixture directory
		t.Fatalf("failed to create provider directory: %v", err)
	}
	providerPath := filepath.Join(providerDir, "provider")
	providerContent := []byte("test provider binary content")
	if err := os.WriteFile(providerPath, providerContent, 0755); err != nil { //nolint:gosec // G306: Test provider binary requires executable permissions
		t.Fatalf("failed to create provider binary: %v", err)
	}

	// Compute the correct checksum
	correctChecksum, err := ComputeChecksum(providerPath)
	if err != nil {
		t.Fatalf("failed to compute checksum: %v", err)
	}

	t.Run("succeeds with valid checksum", func(t *testing.T) {
		lockfilePath := filepath.Join(tmpDir, "valid-checksum.lock.json")
		lockfile := &Lockfile{
			Providers: []Provider{
				{
					Alias:    "configs",
					Type:     "file",
					Version:  "0.2.0",
					OS:       "darwin",
					Arch:     "arm64",
					Path:     "file/0.2.0/darwin-arm64/provider",
					Checksum: correctChecksum,
				},
			},
		}
		if err := lockfile.Save(lockfilePath); err != nil {
			t.Fatalf("failed to save lockfile: %v", err)
		}

		baseDirFunc := func() string {
			return filepath.Join(tmpDir, "providers")
		}
		resolver, err := NewLockfileProviderResolver(lockfilePath, "", baseDirFunc)
		if err != nil {
			t.Fatalf("failed to create resolver: %v", err)
		}

		ctx := context.Background()
		binaryPath, err := resolver.ResolveBinaryPath(ctx, "file")
		if err != nil {
			t.Fatalf("expected success with valid checksum, got error: %v", err)
		}

		if binaryPath != providerPath {
			t.Errorf("expected path %q, got %q", providerPath, binaryPath)
		}
	})

	t.Run("fails with invalid checksum", func(t *testing.T) {
		// Use a different checksum (tampered binary scenario)
		invalidChecksum := "sha256:0000000000000000000000000000000000000000000000000000000000000000"

		lockfilePath := filepath.Join(tmpDir, "invalid-checksum.lock.json")
		lockfile := &Lockfile{
			Providers: []Provider{
				{
					Alias:    "configs",
					Type:     "file",
					Version:  "0.2.0",
					OS:       "darwin",
					Arch:     "arm64",
					Path:     "file/0.2.0/darwin-arm64/provider",
					Checksum: invalidChecksum,
				},
			},
		}
		if err := lockfile.Save(lockfilePath); err != nil {
			t.Fatalf("failed to save lockfile: %v", err)
		}

		baseDirFunc := func() string {
			return filepath.Join(tmpDir, "providers")
		}
		resolver, err := NewLockfileProviderResolver(lockfilePath, "", baseDirFunc)
		if err != nil {
			t.Fatalf("failed to create resolver: %v", err)
		}

		ctx := context.Background()
		_, err = resolver.ResolveBinaryPath(ctx, "file")
		if err == nil {
			t.Fatal("expected error with invalid checksum, got nil")
		}

		// Error should mention checksum validation failure
		errMsg := err.Error()
		if !contains(errMsg, "checksum validation failed") && !contains(errMsg, "mismatch") {
			t.Errorf("expected error to mention checksum validation failure, got: %v", err)
		}
	})

	t.Run("fails with missing checksum", func(t *testing.T) {
		lockfilePath := filepath.Join(tmpDir, "missing-checksum.lock.json")
		lockfile := &Lockfile{
			Providers: []Provider{
				{
					Alias:    "configs",
					Type:     "file",
					Version:  "0.2.0",
					OS:       "darwin",
					Arch:     "arm64",
					Path:     "file/0.2.0/darwin-arm64/provider",
					Checksum: "", // Missing checksum
				},
			},
		}
		if err := lockfile.Save(lockfilePath); err != nil {
			t.Fatalf("failed to save lockfile: %v", err)
		}

		baseDirFunc := func() string {
			return filepath.Join(tmpDir, "providers")
		}
		resolver, err := NewLockfileProviderResolver(lockfilePath, "", baseDirFunc)
		if err != nil {
			t.Fatalf("failed to create resolver: %v", err)
		}

		ctx := context.Background()
		_, err = resolver.ResolveBinaryPath(ctx, "file")
		if err == nil {
			t.Fatal("expected error with missing checksum, got nil")
		}

		// Error should mention missing checksum and security risk
		errMsg := err.Error()
		if !contains(errMsg, "no checksum") && !contains(errMsg, "security risk") {
			t.Errorf("expected error to mention missing checksum and security risk, got: %v", err)
		}
	})

	t.Run("fails when binary is modified after lockfile creation", func(t *testing.T) {
		// This simulates a real attack scenario where the binary is replaced
		lockfilePath := filepath.Join(tmpDir, "tampered-binary.lock.json")
		lockfile := &Lockfile{
			Providers: []Provider{
				{
					Alias:    "configs",
					Type:     "file",
					Version:  "0.2.0",
					OS:       "darwin",
					Arch:     "arm64",
					Path:     "file/0.2.0/darwin-arm64/provider",
					Checksum: correctChecksum, // Checksum of original content
				},
			},
		}
		if err := lockfile.Save(lockfilePath); err != nil {
			t.Fatalf("failed to save lockfile: %v", err)
		}

		// Modify the provider binary (simulate tampering)
		tamperedContent := []byte("malicious modified content")
		if err := os.WriteFile(providerPath, tamperedContent, 0755); err != nil { //nolint:gosec // G306: Test provider binary
			t.Fatalf("failed to tamper with provider binary: %v", err)
		}

		baseDirFunc := func() string {
			return filepath.Join(tmpDir, "providers")
		}
		resolver, err := NewLockfileProviderResolver(lockfilePath, "", baseDirFunc)
		if err != nil {
			t.Fatalf("failed to create resolver: %v", err)
		}

		ctx := context.Background()
		_, err = resolver.ResolveBinaryPath(ctx, "file")
		if err == nil {
			t.Fatal("expected error when binary is tampered, got nil")
		}

		// Error should mention checksum mismatch
		errMsg := err.Error()
		if !contains(errMsg, "mismatch") || !contains(errMsg, "tampered") {
			t.Errorf("expected error to mention checksum mismatch and tampering, got: %v", err)
		}

		// Restore original content for other tests
		if err := os.WriteFile(providerPath, providerContent, 0755); err != nil { //nolint:gosec // G306: Test provider binary
			t.Fatalf("failed to restore provider binary: %v", err)
		}
	})
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && anyContains(s, substr))
}

func anyContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewLockfileProviderResolver_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("returns error when neither lockfile nor manifest exists", func(t *testing.T) {
		lockfilePath := filepath.Join(tmpDir, "nonexistent.lock.json")
		manifestPath := filepath.Join(tmpDir, "nonexistent.yaml")
		baseDirFunc := func() string { return tmpDir }

		_, err := NewLockfileProviderResolver(lockfilePath, manifestPath, baseDirFunc)
		if err == nil {
			t.Fatal("expected error when neither lockfile nor manifest exists")
		}

		expectedMsg := "failed to create resolver: neither lockfile nor manifest found"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("expected error message starting with %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("returns error when baseDirFunc is nil", func(t *testing.T) {
		// Create a valid lockfile
		lockfilePath := filepath.Join(tmpDir, "valid.lock.json")
		lockfile := &Lockfile{
			Providers: []Provider{
				{
					Alias:   "test",
					Type:    "test",
					Version: "1.0.0",
					Path:    "test/1.0.0/darwin-arm64/provider",
				},
			},
		}
		if err := lockfile.Save(lockfilePath); err != nil {
			t.Fatalf("failed to save lockfile: %v", err)
		}

		_, err := NewLockfileProviderResolver(lockfilePath, "", nil)
		if err == nil {
			t.Fatal("expected error when baseDirFunc is nil")
		}

		expectedMsg := "baseDirFunc must not be nil"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}
