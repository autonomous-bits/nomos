//go:build integration
// +build integration

package test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestInitCommand_Migration_Integration tests the migration behavior when users
// attempt to run the removed 'nomos init' command. This verifies that the CLI
// provides clear migration guidance for the v2.0.0 breaking change.
func TestInitCommand_Migration_Integration(t *testing.T) {
	// Build the CLI binary for testing
	binPath := buildCLI(t)
	defer func() { _ = os.Remove(binPath) }()

	tests := []struct {
		name               string
		args               []string
		wantExitCode       int
		wantStderrContains []string
		wantStdoutEmpty    bool
	}{
		{
			name:         "T040: nomos init displays migration message",
			args:         []string{"init"},
			wantExitCode: 1,
			wantStderrContains: []string{
				"Error: The 'init' command has been removed in v2.0.0.",
				"Providers are now automatically downloaded during 'nomos build'.",
				"Migration:",
				"Old: nomos init && nomos build",
				"New: nomos build",
				"For more information, see the migration guide at:",
				"https://github.com/autonomous-bits/nomos/blob/main/docs/guides/migration-v2.md",
			},
			wantStdoutEmpty: true,
		},
		{
			name:         "T042: nomos init --help displays migration message",
			args:         []string{"init", "--help"},
			wantExitCode: 1,
			wantStderrContains: []string{
				"Error: The 'init' command has been removed in v2.0.0.",
				"Providers are now automatically downloaded during 'nomos build'.",
				"Migration:",
				"Old: nomos init && nomos build",
				"New: nomos build",
				"For more information, see the migration guide at:",
				"https://github.com/autonomous-bits/nomos/blob/main/docs/guides/migration-v2.md",
			},
			wantStdoutEmpty: true,
		},
		{
			name:         "nomos init -h also displays migration message",
			args:         []string{"init", "-h"},
			wantExitCode: 1,
			wantStderrContains: []string{
				"Error: The 'init' command has been removed in v2.0.0.",
				"Providers are now automatically downloaded during 'nomos build'.",
			},
			wantStdoutEmpty: true,
		},
		{
			name:         "nomos init with extra args displays migration message",
			args:         []string{"init", "--some-flag"},
			wantExitCode: 1,
			wantStderrContains: []string{
				"Error: The 'init' command has been removed in v2.0.0.",
			},
			wantStdoutEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:gosec,noctx // G204: Test code with controlled binary path and args; context not needed
			cmd := exec.Command(binPath, tt.args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			// Assert exit code
			if exitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
					exitCode, tt.wantExitCode, stdout, stderr)
			}

			// Assert all required content is in stderr
			for _, content := range tt.wantStderrContains {
				if !strings.Contains(stderr, content) {
					t.Errorf("stderr missing required content: %q\nstderr: %s",
						content, stderr)
				}
			}

			// Assert stdout is empty (migration message should be on stderr)
			if tt.wantStdoutEmpty && strings.TrimSpace(stdout) != "" {
				t.Errorf("expected empty stdout, got: %s", stdout)
			}
		})
	}
}

