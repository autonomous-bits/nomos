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

// TestDiscoverCslFiles_SingleFile tests single file path discovery.
func TestDiscoverCslFiles_SingleFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a single .csl file
	configPath := filepath.Join(tempDir, "config.csl")
	if err := os.WriteFile(configPath, []byte("# test"), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Act
	files, err := discoverCslFiles(configPath)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Should return absolute path
	expectedPath, _ := filepath.Abs(configPath)
	if files[0] != expectedPath {
		t.Errorf("got path %q, want %q", files[0], expectedPath)
	}
}

// TestDiscoverCslFiles_DirectoryWithMultipleCslFiles tests directory with multiple .csl files.
func TestDiscoverCslFiles_DirectoryWithMultipleCslFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple .csl files with names that test lexicographic sorting
	files := []string{"z.csl", "a.csl", "config.csl", "main.csl"}
	for _, f := range files {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("# test"), 0600); err != nil {
			t.Fatalf("failed to write file %s: %v", f, err)
		}
	}

	// Act
	discovered, err := discoverCslFiles(tempDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(discovered) != 4 {
		t.Fatalf("expected 4 files, got %d", len(discovered))
	}

	// Verify lexicographic sorting
	expectedOrder := []string{"a.csl", "config.csl", "main.csl", "z.csl"}
	for i, expectedName := range expectedOrder {
		actualName := filepath.Base(discovered[i])
		if actualName != expectedName {
			t.Errorf("file[%d] = %s, want %s", i, actualName, expectedName)
		}
	}

	// Verify absolute paths
	for _, file := range discovered {
		if !filepath.IsAbs(file) {
			t.Errorf("expected absolute path, got %q", file)
		}
	}
}

// TestDiscoverCslFiles_DirectoryWithNonCslFiles tests that non-.csl files are skipped.
func TestDiscoverCslFiles_DirectoryWithNonCslFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create mix of .csl and non-.csl files
	filesToCreate := map[string]string{
		"valid.csl":   "# csl file",
		"readme.md":   "# readme",
		"config.yaml": "key: value",
		"another.csl": "# another csl",
		"test.txt":    "text",
	}

	for name, content := range filesToCreate {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write file %s: %v", name, err)
		}
	}

	// Act
	discovered, err := discoverCslFiles(tempDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only find the .csl files
	if len(discovered) != 2 {
		t.Fatalf("expected 2 .csl files, got %d", len(discovered))
	}

	// Verify only .csl files returned
	for _, file := range discovered {
		baseName := filepath.Base(file)
		if baseName != "another.csl" && baseName != "valid.csl" {
			t.Errorf("unexpected file: %s", baseName)
		}
	}
}

// TestDiscoverCslFiles_EmptyDirectory tests that empty directory returns empty list.
func TestDiscoverCslFiles_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Act: Discover in empty directory
	discovered, err := discoverCslFiles(tempDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(discovered) != 0 {
		t.Errorf("expected empty list, got %d files", len(discovered))
	}
}

// TestDiscoverCslFiles_DirectoryWithSubdirectories tests that subdirectories are ignored.
func TestDiscoverCslFiles_DirectoryWithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create files and subdirectories
	if err := os.WriteFile(filepath.Join(tempDir, "root.csl"), []byte("# root"), 0600); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}

	// Create subdirectory with .csl file (should be ignored - non-recursive)
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.csl"), []byte("# sub"), 0600); err != nil {
		t.Fatalf("failed to write sub file: %v", err)
	}

	// Act
	discovered, err := discoverCslFiles(tempDir)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only find root.csl (non-recursive)
	if len(discovered) != 1 {
		t.Fatalf("expected 1 file (non-recursive), got %d", len(discovered))
	}
	if filepath.Base(discovered[0]) != "root.csl" {
		t.Errorf("expected root.csl, got %s", filepath.Base(discovered[0]))
	}
}

