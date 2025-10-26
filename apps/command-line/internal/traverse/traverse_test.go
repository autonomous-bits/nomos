// Package traverse provides deterministic file discovery for Nomos CLI.
package traverse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/traverse"
)

func TestDiscoverFiles_SingleFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.csl")
	if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	files, err := traverse.DiscoverFiles(filePath)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0] != filePath {
		t.Errorf("expected path %q, got %q", filePath, files[0])
	}
}

func TestDiscoverFiles_SingleNonCSLFile_ReturnsError(t *testing.T) {
	// Create temp file without .csl extension
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	_, err := traverse.DiscoverFiles(filePath)

	// Assert
	if err == nil {
		t.Fatal("expected error for non-.csl file, got nil")
	}
}

func TestDiscoverFiles_Directory_LexicographicOrdering(t *testing.T) {
	// Create temp directory with files in mixed order
	tmpDir := t.TempDir()

	// Create files with names that should sort in specific order
	// Use names that work reliably across filesystems (avoid case sensitivity issues)
	// Expected UTF-8 lexicographic order: 1-first.csl, 2-second.csl, aa.csl, zz.csl, 中文.csl
	testFiles := []string{"zz.csl", "aa.csl", "中文.csl", "1-first.csl", "2-second.csl"}
	expectedOrder := []string{"1-first.csl", "2-second.csl", "aa.csl", "zz.csl", "中文.csl"}

	for _, name := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create test file %q: %v", name, err)
		}
	}

	// Act
	files, err := traverse.DiscoverFiles(tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(files) != len(expectedOrder) {
		t.Logf("Expected files: %v", expectedOrder)
		t.Logf("Got files: %v", files)
		for i, f := range files {
			t.Logf("  [%d] %s (basename: %s)", i, f, filepath.Base(f))
		}
		t.Fatalf("expected %d files, got %d", len(expectedOrder), len(files))
	}

	for i, expected := range expectedOrder {
		expectedPath := filepath.Join(tmpDir, expected)
		if files[i] != expectedPath {
			t.Errorf("index %d: expected %q, got %q", i, expectedPath, files[i])
		}
	}
}

func TestDiscoverFiles_NestedDirectories_RecursiveOrdering(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()

	// Create structure:
	// tmpDir/
	//   a.csl
	//   subdir/
	//     b.csl
	//     deeper/
	//       c.csl
	if err := os.WriteFile(filepath.Join(tmpDir, "a.csl"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create a.csl: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.csl"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create b.csl: %v", err)
	}

	deeperDir := filepath.Join(subDir, "deeper")
	if err := os.MkdirAll(deeperDir, 0755); err != nil {
		t.Fatalf("failed to create deeper dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deeperDir, "c.csl"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create c.csl: %v", err)
	}

	// Act
	files, err := traverse.DiscoverFiles(tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Expected lexicographic order of full paths
	expected := []string{
		filepath.Join(tmpDir, "a.csl"),
		filepath.Join(tmpDir, "subdir", "b.csl"),
		filepath.Join(tmpDir, "subdir", "deeper", "c.csl"),
	}

	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(files))
	}

	for i, expectedPath := range expected {
		if files[i] != expectedPath {
			t.Errorf("index %d: expected %q, got %q", i, expectedPath, files[i])
		}
	}
}

func TestDiscoverFiles_EmptyDirectory_ReturnsError(t *testing.T) {
	// Create empty directory
	tmpDir := t.TempDir()

	// Act
	_, err := traverse.DiscoverFiles(tmpDir)

	// Assert
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
	// Error should mention no .csl files found
	if err.Error() != "no .csl files found" {
		t.Errorf("expected error message 'no .csl files found', got %q", err.Error())
	}
}

func TestDiscoverFiles_DirectoryWithNonCSLFiles_IgnoresThem(t *testing.T) {
	// Create directory with .csl and non-.csl files
	tmpDir := t.TempDir()

	// Create mix of files
	testFiles := map[string]bool{
		"a.csl":      true,  // Should be included
		"b.txt":      false, // Should be ignored
		"c.csl":      true,  // Should be included
		"README.md":  false, // Should be ignored
		"config.yml": false, // Should be ignored
	}

	for name := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file %q: %v", name, err)
		}
	}

	// Act
	files, err := traverse.DiscoverFiles(tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should only have .csl files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Fatalf("expected %d files, got %d", expectedCount, len(files))
	}

	// Verify all returned files end with .csl
	for _, file := range files {
		if filepath.Ext(file) != ".csl" {
			t.Errorf("non-.csl file returned: %q", file)
		}
	}
}

func TestDiscoverFiles_NonexistentPath_ReturnsError(t *testing.T) {
	// Act
	_, err := traverse.DiscoverFiles("/nonexistent/path/to/file.csl")

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestDiscoverFiles_SymlinkToFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create real file
	realFile := filepath.Join(tmpDir, "real.csl")
	if err := os.WriteFile(realFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create real file: %v", err)
	}

	// Create symlink
	symlinkFile := filepath.Join(tmpDir, "link.csl")
	if err := os.Symlink(realFile, symlinkFile); err != nil {
		t.Skipf("skipping symlink test: %v", err)
	}

	// Act
	files, err := traverse.DiscoverFiles(tmpDir)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should discover both files in sorted order
	expectedCount := 2
	if len(files) != expectedCount {
		t.Fatalf("expected %d files, got %d", expectedCount, len(files))
	}

	// Verify ordering: link.csl < real.csl
	if filepath.Base(files[0]) != "link.csl" {
		t.Errorf("expected first file to be link.csl, got %s", filepath.Base(files[0]))
	}
	if filepath.Base(files[1]) != "real.csl" {
		t.Errorf("expected second file to be real.csl, got %s", filepath.Base(files[1]))
	}
}

func TestDiscoverFiles_SymlinkLoop_HandledGracefully(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories for loop
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir1", "dir2")
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	// Create symlink loop: dir2/loop -> dir1
	loopLink := filepath.Join(dir2, "loop")
	if err := os.Symlink(dir1, loopLink); err != nil {
		t.Skipf("skipping symlink loop test: %v", err)
	}

	// Create a valid .csl file to ensure we have something to discover
	if err := os.WriteFile(filepath.Join(dir1, "test.csl"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Act
	files, err := traverse.DiscoverFiles(tmpDir)

	// Assert - should not hang or panic, should handle loop gracefully
	if err != nil {
		// Error is acceptable for symlink loops
		t.Logf("symlink loop produced error (acceptable): %v", err)
	} else {
		// If no error, should have found the test file
		if len(files) < 1 {
			t.Error("expected at least 1 file despite symlink loop")
		}
	}
}
