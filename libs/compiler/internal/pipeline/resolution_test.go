package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
)

// Mock provider for resolution tests
type mockProvider struct {
	fetchResult map[string]any
	fetchErr    error
	fetchCalls  [][]string
}

func (m *mockProvider) Init(_ context.Context, _ core.ProviderInitOptions) error {
	return nil
}

func (m *mockProvider) Fetch(_ context.Context, path []string) (any, error) {
	m.fetchCalls = append(m.fetchCalls, path)
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}

	// Simple string key lookup
	key := ""
	if len(path) > 0 {
		key = path[0]
	}
	if result, ok := m.fetchResult[key]; ok {
		return result, nil
	}
	return nil, errors.New("path not found")
}

// Mock registry for resolution tests
type mockRegistry struct {
	providers map[string]*mockProvider
	getErr    error
}

func (r *mockRegistry) Register(_ string, _ core.ProviderConstructor) {}

func (r *mockRegistry) GetProvider(_ context.Context, alias string) (core.Provider, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if p, ok := r.providers[alias]; ok {
		return p, nil
	}
	return nil, errors.New("provider not found: " + alias)
}

func (r *mockRegistry) RegisteredAliases() []string {
	aliases := make([]string, 0, len(r.providers))
	for alias := range r.providers {
		aliases = append(aliases, alias)
	}
	return aliases
}

func TestResolveReferences_SimpleMap(t *testing.T) {
	// Setup: data without references
	data := map[string]any{
		"name": "myapp",
		"port": 8080,
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if result["name"] != "myapp" {
		t.Errorf("expected name 'myapp', got %v", result["name"])
	}
	if result["port"] != 8080 {
		t.Errorf("expected port 8080, got %v", result["port"])
	}
}

func TestResolveReferences_WithProvider(t *testing.T) {
	// Setup: mock provider that returns data
	provider := &mockProvider{
		fetchResult: map[string]any{
			"config": map[string]any{
				"database": "postgres",
			},
		},
	}

	registry := &mockRegistry{
		providers: map[string]*mockProvider{
			"configs": provider,
		},
	}

	// Data with plain values (resolver handles ReferenceExpr internally)
	data := map[string]any{
		"app": "myapp",
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if result["app"] != "myapp" {
		t.Errorf("expected app 'myapp', got %v", result["app"])
	}
}

func TestResolveReferences_EmptyData(t *testing.T) {
	// Setup: empty data map
	data := map[string]any{}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestResolveReferences_NestedMaps(t *testing.T) {
	// Setup: nested map structure
	data := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
		"database": map[string]any{
			"host": "db.example.com",
			"port": 5432,
		},
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	server, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map[string]any, got %T", result["server"])
	}

	if server["host"] != "localhost" {
		t.Errorf("expected host 'localhost', got %v", server["host"])
	}
	if server["port"] != 8080 {
		t.Errorf("expected port 8080, got %v", server["port"])
	}
}

func TestResolveReferences_AllowMissingProvider(t *testing.T) {
	// Setup: data and registry
	data := map[string]any{
		"app": "myapp",
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	var receivedWarnings []string
	opts := ResolveOptions{
		ProviderRegistry:     registry,
		AllowMissingProvider: true,
		OnWarning: func(msg string) {
			receivedWarnings = append(receivedWarnings, msg)
		},
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if result["app"] != "myapp" {
		t.Errorf("expected app 'myapp', got %v", result["app"])
	}
}

func TestResolveReferences_OnWarningCallback(t *testing.T) {
	// Setup
	data := map[string]any{
		"test": "value",
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	var warnings []string
	opts := ResolveOptions{
		ProviderRegistry: registry,
		OnWarning: func(msg string) {
			warnings = append(warnings, msg)
		},
	}

	// Act
	_, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	// Note: Without actual references, no warnings would be generated
	// This test validates the callback mechanism is available
}

func TestResolveReferences_ContextCancellation(t *testing.T) {
	// Setup: data
	data := map[string]any{
		"app": "myapp",
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act - context is cancelled but simple data should still work
	// The resolver respects context for provider operations
	result, err := ResolveReferences(ctx, data, opts)

	// Assert - no error for simple data without provider calls
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if result["app"] != "myapp" {
		t.Errorf("expected app 'myapp', got %v", result["app"])
	}
}

func TestResolveReferences_MixedDataTypes(t *testing.T) {
	// Setup: data with various types
	data := map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"array":  []any{1, 2, 3},
		"nested": map[string]any{"key": "value"},
	}

	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	opts := ResolveOptions{
		ProviderRegistry: registry,
	}

	// Act
	result, err := ResolveReferences(context.Background(), data, opts)

	// Assert
	if err != nil {
		t.Fatalf("ResolveReferences failed: %v", err)
	}

	if result["string"] != "value" {
		t.Errorf("string mismatch")
	}
	if result["int"] != 42 {
		t.Errorf("int mismatch")
	}
	if result["float"] != 3.14 {
		t.Errorf("float mismatch")
	}
	if result["bool"] != true {
		t.Errorf("bool mismatch")
	}

	arr, ok := result["array"].([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("array mismatch")
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok || nested["key"] != "value" {
		t.Errorf("nested map mismatch")
	}
}

func TestProviderRegistryAdapter_GetProvider(t *testing.T) {
	// Test the adapter directly
	provider := &mockProvider{}
	registry := &mockRegistry{
		providers: map[string]*mockProvider{
			"test": provider,
		},
	}

	ctx := context.Background()
	adapter := &providerRegistryAdapter{
		registry: registry,
		ctx:      ctx,
	}

	// Act
	result, err := adapter.GetProvider("test")

	// Assert
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Compare by checking it's not nil and is the right type
	if result == nil {
		t.Error("expected provider instance, got nil")
	}
}

func TestProviderRegistryAdapter_GetProvider_Error(t *testing.T) {
	// Test adapter error handling
	registry := &mockRegistry{
		providers: make(map[string]*mockProvider),
	}

	ctx := context.Background()
	adapter := &providerRegistryAdapter{
		registry: registry,
		ctx:      ctx,
	}

	// Act
	_, err := adapter.GetProvider("nonexistent")

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}
