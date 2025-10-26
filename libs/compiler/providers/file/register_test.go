package file

import (
	"os"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestRegisterFileProvider_Success tests successful registration.
func TestRegisterFileProvider_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.json"
	// Create test file
	if err := os.WriteFile(testFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	registry := compiler.NewProviderRegistry()

	err := RegisterFileProvider(registry, "file", testFile)
	if err != nil {
		t.Fatalf("RegisterFileProvider failed: %v", err)
	}

	// Verify provider can be retrieved
	provider, err := registry.GetProvider("file")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}

	// Verify provider is a FileProvider
	_, ok := provider.(*FileProvider)
	if !ok {
		t.Errorf("expected *FileProvider, got %T", provider)
	}
}

// TestRegisterFileProvider_NilRegistry tests error with nil registry.
func TestRegisterFileProvider_NilRegistry(t *testing.T) {
	err := RegisterFileProvider(nil, "file", "/tmp/test.json")
	if err == nil {
		t.Fatal("expected error with nil registry")
	}

	if err.Error() != "registry cannot be nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegisterFileProvider_EmptyAlias tests error with empty alias.
func TestRegisterFileProvider_EmptyAlias(t *testing.T) {
	registry := compiler.NewProviderRegistry()

	err := RegisterFileProvider(registry, "", "/tmp/test.json")
	if err == nil {
		t.Fatal("expected error with empty alias")
	}

	if err.Error() != "alias cannot be empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegisterFileProvider_EmptyFilePath tests error with empty file path.
func TestRegisterFileProvider_EmptyFilePath(t *testing.T) {
	registry := compiler.NewProviderRegistry()

	err := RegisterFileProvider(registry, "file", "")
	if err == nil {
		t.Fatal("expected error with empty file path")
	}

	if err.Error() != "file path cannot be empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegisterFileProvider_InvalidFilePath tests error when file doesn't exist.
func TestRegisterFileProvider_InvalidFilePath(t *testing.T) {
	registry := compiler.NewProviderRegistry()

	err := RegisterFileProvider(registry, "file", "/nonexistent/path/12345/file.json")
	if err != nil {
		t.Fatalf("RegisterFileProvider should not fail during registration: %v", err)
	}

	// The error should occur when getting the provider
	_, err = registry.GetProvider("file")
	if err == nil {
		t.Fatal("expected GetProvider to fail with invalid file path")
	}
}

// TestRegisterFileProvider_MultipleAliases tests registering multiple file providers.
func TestRegisterFileProvider_MultipleAliases(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	testFile1 := tmpDir1 + "/test1.json"
	testFile2 := tmpDir2 + "/test2.json"

	// Create test files
	if err := os.WriteFile(testFile1, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}

	registry := compiler.NewProviderRegistry()

	// Register first provider
	if err := RegisterFileProvider(registry, "file1", testFile1); err != nil {
		t.Fatalf("RegisterFileProvider failed for file1: %v", err)
	}

	// Register second provider
	if err := RegisterFileProvider(registry, "file2", testFile2); err != nil {
		t.Fatalf("RegisterFileProvider failed for file2: %v", err)
	}

	// Verify both can be retrieved
	provider1, err := registry.GetProvider("file1")
	if err != nil {
		t.Fatalf("GetProvider failed for file1: %v", err)
	}

	provider2, err := registry.GetProvider("file2")
	if err != nil {
		t.Fatalf("GetProvider failed for file2: %v", err)
	}

	if provider1 == nil || provider2 == nil {
		t.Fatal("expected both providers to be non-nil")
	}

	// Verify they are different instances
	if provider1 == provider2 {
		t.Error("expected different provider instances")
	}
}
