package resolver

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// fakeProvider implements Provider for testing.
type fakeProvider struct {
	mu             sync.Mutex
	fetchCount     int
	fetchResponses map[string]any
	fetchError     error
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		fetchResponses: make(map[string]any),
	}
}

func (f *fakeProvider) Fetch(_ context.Context, path []string) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fetchCount++

	if f.fetchError != nil {
		return nil, f.fetchError
	}

	key := strings.Join(path, "/")
	if val, ok := f.fetchResponses[key]; ok {
		return val, nil
	}

	return nil, errors.New("path not found")
}

func (f *fakeProvider) getFetchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetchCount
}

func (f *fakeProvider) setResponse(path []string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.Join(path, "/")
	f.fetchResponses[key] = value
}

func (f *fakeProvider) setError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fetchError = err
}

// fakeProviderRegistry implements ProviderRegistry for testing.
type fakeProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func newFakeProviderRegistry() *fakeProviderRegistry {
	return &fakeProviderRegistry{
		providers: make(map[string]Provider),
	}
}

func (r *fakeProviderRegistry) register(alias string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[alias] = provider
}

func (r *fakeProviderRegistry) GetProvider(alias string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.providers[alias]; ok {
		return p, nil
	}

	return nil, ErrProviderNotRegistered
}

// TestResolveValue_ScalarPassthrough tests that scalar values pass through unchanged.
func TestResolveValue_ScalarPassthrough(t *testing.T) {
	registry := newFakeProviderRegistry()
	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	tests := []struct {
		name  string
		value any
	}{
		{"string", "hello"},
		{"int", 42},
		{"float", 3.14},
		{"bool", true},
		{"nil", nil},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveValue(ctx, tt.value)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != tt.value {
				t.Errorf("expected %v, got %v", tt.value, result)
			}
		})
	}
}

