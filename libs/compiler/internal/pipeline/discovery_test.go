package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverInputFiles_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.csl")
	if err := os.WriteFile(filePath, []byte("# test"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	files, err := DiscoverInputFiles(filePath)

	if err != nil {
		t.Fatalf("DiscoverInputFiles failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	absPath, _ := filepath.Abs(filePath)
	if files[0] != absPath {
		t.Errorf("expected %q, got %q", absPath, files[0])
	}
}

func TestDiscoverInputFiles_NonCslFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := DiscoverInputFiles(filePath)

	if err == nil {
		t.Fatal("expected error for non-.csl file, got nil")
	}
}

func TestDiscoverInputFiles_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{"a.csl", "z.csl", "config.csl"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("# "+file), 0600); err != nil {
			t.Fatalf("failed to write %s: %v", file, err)
		}
	}

	result, err := DiscoverInputFiles(tmpDir)

	if err != nil {
		t.Fatalf("DiscoverInputFiles failed: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 files, got %d", len(result))
	}

	expectedOrder := []string{"a.csl", "config.csl", "z.csl"}
	for i, expected := range expectedOrder {
		if filepath.Base(result[i]) != expected {
			t.Errorf("file %d: expected %s, got %s", i, expected, filepath.Base(result[i]))
		}
	}
}

func TestDiscoverInputFiles_DirectoryWithNonCslFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cslFiles := []string{"config.csl", "app.csl"}
	for _, file := range cslFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("# "+file), 0600); err != nil {
			t.Fatalf("failed to write %s: %v", file, err)
		}
	}

	nonCslFiles := []string{"README.md", "data.json", ".gitignore"}
	for _, file := range nonCslFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
			t.Fatalf("failed to write %s: %v", file, err)
		}
	}

	result, err := DiscoverInputFiles(tmpDir)

	if err != nil {
		t.Fatalf("DiscoverInputFiles failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}

	for _, filePath := range result {
		if filepath.Ext(filePath) != ".csl" {
			t.Errorf("non-.csl file included: %s", filePath)
		}
	}
}

func TestDiscoverInputFiles_DirectoryWithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	topLevel := []string{"a.csl", "b.csl"}
	for _, file := range topLevel {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("# "+file), 0600); err != nil {
			t.Fatalf("failed to write %s: %v", file, err)
		}
	}

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "nested.csl")
	if err := os.WriteFile(subFile, []byte("# nested"), 0600); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	result, err := DiscoverInputFiles(tmpDir)

	if err != nil {
		t.Fatalf("DiscoverInputFiles failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}

	for _, filePath := range result {
		absDir, _ := filepath.Abs(tmpDir)
		if filepath.Dir(filePath) != absDir {
			t.Errorf("expected top-level file, got: %s", filePath)
		}
	}
}

func TestDiscoverInputFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := DiscoverInputFiles(tmpDir)

	if err != nil {
		t.Fatalf("DiscoverInputFiles failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 files in empty directory, got %d", len(result))
	}
}

func TestDiscoverInputFiles_NonExistentPath(t *testing.T) {
	nonExistent := "/nonexistent/path/to/file.csl"

	_, err := DiscoverInputFiles(nonExistent)

	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}
