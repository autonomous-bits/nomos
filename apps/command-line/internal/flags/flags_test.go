package flags

import (
	"testing"
	"time"
)

// TestBuildFlags_Parse_BasicFlags tests parsing of the core build flags.
func TestBuildFlags_Parse_BasicFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    BuildFlags
		wantErr bool
		errMsg  string
	}{
		{
			name: "path flag short form",
			args: []string{"-p", "/path/to/file.csl"},
			want: BuildFlags{
				Path:   "/path/to/file.csl",
				Format: "json", // Default format
			},
		},
		{
			name: "path flag long form",
			args: []string{"--path", "/path/to/dir"},
			want: BuildFlags{
				Path:   "/path/to/dir",
				Format: "json", // Default format
			},
		},
		{
			name: "format flag json",
			args: []string{"-p", "test.csl", "-f", "json"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json",
			},
		},
		{
			name: "format flag yaml",
			args: []string{"--path", "test.csl", "--format", "yaml"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "yaml",
			},
		},
		{
			name: "format flag hcl",
			args: []string{"-p", "test.csl", "-f", "hcl"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "hcl",
			},
		},
		{
			name: "out flag short form",
			args: []string{"-p", "test.csl", "-o", "output.json"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json", // Default format
				Out:    "output.json",
			},
		},
		{
			name: "out flag long form",
			args: []string{"--path", "test.csl", "--out", "result.yaml"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json", // Default format
				Out:    "result.yaml",
			},
		},
		{
			name: "single var flag",
			args: []string{"-p", "test.csl", "--var", "region=us-west"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json", // Default format
				Vars:   []string{"region=us-west"},
			},
		},
		{
			name: "multiple var flags",
			args: []string{"-p", "test.csl", "--var", "region=us-west", "--var", "env=prod"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json", // Default format
				Vars:   []string{"region=us-west", "env=prod"},
			},
		},
		{
			name: "strict flag",
			args: []string{"-p", "test.csl", "--strict"},
			want: BuildFlags{
				Path:   "test.csl",
				Format: "json", // Default format
				Strict: true,
			},
		},
		{
			name: "allow-missing-provider flag",
			args: []string{"-p", "test.csl", "--allow-missing-provider"},
			want: BuildFlags{
				Path:                 "test.csl",
				Format:               "json", // Default format
				AllowMissingProvider: true,
			},
		},
		{
			name: "timeout-per-provider flag",
			args: []string{"-p", "test.csl", "--timeout-per-provider", "10s"},
			want: BuildFlags{
				Path:               "test.csl",
				Format:             "json", // Default format
				TimeoutPerProvider: "10s",
			},
		},
		{
			name: "max-concurrent-providers flag",
			args: []string{"-p", "test.csl", "--max-concurrent-providers", "5"},
			want: BuildFlags{
				Path:                   "test.csl",
				Format:                 "json", // Default format
				MaxConcurrentProviders: 5,
			},
		},
		{
			name: "verbose flag",
			args: []string{"-p", "test.csl", "--verbose"},
			want: BuildFlags{
				Path:    "test.csl",
				Format:  "json", // Default format
				Verbose: true,
			},
		},
		{
			name: "all flags combined",
			args: []string{
				"-p", "/path/to/configs",
				"-f", "yaml",
				"-o", "snapshot.yaml",
				"--var", "region=eu-west",
				"--var", "env=staging",
				"--strict",
				"--allow-missing-provider",
				"--timeout-per-provider", "30s",
				"--max-concurrent-providers", "10",
				"--verbose",
			},
			want: BuildFlags{
				Path:                   "/path/to/configs",
				Format:                 "yaml",
				Out:                    "snapshot.yaml",
				Vars:                   []string{"region=eu-west", "env=staging"},
				Strict:                 true,
				AllowMissingProvider:   true,
				TimeoutPerProvider:     "30s",
				MaxConcurrentProviders: 10,
				Verbose:                true,
			},
		},
		{
			name:    "missing required path flag",
			args:    []string{},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name:    "invalid format value",
			args:    []string{"-p", "test.csl", "-f", "xml"},
			wantErr: true,
			errMsg:  "format",
		},
		{
			name:    "negative max-concurrent-providers",
			args:    []string{"-p", "test.csl", "--max-concurrent-providers", "-5"},
			wantErr: true,
			errMsg:  "max-concurrent-providers",
		},
		{
			name:    "invalid timeout-per-provider",
			args:    []string{"-p", "test.csl", "--timeout-per-provider", "notaduration"},
			wantErr: true,
			errMsg:  "timeout-per-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}

			// Compare fields
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
			if got.Format != tt.want.Format {
				t.Errorf("Format = %q, want %q", got.Format, tt.want.Format)
			}
			if got.Out != tt.want.Out {
				t.Errorf("Out = %q, want %q", got.Out, tt.want.Out)
			}
			if !stringSliceEqual(got.Vars, tt.want.Vars) {
				t.Errorf("Vars = %v, want %v", got.Vars, tt.want.Vars)
			}
			if got.Strict != tt.want.Strict {
				t.Errorf("Strict = %v, want %v", got.Strict, tt.want.Strict)
			}
			if got.AllowMissingProvider != tt.want.AllowMissingProvider {
				t.Errorf("AllowMissingProvider = %v, want %v", got.AllowMissingProvider, tt.want.AllowMissingProvider)
			}
			if got.TimeoutPerProvider != tt.want.TimeoutPerProvider {
				t.Errorf("TimeoutPerProvider = %q, want %q", got.TimeoutPerProvider, tt.want.TimeoutPerProvider)
			}
			if got.MaxConcurrentProviders != tt.want.MaxConcurrentProviders {
				t.Errorf("MaxConcurrentProviders = %d, want %d", got.MaxConcurrentProviders, tt.want.MaxConcurrentProviders)
			}
			if got.Verbose != tt.want.Verbose {
				t.Errorf("Verbose = %v, want %v", got.Verbose, tt.want.Verbose)
			}
		})
	}
}

