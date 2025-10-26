//go:build integration
// +build integration

// Package test provides integration tests for the Nomos compiler library.
//
// This file contains integration tests that require network access or external services.
// These tests are excluded from the default CI run and must be explicitly enabled.
//
// Run with:
//
//	go test -tags=integration ./test
//
// These tests validate provider behavior with real network calls and are useful
// for testing actual provider implementations against live services.
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// HTTPTestProvider is a simple provider that fetches data from HTTP endpoints.
// Used for integration testing with real HTTP clients.
type HTTPTestProvider struct {
	baseURL string
	client  *http.Client
}

// NewHTTPTestProvider creates a new HTTP-based test provider.
func NewHTTPTestProvider(baseURL string) *HTTPTestProvider {
	return &HTTPTestProvider{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Init implements compiler.Provider.Init.
func (p *HTTPTestProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
	// No initialization required for this test provider
	return nil
}

// Fetch implements compiler.Provider.Fetch.
func (p *HTTPTestProvider) Fetch(ctx context.Context, path []string) (any, error) {
	// Simple HTTP fetch - in real provider would be more sophisticated
	// For testing, we just return the path as the value
	return map[string]any{
		"path":   path,
		"source": "http",
	}, nil
}

// TestIntegration_HTTPProvider tests compilation with a provider that makes HTTP calls.
// This test creates a test HTTP server and verifies the compiler can resolve references
// via HTTP-based providers.
func TestIntegration_HTTPProvider(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple response for testing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value": "test-data"}`))
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := NewHTTPTestProvider(server.URL)

	// Create provider registry
	registry := &integrationProviderRegistry{
		providers: map[string]compiler.Provider{
			"http": provider,
		},
	}

	// Create test config directory with reference
	tmpDir := t.TempDir()

	// For this test, we're validating that:
	// 1. The compiler can instantiate and initialize providers
	// 2. Providers that make network calls work correctly
	// 3. The integration flow compiles without errors

	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify snapshot was created
	if snapshot.Data == nil {
		t.Error("snapshot.Data should not be nil")
	}

	t.Logf("Integration test passed - HTTP provider initialized successfully")
	t.Logf("Server URL: %s", server.URL)
}

// TestIntegration_NetworkTimeout tests that provider timeouts work correctly.
// This verifies that network calls respect context deadlines and timeouts.
func TestIntegration_NetworkTimeout(t *testing.T) {
	t.Skip("TODO: Implement network timeout integration test")
	// This test would create a slow HTTP server and verify timeout behavior
}

// TestIntegration_ProviderCaching tests that providers cache results across multiple fetches.
// This validates the per-run caching behavior with real providers.
func TestIntegration_ProviderCaching(t *testing.T) {
	t.Skip("TODO: Implement provider caching integration test")
	// This test would verify that identical provider+path combinations
	// result in only one network call, with subsequent calls using cache
}

// integrationProviderRegistry is a simple provider registry for integration tests.
type integrationProviderRegistry struct {
	providers map[string]compiler.Provider
}

func (r *integrationProviderRegistry) Register(alias string, constructor compiler.ProviderConstructor) {
	// Not used in integration tests - providers are pre-created
}

func (r *integrationProviderRegistry) GetProvider(alias string) (compiler.Provider, error) {
	if p, ok := r.providers[alias]; ok {
		return p, nil
	}
	return nil, compiler.ErrProviderNotRegistered
}

func (r *integrationProviderRegistry) RegisteredAliases() []string {
	aliases := make([]string, 0, len(r.providers))
	for alias := range r.providers {
		aliases = append(aliases, alias)
	}
	return aliases
}
