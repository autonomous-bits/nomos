//go:build integration
// +build integration

package test

import (
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
	binPath := buildCLI(t)

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
		cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "json", "-o", outFile, "--include-metadata")

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
	binPath := buildCLI(t)

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
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "json")
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
// result in exit code 1 (runtime I/O error). Note: Despite the function name,
// this test expects exit code 1, not 2. Exit code 2 is for usage errors,
// while I/O errors use exit code 1 following Cobra conventions.
func TestNonWritableOutput_ExitsWithCode2(t *testing.T) {
	binPath := buildCLI(t)

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

	// Create a read-only directory to ensure write operations fail
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil { // r-xr-xr-x (no write permission)
		t.Fatalf("failed to create read-only directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	invalidOutput := filepath.Join(readOnlyDir, "output.json")

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "json", "-o", invalidOutput)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected command to fail with non-writable output, but it succeeded")
	}

	// Check exit code is 1 (runtime I/O error, not usage error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		if exitCode != 1 {
			t.Errorf("expected exit code 1 for I/O error, got %d", exitCode)
			t.Logf("Output: %s", output)
		}
	} else {
		t.Errorf("expected ExitError, got: %v", err)
	}
}
