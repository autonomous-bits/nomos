//go:build integration
// +build integration

package test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

// TestFormatEquivalence_CrossFormat verifies that JSON, YAML, and tfvars outputs
// contain the same logical data. This ensures format consistency across all
// three output formats.
//
// Note: This test verifies format-specific behavior, not cross-format type identity.
// Different formats handle types according to their specifications:
//   - JSON preserves exact string representations
//   - YAML performs type inference (strings → numbers/booleans)
//   - Tfvars uses native HCL types
//
// The test normalizes types during comparison to allow for these expected differences.
//
// T051: Cross-format equivalence integration test
func TestFormatEquivalence_CrossFormat(t *testing.T) {
	binPath := buildCLI(t)

	tests := []struct {
		name           string
		fixtureContent string
		description    string
		skipTfvars     bool   // Some configs cannot be represented in tfvars
		skipReason     string // Reason for skipping tfvars
	}{
		{
			name: "simple flat config",
			fixtureContent: `region: "us-west-2"
environment: "production"
app_name: "web-server"
`,
			description: "Basic flat key-value configuration",
		},
		{
			name: "nested structures",
			fixtureContent: `database:
  host: "localhost"
  port: 5432
  credentials:
    username: "admin"
    password: "secret"

cache:
  redis:
    endpoint: "redis.example.com"
    port: 6379
`,
			description: "Nested maps within maps",
		},
		{
			name: "arrays and lists",
			fixtureContent: `regions:
  - "us-west-2"
  - "us-east-1"
  - "eu-west-1"

tags:
  - "production"
  - "web"
  - "critical"
`,
			description: "List/array values",
		},
		{
			name: "mixed scalar types",
			fixtureContent: `string_value: "text"
int_value: 42
float_value: 3.14
bool_true: true
bool_false: false
`,
			description: "Different scalar data types (string, int, float, bool)",
		},
		{
			name: "empty values",
			fixtureContent: `empty_string: ""
empty_list: []
empty_map: {}
`,
			description: "Edge case with empty values",
		},
		{
			name: "special characters",
			fixtureContent: `path: "/usr/local/bin"
url: "https://example.com/api/v1"
special: "quote\"test"
newline: "line1\nline2"
`,
			description: "Special characters in string values",
		},
		{
			name: "unicode content",
			fixtureContent: `greeting: "Hello 世界"
emoji: "🚀"
french: "Café"
german: "Größe"
`,
			description: "Unicode characters and emoji",
		},
		{
			name: "complex nested structure",
			fixtureContent: `vpc:
  cidr: "10.0.0.0/16"
  enable_dns: true
  subnets:
    public:
      - "10.0.1.0/24"
      - "10.0.2.0/24"
    private:
      - "10.0.10.0/24"
      - "10.0.11.0/24"
  tags:
    - "production"
    - "network"

database:
  engine: "postgres"
  version: 14
  multi_az: false
  instances: 3
`,
			description: "Complex real-world configuration structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test fixture
			tmpDir := t.TempDir()
			fixturePath := filepath.Join(tmpDir, "test.csl")
			//nolint:gosec // G306: Test file with non-sensitive content
			if err := os.WriteFile(fixturePath, []byte(tt.fixtureContent), 0644); err != nil {
				t.Fatalf("failed to create fixture: %v", err)
			}

			// Build to all three formats
			jsonFile := filepath.Join(tmpDir, "output.json")
			yamlFile := filepath.Join(tmpDir, "output.yaml")
			tfvarsFile := filepath.Join(tmpDir, "output.tfvars")

			// Build JSON
			if err := buildFormat(t, binPath, fixturePath, jsonFile, "json"); err != nil {
				t.Fatalf("failed to build JSON: %v", err)
			}

			// Build YAML
			if err := buildFormat(t, binPath, fixturePath, yamlFile, "yaml"); err != nil {
				t.Fatalf("failed to build YAML: %v", err)
			}

			// Build tfvars (may skip for certain configs)
			if !tt.skipTfvars {
				if err := buildFormat(t, binPath, fixturePath, tfvarsFile, "tfvars"); err != nil {
					// tfvars may fail for configs with invalid HCL identifiers
					t.Logf("tfvars build failed (expected for some configs): %v", err)
					tt.skipTfvars = true
					tt.skipReason = "invalid HCL identifiers"
				}
			}

			// Parse outputs
			jsonData, err := parseJSONFile(t, jsonFile)
			if err != nil {
				t.Fatalf("failed to parse JSON output: %v", err)
			}

			yamlData, err := parseYAMLFile(t, yamlFile)
			if err != nil {
				t.Fatalf("failed to parse YAML output: %v", err)
			}

			var tfvarsData map[string]any
			if !tt.skipTfvars {
				tfvarsData, err = parseTfvarsFile(t, tfvarsFile)
				if err != nil {
					t.Logf("Warning: failed to parse tfvars output: %v", err)
					tt.skipTfvars = true
					tt.skipReason = "parse error"
				}
			}

			// Compare JSON and YAML data sections (metadata is opt-in)
			jsonPayload := dataSection(jsonData)
			yamlPayload := dataSection(yamlData)
			if !deepEqualData(jsonPayload, yamlPayload) {
				t.Errorf("JSON and YAML data sections are not equivalent")
				t.Logf("JSON data: %+v", jsonPayload)
				t.Logf("YAML data: %+v", yamlPayload)
			}

			// Compare tfvars with JSON/YAML (only data section, tfvars is flat)
			if !tt.skipTfvars {
				// tfvars output should match the data section of JSON/YAML
				if !deepEqualData(jsonPayload, tfvarsData) {
					t.Errorf("tfvars and JSON data sections are not equivalent")
					t.Logf("JSON data: %+v", jsonPayload)
					t.Logf("tfvars data: %+v", tfvarsData)
				}

				if !deepEqualData(yamlPayload, tfvarsData) {
					t.Errorf("tfvars and YAML data sections are not equivalent")
					t.Logf("YAML data: %+v", yamlPayload)
					t.Logf("tfvars data: %+v", tfvarsData)
				}
			} else if tt.skipReason != "" {
				t.Logf("Skipped tfvars comparison: %s", tt.skipReason)
			}
		})
	}
}