// TestHelpCommand_NoInitCommand_Integration tests that 'nomos --help' does NOT
// list the removed 'init' command in the available commands list.
func TestHelpCommand_NoInitCommand_Integration(t *testing.T) {
	// Build the CLI binary for testing
	binPath := buildCLI(t)
	defer func() { _ = os.Remove(binPath) }()

	t.Run("T041: nomos --help does NOT list init command", func(t *testing.T) {
		//nolint:gosec,noctx // G204: Test code with controlled binary path; context not needed
		cmd := exec.Command(binPath, "--help")
		stdout, stderr, exitCode := runCommand(t, cmd)

		// Should succeed
		if exitCode != 0 {
			t.Fatalf("nomos --help failed with exit code %d\nstdout: %s\nstderr: %s",
				exitCode, stdout, stderr)
		}

		// Combine stdout and stderr for comprehensive check
		helpOutput := stdout + stderr

		// Verify help output contains expected sections
		requiredSections := []string{
			"Available Commands:",
			"build",
			"version",
			"help",
		}

		for _, section := range requiredSections {
			if !strings.Contains(helpOutput, section) {
				t.Errorf("help output missing expected section/command: %q\nOutput: %s",
					section, helpOutput)
			}
		}

		// Verify 'init' is NOT listed as a command
		// Check for common patterns that would indicate init is a command:
		// - "init" followed by whitespace and a description
		// - "init" appearing in a command list context
		lines := strings.Split(helpOutput, "\n")
		for _, line := range lines {
			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Check if this looks like a command list entry with 'init'
			// Command entries typically start with whitespace followed by command name
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "init") && len(trimmed) > 4 {
				// If line starts with "init " (init followed by space/description),
				// that means init is listed as a command
				if trimmed[4] == ' ' || trimmed[4] == '\t' {
					t.Errorf("help output should NOT list 'init' as a command, but found: %q",
						line)
				}
			}
		}

		// Additional check: if help output explicitly lists commands in a structured way,
		// verify init is not among them
		if strings.Contains(helpOutput, "Available Commands:") {
			// Extract section after "Available Commands:"
			parts := strings.Split(helpOutput, "Available Commands:")
			if len(parts) > 1 {
				commandsSection := parts[1]
				// Look for next major section (e.g., "Flags:")
				if idx := strings.Index(commandsSection, "Flags:"); idx != -1 {
					commandsSection = commandsSection[:idx]
				}

				// In the commands section, check that 'init' doesn't appear as a command
				commandLines := strings.Split(commandsSection, "\n")
				for _, line := range commandLines {
					trimmed := strings.TrimSpace(line)
					// If line starts with "init " (command pattern), fail
					if strings.HasPrefix(trimmed, "init ") {
						t.Errorf("'init' command found in Available Commands section: %q",
							line)
					}
				}
			}
		}
	})

	t.Run("nomos help also does not list init command", func(t *testing.T) {
		//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
		cmd := exec.Command(binPath, "help")
		stdout, stderr, exitCode := runCommand(t, cmd)

		if exitCode != 0 {
			t.Fatalf("nomos help failed with exit code %d\nstdout: %s\nstderr: %s",
				exitCode, stdout, stderr)
		}

		helpOutput := stdout + stderr

		// Verify help output contains expected commands
		if !strings.Contains(helpOutput, "build") {
			t.Error("help output should contain 'build' command")
		}

		// Verify 'init' is NOT listed as a command
		lines := strings.Split(helpOutput, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "init ") {
				t.Errorf("'init' command should not be listed, but found: %q", line)
			}
		}
	})
}

// TestInitCommand_ErrorMessage_Specificity tests that the migration error
// is specific and actionable, verifying the exact content and structure.
func TestInitCommand_ErrorMessage_Specificity(t *testing.T) {
	// Build the CLI binary for testing
	binPath := buildCLI(t)
	defer func() { _ = os.Remove(binPath) }()

	t.Run("migration message structure is correct", func(t *testing.T) {
		//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
		cmd := exec.Command(binPath, "init")
		_, stderr, exitCode := runCommand(t, cmd)

		if exitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", exitCode)
		}

		// Verify the error message structure matches expectations
		expectedLines := []string{
			"Error: The 'init' command has been removed in v2.0.0.",
			"",
			"Providers are now automatically downloaded during 'nomos build'.",
			"",
			"Migration:",
			"  Old: nomos init && nomos build",
			"  New: nomos build",
		}

		for _, expectedLine := range expectedLines {
			if !strings.Contains(stderr, expectedLine) {
				t.Errorf("stderr missing expected line: %q\nFull stderr:\n%s",
					expectedLine, stderr)
			}
		}

		// Verify URL is present and correctly formatted
		expectedURL := "https://github.com/autonomous-bits/nomos/blob/main/docs/guides/migration-v2.md"
		if !strings.Contains(stderr, expectedURL) {
			t.Errorf("stderr missing migration guide URL: %q\nFull stderr:\n%s",
				expectedURL, stderr)
		}
	})
}
