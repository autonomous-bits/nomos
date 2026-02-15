package test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// writeTestFile is a helper to write test content to a file.
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

// TestCompile_TopLevelScalars tests that inline scalar values at the top level
// are correctly compiled without creating empty-string keys.
func TestCompile_TopLevelScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantData map[string]any
	}{
		{
			name:  "scalar string",
			input: `region: "us-west-2"`,
			wantData: map[string]any{
				"region": "us-west-2",
			},
		},
		{
			name:  "scalar identifier",
			input: `enabled: true`,
			wantData: map[string]any{
				"enabled": "true",
			},
		},
		{
			name: "multiple scalars",
			input: `region: "us-west-2"
environment: "production"`,
			wantData: map[string]any{
				"region":      "us-west-2",
				"environment": "production",
			},
		},
		{
			name: "mixed scalars and maps",
			input: `region: "us-west-2"
database:
  host: "localhost"
  port: "5432"`,
			wantData: map[string]any{
				"region": "us-west-2",
				"database": map[string]any{
					"host": "localhost",
					"port": "5432",
				},
			},
		},
		{
			name: "nested map only",
			input: `database:
  host: "localhost"
  port: "5432"`,
			wantData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": "5432",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with input
			tmpFile := t.TempDir() + "/test.csl"
			if err := writeTestFile(tmpFile, tt.input); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			ctx := context.Background()
			result := compiler.Compile(ctx, compiler.Options{
				Path:             tmpFile,
				ProviderRegistry: compiler.NewProviderRegistry(),
			})

			if result.HasErrors() {
				t.Fatalf("compile failed: %v", result.Error())
			}

			// CRITICAL: Verify NO empty-string keys in output
			for key, value := range result.Snapshot.Data {
				if key == "" {
					t.Errorf("top-level data has empty-string key")
				}
				// Check nested maps
				if m, ok := value.(map[string]any); ok {
					for nestedKey := range m {
						if nestedKey == "" {
							t.Errorf("key %q has empty-string nested key", key)
						}
					}
				}
			}

			// Verify expected data structure
			if !reflect.DeepEqual(result.Snapshot.Data, tt.wantData) {
				t.Errorf("Data mismatch\nGot:  %+v\nWant: %+v",
					result.Snapshot.Data, tt.wantData)
			}
		})
	}
}

// TestCompile_InlineScalarNoEmptyKeys is a focused regression test
// to ensure the empty-string key bug is fixed.
func TestCompile_InlineScalarNoEmptyKeys(t *testing.T) {
	input := `region: "us-west-2"
vpc_id: "vpc-12345"`

	tmpFile := t.TempDir() + "/test.csl"
	if err := writeTestFile(tmpFile, input); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ctx := context.Background()
	result := compiler.Compile(ctx, compiler.Options{
		Path:             tmpFile,
		ProviderRegistry: compiler.NewProviderRegistry(),
	})

	if result.HasErrors() {
		t.Fatalf("compile failed: %v", result.Error())
	}

	// Verify region is a string, not a map with empty-string key
	region, ok := result.Snapshot.Data["region"]
	if !ok {
		t.Fatal("missing 'region' key")
	}

	if regionStr, ok := region.(string); ok {
		if regionStr != "us-west-2" {
			t.Errorf("region = %q, want 'us-west-2'", regionStr)
		}
	} else {
		// Check if it's a map with empty string key (the bug we're fixing)
		if regionMap, ok := region.(map[string]any); ok {
			if _, hasEmpty := regionMap[""]; hasEmpty {
				t.Error("region has empty-string key (bug not fixed)")
			}
			t.Errorf("region should be string, got map: %+v", regionMap)
		} else {
			t.Errorf("region has unexpected type: %T", region)
		}
	}

	// Verify vpc_id
	vpcID, ok := result.Snapshot.Data["vpc_id"]
	if !ok {
		t.Fatal("missing 'vpc_id' key")
	}

	if vpcIDStr, ok := vpcID.(string); ok {
		if vpcIDStr != "vpc-12345" {
			t.Errorf("vpc_id = %q, want 'vpc-12345'", vpcIDStr)
		}
	} else {
		if vpcIDMap, ok := vpcID.(map[string]any); ok {
			if _, hasEmpty := vpcIDMap[""]; hasEmpty {
				t.Error("vpc_id has empty-string key (bug not fixed)")
			}
			t.Errorf("vpc_id should be string, got map: %+v", vpcIDMap)
		} else {
			t.Errorf("vpc_id has unexpected type: %T", vpcID)
		}
	}
}

// TestCompile_ScalarWithReference tests that inline scalar references work correctly.
func TestCompile_ScalarWithReference(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main file with inline scalar reference
	mainFile := tmpDir + "/main.csl"
	input := `source:
  alias: 'network'
  type: 'test'

cidr: @network:.:vpc_cidr`

	if err := writeTestFile(mainFile, input); err != nil {
		t.Fatalf("failed to create main file: %v", err)
	}

	// Create provider registry with test provider that returns network data
	registry := compiler.NewProviderRegistry()
	registry.Register("network", func(_ compiler.ProviderInitOptions) (compiler.Provider, error) {
		return &testProvider{
			data: map[string]any{
				"vpc_cidr": "10.0.0.0/16",
			},
		}, nil
	})

	ctx := context.Background()
	result := compiler.Compile(ctx, compiler.Options{
		Path:             mainFile,
		ProviderRegistry: registry,
	})

	if result.HasErrors() {
		t.Fatalf("compile failed: %v", result.Error())
	}

	// Verify cidr is resolved correctly
	cidr, ok := result.Snapshot.Data["cidr"]
	if !ok {
		t.Fatal("missing 'cidr' key")
	}

	if cidrStr, ok := cidr.(string); ok {
		if cidrStr != "10.0.0.0/16" {
			t.Errorf("cidr = %q, want '10.0.0.0/16'", cidrStr)
		}
	} else {
		t.Errorf("cidr should be string, got type: %T, value: %+v", cidr, cidr)
	}

	// Verify no empty-string keys
	for key, value := range result.Snapshot.Data {
		if key == "" {
			t.Error("top-level data has empty-string key")
		}
		if m, ok := value.(map[string]any); ok {
			for nestedKey := range m {
				if nestedKey == "" {
					t.Errorf("key %q has empty-string nested key", key)
				}
			}
		}
	}
}

// testProvider is a simple test provider that returns pre-configured data.
type testProvider struct {
	data map[string]any
}

func (p *testProvider) Init(_ context.Context, _ compiler.ProviderInitOptions) error {
	return nil
}

func (p *testProvider) Fetch(_ context.Context, path []string) (any, error) {
	if len(path) == 0 {
		return p.data, nil
	}

	// Navigate through the path
	current := p.data
	for i, key := range path {
		val, ok := current[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found", key)
		}

		// If this is the last key, return the value
		if i == len(path)-1 {
			return val, nil
		}

		// Otherwise, try to navigate deeper
		currentMap, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate into non-map value")
		}
		current = currentMap
	}

	return current, nil
}
