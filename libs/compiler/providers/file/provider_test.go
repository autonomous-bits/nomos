package file

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestFileProvider_Init_DirectoryWithCslFiles tests successful initialization with a directory containing .csl files.
func TestFileProvider_Init_DirectoryWithCslFiles(t *testing.T) {
	p := &FileProvider{}
	opts := compiler.ProviderInitOptions{
		Alias: "test",
		Config: map[string]any{
			"directory": "./testdata",
		},
	}

	err := p.Init(context.Background(), opts)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if p.directory == "" {
		t.Error("expected directory to be set")
	}

	if len(p.cslFiles) == 0 {
		t.Error("expected cslFiles to be populated")
	}
}

// TestFileProvider_FetchByName tests fetching a .csl file by base name.
func TestFileProvider_FetchByName(t *testing.T) {
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "test", "./testdata"); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	provider, err := registry.GetProvider("test")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	result, err := provider.Fetch(context.Background(), []string{"network"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// In v0.2.0+, .csl files are parsed and return structured data
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map[string]any, got %T", result)
	}

	// The network.csl file may have different content - just verify we got a map
	// (empty map is ok if the file doesn't have any top-level sections)
	t.Logf("Fetched data: %+v", resultMap)
}

// TestFileProvider_FetchByName_MissingFile tests error when requested file doesn't exist.
func TestFileProvider_FetchByName_MissingFile(t *testing.T) {
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "test", "./testdata"); err != nil {
		t.Fatalf("failed to register provider: %v", err)
	}

	provider, err := registry.GetProvider("test")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	_, err = provider.Fetch(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
