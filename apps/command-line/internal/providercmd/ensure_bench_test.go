package providercmd

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// BenchmarkEnsureProviders_AllCached benchmarks the "Checking providers..." phase
// when all providers are already cached. This verifies the performance goal from
// SC-002 in plan.md: cached provider check overhead should be ≤100ms.
//
// Setup:
// - 5 providers pre-cached in lockfile
// - All binaries exist and pass validation
//
// Benchmark measures:
// - Time for EnsureProviders() to complete with all providers cached
// - Expected: < 100ms per operation for typical multi-provider projects
func BenchmarkEnsureProviders_AllCached(b *testing.B) {
	// Setup: Create temporary directory with cached providers
	tmpDir := b.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create .nomos directory structure
	nomosDir := filepath.Join(tmpDir, ".nomos")
	if err := os.MkdirAll(nomosDir, 0750); err != nil {
		b.Fatalf("failed to create .nomos directory: %v", err)
	}

	// Create test .csl files declaring multiple providers
	providerConfigs := []struct {
		filename string
		alias    string
		version  string
	}{
		{"config1.csl", "provider1", "1.0.0"},
		{"config2.csl", "provider2", "1.0.0"},
		{"config3.csl", "provider3", "1.0.0"},
		{"config4.csl", "provider4", "1.0.0"},
		{"config5.csl", "provider5", "1.0.0"},
	}

	cslPaths := make([]string, 0, len(providerConfigs))
	for _, cfg := range providerConfigs {
		cslPath := filepath.Join(tmpDir, cfg.filename)
		cslContent := "source:\n" +
			"  alias: '" + cfg.alias + "'\n" +
			"  type: 'owner/repo'\n" +
			"  version: '" + cfg.version + "'\n" +
			"\n" +
			"app:\n" +
			"  name: 'test-app'\n"
		if err := os.WriteFile(cslPath, []byte(cslContent), 0600); err != nil {
			b.Fatalf("failed to write %s: %v", cfg.filename, err)
		}
		cslPaths = append(cslPaths, cslPath)
	}

	// Create provider binaries and lockfile
	// Simulate fully cached state
	entries := make([]ProviderEntry, 0, len(providerConfigs))
	for _, cfg := range providerConfigs {
		// Create provider binary directory
		providerDir := filepath.Join(nomosDir, "providers", "owner", "repo", cfg.version,
			runtime.GOOS+"-"+runtime.GOARCH)
		if err := os.MkdirAll(providerDir, 0750); err != nil {
			b.Fatalf("failed to create provider directory: %v", err)
		}

		// Create dummy provider binary
		binaryPath := filepath.Join(providerDir, "provider")
		binaryContent := []byte("#!/bin/sh\necho 'test provider'")
		if err := os.WriteFile(binaryPath, binaryContent, 0700); err != nil { //nolint:gosec // G306: Test binary needs execute permission
			b.Fatalf("failed to write provider binary: %v", err)
		}

		// Compute checksum for validation
		checksum := computeChecksum(binaryContent)

		// Build relative path for lockfile
		relativePath := filepath.Join("owner", "repo", cfg.version,
			runtime.GOOS+"-"+runtime.GOARCH, "provider")

		// Create lockfile entry
		entry := ProviderEntry{
			Alias:    cfg.alias,
			Type:     "owner/repo",
			Version:  cfg.version,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Path:     relativePath,
			Checksum: checksum,
			Size:     int64(len(binaryContent)),
		}
		entries = append(entries, entry)
	}

	// Write lockfile
	lockfile := LockFile{Providers: entries}
	if err := WriteLockFile(lockfile); err != nil {
		b.Fatalf("failed to write lockfile: %v", err)
	}

	// Create ProviderOptions
	opts := ProviderOptions{
		Paths: cslPaths,
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	// Reset timer - only measure EnsureProviders() execution
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		summary, err := EnsureProviders(opts)
		if err != nil {
			b.Fatalf("EnsureProviders failed: %v", err)
		}

		// Verify all providers were cached (no downloads)
		if summary.Downloaded != 0 {
			b.Errorf("Expected 0 downloads (all cached), got %d", summary.Downloaded)
		}
		if summary.Cached != len(providerConfigs) {
			b.Errorf("Expected %d cached, got %d", len(providerConfigs), summary.Cached)
		}
	}

	b.StopTimer()

	// Report results summary
	// The benchmark will print ops/sec automatically
	// Compare elapsed time per operation against 100ms target
}

// BenchmarkEnsureProviders_SingleProvider benchmarks with a single cached provider
// to establish baseline performance for minimal provider checks.
func BenchmarkEnsureProviders_SingleProvider(b *testing.B) {
	tmpDir := b.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create .nomos directory
	nomosDir := filepath.Join(tmpDir, ".nomos")
	if err := os.MkdirAll(nomosDir, 0750); err != nil {
		b.Fatalf("failed to create .nomos directory: %v", err)
	}

	// Create single .csl file
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `source:
  alias: 'test'
  type: 'owner/repo'
  version: '1.0.0'

app:
  name: 'test-app'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0600); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	// Create provider binary
	providerDir := filepath.Join(nomosDir, "providers", "owner", "repo", "1.0.0",
		runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.MkdirAll(providerDir, 0750); err != nil {
		b.Fatalf("failed to create provider directory: %v", err)
	}

	binaryPath := filepath.Join(providerDir, "provider")
	binaryContent := []byte("#!/bin/sh\necho 'test provider'")
	if err := os.WriteFile(binaryPath, binaryContent, 0700); err != nil { //nolint:gosec // G306: Test binary needs execute permission
		b.Fatalf("failed to write provider binary: %v", err)
	}

	// Write lockfile
	checksum := computeChecksum(binaryContent)
	relativePath := filepath.Join("owner", "repo", "1.0.0",
		runtime.GOOS+"-"+runtime.GOARCH, "provider")

	entry := ProviderEntry{
		Alias:    "test",
		Type:     "owner/repo",
		Version:  "1.0.0",
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Path:     relativePath,
		Checksum: checksum,
		Size:     int64(len(binaryContent)),
	}

	lockfile := LockFile{Providers: []ProviderEntry{entry}}
	if err := WriteLockFile(lockfile); err != nil {
		b.Fatalf("failed to write lockfile: %v", err)
	}

	opts := ProviderOptions{
		Paths: []string{cslPath},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		summary, err := EnsureProviders(opts)
		if err != nil {
			b.Fatalf("EnsureProviders failed: %v", err)
		}

		if summary.Cached != 1 {
			b.Errorf("Expected 1 cached, got %d", summary.Cached)
		}
	}

	b.StopTimer()
}

// BenchmarkEnsureProviders_NoProviders benchmarks the case where no providers
// are declared (early exit path).
func BenchmarkEnsureProviders_NoProviders(b *testing.B) {
	tmpDir := b.TempDir()

	// Create .csl file with no provider declarations
	cslPath := filepath.Join(tmpDir, "config.csl")
	cslContent := `app:
  name: 'test-app'
  environment: 'production'
`
	if err := os.WriteFile(cslPath, []byte(cslContent), 0600); err != nil {
		b.Fatalf("failed to write config: %v", err)
	}

	opts := ProviderOptions{
		Paths: []string{cslPath},
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		summary, err := EnsureProviders(opts)
		if err != nil {
			b.Fatalf("EnsureProviders failed: %v", err)
		}

		if summary.Total != 0 {
			b.Errorf("Expected 0 total providers, got %d", summary.Total)
		}
	}

	b.StopTimer()
}

// computeChecksum computes SHA256 checksum for benchmark test setup.
func computeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