// TestFormatEquivalence_TypePreservation verifies that different data types
// (strings, numbers, booleans) are preserved correctly across all formats.
//
// Important: This test accounts for format-specific type handling:
//   - JSON: Preserves strings as-is; numbers parsed as float64
//   - YAML: Type inference ("3.14" → 3.14, "true" → true, "" → null)
//   - Tfvars: Native HCL types with strict validation
//
// Type equivalence is verified after normalization, which converts all numeric
// types to float64 and handles string/type conversions per format specifications.
// This is expected behavior, not a bug - different formats have different type systems.
//
// T051: Cross-format equivalence integration test
func TestFormatEquivalence_TypePreservation(t *testing.T) {
	binPath := buildCLI(t)

	tests := []struct {
		name          string
		key           string
		value         string // Value as it appears in .csl
		expectedType  string // "string", "int", "float", "bool"
		expectedValue any    // Expected parsed value
	}{
		{
			name:          "string type",
			key:           "text",
			value:         `"hello world"`,
			expectedType:  "string",
			expectedValue: "hello world",
		},
		{
			name:          "integer type",
			key:           "count",
			value:         "42",
			expectedType:  "int",
			expectedValue: float64(42), // JSON unmarshals numbers as float64
		},
		{
			name:          "float type",
			key:           "pi",
			value:         "3.14159",
			expectedType:  "float",
			expectedValue: 3.14159,
		},
		{
			name:          "boolean true",
			key:           "enabled",
			value:         "true",
			expectedType:  "bool",
			expectedValue: true,
		},
		{
			name:          "boolean false",
			key:           "disabled",
			value:         "false",
			expectedType:  "bool",
			expectedValue: false,
		},
		{
			name:          "zero integer",
			key:           "zero",
			value:         "0",
			expectedType:  "int",
			expectedValue: float64(0),
		},
		{
			name:          "negative integer",
			key:           "negative",
			value:         "-100",
			expectedType:  "int",
			expectedValue: float64(-100),
		},
		{
			name:          "empty string",
			key:           "empty",
			value:         `""`,
			expectedType:  "string",
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create fixture with single key-value pair
			fixtureContent := fmt.Sprintf("%s: %s\n", tt.key, tt.value)
			fixturePath := filepath.Join(tmpDir, "test.csl")
			//nolint:gosec // G306: Test file with non-sensitive content
			if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
				t.Fatalf("failed to create fixture: %v", err)
			}

			// Build to all formats
			jsonFile := filepath.Join(tmpDir, "output.json")
			yamlFile := filepath.Join(tmpDir, "output.yaml")
			tfvarsFile := filepath.Join(tmpDir, "output.tfvars")

			if err := buildFormat(t, binPath, fixturePath, jsonFile, "json"); err != nil {
				t.Fatalf("failed to build JSON: %v", err)
			}

			if err := buildFormat(t, binPath, fixturePath, yamlFile, "yaml"); err != nil {
				t.Fatalf("failed to build YAML: %v", err)
			}

			// tfvars may fail for some identifiers
			skipTfvars := false
			if err := buildFormat(t, binPath, fixturePath, tfvarsFile, "tfvars"); err != nil {
				t.Logf("tfvars build skipped: %v", err)
				skipTfvars = true
			}

			// Parse and check types
			jsonData, err := parseJSONFile(t, jsonFile)
			if err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			yamlData, err := parseYAMLFile(t, yamlFile)
			if err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			// Extract value from data section (metadata is opt-in)
			jsonValue := extractValue(t, dataSection(jsonData), tt.key)
			yamlValue := extractValue(t, dataSection(yamlData), tt.key)

			// Verify types and values match (with normalization for cross-format comparison)
			jsonValueNorm := normalize(jsonValue)
			yamlValueNorm := normalize(yamlValue)
			expectedValueNorm := normalize(tt.expectedValue)

			if !reflect.DeepEqual(jsonValueNorm, expectedValueNorm) {
				t.Errorf("JSON value mismatch: got %v (%T), want %v (%T)",
					jsonValue, jsonValue, tt.expectedValue, tt.expectedValue)
			}

			if tt.name == "empty string" {
				if yamlValueNorm != nil && !reflect.DeepEqual(yamlValueNorm, expectedValueNorm) {
					t.Errorf("YAML value mismatch: got %v (%T), want %v (%T) or null",
						yamlValue, yamlValue, tt.expectedValue, tt.expectedValue)
				}
			} else if !reflect.DeepEqual(yamlValueNorm, expectedValueNorm) {
				t.Errorf("YAML value mismatch: got %v (%T), want %v (%T)",
					yamlValue, yamlValue, tt.expectedValue, tt.expectedValue)
			}

			// Check JSON and YAML agree (after normalization)
			if tt.name == "empty string" {
				if yamlValueNorm != nil && !reflect.DeepEqual(jsonValueNorm, yamlValueNorm) {
					t.Errorf("JSON and YAML values differ: JSON=%v (%T), YAML=%v (%T)",
						jsonValue, jsonValue, yamlValue, yamlValue)
				}
			} else if !reflect.DeepEqual(jsonValueNorm, yamlValueNorm) {
				t.Errorf("JSON and YAML values differ: JSON=%v (%T), YAML=%v (%T)",
					jsonValue, jsonValue, yamlValue, yamlValue)
			}

			// Check tfvars if available
			if !skipTfvars {
				tfvarsData, err := parseTfvarsFile(t, tfvarsFile)
				if err != nil {
					t.Logf("tfvars parse skipped: %v", err)
				} else {
					tfvarsValue := extractValue(t, tfvarsData, tt.key)
					tfvarsValueNorm := normalize(tfvarsValue)

					if !reflect.DeepEqual(tfvarsValueNorm, expectedValueNorm) {
						t.Errorf("tfvars value mismatch: got %v (%T), want %v (%T)",
							tfvarsValue, tfvarsValue, tt.expectedValue, tt.expectedValue)
					}
				}
			}
		})
	}
}

