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

	// Create lockfile
	lockfilePath := filepath.Join(tmpDir, "providers.lock.json")
	lockfile := &Lockfile{
		Providers: []Provider{
			{
				Alias:   "configs",
				Type:    "file",
				Version: "0.2.0",
				OS:      "darwin",
				Arch:    "arm64",
				Path:    "file/0.2.0/darwin-arm64/provider",
			},
		},
	}
	if err := lockfile.Save(lockfilePath); err != nil {
		t.Fatalf("failed to save lockfile: %v", err)
	}

	// Create the provider binary (empty file for testing)
	providerDir := filepath.Join(tmpDir, "providers", "file", "0.2.0", "darwin-arm64")
	if err := os.MkdirAll(providerDir, 0755); err != nil { //nolint:gosec // G301: Test fixture directory
		t.Fatalf("failed to create provider directory: %v", err)
	}
	providerPath := filepath.Join(providerDir, "provider")
	if err := os.WriteFile(providerPath, []byte{}, 0755); err != nil { //nolint:gosec // G306: Test provider binary requires executable permissions
		t.Fatalf("failed to create provider binary: %v", err)
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
