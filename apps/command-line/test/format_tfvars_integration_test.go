//go:build integration
// +build integration

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
)

// TestFormatIntegration_Tfvars tests that the CLI correctly generates HCL tfvars output
// when the --format tfvars flag is specified.
//
// T032: Integration test for tfvars CLI flag
func TestFormatIntegration_Tfvars(t *testing.T) {
	binPath := buildCLI(t)

	// Create a test fixture
	tmpFixtureDir := t.TempDir()
	fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
	fixtureContent := `region: "us-west-2"

vpc:
  cidr: "10.0.0.0/16"
  enable_dns: true
  tag_env: "production"
  tag_type: "web"

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
	outFile := filepath.Join(tmpDir, "output.tfvars")

	// Run the build command with --format tfvars
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "tfvars", "-o", outFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build command failed: %v\nOutput: %s", err, output)
	}

	// Verify output file was created
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatalf("output file was not created: %s", outFile)
	}

	// Read and verify the HCL output
	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Verify it's valid HCL
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(content, "test.tfvars")
	if diags.HasErrors() {
		t.Fatalf("failed to parse HCL output: %v\nContent:\n%s", diags, content)
	}

	contentStr := strings.TrimSpace(string(content))

	// Verify HCL syntax (= not :)
	if !strings.Contains(contentStr, " = ") {
		t.Error("HCL output missing assignment syntax (=)")
	}

	// Verify NOT YAML/JSON
	if strings.Contains(contentStr, "data:") || strings.Contains(contentStr, "metadata:") {
		t.Error("HCL output contains YAML-style sections (data:/metadata:)")
	}
	if strings.HasPrefix(contentStr, "{") {
		t.Error("HCL output appears to be JSON format")
	}

	// Verify content is NOT wrapped in data/metadata sections
	// Tfvars should be flat variable declarations
	if strings.Contains(contentStr, "data =") || strings.Contains(contentStr, "metadata =") {
		t.Error("tfvars output should not contain data/metadata wrappers")
	}
}

// TestFormatIntegration_Tfvars_Deterministic tests that tfvars output is deterministic
// across multiple runs (same input produces identical output).
func TestFormatIntegration_Tfvars_Deterministic(t *testing.T) {
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
		outFile := filepath.Join(tmpDir, "output-"+string(rune('0'+i))+".tfvars")

		//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
		cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "tfvars", "-o", outFile)

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

	// Verify all outputs are byte-for-byte identical
	// Tfvars doesn't include timestamps, so should be fully deterministic
	for i := 1; i < len(outputs); i++ {
		if string(outputs[i]) != string(outputs[0]) {
			t.Errorf("run %d: output differs from run 0", i)
			t.Logf("Run 0:\n%s", outputs[0])
			t.Logf("Run %d:\n%s", i, outputs[i])
		}
	}
}

// TestFormatIntegration_Tfvars_KeyOrdering tests that map keys are sorted
// alphabetically in the tfvars output.
func TestFormatIntegration_Tfvars_KeyOrdering(t *testing.T) {
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
	outFile := filepath.Join(tmpDir, "output.tfvars")
	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "tfvars", "-o", outFile)
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

	// Find positions of our keys (they should be top-level variable declarations)
	alphaPos := strings.Index(outputStr, "alpha ")
	middlePos := strings.Index(outputStr, "middle ")
	zebraPos := strings.Index(outputStr, "zebra ")

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Fatalf("expected keys not found in output")
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("keys not in sorted order: alpha=%d, middle=%d, zebra=%d", alphaPos, middlePos, zebraPos)
		t.Logf("Output:\n%s", outputStr)
	}
}

// TestFormatIntegration_Tfvars_CaseInsensitive tests that format flag accepts
// case-insensitive values (tfvars, TFVARS, Tfvars should all work).
func TestFormatIntegration_Tfvars_CaseInsensitive(t *testing.T) {
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
	testCases := []string{"tfvars", "TFVARS", "Tfvars", "TfVars"}

	tmpDir := t.TempDir()
	for _, formatValue := range testCases {
		t.Run(formatValue, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, "output-"+formatValue+".tfvars")

			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", formatValue, "-o", outFile)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build command failed for format %q: %v\nOutput: %s", formatValue, err, output)
			}

			// Verify output is valid HCL
			//nolint:gosec // G304: Reading test output file from controlled location
			content, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("failed to read output for format %q: %v", formatValue, err)
			}

			parser := hclparse.NewParser()
			_, diags := parser.ParseHCL(content, "test.tfvars")
			if diags.HasErrors() {
				t.Errorf("failed to parse HCL output for format %q: %v\nContent:\n%s", formatValue, diags, content)
			}
		})
	}
}

// TestFormatIntegration_Tfvars_InvalidKeys tests error handling for invalid HCL keys.
// HCL variable names must follow specific rules (no spaces, dots, etc.).
func TestFormatIntegration_Tfvars_InvalidKeys(t *testing.T) {
	binPath := buildCLI(t)

	testCases := []struct {
		name           string
		fixtureContent string
		expectedInErr  string
	}{
		{
			name: "key with spaces",
			fixtureContent: `my key:
	value: "test"
`,
			expectedInErr: "invalid",
		},
		{
			name: "key with dots",
			fixtureContent: `my.key:
	value: "test"
`,
			expectedInErr: "invalid",
		},
		{
			name: "key starting with number",
			fixtureContent: `123key:
	value: "test"
`,
			expectedInErr: "invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fixture with invalid key
			tmpFixtureDir := t.TempDir()
			fixturePath := filepath.Join(tmpFixtureDir, "test.csl")
			//nolint:gosec // G306: Test file with non-sensitive content
			if err := os.WriteFile(fixturePath, []byte(tc.fixtureContent), 0644); err != nil {
				t.Fatalf("failed to create fixture: %v", err)
			}

			// Run CLI and expect error
			tmpDir := t.TempDir()
			outFile := filepath.Join(tmpDir, "output.tfvars")
			//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
			cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", "tfvars", "-o", outFile)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("expected command to fail for invalid key, but it succeeded")
				t.Logf("Output:\n%s", output)

				// For debugging, show what was generated
				//nolint:gosec // G304: Reading test output file from controlled location
				if content, readErr := os.ReadFile(outFile); readErr == nil {
					t.Logf("Generated content:\n%s", content)
				}
			} else {
				// Verify error message mentions invalid key
				outputStr := string(output)
				if !strings.Contains(strings.ToLower(outputStr), tc.expectedInErr) {
					t.Errorf("expected error message to contain %q, got: %s", tc.expectedInErr, outputStr)
				}
			}
		})
	}
}
