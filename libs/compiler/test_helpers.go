package compiler

import (
	"os"
)

// writeFile is a shared helper that writes content to a file.
// Used by both unit and integration tests.
//
//nolint:unused // Shared helper function across test files with different build tags
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}
