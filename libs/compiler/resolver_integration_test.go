package compiler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestResolveReferences_Integration tests the reference resolution integration.
func TestResolveReferences_Integration(t *testing.T) {
	t.Run("resolves references in data", func(t *testing.T) {
		// Setup provider registry with test provider
		registry := NewProviderRegistry()
		registry.Register("config", func(_ ProviderInitOptions) (Provider, error) {
			return &testProvider{
				responses: map[string]any{
					"db/host": "localhost",
					"db/port": 5432,
				},
			}, nil
		})

		// Create data with references
		data := map[string]any{
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

		opts := Options{
			Path:             "/test",
			ProviderRegistry: registry,
		}

		var warnings []string
		ctx := context.Background()

		// Resolve references
		resolved, err := resolveReferences(ctx, data, opts, &warnings)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}

		// Check resolved values
		dbMap := resolved["database"].(map[string]any)
		if dbMap["host"] != "localhost" {
			t.Errorf("expected host='localhost', got %v", dbMap["host"])
		}
		if dbMap["port"] != 5432 {
			t.Errorf("expected port=5432, got %v", dbMap["port"])
		}
		if resolved["static"] != "value" {
			t.Errorf("expected static='value', got %v", resolved["static"])
		}
	})

	t.Run("handles missing provider with AllowMissingProvider", func(t *testing.T) {
		// Setup empty registry
		registry := NewProviderRegistry()

		// Create data with reference to missing provider
		data := map[string]any{
			"value": &ast.ReferenceExpr{
				Alias:      "missing",
				Path:       []string{"key"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
			},
		}

		opts := Options{
			Path:                 "/test",
			ProviderRegistry:     registry,
			AllowMissingProvider: true,
		}

		var warnings []string
		ctx := context.Background()

		// Resolve references
		resolved, err := resolveReferences(ctx, data, opts, &warnings)

		// Assert - should not error
		if err != nil {
			t.Fatalf("expected no error with AllowMissingProvider, got %v", err)
		}

		// Should have recorded warning
		if len(warnings) != 1 {
			t.Errorf("expected 1 warning, got %d", len(warnings))
		}

		// Value should be nil
		if resolved["value"] != nil {
			t.Errorf("expected nil value for missing provider, got %v", resolved["value"])
		}
	})

	t.Run("errors on missing provider without AllowMissingProvider", func(t *testing.T) {
		// Setup empty registry
		registry := NewProviderRegistry()

		// Create data with reference to missing provider
		data := map[string]any{
			"value": &ast.ReferenceExpr{
				Alias:      "missing",
				Path:       []string{"key"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
			},
		}

		opts := Options{
			Path:                 "/test",
			ProviderRegistry:     registry,
			AllowMissingProvider: false, // Explicitly false
		}

		var warnings []string
		ctx := context.Background()

		// Resolve references
		_, err := resolveReferences(ctx, data, opts, &warnings)

		// Assert - should error
		if err == nil {
			t.Fatal("expected error for missing provider, got nil")
		}

		// Error should mention provider not registered
		errMsg := err.Error()
		if !strings.Contains(errMsg, "provider not registered") && !strings.Contains(errMsg, "missing") {
			t.Errorf("error should mention provider issue, got: %v", err)
		}
	})

	t.Run("caches provider fetch results", func(t *testing.T) {
		// Setup provider that tracks fetch calls
		provider := &testProvider{
			responses: map[string]any{
				"config/value": "cached-data",
			},
		}

		registry := NewProviderRegistry()
		registry.Register("source", func(_ ProviderInitOptions) (Provider, error) {
			return provider, nil
		})

		// Create data with same reference twice
		data := map[string]any{
			"first": &ast.ReferenceExpr{
				Alias:      "source",
				Path:       []string{"config", "value"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 1, StartCol: 1},
			},
			"second": &ast.ReferenceExpr{
				Alias:      "source",
				Path:       []string{"config", "value"},
				SourceSpan: ast.SourceSpan{Filename: "test.csl", StartLine: 2, StartCol: 1},
			},
		}

		opts := Options{
			Path:             "/test",
			ProviderRegistry: registry,
		}

		var warnings []string
		ctx := context.Background()

		// Resolve references
		resolved, err := resolveReferences(ctx, data, opts, &warnings)

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Both values should be resolved
		if resolved["first"] != "cached-data" {
			t.Errorf("expected first='cached-data', got %v", resolved["first"])
		}
		if resolved["second"] != "cached-data" {
			t.Errorf("expected second='cached-data', got %v", resolved["second"])
		}

		// Provider should have been called only once (cached)
		if provider.fetchCount != 1 {
			t.Errorf("expected 1 fetch call (cached), got %d", provider.fetchCount)
		}
	})
}

// testProvider is a simple provider for integration testing.
type testProvider struct {
	responses  map[string]any
	fetchCount int
}

func (p *testProvider) Init(_ context.Context, _ ProviderInitOptions) error {
	return nil
}

func (p *testProvider) Fetch(_ context.Context, path []string) (any, error) {
	p.fetchCount++

	// Build key from path
	var key string
	if len(path) > 0 {
		key = path[0]
		for i := 1; i < len(path); i++ {
			key += "/" + path[i]
		}
	}

	if val, ok := p.responses[key]; ok {
		return val, nil
	}

	return nil, errors.New("not found")
}