// TestDiscoverCslFiles_NonExistentPath tests error for non-existent path.
func TestDiscoverCslFiles_NonExistentPath(t *testing.T) {
	// Act
	_, err := discoverCslFiles("/nonexistent/path/to/file.csl")

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

// TestDiscoverCslFiles_NonCslFile tests error for non-.csl file.
func TestDiscoverCslFiles_NonCslFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create non-.csl file
	txtPath := filepath.Join(tempDir, "config.txt")
	if err := os.WriteFile(txtPath, []byte("text"), 0600); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Act
	_, err := discoverCslFiles(txtPath)

	// Assert
	if err == nil {
		t.Fatal("expected error for non-.csl file, got nil")
	}
}

// TestDiscoverCslFiles_TableDriven tests various scenarios using table-driven pattern.
func TestDiscoverCslFiles_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) string // Returns path to test
		wantCount    int
		wantErr      bool
		wantBasename []string // Expected basenames in sorted order
	}{
		{
			name: "single .csl file",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "single.csl")
				writeTestFile(t, path, "# test")
				return path
			},
			wantCount:    1,
			wantErr:      false,
			wantBasename: []string{"single.csl"},
		},
		{
			name: "directory with sorted files",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeTestFile(t, filepath.Join(dir, "c.csl"), "# c")
				writeTestFile(t, filepath.Join(dir, "a.csl"), "# a")
				writeTestFile(t, filepath.Join(dir, "b.csl"), "# b")
				return dir
			},
			wantCount:    3,
			wantErr:      false,
			wantBasename: []string{"a.csl", "b.csl", "c.csl"},
		},
		{
			name: "empty directory",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantCount:    0,
			wantErr:      false,
			wantBasename: []string{},
		},
		{
			name: "directory with mixed files",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeTestFile(t, filepath.Join(dir, "valid.csl"), "# csl")
				writeTestFile(t, filepath.Join(dir, "readme.md"), "# md")
				writeTestFile(t, filepath.Join(dir, "data.json"), "{}")
				return dir
			},
			wantCount:    1,
			wantErr:      false,
			wantBasename: []string{"valid.csl"},
		},
		{
			name: "non-existent file",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				return "/nonexistent/file.csl"
			},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "non-.csl file",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				path := filepath.Join(dir, "file.txt")
				writeTestFile(t, path, "text")
				return path
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := tt.setupFunc(t)

			// Act
			files, err := discoverCslFiles(path)

			// Assert error
			if (err != nil) != tt.wantErr {
				t.Errorf("discoverCslFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Don't check files if error expected
			}

			// Assert count
			if len(files) != tt.wantCount {
				t.Errorf("got %d files, want %d", len(files), tt.wantCount)
			}

			// Assert basenames and order
			for i, wantBase := range tt.wantBasename {
				if i >= len(files) {
					t.Errorf("missing file at index %d", i)
					continue
				}
				gotBase := filepath.Base(files[i])
				if gotBase != wantBase {
					t.Errorf("file[%d] basename = %s, want %s", i, gotBase, wantBase)
				}
			}

			// Assert all paths are absolute
			for _, file := range files {
				if !filepath.IsAbs(file) {
					t.Errorf("expected absolute path, got %q", file)
				}
			}
		})
	}
}