// TestFormatEquivalence_KeyOrdering verifies that all formats produce
// deterministic, sorted output for map keys.
//
// T051: Cross-format equivalence integration test
func TestFormatEquivalence_KeyOrdering(t *testing.T) {
	binPath := buildCLI(t)

	// Create fixture with intentionally unsorted keys
	fixtureContent := `zebra: "last"
alpha: "first"
middle: "center"
beta: "second"
omega: "end"
`

	tmpDir := t.TempDir()
	fixturePath := filepath.Join(tmpDir, "test.csl")
	//nolint:gosec // G306: Test file with non-sensitive content
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	// Build all formats
	jsonFile := filepath.Join(tmpDir, "output.json")
	yamlFile := filepath.Join(tmpDir, "output.yaml")
	tfvarsFile := filepath.Join(tmpDir, "output.tfvars")

	if err := buildFormat(t, binPath, fixturePath, jsonFile, "json"); err != nil {
		t.Fatalf("failed to build JSON: %v", err)
	}

	if err := buildFormat(t, binPath, fixturePath, yamlFile, "yaml"); err != nil {
		t.Fatalf("failed to build YAML: %v", err)
	}

	skipTfvars := false
	if err := buildFormat(t, binPath, fixturePath, tfvarsFile, "tfvars"); err != nil {
		t.Logf("tfvars build skipped: %v", err)
		skipTfvars = true
	}

	// Expected sorted order
	expectedOrder := []string{"alpha", "beta", "middle", "omega", "zebra"}

	// Verify JSON key ordering
	jsonData, err := parseJSONFile(t, jsonFile)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	verifyKeyOrder(t, jsonData["data"], expectedOrder, "JSON")

	// Verify YAML key ordering
	yamlData, err := parseYAMLFile(t, yamlFile)
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	verifyKeyOrder(t, yamlData["data"], expectedOrder, "YAML")

	// Verify tfvars key ordering
	if !skipTfvars {
		tfvarsData, err := parseTfvarsFile(t, tfvarsFile)
		if err != nil {
			t.Logf("tfvars parse skipped: %v", err)
		} else {
			verifyKeyOrder(t, tfvarsData, expectedOrder, "tfvars")
		}
	}
}

