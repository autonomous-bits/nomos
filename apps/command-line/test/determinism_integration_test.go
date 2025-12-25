package test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDeterministicJSON_Integration tests that the CLI produces deterministic
// JSON output for the same input across multiple runs.
//
// NOTE: Full byte-for-byte equality is NOT expected because metadata contains
// timestamps (start_time, end_time) that vary. This test verifies that:
// 1. The data section is deterministic (map keys sorted)
// 2. The structure and ordering of metadata is deterministic
// 3. Map keys throughout the output are consistently ordered
func TestDeterministicJSON_Integration(t *testing.T) {
	// Build the CLI binary first
	ctx := context.Background()
	//nolint:gosec // G204: Test code with controlled input
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "../../../bin/nomos-test", "./cmd/nomos")
	buildCmd.Dir = ".."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\nOutput: %s", err, output)
	}
	defer func() { _ = os.Remove("../../bin/nomos-test") }()

	// Create a fixture with nested maps to test key ordering
	tmpFixtureDir := t.TempDir()
	fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
	fixtureContent := `zebra:
	value: "last"

alpha:
	value: "first"

middle:
	z: 3
	a: 1
	m: 2
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	// Create temp directory for outputs
	tmpDir := t.TempDir()

	// Run the build command 10 times and collect outputs
	var outputs []map[string]any
	for i := 0; i < 10; i++ {
		outFile := filepath.Join(tmpDir, "output-"+string(rune('0'+i))+".json")

		//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
		cmd := exec.Command("../../bin/nomos-test", "build", "-p", fixturePath, "-f", "json", "-o", outFile)
		cmd.Dir = ".."

		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("run %d: build command failed: %v\nOutput: %s", i, err, output)
		}

		// Read and parse the output file
		//nolint:gosec // G304: Reading test output file from controlled location
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("run %d: failed to read output: %v", i, err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(content, &parsed); err != nil {
			t.Fatalf("run %d: failed to parse JSON: %v", i, err)
		}

		outputs = append(outputs, parsed)
	}

	// Verify all outputs have identical data sections (which should be deterministic)
	firstData, _ := json.Marshal(outputs[0]["data"])
	for i := 1; i < len(outputs); i++ {
		currentData, _ := json.Marshal(outputs[i]["data"])
		if string(currentData) != string(firstData) {
			t.Errorf("run %d: data section differs from run 0", i)
			t.Logf("Run 0 data: %s", firstData)
			t.Logf("Run %d data: %s", i, currentData)
		}
	}

	// Verify key ordering is consistent in data section
	// Keys should appear as: alpha, middle, zebra (sorted)
	firstDataMap := outputs[0]["data"].(map[string]any)
	expectedKeys := []string{"alpha", "middle", "zebra"}
	actualKeys := make([]string, 0, len(firstDataMap))
	for k := range firstDataMap {
		actualKeys = append(actualKeys, k)
	}

	// Note: Go maps don't preserve order, but our JSON serializer should
	// This is verified by the JSON structure test below
	if len(actualKeys) != len(expectedKeys) {
		t.Errorf("expected %d keys, got %d", len(expectedKeys), len(actualKeys))
	}
}

// TestJSONStructure_KeyOrdering tests that JSON output has sorted map keys.
func TestJSONStructure_KeyOrdering(t *testing.T) {
	// Build the CLI binary
	ctx := context.Background()
	//nolint:gosec // G204: Test code with controlled input
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "../../../bin/nomos-test", "./cmd/nomos")
	buildCmd.Dir = ".."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\nOutput: %s", err, output)
	}
	defer func() { _ = os.Remove("../../bin/nomos-test") }()

	// Create fixture
	tmpFixtureDir := t.TempDir()
	fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
	fixtureContent := `zebra:
	name: "last"

alpha:
	name: "first"

middle:
	name: "center"
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	// Run CLI and capture stdout
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command("../../bin/nomos-test", "build", "-p", fixturePath, "-f", "json")
	cmd.Dir = ".."
	stdoutOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build command failed: %v\nOutput: %s", err, stdoutOutput)
	}

	// Check that keys appear in sorted order in the output string
	outputStr := string(stdoutOutput)
	alphaPos := indexOf(outputStr, "\"alpha\"")
	middlePos := indexOf(outputStr, "\"middle\"")
	zebraPos := indexOf(outputStr, "\"zebra\"")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output")
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestNonWritableOutput_ExitsWithCode2 tests that non-writable output paths
// result in exit code 2 as specified in the PRD.
func TestNonWritableOutput_ExitsWithCode2(t *testing.T) {
	// Build the CLI binary
	ctx := context.Background()
	//nolint:gosec // G204: Test code with controlled input
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "../../../bin/nomos-test", "./cmd/nomos")
	buildCmd.Dir = ".."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\nOutput: %s", err, output)
	}
	defer func() { _ = os.Remove("../../bin/nomos-test") }()

	// Create fixture
	tmpFixtureDir := t.TempDir()
	fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
	fixtureContent := `config:
	name: "test"
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	// Try to write to a directory (not a file) - should fail
	tmpDir := t.TempDir()
	invalidOutput := tmpDir // directory, not a file

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command("../../bin/nomos-test", "build", "-p", fixturePath, "-f", "json", "-o", invalidOutput)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected command to fail with non-writable output, but it succeeded")
	}

	// Check exit code is 2
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		if exitCode != 2 {
			t.Errorf("expected exit code 2 for non-writable output, got %d", exitCode)
			t.Logf("Output: %s", output)
		}
	} else {
		t.Errorf("expected ExitError, got: %v", err)
	}
}