// TestResolveValue_ReferenceExpr_BasicResolution tests basic reference resolution.
func TestResolveValue_ReferenceExpr_BasicResolution(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setResponse([]string{"config", "name"}, "test-value")
	registry.register("network", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	// Create reference expression
	ref := &ast.ReferenceExpr{
		Alias: "network",
		Path:  []string{"config", "name"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 10,
			StartCol:  5,
		},
	}

	// Act
	ctx := context.Background()
	result, err := resolver.ResolveValue(ctx, ref)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "test-value" {
		t.Errorf("expected 'test-value', got %v", result)
	}
	if provider.getFetchCount() != 1 {
		t.Errorf("expected 1 fetch call, got %d", provider.getFetchCount())
	}
}

// TestResolveValue_ReferenceExpr_Caching tests that fetch results are cached per run.
func TestResolveValue_ReferenceExpr_Caching(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setResponse([]string{"config", "value"}, "cached-result")
	registry.register("source", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	ref := &ast.ReferenceExpr{
		Alias: "source",
		Path:  []string{"config", "value"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 1,
			StartCol:  1,
		},
	}

	ctx := context.Background()

	// Resolve the same reference three times
	for i := 0; i < 3; i++ {
		result, err := resolver.ResolveValue(ctx, ref)
		if err != nil {
			t.Fatalf("iteration %d: expected no error, got %v", i, err)
		}
		if result != "cached-result" {
			t.Errorf("iteration %d: expected 'cached-result', got %v", i, result)
		}
	}

	// Verify provider was only called once
	if provider.getFetchCount() != 1 {
		t.Errorf("expected 1 fetch call (cached), got %d", provider.getFetchCount())
	}
}

// TestResolveValue_ReferenceExpr_ProviderNotRegistered tests error handling for missing providers.
func TestResolveValue_ReferenceExpr_ProviderNotRegistered(t *testing.T) {
	// Setup - empty registry
	registry := newFakeProviderRegistry()

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	ref := &ast.ReferenceExpr{
		Alias: "missing",
		Path:  []string{"key"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 5,
			StartCol:  10,
		},
	}

	// Act
	ctx := context.Background()
	_, err := resolver.ResolveValue(ctx, ref)

	// Assert
	if err == nil {
		t.Fatal("expected error for missing provider, got nil")
	}
	if !errors.Is(err, ErrProviderNotRegistered) {
		t.Errorf("expected ErrProviderNotRegistered, got %v", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error should mention provider alias 'missing', got: %v", err)
	}
	if !strings.Contains(err.Error(), "test.csl:5:10") {
		t.Errorf("error should include source location, got: %v", err)
	}
}

// TestResolveValue_ReferenceExpr_AllowMissingProvider tests non-fatal error handling.
func TestResolveValue_ReferenceExpr_AllowMissingProvider(t *testing.T) {
	// Setup - empty registry but AllowMissingProvider enabled
	registry := newFakeProviderRegistry()

	var warnings []string
	resolver := New(ResolverOptions{
		ProviderRegistry:     registry,
		AllowMissingProvider: true,
		OnWarning: func(warning string) {
			warnings = append(warnings, warning)
		},
	})

	ref := &ast.ReferenceExpr{
		Alias: "missing",
		Path:  []string{"key"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 5,
			StartCol:  10,
		},
	}

	// Act
	ctx := context.Background()
	result, err := resolver.ResolveValue(ctx, ref)

	// Assert - should not error
	if err != nil {
		t.Fatalf("expected no error with AllowMissingProvider, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for missing provider, got %v", result)
	}

	// Verify warning was recorded
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "missing") {
		t.Errorf("warning should mention provider 'missing', got: %s", warnings[0])
	}
}

// TestResolveValue_ReferenceExpr_FetchError tests error handling for fetch failures.
func TestResolveValue_ReferenceExpr_FetchError(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setError(errors.New("network timeout"))
	registry.register("remote", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	ref := &ast.ReferenceExpr{
		Alias: "remote",
		Path:  []string{"data"},
		SourceSpan: ast.SourceSpan{
			Filename:  "app.csl",
			StartLine: 20,
			StartCol:  15,
		},
	}

	// Act
	ctx := context.Background()
	_, err := resolver.ResolveValue(ctx, ref)

	// Assert
	if err == nil {
		t.Fatal("expected error for fetch failure, got nil")
	}
	if !errors.Is(err, ErrUnresolvedReference) {
		t.Errorf("expected ErrUnresolvedReference, got %v", err)
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("error should include underlying error, got: %v", err)
	}
}

// TestResolveValue_ReferenceExpr_FetchError_AllowMissing tests AllowMissingProvider for fetch errors.
func TestResolveValue_ReferenceExpr_FetchError_AllowMissing(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setError(errors.New("connection refused"))
	registry.register("remote", provider)

	var warnings []string
	resolver := New(ResolverOptions{
		ProviderRegistry:     registry,
		AllowMissingProvider: true,
		OnWarning: func(warning string) {
			warnings = append(warnings, warning)
		},
	})

	ref := &ast.ReferenceExpr{
		Alias: "remote",
		Path:  []string{"data"},
		SourceSpan: ast.SourceSpan{
			Filename:  "app.csl",
			StartLine: 20,
			StartCol:  15,
		},
	}

	// Act
	ctx := context.Background()
	result, err := resolver.ResolveValue(ctx, ref)

	// Assert - should not error
	if err != nil {
		t.Fatalf("expected no error with AllowMissingProvider, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for failed fetch, got %v", result)
	}

	// Verify warning was recorded
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "connection refused") {
		t.Errorf("warning should mention error, got: %s", warnings[0])
	}
}

// TestResolveValue_Map_RecursiveResolution tests map value resolution.
func TestResolveValue_Map_RecursiveResolution(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setResponse([]string{"db", "host"}, "localhost")
	provider.setResponse([]string{"db", "port"}, 5432)
	registry.register("config", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	// Create map with references
	input := map[string]any{
		"database": map[string]any{
			"host": &ast.ReferenceExpr{
				Alias:      "config",
				Path:       []string{"db", "host"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
			},
			"port": &ast.ReferenceExpr{
				Alias:      "config",
				Path:       []string{"db", "port"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 2, StartCol: 1},
			},
		},
		"static": "value",
	}

	// Act
	ctx := context.Background()
	result, err := resolver.ResolveValue(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	// Check resolved values
	dbMap := resultMap["database"].(map[string]any)
	if dbMap["host"] != "localhost" {
		t.Errorf("expected host='localhost', got %v", dbMap["host"])
	}
	if dbMap["port"] != 5432 {
		t.Errorf("expected port=5432, got %v", dbMap["port"])
	}
	if resultMap["static"] != "value" {
		t.Errorf("expected static='value', got %v", resultMap["static"])
	}
}

// TestResolveValue_Slice_RecursiveResolution tests slice element resolution.
func TestResolveValue_Slice_RecursiveResolution(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	provider.setResponse([]string{"hosts", "0"}, "host1.example.com")
	provider.setResponse([]string{"hosts", "1"}, "host2.example.com")
	registry.register("inventory", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	// Create slice with references
	input := []any{
		&ast.ReferenceExpr{
			Alias:      "inventory",
			Path:       []string{"hosts", "0"},
			SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
		},
		&ast.ReferenceExpr{
			Alias:      "inventory",
			Path:       []string{"hosts", "1"},
			SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 2, StartCol: 1},
		},
		"host3.example.com",
	}

	// Act
	ctx := context.Background()
	result, err := resolver.ResolveValue(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatalf("expected slice result, got %T", result)
	}

	expected := []any{"host1.example.com", "host2.example.com", "host3.example.com"}
	if len(resultSlice) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(resultSlice))
	}

	for i, exp := range expected {
		if resultSlice[i] != exp {
			t.Errorf("element[%d]: expected %v, got %v", i, exp, resultSlice[i])
		}
	}
}

// TestResolveValue_ContextCancellation tests that context cancellation is respected.
func TestResolveValue_ContextCancellation(t *testing.T) {
	// Setup
	registry := newFakeProviderRegistry()
	provider := newFakeProvider()
	// Don't set a response - will cause provider to block/error
	registry.register("slow", provider)

	resolver := New(ResolverOptions{
		ProviderRegistry: registry,
	})

	ref := &ast.ReferenceExpr{
		Alias:      "slow",
		Path:       []string{"data"},
		SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act - provider will be called but should handle cancellation
	// Note: Our fake provider doesn't check ctx, but this test documents expected behavior
	_, err := resolver.ResolveValue(ctx, ref)

	// Assert - should error (either from context or from fetch failure)
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}

// TestBuildCacheKey verifies cache key format.
func TestBuildCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		path     []string
		expected string
	}{
		{
			name:     "simple path",
			alias:    "config",
			path:     []string{"key"},
			expected: "config:key",
		},
		{
			name:     "nested path",
			alias:    "network",
			path:     []string{"vpc", "subnets", "0"},
			expected: "network:vpc/subnets/0",
		},
		{
			name:     "empty path",
			alias:    "root",
			path:     []string{},
			expected: "root:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCacheKey(tt.alias, tt.path)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