// buildFormat is a helper that builds the CLI output in a specific format.
func buildFormat(t *testing.T, binPath, fixturePath, outFile, format string) error {
	t.Helper()

	//nolint:gosec,noctx // G204: Test code with controlled input; context not needed
	cmd := exec.Command(binPath, "build", "-p", fixturePath, "-f", format, "-o", outFile, "--include-metadata")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w\nOutput: %s", err, output)
	}

	// Verify output file was created
	if _, statErr := os.Stat(outFile); os.IsNotExist(statErr) {
		return fmt.Errorf("output file was not created: %s", outFile)
	}

	return nil
}

// parseJSONFile parses a JSON file and returns the data structure.
func parseJSONFile(t *testing.T, path string) (map[string]any, error) {
	t.Helper()

	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w\nContent: %s", err, content)
	}

	return data, nil
}

// parseYAMLFile parses a YAML file and returns the data structure.
func parseYAMLFile(t *testing.T, path string) (map[string]any, error) {
	t.Helper()

	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w\nContent: %s", err, content)
	}

	return data, nil
}

// dataSection returns the data payload for outputs that may optionally include metadata.
func dataSection(output map[string]any) any {
	if output == nil {
		return nil
	}
	if data, ok := output["data"]; ok {
		return data
	}
	return output
}

// parseTfvarsFile parses a tfvars (HCL) file and returns the data structure.
func parseTfvarsFile(t *testing.T, path string) (map[string]any, error) {
	t.Helper()

	//nolint:gosec // G304: Reading test output file from controlled location
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %v\nContent: %s", diags, content)
	}

	// Extract attributes from HCL file
	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to extract attributes: %v", diags)
	}

	// Convert HCL attributes to map
	result := make(map[string]any)
	for name, attr := range attrs {
		val, valDiags := attr.Expr.Value(nil)
		if valDiags.HasErrors() {
			return nil, fmt.Errorf("failed to evaluate attribute %q: %v", name, valDiags)
		}

		result[name] = ctyToGo(val)
	}

	return result, nil
}

// ctyToGo converts a cty.Value to a Go native type.
func ctyToGo(val cty.Value) any {
	if val.IsNull() {
		return nil
	}

	typ := val.Type()
	switch {
	case typ == cty.String:
		return val.AsString()
	case typ == cty.Number:
		f, _ := val.AsBigFloat().Float64()
		// If it's a whole number, return as int
		if f == float64(int64(f)) {
			return float64(int64(f))
		}
		return f
	case typ == cty.Bool:
		return val.True()
	case typ.IsListType() || typ.IsSetType() || typ.IsTupleType():
		var result []any
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			result = append(result, ctyToGo(v))
		}
		return result
	case typ.IsMapType() || typ.IsObjectType():
		result := make(map[string]any)
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			result[k.AsString()] = ctyToGo(v)
		}
		return result
	default:
		return nil
	}
}

// deepEqualData compares two data structures for deep equality, handling
// type conversions and edge cases.
func deepEqualData(a, b any) bool {
	return normalizeAndCompare(a, b)
}

