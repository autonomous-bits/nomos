package file

import (
	"context"
	"strings"
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

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("expected result to be string, got %T", result)
	}

	if !strings.Contains(resultStr, "network") {
		t.Errorf("expected result to contain 'network', got: %s", resultStr)
	}
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