// TestBuildFlags_ToCompilerOptions tests conversion of BuildFlags to compiler.Options.
func TestBuildFlags_ToCompilerOptions(t *testing.T) {
	flags := BuildFlags{
		Path:                   "/test/path",
		Vars:                   []string{"key1=value1", "key2=value2"},
		AllowMissingProvider:   true,
		TimeoutPerProvider:     "5s",
		MaxConcurrentProviders: 10,
	}

	opts, err := flags.ToCompilerOptions()
	if err != nil {
		t.Fatalf("ToCompilerOptions() unexpected error = %v", err)
	}

	if opts.Path != "/test/path" {
		t.Errorf("Options.Path = %q, want %q", opts.Path, "/test/path")
	}

	if !opts.AllowMissingProvider {
		t.Errorf("Options.AllowMissingProvider = false, want true")
	}

	expectedTimeout := 5 * time.Second
	if opts.Timeouts.PerProviderFetch != expectedTimeout {
		t.Errorf("Options.Timeouts.PerProviderFetch = %v, want %v", opts.Timeouts.PerProviderFetch, expectedTimeout)
	}

	if opts.Timeouts.MaxConcurrentProviders != 10 {
		t.Errorf("Options.Timeouts.MaxConcurrentProviders = %d, want 10", opts.Timeouts.MaxConcurrentProviders)
	}

	// Check vars map
	if opts.Vars == nil {
		t.Fatal("Options.Vars is nil")
	}
	if opts.Vars["key1"] != "value1" {
		t.Errorf("Options.Vars[key1] = %v, want value1", opts.Vars["key1"])
	}
	if opts.Vars["key2"] != "value2" {
		t.Errorf("Options.Vars[key2] = %v, want value2", opts.Vars["key2"])
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && containsAtAnyPosition(s, substr)))
}

func containsAtAnyPosition(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
