// Package test provides integration tests for the Nomos CLI help text.
package test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestHelpText verifies that the main help output contains all required content.
func TestHelpText(t *testing.T) {
	// Build CLI binary for testing
	binPath := buildCLI(t)

	//nolint:gosec,noctx // G204: Test code with controlled binary path; context not needed
	cmd := exec.Command(binPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run nomos --help: %v\noutput: %s", err, output)
	}

	helpText := string(output)

	// Required sections and keywords that must appear in main help
	requiredContent := []string{
		"Nomos CLI",
		"Usage:",
		"Commands:",
		"build",
		"Compile", // Capital C as it appears in the help text
		"configuration snapshots",
		"Global Options:",
		"--help",
		"Examples:",
		"nomos build",
	}

	for _, content := range requiredContent {
		if !strings.Contains(helpText, content) {
			t.Errorf("main help text missing required content: %q\n\nFull output:\n%s", content, helpText)
		}
	}
}

// TestBuildHelpText verifies that build --help contains all required flags and examples.
func TestBuildHelpText(t *testing.T) {
	// Build CLI binary for testing
	binPath := buildCLI(t)

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run nomos build --help: %v\noutput: %s", err, output)
	}

	helpText := string(output)

	// All required flags that must be documented
	requiredFlags := []string{
		"-p, --path",
		"-f, --format",
		"-o, --out",
		"--var",
		"--strict",
		"--allow-missing-provider",
		"--timeout-per-provider",
		"--max-concurrent-providers",
		"--verbose",
		"-h, --help",
	}

	for _, flag := range requiredFlags {
		if !strings.Contains(helpText, flag) {
			t.Errorf("build help text missing required flag: %q\n\nFull output:\n%s", flag, helpText)
		}
	}

	// Required sections
	requiredSections := []string{
		"Usage:",
		"Options:",
		"Exit Codes:",
		"Examples:",
		"File Discovery:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(helpText, section) {
			t.Errorf("build help text missing required section: %q\n\nFull output:\n%s", section, helpText)
		}
	}

	// Required keywords in help text (networking and determinism notes)
	requiredKeywords := []string{
		"deterministic",
		"lexicographic",
		"network",
		"offline",
	}

	for _, keyword := range requiredKeywords {
		if !strings.Contains(strings.ToLower(helpText), strings.ToLower(keyword)) {
			t.Errorf("build help text missing important keyword: %q\n\nFull output:\n%s", keyword, helpText)
		}
	}

	// Example commands that should be present (at least one form of path flag)
	exampleCommands := []string{
		"nomos build",
		".csl",
		"--var",
		"--strict",
	}

	for _, example := range exampleCommands {
		if !strings.Contains(helpText, example) {
			t.Errorf("build help text missing example command pattern: %q\n\nFull output:\n%s", example, helpText)
		}
	}

	// Check that path flag appears in examples (either -p or --path)
	if !strings.Contains(helpText, "-p ") && !strings.Contains(helpText, "--path ") {
		t.Errorf("build help text missing path flag (-p or --path) in examples\n\nFull output:\n%s", helpText)
	}
}

// TestHelpConsistency verifies that help text doesn't have obvious duplications or inconsistencies.
func TestHelpConsistency(t *testing.T) {
	// Build CLI binary for testing
	binPath := buildCLI(t)

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run nomos build --help: %v\noutput: %s", err, output)
	}

	helpText := string(output)

	// Check for duplicate "Examples:" section which was observed in the code
	examplesCount := strings.Count(helpText, "Examples:")
	if examplesCount > 1 {
		t.Errorf("build help has duplicate 'Examples:' sections (found %d)", examplesCount)
	}

	// Check for duplicate "Exit Codes:" section
	exitCodesCount := strings.Count(helpText, "Exit Codes:")
	if exitCodesCount > 1 {
		t.Errorf("build help has duplicate 'Exit Codes:' sections (found %d)", exitCodesCount)
	}
}
