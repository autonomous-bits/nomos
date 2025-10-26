//go:build integration

package file

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestIntegration_FileProvider_FetchJSON tests fetching JSON files in integration scenario.
func TestIntegration_FileProvider_FetchJSON(t *testing.T) {
	ctx := context.Background()

	// Create registry and register file provider
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "file", "testdata"); err != nil {
		t.Fatalf("RegisterFileProvider failed: %v", err)
	}

	// Get provider
	provider, err := registry.GetProvider("file")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Fetch JSON file
	result, err := provider.Fetch(ctx, []string{"config.json"})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify structure
	data, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	// Verify database config
	db, ok := data["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database to be map, got %T", data["database"])
	}

	if db["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", db["host"])
	}

	if port, ok := db["port"].(float64); !ok || port != 5432 {
		t.Errorf("expected port=5432, got %v", db["port"])
	}

	// Verify cache config
	cache, ok := data["cache"].(map[string]any)
	if !ok {
		t.Fatalf("expected cache to be map, got %T", data["cache"])
	}

	if enabled, ok := cache["enabled"].(bool); !ok || !enabled {
		t.Errorf("expected enabled=true, got %v", cache["enabled"])
	}
}

// TestIntegration_FileProvider_FetchYAML tests fetching YAML files in integration scenario.
func TestIntegration_FileProvider_FetchYAML(t *testing.T) {
	ctx := context.Background()

	// Create registry and register file provider
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "file", "testdata"); err != nil {
		t.Fatalf("RegisterFileProvider failed: %v", err)
	}

	// Get provider
	provider, err := registry.GetProvider("file")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Fetch YAML file
	result, err := provider.Fetch(ctx, []string{"network.yaml"})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify structure
	data, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	// Verify network config
	network, ok := data["network"].(map[string]any)
	if !ok {
		t.Fatalf("expected network to be map, got %T", data["network"])
	}

	vpc, ok := network["vpc"].(map[string]any)
	if !ok {
		t.Fatalf("expected vpc to be map, got %T", network["vpc"])
	}

	if vpc["cidr"] != "10.0.0.0/16" {
		t.Errorf("expected cidr=10.0.0.0/16, got %v", vpc["cidr"])
	}

	// Verify subnets
	subnets, ok := vpc["subnets"].([]any)
	if !ok {
		t.Fatalf("expected subnets to be array, got %T", vpc["subnets"])
	}

	if len(subnets) != 2 {
		t.Errorf("expected 2 subnets, got %d", len(subnets))
	}

	// Verify first subnet
	if len(subnets) > 0 {
		subnet1, ok := subnets[0].(map[string]any)
		if !ok {
			t.Fatalf("expected subnet to be map, got %T", subnets[0])
		}

		if subnet1["name"] != "public" {
			t.Errorf("expected name=public, got %v", subnet1["name"])
		}
	}
}

// TestIntegration_FileProvider_Caching tests that fetch results are cached.
func TestIntegration_FileProvider_Caching(t *testing.T) {
	ctx := context.Background()

	// Create registry and register file provider
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "file", "testdata"); err != nil {
		t.Fatalf("RegisterFileProvider failed: %v", err)
	}

	// Get provider (should initialize once)
	provider1, err := registry.GetProvider("file")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Get provider again (should return cached instance)
	provider2, err := registry.GetProvider("file")
	if err != nil {
		t.Fatalf("GetProvider failed second time: %v", err)
	}

	// Verify same instance
	if provider1 != provider2 {
		t.Error("expected same provider instance from registry cache")
	}

	// Fetch same file twice
	result1, err := provider1.Fetch(ctx, []string{"config.json"})
	if err != nil {
		t.Fatalf("Fetch failed first time: %v", err)
	}

	result2, err := provider1.Fetch(ctx, []string{"config.json"})
	if err != nil {
		t.Fatalf("Fetch failed second time: %v", err)
	}

	// Both should succeed (implementation doesn't cache fetch results at provider level,
	// but registry caches provider instances)
	if result1 == nil || result2 == nil {
		t.Error("expected both fetch results to be non-nil")
	}
}

// TestIntegration_FileProvider_Info tests provider metadata.
func TestIntegration_FileProvider_Info(t *testing.T) {
	// Create registry and register file provider
	registry := compiler.NewProviderRegistry()
	if err := RegisterFileProvider(registry, "myfiles", "testdata"); err != nil {
		t.Fatalf("RegisterFileProvider failed: %v", err)
	}

	// Get provider
	provider, err := registry.GetProvider("myfiles")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Check if provider implements ProviderWithInfo
	providerWithInfo, ok := provider.(compiler.ProviderWithInfo)
	if !ok {
		t.Fatal("expected provider to implement ProviderWithInfo")
	}

	alias, version := providerWithInfo.Info()

	if alias != "myfiles" {
		t.Errorf("expected alias=myfiles, got %s", alias)
	}

	if version == "" {
		t.Error("expected version to be non-empty")
	}
}
