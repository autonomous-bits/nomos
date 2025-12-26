//go:build integration
// +build integration

package test

import (
	"context"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/options"
	"github.com/autonomous-bits/nomos/libs/compiler"
)

// mockCompiler is a test double for the compiler that records Options passed to Compile.
type mockCompiler struct {
	// RecordedOptions holds the compiler.Options passed to Compile
	RecordedOptions *compiler.Options
	// ReturnSnapshot is the snapshot to return from Compile
	ReturnSnapshot compiler.Snapshot
	// ReturnError is the error to return from Compile
	ReturnError error
}

// Compile records the options and returns the configured response.
func (m *mockCompiler) Compile(_ context.Context, opts compiler.Options) (compiler.Snapshot, error) {
	m.RecordedOptions = &opts
	return m.ReturnSnapshot, m.ReturnError
}

// TestOptionsBuilder_Integration verifies options are correctly built and passed to compiler.
func TestOptionsBuilder_Integration(t *testing.T) {
	tests := []struct {
		name                   string
		path                   string
		vars                   []string
		timeoutPerProvider     string
		maxConcurrentProviders int
		allowMissingProvider   bool
		wantPath               string
		wantVars               map[string]any
		wantTimeout            time.Duration
		wantMaxConcurrent      int
		wantAllowMissing       bool
	}{
		{
			name:                   "minimal options",
			path:                   "/path/to/test.csl",
			vars:                   nil,
			timeoutPerProvider:     "",
			maxConcurrentProviders: 0,
			allowMissingProvider:   false,
			wantPath:               "/path/to/test.csl",
			wantVars:               map[string]any{},
			wantTimeout:            0,
			wantMaxConcurrent:      0,
			wantAllowMissing:       false,
		},
		{
			name:                   "with vars",
			path:                   "/path/to/dir",
			vars:                   []string{"env=prod", "region=us-west"},
			timeoutPerProvider:     "",
			maxConcurrentProviders: 0,
			allowMissingProvider:   false,
			wantPath:               "/path/to/dir",
			wantVars:               map[string]any{"env": "prod", "region": "us-west"},
			wantTimeout:            0,
			wantMaxConcurrent:      0,
			wantAllowMissing:       false,
		},
		{
			name:                   "with timeout and concurrency",
			path:                   "/path",
			vars:                   nil,
			timeoutPerProvider:     "30s",
			maxConcurrentProviders: 5,
			allowMissingProvider:   false,
			wantPath:               "/path",
			wantVars:               map[string]any{},
			wantTimeout:            30 * time.Second,
			wantMaxConcurrent:      5,
			wantAllowMissing:       false,
		},
		{
			name:                   "with allow missing provider",
			path:                   "/path",
			vars:                   nil,
			timeoutPerProvider:     "",
			maxConcurrentProviders: 0,
			allowMissingProvider:   true,
			wantPath:               "/path",
			wantVars:               map[string]any{},
			wantTimeout:            0,
			wantMaxConcurrent:      0,
			wantAllowMissing:       true,
		},
		{
			name:                   "full options",
			path:                   "/full/path.csl",
			vars:                   []string{"a=1", "b=2", "c=3"},
			timeoutPerProvider:     "1m",
			maxConcurrentProviders: 10,
			allowMissingProvider:   true,
			wantPath:               "/full/path.csl",
			wantVars:               map[string]any{"a": "1", "b": "2", "c": "3"},
			wantTimeout:            60 * time.Second,
			wantMaxConcurrent:      10,
			wantAllowMissing:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider registries
			pr, ptr := options.NewProviderRegistries()

			// Build options
			opts, err := options.BuildOptions(options.BuildParams{
				Path:                   tt.path,
				Vars:                   tt.vars,
				TimeoutPerProvider:     tt.timeoutPerProvider,
				MaxConcurrentProviders: tt.maxConcurrentProviders,
				AllowMissingProvider:   tt.allowMissingProvider,
				ProviderRegistry:       pr,
				ProviderTypeRegistry:   ptr,
			})

			if err != nil {
				t.Fatalf("BuildOptions() failed: %v", err)
			}

			// Create mock compiler
			mock := &mockCompiler{
				ReturnSnapshot: compiler.Snapshot{
					Data: map[string]any{"test": "data"},
					Metadata: compiler.Metadata{
						InputFiles: []string{tt.path},
					},
				},
			}

			// Simulate calling compiler with built options
			_, _ = mock.Compile(context.Background(), opts)

			// Verify the recorded options match expected values
			if mock.RecordedOptions == nil {
				t.Fatal("expected Options to be recorded in mock compiler")
			}

			recorded := mock.RecordedOptions

			// Verify Path
			if recorded.Path != tt.wantPath {
				t.Errorf("recorded Path = %q, want %q", recorded.Path, tt.wantPath)
			}

			// Verify Vars
			if len(recorded.Vars) != len(tt.wantVars) {
				t.Errorf("recorded Vars length = %d, want %d", len(recorded.Vars), len(tt.wantVars))
			}
			for k, v := range tt.wantVars {
				if recorded.Vars[k] != v {
					t.Errorf("recorded Vars[%q] = %v, want %v", k, recorded.Vars[k], v)
				}
			}

			// Verify Timeouts
			if recorded.Timeouts.PerProviderFetch != tt.wantTimeout {
				t.Errorf("recorded PerProviderFetch = %v, want %v",
					recorded.Timeouts.PerProviderFetch, tt.wantTimeout)
			}
			if recorded.Timeouts.MaxConcurrentProviders != tt.wantMaxConcurrent {
				t.Errorf("recorded MaxConcurrentProviders = %d, want %d",
					recorded.Timeouts.MaxConcurrentProviders, tt.wantMaxConcurrent)
			}

			// Verify AllowMissingProvider
			if recorded.AllowMissingProvider != tt.wantAllowMissing {
				t.Errorf("recorded AllowMissingProvider = %v, want %v",
					recorded.AllowMissingProvider, tt.wantAllowMissing)
			}

			// Verify provider registries are set
			if recorded.ProviderRegistry == nil {
				t.Error("expected ProviderRegistry to be set")
			}
			if recorded.ProviderTypeRegistry == nil {
				t.Error("expected ProviderTypeRegistry to be set")
			}

			// Verify provider registries are empty (no-network default)
			aliases := recorded.ProviderRegistry.RegisteredAliases()
			if len(aliases) != 0 {
				t.Errorf("expected no registered providers (no-network default), got %d", len(aliases))
			}
		})
	}
}

