//go:build integration
// +build integration

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestFormatIntegration_YAML tests that the CLI correctly generates YAML output
// when the --format yaml flag is specified.
//
// T019: Integration test for YAML CLI flag
func TestFormatIntegration_YAML(t *testing.T) {
	binPath := buildCLI(t)

	// Create a test fixture
	tmpFixtureDir := t.TempDir()
	fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
	fixtureContent := `region: "us-west-2"

vpc:
  cidr: "10.0.0.0/16"
  enable_dns: true
  tags:
    - "production"
    - "web"

database:
  engine: "postgres"
  version: 14
  multi_az: false
`
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	// Create temp directory for output
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.yaml")

	// Run the build command with --format yaml
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "yaml", "-o", outFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build command failed: %v\nOutput: %s", err, output)
	}

	// Verify output file was created
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatalf("output file was not created: %s", outFile)
	}

	// Read and verify the YAML output
	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Verify it's valid YAML
	var parsed map[string]any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("failed to parse YAML output: %v\nContent: %s", err, content)
	}

	// Verify the structure contains expected content (metadata is opt-in)
	if parsed["region"] == nil {
		t.Error("YAML output missing expected top-level data")
	}
	if parsed["metadata"] != nil {
		t.Error("YAML output should not include 'metadata' by default")
	}

	// Verify content is not JSON (should not start with '{')
	contentStr := strings.TrimSpace(string(content))
	if strings.HasPrefix(contentStr, "{") {
		t.Error("YAML output appears to be JSON format")
	}

	// Verify YAML-specific syntax is present
	if !strings.Contains(contentStr, "region:") {
		t.Error("YAML output missing expected YAML syntax (key: value)")
	}
}

// TestFormatIntegration_YAML_Deterministic tests that YAML output is deterministic
// across multiple runs (same input produces identical output).
func TestFormatIntegration_YAML_Deterministic(t *testing.T) {
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

	// Run the build command 5 times and collect outputs
	var outputs [][]byte
	for i := 0; i < 5; i++ {
		outFile := filepath.Join(tmpDir, "output-"+string(rune('0'+i))+".yaml")

		//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
		cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "yaml", "-o", outFile)

		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("run %d: build command failed: %v\nOutput: %s", i, err, output)
		}

		// Read the output file
		//nolint:gosec // G304: Reading test output file from controlled location
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("run %d: failed to read output: %v", i, err)
		}

		outputs = append(outputs, content)
	}

	// Verify all outputs have identical data sections
	// Note: Metadata sections may differ due to timestamps
	for i := 0; i < len(outputs); i++ {
		var parsed map[string]any
		if err := yaml.Unmarshal(outputs[i], &parsed); err != nil {
			t.Fatalf("run %d: failed to parse YAML: %v", i, err)
		}

		// Extract data section
		dataYAML, err := yaml.Marshal(parsed["data"])
		if err != nil {
			t.Fatalf("run %d: failed to marshal data section: %v", i, err)
		}

		if i > 0 {
			var firstParsed map[string]any
			if err := yaml.Unmarshal(outputs[0], &firstParsed); err != nil {
				t.Fatalf("run 0: failed to parse YAML: %v", err)
			}
			firstDataYAML, _ := yaml.Marshal(firstParsed["data"])

			if string(dataYAML) != string(firstDataYAML) {
				t.Errorf("run %d: data section differs from run 0", i)
			}
		}
	}
}

// TestFormatIntegration_YAML_KeyOrdering tests that map keys are sorted
// alphabetically in the YAML output.
func TestFormatIntegration_YAML_KeyOrdering(t *testing.T) {
	binPath := buildCLI(t)

	// Create fixture with unsorted keys
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

	// Run CLI and capture output to file
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.yaml")
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "yaml", "-o", outFile)
	if combinedOutput, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build command failed: %v\nOutput: %s", err, combinedOutput)
	}

	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Check that keys appear in sorted order in the output string
	outputStr := string(content)

	// Find positions of our keys in the top-level output
	alphaPos := strings.Index(outputStr, "alpha:")
	middlePos := strings.Index(outputStr, "middle:")
	zebraPos := strings.Index(outputStr, "zebra:")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output")
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}
}

// TestFormatIntegration_YAML_CaseInsensitive tests that format flag accepts
// case-insensitive values (yaml, YAML, Yaml should all work).
func TestFormatIntegration_YAML_CaseInsensitive(t *testing.T) {
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

	// Test different case variations
	testCases := []string{"yaml", "YAML", "Yaml", "YaML"}

	tmpDir := t.TempDir()
	for _, formatValue := range testCases {
		t.Run(formatValue, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+formatValue+".yaml")

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", formatValue, "-o", outFile)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build command failed for format %q: %v\nOutput: %s", formatValue, err, output)
			}

			// Verify output is valid YAML
			//nolint:gosec // G304: Reading test output file from controlled location
			content, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("failed to read output for format %q: %v", formatValue, err)
			}

			var parsed map[string]any
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("failed to parse YAML output for format %q: %v", formatValue, err)
			}
		})
	}
}
