package file

import (
	"os"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestRegisterFileProvider_NilRegistry tests error with nil registry.
func TestRegisterFileProvider_NilRegistry(t *testing.T) {
	err := RegisterFileProvider(nil, "file", "/tmp/test-directory")
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

	err := RegisterFileProvider(registry, "", "./testdata")
	if err == nil {
		t.Fatal("expected error with empty alias")
	}

	if err.Error() != "alias cannot be empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegisterFileProvider_EmptyDirectory tests error with empty directory path.
func TestRegisterFileProvider_EmptyDirectory(t *testing.T) {
	registry := compiler.NewProviderRegistry()

	err := RegisterFileProvider(registry, "test", "")
	if err == nil {
		t.Fatal("expected error with empty directory path")
	}

	if err.Error() != "directory path cannot be empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

// NEW TESTS FOR DIRECTORY-BASED PROVIDER (v0.2.0)

// TestRegisterFileProvider_DirectoryWithCslFiles tests successful registration with a directory containing .csl files.
func TestRegisterFileProvider_DirectoryWithCslFiles(t *testing.T) {
	// Arrange
	registry := compiler.NewProviderRegistry()

	// Use testdata directory which contains network.csl and database.csl
	testDir := "./testdata"

	// Act
	err := RegisterFileProvider(registry, "test-provider", testDir)

	// Assert - this should FAIL initially (RED phase)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify provider was registered
	provider, err := registry.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("failed to get registered provider: %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
}

// TestRegisterFileProvider_DirectoryNotExists_New tests registration fails when directory doesn't exist.
func TestRegisterFileProvider_DirectoryNotExists_New(t *testing.T) {
	// Arrange
	registry := compiler.NewProviderRegistry()
	nonExistentDir := "/nonexistent/path/12345"

	// Act
	err := RegisterFileProvider(registry, "test", nonExistentDir)

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// TestRegisterFileProvider_PathIsFileShouldFail tests registration fails when path points to a file.
func TestRegisterFileProvider_PathIsFileShouldFail(t *testing.T) {
	// Arrange
	registry := compiler.NewProviderRegistry()
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.csl"

	// Create a test file
	if err := os.WriteFile(testFile, []byte("// test file"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	err := RegisterFileProvider(registry, "test", testFile)

	// Assert
	if err == nil {
		t.Fatal("expected error when path is a file, got nil")
	}
}

// TestRegisterFileProvider_DirectoryNoCslFiles tests registration fails when directory has no .csl files.
func TestRegisterFileProvider_DirectoryNoCslFiles(t *testing.T) {
	// Arrange
	registry := compiler.NewProviderRegistry()
	tmpDir := t.TempDir()

	// Create a non-.csl file
	testFile := tmpDir + "/test.yaml"
	if err := os.WriteFile(testFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	err := RegisterFileProvider(registry, "test", tmpDir)

	// Assert
	if err == nil {
		t.Fatal("expected error when directory has no .csl files, got nil")
	}
}
