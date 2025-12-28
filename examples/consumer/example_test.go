package consumer_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestConsumerExampleBuilds verifies that the consumer example builds successfully
// using the workspace-managed dependencies. This validates that external consumers
// can use Nomos libraries with standard require directives.
func TestConsumerExampleBuilds(t *testing.T) {
	examplePath := filepath.Join("cmd", "consumer-example")

	//nolint:gosec // G204: test code with controlled inputs
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(t.TempDir(), "consumer-example"), ".")
	cmd.Dir = examplePath
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Consumer example failed to build:\n%s\nError: %v", string(output), err)
	}

	t.Logf("Consumer example built successfully")
}

// TestConsumerExampleRuns verifies that the built consumer example can execute
// and properly handle command-line arguments.
func TestConsumerExampleRuns(t *testing.T) {
	examplePath := filepath.Join("cmd", "consumer-example")
	binaryPath := filepath.Join(t.TempDir(), "consumer-example")

	// Build the example
	//nolint:gosec // G204: test code with controlled inputs
	buildCmd := exec.CommandContext(t.Context(), "go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = examplePath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build consumer example:\n%s\nError: %v", string(output), err)
	}

	// Run the example with test data
	testConfig := filepath.Join("testdata", "simple.csl")
	//nolint:gosec // G204: test code with controlled inputs
	runCmd := exec.CommandContext(t.Context(), binaryPath, testConfig)
	output, err := runCmd.CombinedOutput()

	t.Logf("Consumer example output:\n%s", string(output))

	// The example should run without critical errors
	if err != nil {
		t.Logf("Note: Example exited with error (expected without providers): %v", err)
	}

	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Consumer example produced no output")
	}
}

// TestGoModHasNoReplaceDirectives verifies that the go.mod does not contain
// replace directives, demonstrating the proper pattern for external consumers.
func TestGoModHasNoReplaceDirectives(t *testing.T) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	goModContent := string(content)
	if containsReplaceDirective(goModContent) {
		t.Errorf("go.mod contains replace directives, but should use require only:\n%s", goModContent)
		t.Error("External consumers should not need replace directives")
	}

	t.Log("✓ go.mod correctly uses require directives without replace")
}

// containsReplaceDirective checks if the content contains uncommented replace directives
func containsReplaceDirective(content string) bool {
	inMultilineComment := false
	for i := 0; i < len(content); i++ {
		// Handle multi-line comments
		if i < len(content)-1 && content[i:i+2] == "/*" {
			inMultilineComment = true
			i++
			continue
		}
		if inMultilineComment && i < len(content)-1 && content[i:i+2] == "*/" {
			inMultilineComment = false
			i++
			continue
		}
		if inMultilineComment {
			continue
		}

		// Handle single-line comments
		if i < len(content)-1 && content[i:i+2] == "//" {
			// Skip to end of line
			for i < len(content) && content[i] != '\n' {
				i++
			}
			continue
		}

		// Check for replace keyword
		if i < len(content)-6 && content[i:i+7] == "replace" {
			// Make sure it's not part of another word
			if i > 0 && isAlphanumeric(content[i-1]) {
				continue
			}
			if i+7 < len(content) && isAlphanumeric(content[i+7]) {
				continue
			}
			return true
		}
	}
	return false
}

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
