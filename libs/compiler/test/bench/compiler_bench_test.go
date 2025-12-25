// Package bench provides performance benchmarks for the Nomos compiler library.
//
// These benchmarks measure merge semantics, reference resolution, and overall
// compilation performance to ensure acceptable performance characteristics.
//
// Run with:
//
//	go test -bench=. -benchmem ./test/bench
//
// Hardware context for reproducible results:
// Benchmark results are machine-dependent. See README.md for documented baseline.
package bench

import (
	"context"
	"fmt"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
)

// BenchmarkMergeSmall benchmarks deep-merge of small configuration maps.
// This represents typical single-file config with a few keys.
func BenchmarkMergeSmall(b *testing.B) {
	dst := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"cache": map[string]any{
			"enabled": true,
		},
	}

	src := map[string]any{
		"database": map[string]any{
			"port":     5433,
			"username": "admin",
		},
		"logging": map[string]any{
			"level": "info",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = compiler.DeepMerge(dst, src)
	}
}

// BenchmarkMergeLarge benchmarks deep-merge of larger configuration maps.
// This represents multi-file configs with many nested sections.
func BenchmarkMergeLarge(b *testing.B) {
	// Generate large nested configuration
	dst := generateLargeConfig(100, 3) // 100 top-level keys, 3 levels deep
	src := generateLargeConfig(100, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = compiler.DeepMerge(dst, src)
	}
}

// BenchmarkMergeWithProvenance benchmarks merge with provenance tracking.
func BenchmarkMergeWithProvenance(b *testing.B) {
	dst := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"cache": map[string]any{
			"enabled": true,
		},
	}

	src := map[string]any{
		"database": map[string]any{
			"port":     5433,
			"username": "admin",
		},
		"logging": map[string]any{
			"level": "info",
		},
	}

	dstFile := "base.csl"
	srcFile := "config.csl"
	provenance := make(map[string]compiler.Provenance)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset provenance for each iteration
		for k := range provenance {
			delete(provenance, k)
		}
		_ = compiler.DeepMergeWithProvenance(dst, dstFile, src, srcFile, provenance)
	}
}

// BenchmarkReferenceResolution benchmarks resolution of inline references.
// Tests provider fetch and caching performance.
func BenchmarkReferenceResolution(b *testing.B) {
	tests := []struct {
		name      string
		numRefs   int
		cacheHits bool // Whether references are identical (cache hits)
	}{
		{"single_ref", 1, false},
		{"10_unique_refs", 10, false},
		{"10_cached_refs", 10, true},
		{"100_cached_refs", 100, true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			ctx := context.Background()

			// Create fake provider with responses
			provider := fakes.NewFakeProvider("test")
			for i := 0; i < tt.numRefs; i++ {
				path := fmt.Sprintf("config/key%d", i)
				provider.FetchResponses[path] = fmt.Sprintf("value%d", i)
			}

			if tt.cacheHits {
				// All references point to the same path (cache hits)
				provider.FetchResponses["config/key0"] = "cached_value"
			}

			// Reset provider counters before benchmark
			provider.Reset()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// This benchmarks the internal reference resolution logic
				// In actual use, this would be part of the Compile call
				// For now, we simulate by calling provider directly
				for j := 0; j < tt.numRefs; j++ {
					var path []string
					if tt.cacheHits {
						path = []string{"config", "key0"}
					} else {
						path = []string{"config", fmt.Sprintf("key%d", j)}
					}
					_, _ = provider.Fetch(ctx, path)
				}
				provider.Reset()
			}
		})
	}
}

// BenchmarkCompileEmpty benchmarks compiling an empty directory.
// Tests compilation overhead without actual parsing.
func BenchmarkCompileEmpty(b *testing.B) {
	tmpDir := b.TempDir()
	ctx := context.Background()

	registry := &benchProviderRegistry{
		provider: fakes.NewFakeProvider("test"),
	}

	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: registry,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.Compile(ctx, opts)
	}
}

// Helper functions

// generateLargeConfig creates a nested map with specified depth and keys per level.
func generateLargeConfig(keysPerLevel int, depth int) map[string]any {
	if depth <= 0 {
		return map[string]any{"value": "leaf"}
	}

	result := make(map[string]any)
	for i := 0; i < keysPerLevel; i++ {
		key := fmt.Sprintf("key%d", i)
		result[key] = generateLargeConfig(keysPerLevel/2, depth-1)
	}
	return result
}

// benchProviderRegistry is a minimal provider registry for benchmarks.
type benchProviderRegistry struct {
	provider compiler.Provider
}

func (r *benchProviderRegistry) Register(_ string, _ compiler.ProviderConstructor) {
	// No-op for benchmarks
}

func (r *benchProviderRegistry) GetProvider(_ context.Context, _ string) (compiler.Provider, error) {
	return r.provider, nil
}

func (r *benchProviderRegistry) RegisteredAliases() []string {
	return []string{"test"}
}
