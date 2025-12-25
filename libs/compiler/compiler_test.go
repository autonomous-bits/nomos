package compiler

import (
	"context"
	"errors"
	"os"
	"testing"
) // mockProviderRegistry is a minimal fake for testing.
type mockProviderRegistry struct {
	aliases []string
}

func (m *mockProviderRegistry) Register(alias string, _ ProviderConstructor) {
	// No-op for basic tests
	m.aliases = append(m.aliases, alias)
}

func (m *mockProviderRegistry) GetProvider(_ context.Context, _ string) (Provider, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProviderRegistry) RegisteredAliases() []string {
	return m.aliases
}

// TestCompile_OptionsValidation tests that Compile validates required options.
func TestCompile_OptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		opts        Options
		expectError string
	}{
		{
			name:        "nil context",
			ctx:         nil,
			opts:        Options{Path: "/some/path", ProviderRegistry: &mockProviderRegistry{}},
			expectError: "context must not be nil",
		},
		{
			name:        "empty Path",
			ctx:         context.Background(),
			opts:        Options{Path: "", ProviderRegistry: &mockProviderRegistry{}},
			expectError: "options.Path must not be empty",
		},
		{
			name:        "nil ProviderRegistry",
			ctx:         context.Background(),
			opts:        Options{Path: "/some/path", ProviderRegistry: nil},
			expectError: "options.ProviderRegistry must not be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.ctx, tt.opts)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.expectError {
				t.Errorf("expected error %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

// TestCompile_DeterministicDirectoryTraversal tests that directory traversal
// occurs in lexicographic order and that InputFiles are populated correctly.
func TestCompile_DeterministicDirectoryTraversal(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create files in non-lexicographic order
	files := []string{"z.csl", "a.csl", "m.csl"}
	for _, f := range files {
		path := tmpDir + "/" + f
		if err := writeFile(path, "# test"); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Compile the directory
	ctx := context.Background()
	opts := Options{
		Path:             tmpDir,
		ProviderRegistry: &mockProviderRegistry{},
	}

	snapshot, err := Compile(ctx, opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify files are in lexicographic order
	expected := []string{tmpDir + "/a.csl", tmpDir + "/m.csl", tmpDir + "/z.csl"}
	if len(snapshot.Metadata.InputFiles) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(snapshot.Metadata.InputFiles))
	}

	for i, expectedPath := range expected {
		if snapshot.Metadata.InputFiles[i] != expectedPath {
			t.Errorf("file[%d]: expected %q, got %q", i, expectedPath, snapshot.Metadata.InputFiles[i])
		}
	}
}

// writeFile is a helper to write content to a file.
func writeFile(path, content string) error {
	file, err := os.Create(path) //nolint:gosec // G304: Path is from test temp directory
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	_, err = file.WriteString(content)
	return err
}
