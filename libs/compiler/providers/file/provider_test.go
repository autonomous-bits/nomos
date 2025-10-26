package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestFileProvider_Init_ValidFile tests successful initialization with a valid file.
func TestFileProvider_Init_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")

	// Create a test file
	if err := os.WriteFile(testFile, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": testFile,
		},
	}

	err := p.Init(context.Background(), opts)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if p.filePath == "" {
		t.Error("expected filePath to be set")
	}
}

// TestFileProvider_Init_MissingFile tests initialization fails when file config is missing.
func TestFileProvider_Init_MissingFile(t *testing.T) {
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias:  "file",
		Config: map[string]any{},
	}

	err := p.Init(context.Background(), opts)
	if err == nil {
		t.Fatal("expected Init to fail with missing file")
	}

	if !strings.Contains(err.Error(), "file") {
		t.Errorf("expected error to mention file, got: %v", err)
	}
}

// TestFileProvider_Init_InvalidFile tests initialization fails with non-existent file.
func TestFileProvider_Init_InvalidFile(t *testing.T) {
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": "/nonexistent/path/12345/file.json",
		},
	}

	err := p.Init(context.Background(), opts)
	if err == nil {
		t.Fatal("expected Init to fail with invalid file")
	}
}

// TestFileProvider_Init_DirectoryPath tests initialization fails when file path points to a directory.
func TestFileProvider_Init_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()

	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": tmpDir,
		},
	}

	err := p.Init(context.Background(), opts)
	if err == nil {
		t.Fatal("expected Init to fail when file points to directory")
	}

	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("expected error to mention directory, got: %v", err)
	}
}

// TestFileProvider_Fetch_JSON tests fetching and parsing a JSON file.
func TestFileProvider_Fetch_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSON file
	jsonContent := `{"key": "value", "number": 42, "nested": {"item": "test"}}`
	jsonPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": jsonPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Fetch entire file (no path)
	ctx := context.Background()
	result, err := p.Fetch(ctx, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify result
	data, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if data["key"] != "value" {
		t.Errorf("expected key=value, got %v", data["key"])
	}

	if num, ok := data["number"].(float64); !ok || num != 42 {
		t.Errorf("expected number=42, got %v", data["number"])
	}

	// Fetch nested path
	nestedResult, err := p.Fetch(ctx, []string{"nested", "item"})
	if err != nil {
		t.Fatalf("Fetch nested failed: %v", err)
	}

	if nestedResult != "test" {
		t.Errorf("expected nested.item=test, got %v", nestedResult)
	}
}

// TestFileProvider_Fetch_YAML tests fetching and parsing a YAML file.
func TestFileProvider_Fetch_YAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test YAML file
	yamlContent := `
key: value
nested:
  item: test
list:
  - one
  - two
`
	yamlPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": yamlPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Fetch entire file
	ctx := context.Background()
	result, err := p.Fetch(ctx, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify result
	data, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if data["key"] != "value" {
		t.Errorf("expected key=value, got %v", data["key"])
	}

	// Fetch nested path
	nestedResult, err := p.Fetch(ctx, []string{"nested", "item"})
	if err != nil {
		t.Fatalf("Fetch nested failed: %v", err)
	}

	if nestedResult != "test" {
		t.Errorf("expected nested.item=test, got %v", nestedResult)
	}
}

// TestFileProvider_Fetch_UnsupportedFormat tests error for unsupported file formats.
func TestFileProvider_Fetch_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with unsupported format
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("plain text"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": txtPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Fetch file
	_, err := p.Fetch(context.Background(), nil)
	if err == nil {
		t.Fatal("expected Fetch to fail for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported") && !strings.Contains(err.Error(), "format") {
		t.Errorf("expected unsupported format error, got: %v", err)
	}
}

// TestFileProvider_Fetch_InvalidPath tests error when navigating to invalid path within file.
func TestFileProvider_Fetch_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSON file
	jsonPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": jsonPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Fetch non-existent path within file
	_, err := p.Fetch(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected Fetch to fail for non-existent path")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestFileProvider_Fetch_ContextCancellation tests context cancellation handling.
func TestFileProvider_Fetch_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	jsonPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": jsonPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Fetch should respect cancelled context
	_, err := p.Fetch(ctx, nil)
	if err == nil {
		t.Fatal("expected Fetch to fail with cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestFileProvider_Fetch_NestedPath tests fetching nested data within a file.
func TestFileProvider_Fetch_NestedPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with nested structure
	jsonPath := filepath.Join(tmpDir, "config.json")
	jsonContent := `{
		"configs": {
			"network": {
				"vpc": {
					"cidr": "10.0.0.0/16"
				}
			}
		}
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": jsonPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Fetch nested path
	result, err := p.Fetch(context.Background(), []string{"configs", "network", "vpc", "cidr"})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if result != "10.0.0.0/16" {
		t.Errorf("expected cidr=10.0.0.0/16, got %v", result)
	}
}

// TestFileProvider_Fetch_Timeout tests timeout handling.
func TestFileProvider_Fetch_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	jsonPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonPath, []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Initialize provider
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"file": jsonPath,
		},
	}
	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow timeout to expire
	time.Sleep(2 * time.Millisecond)

	// Fetch should respect timeout
	_, err := p.Fetch(ctx, nil)
	if err == nil {
		t.Fatal("expected Fetch to fail with timeout")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

// TestFileProvider_Info tests the Info method returns correct metadata.
func TestFileProvider_Info(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(testFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "myfiles",
		Config: map[string]any{
			"file": testFile,
		},
	}

	if err := p.Init(context.Background(), opts); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	alias, version := p.Info()

	if alias != "myfiles" {
		t.Errorf("expected alias=myfiles, got %s", alias)
	}

	if version != "v0.1.0" {
		t.Errorf("expected version=v0.1.0, got %s", version)
	}
}