// TestDiscoverProviders_MixedFileAndDirectoryPaths tests discovering from mixed paths.
func TestDiscoverProviders_MixedFileAndDirectoryPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create standalone file
	standalonePath := filepath.Join(tempDir, "standalone.csl")
	standaloneContent := `source:
  alias: 'standalone'
  type: 'owner/repo'
  version: '1.0.0'
`
	writeTestFile(t, standalonePath, standaloneContent)

	// Create directory with multiple files
	subDir := filepath.Join(tempDir, "configs")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	config1Content := `source:
  alias: 'file1'
  type: 'owner/repo1'
  version: '1.1.0'
`
	writeTestFile(t, filepath.Join(subDir, "config1.csl"), config1Content)

	config2Content := `source:
  alias: 'file2'
  type: 'owner/repo2'
  version: '1.2.0'
`
	writeTestFile(t, filepath.Join(subDir, "config2.csl"), config2Content)

	// Act: Discover from both standalone file and directory
	providers, err := DiscoverProviders([]string{standalonePath, subDir})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find all 3 providers
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}

	// Verify all providers discovered
	aliases := make(map[string]bool)
	for _, p := range providers {
		aliases[p.Alias] = true
	}

	expectedAliases := []string{"standalone", "file1", "file2"}
	for _, alias := range expectedAliases {
		if !aliases[alias] {
			t.Errorf("provider with alias %q not found", alias)
		}
	}
}

// TestDiscoverProviders_DirectoryWithMultipleProviders tests directory discovery.
func TestDiscoverProviders_DirectoryWithMultipleProviders(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple config files in directory
	configs := map[string]string{
		"a.csl": `source:
  alias: 'provider-a'
  type: 'owner/repo-a'
  version: '1.0.0'
`,
		"b.csl": `source:
  alias: 'provider-b'
  type: 'owner/repo-b'
  version: '2.0.0'
`,
		"c.csl": `source:
  alias: 'provider-c'
  type: 'owner/repo-c'
  version: '3.0.0'
`,
	}

	for name, content := range configs {
		writeTestFile(t, filepath.Join(tempDir, name), content)
	}

	// Act: Discover from directory
	providers, err := DiscoverProviders([]string{tempDir})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}

	// Verify files processed in alphabetical order (deterministic)
	expectedOrder := []string{"provider-a", "provider-b", "provider-c"}
	for i, expectedAlias := range expectedOrder {
		if providers[i].Alias != expectedAlias {
			t.Errorf("provider[%d].alias = %s, want %s", i, providers[i].Alias, expectedAlias)
		}
	}
}

// TestDiscoverProviders_DirectoryWithNonCslFiles tests that non-.csl files are ignored.
func TestDiscoverProviders_DirectoryWithNonCslFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid .csl file
	validContent := `source:
  alias: 'valid'
  type: 'owner/repo'
  version: '1.0.0'
`
	writeTestFile(t, filepath.Join(tempDir, "valid.csl"), validContent)

	// Create non-.csl files that should be ignored
	writeTestFile(t, filepath.Join(tempDir, "README.md"), "# Documentation")
	writeTestFile(t, filepath.Join(tempDir, "config.yaml"), "key: value")
	writeTestFile(t, filepath.Join(tempDir, ".gitignore"), "*.log")

	// Act
	providers, err := DiscoverProviders([]string{tempDir})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only find the one .csl file
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}

	if providers[0].Alias != "valid" {
		t.Errorf("alias = %s, want 'valid'", providers[0].Alias)
	}
}

// TestDiscoverProviders_EmptyDirectory tests that empty directory returns no providers.
func TestDiscoverProviders_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Act: Discover from empty directory
	providers, err := DiscoverProviders([]string{tempDir})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

// TestDiscoverProviders_NonExistentDirectory tests error for non-existent directory.
func TestDiscoverProviders_NonExistentDirectory(t *testing.T) {
	// Act
	_, err := DiscoverProviders([]string{"/nonexistent/directory"})

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// TestDiscoverProviders_NonCslFilePath tests error for non-.csl file path.
func TestDiscoverProviders_NonCslFilePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create non-.csl file
	txtPath := filepath.Join(tempDir, "config.txt")
	writeTestFile(t, txtPath, "some content")

	// Act
	_, err := DiscoverProviders([]string{txtPath})

	// Assert
	if err == nil {
		t.Fatal("expected error for non-.csl file, got nil")
	}
}

// Helper function: write test file with proper error handling and t.Helper()
func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
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
