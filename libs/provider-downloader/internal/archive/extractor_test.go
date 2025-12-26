package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// TestTarGzExtractor_Extract tests successful extraction of tar.gz archives.
func TestTarGzExtractor_Extract(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	providerContent := []byte("#!/bin/bash\necho 'provider'\n")
	if err := createTarGzFile(archivePath, map[string][]byte{
		"provider": providerContent,
	}); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	destDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(destDir, 0755); err != nil { //nolint:gosec // G301: Test directory
		t.Fatalf("failed to create dest dir: %v", err)
	}

	extractor := &TarGzExtractor{}
	extractedPath, err := extractor.Extract(archivePath, destDir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPath := filepath.Join(destDir, "provider")
	if extractedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, extractedPath)
	}

	content, err := os.ReadFile(extractedPath) //nolint:gosec // G304: Test file read
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(content) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, content)
	}
}

// TestZipExtractor_Extract tests successful extraction of zip archives.
func TestZipExtractor_Extract(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	providerContent := []byte("#!/bin/bash\necho 'provider'\n")
	if err := createZipFile(archivePath, map[string][]byte{
		"provider": providerContent,
	}); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	destDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(destDir, 0755); err != nil { //nolint:gosec // G301: Test directory
		t.Fatalf("failed to create dest dir: %v", err)
	}

	extractor := &ZipExtractor{}
	extractedPath, err := extractor.Extract(archivePath, destDir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPath := filepath.Join(destDir, "provider")
	if extractedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, extractedPath)
	}

	content, err := os.ReadFile(extractedPath) //nolint:gosec // G304: Test file read
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(content) != string(providerContent) {
		t.Errorf("content mismatch: expected %q, got %q", providerContent, content)
	}
}

// TestGetExtractor tests the extractor factory function.
func TestGetExtractor(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
	}{
		{"tar.gz", "provider.tar.gz", false},
		{"tgz", "provider.tgz", false},
		{"zip", "provider.zip", false},
		{"unsupported", "provider.rar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := GetExtractor(tt.filename)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if extractor == nil {
				t.Fatal("expected non-nil extractor")
			}
		})
	}
}

// createTarGzFile creates a tar.gz archive with the given files.
func createTarGzFile(path string, files map[string][]byte) error {
	f, err := os.Create(path) //nolint:gosec // G304: Test file creation
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzw := gzip.NewWriter(f)
	defer func() { _ = gzw.Close() }()

	tw := tar.NewWriter(gzw)
	defer func() { _ = tw.Close() }()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(content); err != nil {
			return err
		}
	}

	return nil
}

// createZipFile creates a zip archive with the given files.
func createZipFile(path string, files map[string][]byte) error {
	f, err := os.Create(path) //nolint:gosec // G304: Test file creation
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(content); err != nil {
			return err
		}
	}

	return nil
}
