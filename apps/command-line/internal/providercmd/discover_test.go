package providercmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverProviders_WithVersion tests that DiscoverProviders correctly
// extracts version from SourceDecl.Version field (not from Config map).
func TestDiscoverProviders_WithVersion(t *testing.T) {
	// Arrange: Create temp config file with versioned source declaration
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.csl")

	configContent := `source:
  alias: 'testprovider'
  type: 'owner/repo'
  version: '1.2.3'
  some_config: 'value'
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act: Discover providers
	providers, err := DiscoverProviders([]string{configPath})
	if err != nil {
		t.Fatalf("DiscoverProviders failed: %v", err)
	}

	// Assert: Provider discovered with correct version
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}

	p := providers[0]
	if p.Alias != "testprovider" {
		t.Errorf("alias = %q, want %q", p.Alias, "testprovider")
	}
	if p.Type != "owner/repo" {
		t.Errorf("type = %q, want %q", p.Type, "owner/repo")
	}
	if p.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", p.Version, "1.2.3")
	}

	// Verify version is NOT in config map
	if _, hasVersion := p.Config["version"]; hasVersion {
		t.Error("version should not be in Config map (should be in Version field)")
	}

	// Verify other config is still in map
	if val, ok := p.Config["some_config"]; !ok || val != "value" {
		t.Errorf("some_config = %v, want 'value'", val)
	}
}

// TestDiscoverProviders_WithoutVersion tests that unversioned sources work correctly.
func TestDiscoverProviders_WithoutVersion(t *testing.T) {
	// Arrange: Create temp config file without version
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.csl")

	configContent := `source:
  alias: 'legacy'
  type: 'owner/repo'
  path: './data'
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act: Discover providers
	providers, err := DiscoverProviders([]string{configPath})
	if err != nil {
		t.Fatalf("DiscoverProviders failed: %v", err)
	}

	// Assert: Provider discovered with empty version
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}

	p := providers[0]
	if p.Version != "" {
		t.Errorf("version = %q, want empty string", p.Version)
	}
}

// TestDiscoverProviders_MultipleProviders tests discovering multiple providers.
func TestDiscoverProviders_MultipleProviders(t *testing.T) {
	// Arrange: Create temp config file with multiple sources
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.csl")

	configContent := `source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './data'

source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
  backend_type: 'azurerm'
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act: Discover providers
	providers, err := DiscoverProviders([]string{configPath})
	if err != nil {
		t.Fatalf("DiscoverProviders failed: %v", err)
	}

	// Assert: Both providers discovered
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	// Verify first provider
	if providers[0].Alias != "configs" {
		t.Errorf("provider[0].alias = %q, want %q", providers[0].Alias, "configs")
	}
	if providers[0].Version != "0.1.1" {
		t.Errorf("provider[0].version = %q, want %q", providers[0].Version, "0.1.1")
	}

	// Verify second provider
	if providers[1].Alias != "tfstate" {
		t.Errorf("provider[1].alias = %q, want %q", providers[1].Alias, "tfstate")
	}
	if providers[1].Version != "0.1.2" {
		t.Errorf("provider[1].version = %q, want %q", providers[1].Version, "0.1.2")
	}
}

// TestDiscoverProviders_SkipsDuplicates tests that duplicate aliases are skipped.
func TestDiscoverProviders_SkipsDuplicates(t *testing.T) {
	// Arrange: Create temp config file with duplicate alias
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.csl")

	configContent := `source:
  alias: 'provider'
  type: 'owner/repo1'
  version: '1.0.0'

source:
  alias: 'provider'
  type: 'owner/repo2'
  version: '2.0.0'
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act: Discover providers
	providers, err := DiscoverProviders([]string{configPath})
	if err != nil {
		t.Fatalf("DiscoverProviders failed: %v", err)
	}

	// Assert: Only first provider kept (duplicates skipped)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider (duplicate skipped), got %d", len(providers))
	}

	p := providers[0]
	if p.Type != "owner/repo1" {
		t.Errorf("type = %q, want %q (first occurrence)", p.Type, "owner/repo1")
	}
	if p.Version != "1.0.0" {
		t.Errorf("version = %q, want %q (first occurrence)", p.Version, "1.0.0")
	}
}

// TestDiscoverProviders_InvalidFile tests error handling for non-existent files.
func TestDiscoverProviders_InvalidFile(t *testing.T) {
	// Act: Try to discover from non-existent file
	_, err := DiscoverProviders([]string{"nonexistent.csl"})

	// Assert: Error returned
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// TestDiscoverProviders_ParseError tests error handling for invalid syntax.
func TestDiscoverProviders_ParseError(t *testing.T) {
	// Arrange: Create temp file with invalid syntax
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.csl")

	configContent := `source invalid syntax !!! this should fail`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act: Try to discover from invalid file
	_, err := DiscoverProviders([]string{configPath})

	// Assert: Error returned
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

// TestDiscoverProviders_MultipleFiles tests discovering from multiple files.
func TestDiscoverProviders_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create first config file
	config1Path := filepath.Join(tempDir, "config1.csl")
	config1Content := `source:
  alias: 'file'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
`
	if err := os.WriteFile(config1Path, []byte(config1Content), 0600); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}

	// Create second config file
	config2Path := filepath.Join(tempDir, "config2.csl")
	config2Content := `source:
  alias: 'tfstate'
  type: 'autonomous-bits/nomos-provider-terraform-remote-state'
  version: '0.1.2'
`
	if err := os.WriteFile(config2Path, []byte(config2Content), 0600); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	// Act: Discover from multiple files
	providers, err := DiscoverProviders([]string{config1Path, config2Path})
	if err != nil {
		t.Fatalf("DiscoverProviders failed: %v", err)
	}

	// Assert: Both providers discovered
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	aliases := []string{providers[0].Alias, providers[1].Alias}
	if !containsString(aliases, "file") {
		t.Error("'file' provider not found")
	}
	if !containsString(aliases, "tfstate") {
		t.Error("'tfstate' provider not found")
	}
}

// TestDiscoverProviders_EmptyFile tests discovering from empty file.
func TestDiscoverProviders_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.csl")

	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	providers, err := DiscoverProviders([]string{configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

// Helper function
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