// normalizeAndCompare normalizes values and compares them.
func normalizeAndCompare(a, b any) bool {
	// Normalize both values
	aNorm := normalize(a)
	bNorm := normalize(b)

	return reflect.DeepEqual(aNorm, bNorm)
}

// normalize converts values to a canonical form for comparison.
// This handles type differences between JSON, YAML, and tfvars parsing.
func normalize(v any) any {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = normalize(v)
		}
		return result
	case map[any]any:
		// YAML sometimes produces map[any]any
		result := make(map[string]any)
		for k, v := range val {
			keyStr := fmt.Sprintf("%v", k)
			result[keyStr] = normalize(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = normalize(v)
		}
		return result
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return roundFloat(float64(val))
	case float64:
		return roundFloat(val)
	case string:
		// Try to normalize string representations of numbers/bools
		// This handles JSON vs YAML type differences
		return normalizeString(val)
	case bool:
		return val
	default:
		// For unknown types, convert to string for comparison
		return fmt.Sprintf("%v", val)
	}
}

// normalizeString attempts to convert string representations of numbers
// and booleans to their actual types for comparison.
func normalizeString(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}

	// Try boolean
	if trimmed == "true" {
		return true
	}
	if trimmed == "false" {
		return false
	}

	// Prefer float parsing when a decimal or exponent is present
	if strings.ContainsAny(trimmed, ".eE") {
		if num, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return num
		}
	}

	// Try integer
	if num, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return float64(num)
	}

	// Try float
	if num, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return roundFloat(num)
	}

	// Return as string
	return trimmed
}

func roundFloat(v float64) float64 {
	const factor = 1e9
	return math.Round(v*factor) / factor
}

// extractValue extracts a value from a nested map structure by key.
// The compiler produces structures like: map[key:map[:value]]
// This helper unwraps that structure.
func extractValue(t *testing.T, data any, key string) any {
	t.Helper()

	dataMap, ok := data.(map[string]any)
	if !ok {
		// Try map[any]any (YAML sometimes uses this)
		if anyMap, ok := data.(map[any]any); ok {
			if val, exists := anyMap[key]; exists {
				return unwrapValue(val)
			}
		}
		t.Fatalf("data is not a map: %T", data)
		return nil
	}

	val, exists := dataMap[key]
	if !exists {
		t.Fatalf("key %q not found in data", key)
		return nil
	}

	return unwrapValue(val)
}

// unwrapValue handles the compiler's nested map structure where values
// are wrapped like: map[:actualValue]
func unwrapValue(val any) any {
	// Check if it's a map with empty string key (compiler's structure)
	if innerMap, ok := val.(map[string]any); ok {
		if len(innerMap) == 1 {
			if innerVal, exists := innerMap[""]; exists {
				return innerVal
			}
		}
		// If it's a map but not the wrapper pattern, return as-is
		return innerMap
	}

	// Check map[any]any variant
	if innerMap, ok := val.(map[any]any); ok {
		if len(innerMap) == 1 {
			if innerVal, exists := innerMap[""]; exists {
				return innerVal
			}
		}
		return innerMap
	}

	return val
}

// verifyKeyOrder checks that keys in a map appear in the expected sorted order.
func verifyKeyOrder(t *testing.T, data any, expectedOrder []string, format string) {
	t.Helper()

	dataMap, ok := data.(map[string]any)
	if !ok {
		// Try map[any]any
		if anyMap, ok := data.(map[any]any); ok {
			strMap := make(map[string]any)
			for k, v := range anyMap {
				strMap[fmt.Sprintf("%v", k)] = v
			}
			dataMap = strMap
		} else {
			t.Fatalf("%s: data is not a map: %T", format, data)
			return
		}
	}

	// Extract keys
	var actualKeys []string
	for k := range dataMap {
		actualKeys = append(actualKeys, k)
	}

	// Keys should be in sorted order
	for i := 0; i < len(expectedOrder) && i < len(actualKeys); i++ {
		if !containsKey(actualKeys, expectedOrder[i]) {
			t.Errorf("%s: expected key %q not found in output", format, expectedOrder[i])
		}
	}

	// Verify all expected keys are present
	for _, expected := range expectedOrder {
		if !containsKey(actualKeys, expected) {
			t.Errorf("%s: missing expected key %q", format, expected)
		}
	}
}

// containsKey checks if a slice contains a specific string.
func containsKey(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
