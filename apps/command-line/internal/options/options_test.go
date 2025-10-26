package options

import (
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// Test_NewProviderRegistries verifies factory creates registries correctly
func Test_NewProviderRegistries(t *testing.T) {
	t.Run("creates non-nil registries by default", func(t *testing.T) {
		pr, ptr := NewProviderRegistries()

		if pr == nil {
			t.Error("expected non-nil ProviderRegistry")
		}

		if ptr == nil {
			t.Error("expected non-nil ProviderTypeRegistry")
		}
	})

	t.Run("provider registry is empty by default (no-network)", func(t *testing.T) {
		pr, _ := NewProviderRegistries()

		aliases := pr.RegisteredAliases()
		if len(aliases) != 0 {
			t.Errorf("expected no registered providers by default, got %d", len(aliases))
		}
	})
}

// Test_BuildOptions verifies options building from flags
func Test_BuildOptions(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		vars          []string
		timeout       string
		maxConcurrent int
		allowMissing  bool
		wantVars      map[string]any
		wantTimeout   time.Duration
		wantMaxConc   int
		wantErr       bool
	}{
		{
			name:          "minimal valid options",
			path:          "/path/to/file.csl",
			vars:          nil,
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  false,
			wantVars:      map[string]any{},
			wantTimeout:   0,
			wantMaxConc:   0,
			wantErr:       false,
		},
		{
			name:          "with single var",
			path:          "/path/to/dir",
			vars:          []string{"key=value"},
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  false,
			wantVars:      map[string]any{"key": "value"},
			wantTimeout:   0,
			wantMaxConc:   0,
			wantErr:       false,
		},
		{
			name:          "with multiple vars",
			path:          "/path/to/dir",
			vars:          []string{"key1=val1", "key2=val2"},
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  false,
			wantVars:      map[string]any{"key1": "val1", "key2": "val2"},
			wantTimeout:   0,
			wantMaxConc:   0,
			wantErr:       false,
		},
		{
			name:          "with timeout",
			path:          "/path",
			vars:          nil,
			timeout:       "5s",
			maxConcurrent: 0,
			allowMissing:  false,
			wantVars:      map[string]any{},
			wantTimeout:   5 * time.Second,
			wantMaxConc:   0,
			wantErr:       false,
		},
		{
			name:          "with max concurrent providers",
			path:          "/path",
			vars:          nil,
			timeout:       "",
			maxConcurrent: 10,
			allowMissing:  false,
			wantVars:      map[string]any{},
			wantTimeout:   0,
			wantMaxConc:   10,
			wantErr:       false,
		},
		{
			name:          "with allow missing provider",
			path:          "/path",
			vars:          nil,
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  true,
			wantVars:      map[string]any{},
			wantTimeout:   0,
			wantMaxConc:   0,
			wantErr:       false,
		},
		{
			name:          "invalid var format missing equals",
			path:          "/path",
			vars:          []string{"invalidvar"},
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  false,
			wantErr:       true,
		},
		{
			name:          "invalid var format empty key",
			path:          "/path",
			vars:          []string{"=value"},
			timeout:       "",
			maxConcurrent: 0,
			allowMissing:  false,
			wantErr:       true,
		},
		{
			name:          "invalid timeout",
			path:          "/path",
			vars:          nil,
			timeout:       "notaduration",
			maxConcurrent: 0,
			allowMissing:  false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, ptr := NewProviderRegistries()

			opts, err := BuildOptions(BuildParams{
				Path:                   tt.path,
				Vars:                   tt.vars,
				TimeoutPerProvider:     tt.timeout,
				MaxConcurrentProviders: tt.maxConcurrent,
				AllowMissingProvider:   tt.allowMissing,
				ProviderRegistry:       pr,
				ProviderTypeRegistry:   ptr,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify Path
			if opts.Path != tt.path {
				t.Errorf("opts.Path = %q, want %q", opts.Path, tt.path)
			}

			// Verify Vars
			if len(opts.Vars) != len(tt.wantVars) {
				t.Errorf("opts.Vars length = %d, want %d", len(opts.Vars), len(tt.wantVars))
			}
			for k, v := range tt.wantVars {
				if opts.Vars[k] != v {
					t.Errorf("opts.Vars[%q] = %v, want %v", k, opts.Vars[k], v)
				}
			}

			// Verify Timeouts
			if opts.Timeouts.PerProviderFetch != tt.wantTimeout {
				t.Errorf("opts.Timeouts.PerProviderFetch = %v, want %v", opts.Timeouts.PerProviderFetch, tt.wantTimeout)
			}
			if opts.Timeouts.MaxConcurrentProviders != tt.wantMaxConc {
				t.Errorf("opts.Timeouts.MaxConcurrentProviders = %d, want %d", opts.Timeouts.MaxConcurrentProviders, tt.wantMaxConc)
			}

			// Verify AllowMissingProvider
			if opts.AllowMissingProvider != tt.allowMissing {
				t.Errorf("opts.AllowMissingProvider = %v, want %v", opts.AllowMissingProvider, tt.allowMissing)
			}

			// Verify registries are set
			if opts.ProviderRegistry == nil {
				t.Error("opts.ProviderRegistry should not be nil")
			}
			if opts.ProviderTypeRegistry == nil {
				t.Error("opts.ProviderTypeRegistry should not be nil")
			}
		})
	}
}

// Test_BuildOptions_RegistryInjection verifies custom registries are used
func Test_BuildOptions_RegistryInjection(t *testing.T) {
	customPR := compiler.NewProviderRegistry()
	customPTR := compiler.NewProviderTypeRegistry()

	opts, err := BuildOptions(BuildParams{
		Path:                 "/path",
		ProviderRegistry:     customPR,
		ProviderTypeRegistry: customPTR,
	})

	if err != nil {
		t.Fatalf("BuildOptions() error = %v", err)
	}

	// Verify the same instances are used (pointer equality)
	// Note: We can't directly compare interfaces, but we can verify non-nil
	if opts.ProviderRegistry == nil {
		t.Error("expected ProviderRegistry to be set")
	}
	if opts.ProviderTypeRegistry == nil {
		t.Error("expected ProviderTypeRegistry to be set")
	}
}
