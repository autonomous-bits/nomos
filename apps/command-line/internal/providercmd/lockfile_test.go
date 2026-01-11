package providercmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadLockFile tests reading the lockfile from disk.
func TestReadLockFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string // Returns temp dir
		wantErr     bool
		errContains string
		validate    func(t *testing.T, lock *LockFile)
	}{
		{
			name: "valid lockfile",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				lockDir := filepath.Join(tmpDir, ".nomos")
				if err := os.MkdirAll(lockDir, 0750); err != nil {
					t.Fatalf("failed to create .nomos dir: %v", err)
				}

				lock := LockFile{
					Timestamp: "2026-01-10T10:00:00Z",
					Providers: []ProviderEntry{
						{
							Alias:    "aws",
							Type:     "autonomous-bits/nomos-provider-aws",
							Version:  "1.0.0",
							OS:       "darwin",
							Arch:     "arm64",
							Checksum: "abc123",
							Size:     1024,
							Path:     "autonomous-bits/nomos-provider-aws/1.0.0/darwin-arm64/provider",
						},
					},
				}

				data, _ := json.MarshalIndent(lock, "", "  ")
				lockPath := filepath.Join(lockDir, "providers.lock.json")
				if err := os.WriteFile(lockPath, data, 0600); err != nil {
					t.Fatalf("failed to write lockfile: %v", err)
				}

				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, lock *LockFile) {
				t.Helper()
				if lock == nil {
					t.Fatal("expected lockfile, got nil")
				}
				if len(lock.Providers) != 1 {
					t.Errorf("got %d providers, want 1", len(lock.Providers))
				}
				if lock.Providers[0].Alias != "aws" {
					t.Errorf("alias = %q, want %q", lock.Providers[0].Alias, "aws")
				}
			},
		},
		{
			name: "lockfile doesn't exist",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr:     true,
			errContains: "failed to read lockfile",
		},
		{
			name: "invalid json",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				lockDir := filepath.Join(tmpDir, ".nomos")
				if err := os.MkdirAll(lockDir, 0750); err != nil {
					t.Fatalf("failed to create .nomos dir: %v", err)
				}

				lockPath := filepath.Join(lockDir, "providers.lock.json")
				if err := os.WriteFile(lockPath, []byte("invalid json {{{"), 0600); err != nil {
					t.Fatalf("failed to write invalid lockfile: %v", err)
				}

				return tmpDir
			},
			wantErr:     true,
			errContains: "failed to parse lockfile JSON",
		},
		{
			name: "empty providers array",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				lockDir := filepath.Join(tmpDir, ".nomos")
				if err := os.MkdirAll(lockDir, 0750); err != nil {
					t.Fatalf("failed to create .nomos dir: %v", err)
				}

				lock := LockFile{
					Timestamp: "2026-01-10T10:00:00Z",
					Providers: []ProviderEntry{},
				}

				data, _ := json.MarshalIndent(lock, "", "  ")
				lockPath := filepath.Join(lockDir, "providers.lock.json")
				if err := os.WriteFile(lockPath, data, 0600); err != nil {
					t.Fatalf("failed to write lockfile: %v", err)
				}

				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, lock *LockFile) {
				t.Helper()
				if lock == nil {
					t.Fatal("expected lockfile, got nil")
				}
				if len(lock.Providers) != 0 {
					t.Errorf("got %d providers, want 0", len(lock.Providers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tt.setupFunc(t)
			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to chdir: %v", err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			got, err := ReadLockFile()

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadLockFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

// TestWriteLockFile tests writing the lockfile to disk atomically.
func TestWriteLockFile(t *testing.T) {
	tests := []struct {
		name     string
		lock     LockFile
		wantErr  bool
		validate func(t *testing.T, tmpDir string)
	}{
		{
			name: "write new lockfile",
			lock: LockFile{
				Timestamp: "", // Should be auto-set
				Providers: []ProviderEntry{
					{
						Alias:    "file",
						Type:     "autonomous-bits/nomos-provider-file",
						Version:  "0.1.1",
						OS:       "linux",
						Arch:     "amd64",
						Checksum: "def456",
						Size:     2048,
						Path:     "autonomous-bits/nomos-provider-file/0.1.1/linux-amd64/provider",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string) {
				t.Helper()
				lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")

				// Verify file exists
				if _, err := os.Stat(lockPath); err != nil {
					t.Fatalf("lockfile not created: %v", err)
				}

				// Read and verify content
				data, err := os.ReadFile(lockPath) //nolint:gosec // G304: Reading from controlled test temp directory //nolint:gosec // G304: Reading from controlled test temp directory
				if err != nil {
					t.Fatalf("failed to read lockfile: %v", err)
				}

				var lock LockFile
				if err := json.Unmarshal(data, &lock); err != nil {
					t.Fatalf("failed to parse lockfile: %v", err)
				}

				if lock.Timestamp == "" {
					t.Error("timestamp not set")
				}

				if len(lock.Providers) != 1 {
					t.Errorf("got %d providers, want 1", len(lock.Providers))
				}
			},
		},
		{
			name: "overwrite existing lockfile",
			lock: LockFile{
				Timestamp: "2026-01-10T12:00:00Z",
				Providers: []ProviderEntry{
					{
						Alias:   "new",
						Type:    "owner/repo",
						Version: "2.0.0",
						OS:      "darwin",
						Arch:    "arm64",
						Path:    "owner/repo/2.0.0/darwin-arm64/provider",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string) {
				t.Helper()
				lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")

				data, err := os.ReadFile(lockPath) //nolint:gosec // G304: Reading from controlled test temp directory //nolint:gosec // G304: Reading from controlled test temp directory
				if err != nil {
					t.Fatalf("failed to read lockfile: %v", err)
				}

				var lock LockFile
				if err := json.Unmarshal(data, &lock); err != nil {
					t.Fatalf("failed to parse lockfile: %v", err)
				}

				if lock.Providers[0].Alias != "new" {
					t.Errorf("alias = %q, want %q", lock.Providers[0].Alias, "new")
				}
			},
		},
		{
			name: "empty providers array",
			lock: LockFile{
				Timestamp: "2026-01-10T12:00:00Z",
				Providers: []ProviderEntry{},
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string) {
				t.Helper()
				lockPath := filepath.Join(tmpDir, ".nomos", "providers.lock.json")

				data, err := os.ReadFile(lockPath) //nolint:gosec // G304: Reading from controlled test temp directory
				if err != nil {
					t.Fatalf("failed to read lockfile: %v", err)
				}

				var lock LockFile
				if err := json.Unmarshal(data, &lock); err != nil {
					t.Fatalf("failed to parse lockfile: %v", err)
				}

				if len(lock.Providers) != 0 {
					t.Errorf("got %d providers, want 0", len(lock.Providers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to chdir: %v", err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			err := WriteLockFile(tt.lock)

			if (err != nil) != tt.wantErr {
				t.Errorf("WriteLockFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, tmpDir)
			}
		})
	}
}

// TestWriteLockFile_Atomic tests that WriteLockFile uses atomic write pattern.
func TestWriteLockFile_Atomic(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Write initial lockfile
	lock1 := LockFile{
		Timestamp: "2026-01-10T10:00:00Z",
		Providers: []ProviderEntry{
			{Alias: "provider1", Type: "owner/repo1", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
		},
	}
	if err := WriteLockFile(lock1); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	// Overwrite with new lockfile
	lock2 := LockFile{
		Timestamp: "2026-01-10T11:00:00Z",
		Providers: []ProviderEntry{
			{Alias: "provider2", Type: "owner/repo2", Version: "2.0.0", OS: "darwin", Arch: "arm64", Path: "path2"},
		},
	}
	if err := WriteLockFile(lock2); err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	// Verify no temp files left behind
	lockDir := filepath.Join(tmpDir, ".nomos")
	entries, err := os.ReadDir(lockDir)
	if err != nil {
		t.Fatalf("failed to read lock dir: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() != "providers.lock.json" {
			t.Errorf("unexpected file in lock dir: %s", entry.Name())
		}
	}

	// Verify final content is lock2
	lockPath := filepath.Join(lockDir, "providers.lock.json")
	data, err := os.ReadFile(lockPath) //nolint:gosec // G304: Reading from controlled test temp directory //nolint:gosec // G304: Reading from controlled test temp directory
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}

	var finalLock LockFile
	if err := json.Unmarshal(data, &finalLock); err != nil {
		t.Fatalf("failed to parse lockfile: %v", err)
	}

	if finalLock.Providers[0].Alias != "provider2" {
		t.Errorf("lockfile not overwritten correctly, got alias %q", finalLock.Providers[0].Alias)
	}
}

// TestMergeLockFiles tests merging existing and new provider entries.
func TestMergeLockFiles(t *testing.T) {
	tests := []struct {
		name       string
		existing   *LockFile
		newEntries []ProviderEntry
		wantCount  int
		validate   func(t *testing.T, merged LockFile)
	}{
		{
			name:     "merge with nil existing",
			existing: nil,
			newEntries: []ProviderEntry{
				{Alias: "new1", Type: "owner/repo1", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
			},
			wantCount: 1,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				if merged.Providers[0].Alias != "new1" {
					t.Errorf("alias = %q, want %q", merged.Providers[0].Alias, "new1")
				}
			},
		},
		{
			name: "merge with empty existing",
			existing: &LockFile{
				Timestamp: "2026-01-10T10:00:00Z",
				Providers: []ProviderEntry{},
			},
			newEntries: []ProviderEntry{
				{Alias: "new1", Type: "owner/repo1", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
			},
			wantCount: 1,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				if merged.Providers[0].Alias != "new1" {
					t.Errorf("alias = %q, want %q", merged.Providers[0].Alias, "new1")
				}
			},
		},
		{
			name: "append new entry",
			existing: &LockFile{
				Timestamp: "2026-01-10T10:00:00Z",
				Providers: []ProviderEntry{
					{Alias: "existing", Type: "owner/repo1", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
				},
			},
			newEntries: []ProviderEntry{
				{Alias: "new", Type: "owner/repo2", Version: "2.0.0", OS: "darwin", Arch: "arm64", Path: "path2"},
			},
			wantCount: 2,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				aliases := []string{merged.Providers[0].Alias, merged.Providers[1].Alias}
				if !containsString(aliases, "existing") {
					t.Error("existing provider not preserved")
				}
				if !containsString(aliases, "new") {
					t.Error("new provider not added")
				}
			},
		},
		{
			name: "update existing entry",
			existing: &LockFile{
				Timestamp: "2026-01-10T10:00:00Z",
				Providers: []ProviderEntry{
					{Alias: "aws", Type: "owner/repo", Version: "1.0.0", OS: "linux", Arch: "amd64", Checksum: "old", Path: "path1"},
				},
			},
			newEntries: []ProviderEntry{
				{Alias: "aws", Type: "owner/repo", Version: "1.0.0", OS: "linux", Arch: "amd64", Checksum: "new", Path: "path1"},
			},
			wantCount: 1,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				if merged.Providers[0].Checksum != "new" {
					t.Errorf("checksum = %q, want %q (should be updated)", merged.Providers[0].Checksum, "new")
				}
			},
		},
		{
			name: "different os/arch creates separate entry",
			existing: &LockFile{
				Timestamp: "2026-01-10T10:00:00Z",
				Providers: []ProviderEntry{
					{Alias: "aws", Type: "owner/repo", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
				},
			},
			newEntries: []ProviderEntry{
				{Alias: "aws", Type: "owner/repo", Version: "1.0.0", OS: "darwin", Arch: "arm64", Path: "path2"},
			},
			wantCount: 2,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				osArch := []string{
					merged.Providers[0].OS + "-" + merged.Providers[0].Arch,
					merged.Providers[1].OS + "-" + merged.Providers[1].Arch,
				}
				if !containsString(osArch, "linux-amd64") {
					t.Error("linux-amd64 entry not preserved")
				}
				if !containsString(osArch, "darwin-arm64") {
					t.Error("darwin-arm64 entry not added")
				}
			},
		},
		{
			name: "merge sets fresh timestamp",
			existing: &LockFile{
				Timestamp: "2026-01-10T10:00:00Z",
				Providers: []ProviderEntry{},
			},
			newEntries: []ProviderEntry{
				{Alias: "new", Type: "owner/repo", Version: "1.0.0", OS: "linux", Arch: "amd64", Path: "path1"},
			},
			wantCount: 1,
			validate: func(t *testing.T, merged LockFile) {
				t.Helper()
				if merged.Timestamp == "2026-01-10T10:00:00Z" {
					t.Error("timestamp not updated (should be fresh)")
				}
				// Verify it's a valid RFC3339 timestamp
				if _, err := time.Parse(time.RFC3339, merged.Timestamp); err != nil {
					t.Errorf("invalid timestamp format: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := MergeLockFiles(tt.existing, tt.newEntries)

			if len(merged.Providers) != tt.wantCount {
				t.Errorf("got %d providers, want %d", len(merged.Providers), tt.wantCount)
			}

			if tt.validate != nil {
				tt.validate(t, merged)
			}
		})
	}
}

// Helper function to check if error message contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
