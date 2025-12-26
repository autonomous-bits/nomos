package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildCLI builds the nomos CLI binary and returns its path.
// This helper is shared across unit and integration tests.
func buildCLI(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "nomos")

	// Build from the apps/command-line directory
	//nolint:gosec,noctx // G204: Test helper, controlled input; context not needed for build
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/nomos")
	cmd.Dir = ".." // One level up from test/ to apps/command-line/
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\noutput: %s", err, output)
	}

	return binPath
}

// runCommand runs a command and returns stdout, stderr, and exit code.
// This helper is shared across unit and integration tests.
func runCommand(t *testing.T, cmd *exec.Cmd) (stdout, stderr string, exitCode int) {
	t.Helper()

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running command: %v", err)
		}
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode
}

// createBasicFixture creates a minimal valid .csl fixture and returns its path.
// This helper is shared across unit and integration tests.
//
//nolint:unused // Helper function for tests, may be used in future
func createBasicFixture(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")

	// Create a simple valid Nomos configuration using correct syntax
	content := `config:
	name: "test"
	value: 42
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	return fixturePath
}
