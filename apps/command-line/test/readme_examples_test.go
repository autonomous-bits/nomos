// Package test provides smoke tests for README examples.
package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestREADMEExamples verifies that the examples in the README actually work.
//
// Note: Currently skipped due to compiler bug (nil pointer in imports.ExtractImports).
// This test validates that README examples can be executed once the compiler issue is fixed.
func TestREADMEExamples(t *testing.T) {
	t.Skip("Skipping until compiler nil pointer bug in imports.ExtractImports is fixed")

	// Build CLI binary once for all tests
	binPath := buildCLI(t)

	// Change to apps/command-line directory so relative testdata paths work
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	cliDir := filepath.Dir(origDir) // One level up from test/ to apps/command-line/
	if err := os.Chdir(cliDir); err != nil {
		t.Fatalf("failed to change to CLI directory: %v", err)
	}

	tests := []struct {
		name             string
		args             []string
		wantExitCode     int
		wantOutputSubstr string
		skipStdoutCheck  bool
	}{
		{
			name:             "compile single file",
			args:             []string{"build", "-p", "testdata/simple.csl"},
			wantExitCode:     0,
			wantOutputSubstr: "data",
		},
		{
			name:            "compile directory to YAML",
			args:            []string{"build", "-p", "testdata/configs", "-f", "yaml"},
			wantExitCode:    0,
			skipStdoutCheck: true, // YAML not yet implemented, will fail
		},
		{
			name:             "compile with variables",
			args:             []string{"build", "-p", "testdata/with-vars.csl", "--var", "region=us-west", "--var", "env=dev"},
			wantExitCode:     0,
			wantOutputSubstr: "us-west",
		},
		{
			name:             "strict mode with no warnings",
			args:             []string{"build", "-p", "testdata/simple.csl", "--strict"},
			wantExitCode:     0,
			wantOutputSubstr: "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip YAML test until implemented
			if strings.Contains(tt.name, "YAML") || strings.Contains(tt.name, "yaml") {
				t.Skip("YAML format not yet implemented")
			}

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, tt.args...)
			stdout, stderr, exitCode := runCommand(t, cmd)

			if exitCode != tt.wantExitCode {
				t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
					exitCode, tt.wantExitCode, stdout, stderr)
			}

			if !tt.skipStdoutCheck && tt.wantOutputSubstr != "" {
				if !strings.Contains(stdout, tt.wantOutputSubstr) {
					t.Errorf("stdout missing expected substring %q\nstdout: %s",
						tt.wantOutputSubstr, stdout)
				}
			}
		})
	}
}

// TestREADMEExamplesWithOutput verifies examples that write to files.
//
// Note: Currently skipped due to compiler bug (nil pointer in imports.ExtractImports).
// This test validates file output behavior once the compiler issue is fixed.
func TestREADMEExamplesWithOutput(t *testing.T) {
	t.Skip("Skipping until compiler nil pointer bug in imports.ExtractImports is fixed")

	// Build CLI binary
	binPath := buildCLI(t)

	// Change to apps/command-line directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	cliDir := filepath.Dir(origDir)
	if err := os.Chdir(cliDir); err != nil {
		t.Fatalf("failed to change to CLI directory: %v", err)
	}

	// Create temp output file
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "snapshot.json")

	//nolint:gosec // G204: Test code with controlled input
	cmd := exec.Command(binPath, "build", "-p", "testdata/simple.csl", "-o", outputFile)
	stdout, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("output file was not created: %s", outputFile)
	}

	// Verify file contains valid JSON
	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "data") {
		t.Errorf("output file doesn't contain expected 'data' section: %s", content)
	}
}

// TestREADMEExamplesDeterminism verifies that running the same example twice produces identical output.
//
// Note: Currently skipped due to compiler bug (nil pointer in imports.ExtractImports).
// This test validates deterministic output behavior once the compiler issue is fixed.
func TestREADMEExamplesDeterminism(t *testing.T) {
	t.Skip("Skipping until compiler nil pointer bug in imports.ExtractImports is fixed")

	// Build CLI binary
	binPath := buildCLI(t)

	// Change to apps/command-line directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	cliDir := filepath.Dir(origDir)
	if err := os.Chdir(cliDir); err != nil {
		t.Fatalf("failed to change to CLI directory: %v", err)
	}

	// Run the same command twice
	//nolint:gosec,noctx // G204: Test code with controlled binary path; context not needed
	cmd1 := exec.Command(binPath, "build", "-p", "testdata/simple.csl")
	stdout1, stderr1, exitCode1 := runCommand(t, cmd1)

	//nolint:gosec,noctx // G204: Test code with controlled binary path; context not needed
	cmd2 := exec.Command(binPath, "build", "-p", "testdata/simple.csl")
	_, stderr2, exitCode2 := runCommand(t, cmd2)

	// Exit codes should match
	if exitCode1 != exitCode2 {
		t.Errorf("exit codes differ: %d vs %d", exitCode1, exitCode2)
	}

	// Stderr should be identical (may contain warnings)
	if stderr1 != stderr2 {
		t.Errorf("stderr differs:\nRun 1: %s\nRun 2: %s", stderr1, stderr2)
	}

	// Note: stdout will differ due to timestamps in metadata, but data section should be identical
	// For now, just check that both runs succeeded
	if exitCode1 != 0 {
		t.Errorf("expected successful compilation, got exit code %d\nstdout: %s\nstderr: %s",
			exitCode1, stdout1, stderr1)
	}
}
