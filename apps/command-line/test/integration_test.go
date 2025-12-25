package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildCommand_Integration tests the nomos build command end-to-end.
func TestBuildCommand_Integration(t *testing.T) {
	// Build the CLI binary first
	binPath := buildCLI(t)
	defer func() { _ = os.Remove(binPath) }()

	tests := []struct {
		name           string
		args           []string
		setupFixture   func(t *testing.T) string
		wantExitCode   int
		wantStdoutCont string
		wantStderrCont string
	}{
		{
			name:           "basic invocation with valid fixture",
			args:           []string{"build", "-p", "testdata/fixture-basic/test.csl", "-f", "json"},
			setupFixture:   createBasicFixture,
			wantExitCode:   0,
			wantStdoutCont: "config", // Should output JSON with config section
		},
		{
			name: "help flag shows usage",
			args: []string{"build", "--help"},
			setupFixture: func(_ *testing.T) string {
				return "" // No fixture needed
			},
			wantExitCode:   0,
			wantStdoutCont: "Compile Nomos .csl files",
		},
		{
			name: "missing path flag returns exit code 2",
			args: []string{"build"},
			setupFixture: func(_ *testing.T) string {
				return "" // No fixture needed
			},
			wantExitCode:   2,
			wantStderrCont: "path is required",
		},
		{
			name:           "invalid format returns exit code 2",
			args:           []string{"build", "-p", "test.csl", "-f", "xml"},
			setupFixture:   createBasicFixture,
			wantExitCode:   2,
			wantStderrCont: "format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fixture if needed
			fixturePath := ""
			if tt.setupFixture != nil {
				fixturePath = tt.setupFixture(t)
				if fixturePath != "" {
					defer func() { _ = os.RemoveAll(filepath.Dir(fixturePath)) }()

					// Replace placeholder path in args
					for i, arg := range tt.args {
						if strings.Contains(arg, "testdata/") || arg == "test.csl" {
							tt.args[i] = fixturePath
						}
					}
				}
			}

			//nolint:gosec,noctx // G204: Test code with controlled binary path and args; context not needed
			// Run the CLI
			cmd := exec.Command(binPath, tt.args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			// Assert exit code
			if exitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
					exitCode, tt.wantExitCode, stdout, stderr)
			}

			// Assert stdout content
			if tt.wantStdoutCont != "" && !strings.Contains(stdout, tt.wantStdoutCont) {
				t.Errorf("stdout does not contain %q\nstdout: %s",
					tt.wantStdoutCont, stdout)
			}

			// Assert stderr content
			if tt.wantStderrCont != "" && !strings.Contains(stderr, tt.wantStderrCont) {
				t.Errorf("stderr does not contain %q\nstderr: %s",
					tt.wantStderrCont, stderr)
			}
		})
	}
}

// buildCLI builds the nomos CLI binary and returns its path.
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
