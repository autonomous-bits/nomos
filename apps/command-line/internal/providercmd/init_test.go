package providercmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLockfileEntry_IncludesAllGitHubMetadata tests that lockfile entries
// contain all required GitHub metadata fields including size.
// RED: This test will fail until we add Size field to ProviderEntry.
func TestLockfileEntry_IncludesAllGitHubMetadata(t *testing.T) {
	// Arrange: Create a lock entry as would be generated from InstallResult
	entry := ProviderEntry{
		Alias:    "configs",
		Type:     "file",
		Version:  "1.0.0",
		OS:       "darwin",
		Arch:     "arm64",
		Checksum: "sha256:abc123def456",
		Path:     ".nomos/providers/autonomous-bits/nomos-provider-file/1.0.0/darwin-arm64/provider",
		Source: map[string]interface{}{
			"github": map[string]interface{}{
				"owner":       "autonomous-bits",
				"repo":        "nomos-provider-file",
				"release_tag": "v1.0.0",
				"asset":       "nomos-provider-file-darwin-arm64",
			},
		},
		Size: 2048576, // 2MB binary
	}

	// Act: Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal entry: %v", err)
	}

	// Assert: Unmarshal and verify all fields present
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify GitHub metadata
	source, ok := parsed["source"].(map[string]interface{})
	if !ok {
		t.Fatal("source field missing or wrong type")
	}

	github, ok := source["github"].(map[string]interface{})
	if !ok {
		t.Fatal("source.github field missing or wrong type")
	}

	requiredFields := map[string]string{
		"owner":       "autonomous-bits",
		"repo":        "nomos-provider-file",
		"release_tag": "v1.0.0",
		"asset":       "nomos-provider-file-darwin-arm64",
	}

	for field, expected := range requiredFields {
		actual, ok := github[field].(string)
		if !ok {
			t.Errorf("github.%s missing or wrong type", field)
			continue
		}
		if actual != expected {
			t.Errorf("github.%s = %q, want %q", field, actual, expected)
		}
	}

	// Verify size field exists
	size, ok := parsed["size"].(float64)
	if !ok {
		t.Fatal("size field missing or wrong type")
	}
	if int64(size) != 2048576 {
		t.Errorf("size = %d, want %d", int64(size), 2048576)
	}

	// Verify checksum
	checksum, ok := parsed["checksum"].(string)
	if !ok {
		t.Fatal("checksum field missing")
	}
	if checksum != "sha256:abc123def456" {
		t.Errorf("checksum = %q, want %q", checksum, "sha256:abc123def456")
	}
}

// TestLockFile_IncludesTimestamp tests that the lockfile structure
// includes a timestamp field.
// RED: This test will fail until we add Timestamp field to LockFile.
func TestLockFile_IncludesTimestamp(t *testing.T) {
	// Arrange: Create lockfile with timestamp
	lockFile := LockFile{
		Timestamp: "2025-11-02T10:30:00Z",
		Providers: []ProviderEntry{
			{
				Alias:   "configs",
				Type:    "file",
				Version: "1.0.0",
				OS:      "darwin",
				Arch:    "arm64",
				Path:    ".nomos/providers/autonomous-bits/nomos-provider-file/1.0.0/darwin-arm64/provider",
			},
		},
	}

	// Act: Marshal to JSON
	data, err := json.Marshal(lockFile)
	if err != nil {
		t.Fatalf("failed to marshal lockfile: %v", err)
	}

	// Assert: Verify timestamp in JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	timestamp, ok := parsed["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp field missing or wrong type")
	}

	if timestamp != "2025-11-02T10:30:00Z" {
		t.Errorf("timestamp = %q, want %q", timestamp, "2025-11-02T10:30:00Z")
	}
}

// TestWriteLockFile_AtomicWrite tests that lockfile writes are atomic
// using temp file + rename pattern.
// RED: This test will fail until we implement atomic writes.
func TestWriteLockFile_AtomicWrite(t *testing.T) {
	// Arrange: Setup temp directory
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	lockFile := LockFile{
		Providers: []ProviderEntry{
			{
				Alias:   "test",
				Type:    "file",
				Version: "1.0.0",
				OS:      "linux",
				Arch:    "amd64",
				Path:    ".nomos/providers/file/1.0.0/linux-amd64/provider",
			},
		},
	}

	// Act: Write lockfile
	err := writeLockFile(lockFile)
	if err != nil {
		t.Fatalf("writeLockFile failed: %v", err)
	}

	// Assert: Verify no temp files left behind
	nomosDir := filepath.Join(tempDir, ".nomos")
	entries, err := os.ReadDir(nomosDir)
	if err != nil {
		t.Fatalf("failed to read .nomos dir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Check for common temp file patterns
		if filepath.Ext(name) == ".tmp" || name[0] == '.' && name != "." && name != ".." {
			t.Errorf("temp file left behind: %s", name)
		}
	}

	// Verify lockfile exists and is readable
	lockPath := filepath.Join(nomosDir, "providers.lock.json")
	//nolint:gosec // G304: Test code reading test-generated lockfile
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var parsed LockFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse lockfile: %v", err)
	}

	if len(parsed.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(parsed.Providers))
	}
}

// TestReadLockFile_SkipsIdenticalProvider tests that when a lockfile
// entry matches a provider to install, we skip the download.
// RED: This test will fail until we implement skip logic.
func TestReadLockFile_SkipsIdenticalProvider(t *testing.T) {
	// Arrange: Setup temp directory with existing lockfile
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Create existing lockfile
	existingLock := LockFile{
		Providers: []ProviderEntry{
			{
				Alias:    "configs",
				Type:     "autonomous-bits/nomos-provider-file",
				Version:  "1.0.0",
				OS:       "darwin",
				Arch:     "arm64",
				Checksum: "sha256:existing123",
				Path:     ".nomos/providers/autonomous-bits/nomos-provider-file/1.0.0/darwin-arm64/provider",
			},
		},
	}

	if err := writeLockFile(existingLock); err != nil {
		t.Fatalf("failed to write initial lockfile: %v", err)
	}

	// Act: Try to install same provider (should skip)
	// This is a placeholder - actual implementation will need to integrate
	// with discoverProviders and Run() logic.
	// For now, we just test that reading the lockfile works.

	lockPath := filepath.Join(".nomos", "providers.lock.json")
	//nolint:gosec // G304: Test code reading test-generated lockfile
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var readLock LockFile
	if err := json.Unmarshal(data, &readLock); err != nil {
		t.Fatalf("failed to unmarshal lockfile: %v", err)
	}

	// Assert: Lockfile can be read and contains expected entry
	if len(readLock.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(readLock.Providers))
	}

	provider := readLock.Providers[0]
	if provider.Alias != "configs" {
		t.Errorf("alias = %q, want %q", provider.Alias, "configs")
	}
	if provider.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", provider.Version, "1.0.0")
	}

	// TODO: Extend this test to verify Run() skips installation when
	// provider already exists in lockfile with matching version/checksum.
}
