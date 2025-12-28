//go:build integration
// +build integration

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
			wantStdoutCont: "Build compiles", // Cobra uses "Build compiles" in Long description
		},
		{
			name: "missing path flag returns exit code 2",
			args: []string{"build"},
			setupFixture: func(_ *testing.T) string {
				return "" // No fixture needed
			},
			wantExitCode:   1,               // Cobra returns 1 for all errors
			wantStderrCont: "required flag", // Cobra error message
		},
		{
			name:           "invalid format returns exit code 2",
			args:           []string{"build", "-p", "test.csl", "-f", "xml"},
			setupFixture:   createBasicFixture,
			wantExitCode:   1, // Cobra returns 1 for all errors
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