// TestOptionsBuilder_CustomRegistries verifies custom registries are preserved.
func TestOptionsBuilder_CustomRegistries(t *testing.T) {
	// Create custom registries with test providers
	customPR := compiler.NewProviderRegistry()
	customPR.Register("test-provider", func(_ compiler.ProviderInitOptions) (compiler.Provider, error) {
		return nil, nil // Minimal test provider
	})

	customPTR := compiler.NewProviderTypeRegistry()

	// Build options with custom registries
	opts, err := options.BuildOptions(options.BuildParams{
		Path:                 "/test/path",
		ProviderRegistry:     customPR,
		ProviderTypeRegistry: customPTR,
	})

	if err != nil {
		t.Fatalf("BuildOptions() failed: %v", err)
	}

	// Create mock compiler and pass options
	mock := &mockCompiler{
		ReturnSnapshot: compiler.Snapshot{},
	}
	_, _ = mock.Compile(context.Background(), opts)

	// Verify custom registries are preserved
	if mock.RecordedOptions.ProviderRegistry == nil {
		t.Fatal("expected ProviderRegistry to be set")
	}

	aliases := mock.RecordedOptions.ProviderRegistry.RegisteredAliases()
	if len(aliases) != 1 || aliases[0] != "test-provider" {
		t.Errorf("expected custom provider registry with 'test-provider', got %v", aliases)
	}
}
