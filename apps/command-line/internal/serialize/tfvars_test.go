package serialize

import (
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestToTfvars_Determinism tests that ToTfvars produces byte-for-byte identical
// output for the same logical snapshot across 10 independent invocations.
// This ensures consistent builds and reproducibility in CI/CD pipelines.
//
// T021: Determinism test (10 runs produce identical output)
func TestToTfvars_Determinism(t *testing.T) {
	// Create a snapshot with mixed types and nested maps to test key ordering
	snapshot := compiler.Snapshot{
		Data: map[string]any{
			"region":  "us-west-2",
			"count":   10,
			"enabled": true,
			"price":   3.14,
			"zones":   []any{"us-west-2a", "us-west-2b"},
			"vpc": map[string]any{
				"cidr": "10.0.0.0/16",
				"tags": map[string]any{
					"Name":        "main-vpc",
					"Environment": "production",
				},
			},
			"zebra": "last",
			"alpha": "first",
			"middle": map[string]any{
				"z": 3,
				"a": 1,
				"m": 2,
			},
		},
	}

	// Serialize 10 times and compare bytes
	var firstOutput []byte
	for i := 0; i < 10; i++ {
		output, err := ToTfvars(snapshot)
		if err != nil {
			t.Fatalf("iteration %d: ToTfvars failed: %v", i, err)
		}

		if i == 0 {
			firstOutput = output
		} else if string(output) != string(firstOutput) {
			t.Errorf("iteration %d: output differs from first iteration", i)
			t.Logf("First:\n%s", firstOutput)
			t.Logf("Current:\n%s", output)
		}
	}
}

// TestToTfvars_KeyOrdering tests that map keys are sorted alphabetically
// in the HCL .tfvars output for deterministic builds. This test verifies
// both top-level and nested key ordering.
//
// T022: Key ordering test (map keys sorted alphabetically)
func TestToTfvars_KeyOrdering(t *testing.T) {
	tests := []struct {
		name           string
		snapshot       compiler.Snapshot
		expectedOrder  []string // Keys in expected alphabetical order
		checkNested    bool
		nestedParent   string
		nestedExpected []string
	}{
		{
			name: "top-level keys sorted",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"zebra":  1,
					"alpha":  2,
					"middle": 3,
					"beta":   4,
				},
			},
			expectedOrder: []string{"alpha", "beta", "middle", "zebra"},
			checkNested:   false,
		},
		{
			name: "nested keys sorted",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"config": map[string]any{
						"zebra":  "z",
						"alpha":  "a",
						"middle": "m",
					},
				},
			},
			expectedOrder:  []string{"config"},
			checkNested:    true,
			nestedParent:   "config",
			nestedExpected: []string{"alpha", "middle", "zebra"},
		},
		{
			name: "deeply nested keys sorted",
			snapshot: compiler.Snapshot{
				Data: map[string]any{
					"root": map[string]any{
						"level2": map[string]any{
							"z_key": "value1",
							"a_key": "value2",
							"m_key": "value3",
						},
					},
				},
			},
			expectedOrder:  []string{"root"},
			checkNested:    true,
			nestedParent:   "level2",
			nestedExpected: []string{"a_key", "m_key", "z_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ToTfvars(tt.snapshot)
			if err != nil {
				t.Fatalf("ToTfvars failed: %v", err)
			}

			outputStr := string(output)

			// Check that top-level keys appear in sorted order
			var positions []int
			for _, key := range tt.expectedOrder {
				pos := indexOf(outputStr, key)
				if pos == -1 {
					t.Errorf("expected key %q not found in output:\n%s", key, outputStr)
					return
				}
				positions = append(positions, pos)
			}

			// Verify positions are in ascending order
			for i := 1; i < len(positions); i++ {
				if positions[i] <= positions[i-1] {
					t.Errorf("keys not in sorted order: %q appears before %q",
						tt.expectedOrder[i], tt.expectedOrder[i-1])
					t.Logf("Output:\n%s", outputStr)
				}
			}

			// Check nested keys if required
			if tt.checkNested {
				parentPos := indexOf(outputStr, tt.nestedParent)
				if parentPos == -1 {
					t.Fatalf("nested parent %q not found in output", tt.nestedParent)
				}

				var nestedPositions []int
				for _, key := range tt.nestedExpected {
					pos := indexOfAfter(outputStr, key, parentPos)
					if pos == -1 {
						t.Errorf("nested key %q not found after parent %q", key, tt.nestedParent)
						return
					}
					nestedPositions = append(nestedPositions, pos)
				}

				// Verify nested positions are in ascending order
				for i := 1; i < len(nestedPositions); i++ {
					if nestedPositions[i] <= nestedPositions[i-1] {
						t.Errorf("nested keys not in sorted order: %q appears before %q",
							tt.nestedExpected[i], tt.nestedExpected[i-1])
						t.Logf("Output:\n%s", outputStr)
					}
				}
			}
		})
	}
}

// TestToTfvars_InvalidKeys tests that keys not matching HCL identifier rules
// are rejected with descriptive error messages. HCL identifiers must match
// the pattern [a-zA-Z_][a-zA-Z0-9_-]* (start with letter/underscore, contain
// only letters, digits, underscores, hyphens).
//
// T023: Invalid keys test (keys with spaces/special chars rejected)
func TestToTfvars_InvalidKeys(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]any
		wantErr        bool
		invalidKeys    []string // Keys expected to be flagged as invalid
		errMustContain []string // Substrings that must appear in error message
	}{
		{
			name: "key with spaces",
			data: map[string]any{
				"my key": "value",
			},
			wantErr:        true,
			invalidKeys:    []string{"my key"},
			errMustContain: []string{"invalid", "key", "my key"},
		},
		{
			name: "key with dots",
			data: map[string]any{
				"my.key": "value",
			},
			wantErr:        true,
			invalidKeys:    []string{"my.key"},
			errMustContain: []string{"invalid", "key", "my.key"},
		},
		{
			name: "key with @ symbol",
			data: map[string]any{
				"my@key": "value",
			},
			wantErr:        true,
			invalidKeys:    []string{"my@key"},
			errMustContain: []string{"invalid", "key", "my@key"},
		},
		{
			name: "key with $ symbol",
			data: map[string]any{
				"my$var": "value",
			},
			wantErr:        true,
			invalidKeys:    []string{"my$var"},
			errMustContain: []string{"invalid", "key", "my$var"},
		},
		{
			name: "key starting with number",
			data: map[string]any{
				"123key": "value",
			},
			wantErr:        true,
			invalidKeys:    []string{"123key"},
			errMustContain: []string{"invalid", "key", "123key"},
		},
		{
			name: "multiple invalid keys",
			data: map[string]any{
				"valid_key":   "ok",
				"invalid key": "bad1",
				"bad.key":     "bad2",
				"another_ok":  "ok",
			},
			wantErr:        true,
			invalidKeys:    []string{"invalid key", "bad.key"},
			errMustContain: []string{"invalid", "key"},
		},
		{
			name: "valid keys with underscores",
			data: map[string]any{
				"my_key":   "value1",
				"_private": "value2",
				"key_123":  "value3",
			},
			wantErr: false,
		},
		{
			name: "valid keys with hyphens",
			data: map[string]any{
				"my-key":  "value1",
				"key-123": "value2",
				"a-b-c-d": "value3",
			},
			wantErr: false,
		},
		{
			name: "valid keys alphanumeric",
			data: map[string]any{
				"key":    "value1",
				"Key123": "value2",
				"myKey":  "value3",
				"KEY":    "value4",
			},
			wantErr: false,
		},
		{
			name: "nested invalid keys",
			data: map[string]any{
				"valid": map[string]any{
					"invalid.nested": "value",
				},
			},
			wantErr:        true,
			invalidKeys:    []string{"invalid.nested"},
			errMustContain: []string{"invalid", "key", "invalid.nested"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
			}

			_, err := ToTfvars(snapshot)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error for invalid key(s), got nil")
				}

				// Check that error message contains expected substrings
				errMsg := strings.ToLower(err.Error())
				for _, substr := range tt.errMustContain {
					if !strings.Contains(errMsg, strings.ToLower(substr)) {
						t.Errorf("error message should contain %q, got: %v", substr, err)
					}
				}

				// Verify that all invalid keys are mentioned in the error
				for _, invalidKey := range tt.invalidKeys {
					if !strings.Contains(err.Error(), invalidKey) {
						t.Errorf("error message should mention invalid key %q, got: %v", invalidKey, err)
					}
				}
			} else if err != nil {
				t.Errorf("expected no error for valid keys, got: %v", err)
			}
		})
	}
}

// TestToTfvars_TypePreservation tests that different data types are correctly// represented in HCL .tfvars format with proper syntax.
//
// T024: Type preservation test (numbers, bools, strings correctly formatted)
func TestToTfvars_TypePreservation(t *testing.T) {
	tests := []struct {
		name              string
		data              map[string]any
		expectedSnippets  []string // Substrings expected in output
		forbiddenSnippets []string // Substrings that should NOT appear
	}{
		{
			name: "string values",
			data: map[string]any{
				"region": "us-west-2",
				"name":   "my-instance",
			},
			expectedSnippets: []string{
				`region = "us-west-2"`,
				`name = "my-instance"`,
			},
		},
		{
			name: "integer values",
			data: map[string]any{
				"count": 42,
				"port":  8080,
			},
			expectedSnippets: []string{
				`count = 42`,
				`port = 8080`,
			},
			forbiddenSnippets: []string{
				`"42"`, // Numbers should not be quoted
				`"8080"`,
			},
		},
		{
			name: "float values",
			data: map[string]any{
				"price": 3.14,
				"ratio": 0.5,
			},
			expectedSnippets: []string{
				`price = 3.14`,
				`ratio = 0.5`,
			},
			forbiddenSnippets: []string{
				`"3.14"`,
				`"0.5"`,
			},
		},
		{
			name: "boolean values",
			data: map[string]any{
				"enabled":  true,
				"disabled": false,
			},
			expectedSnippets: []string{
				`enabled = true`,
				`disabled = false`,
			},
			forbiddenSnippets: []string{
				`"true"`, // Bools should not be quoted
				`"false"`,
			},
		},
		{
			name: "list values",
			data: map[string]any{
				"zones": []any{"us-west-2a", "us-west-2b", "us-west-2c"},
				"ports": []any{80, 443, 8080},
			},
			expectedSnippets: []string{
				`zones = [`,
				`"us-west-2a"`,
				`"us-west-2b"`,
				`"us-west-2c"`,
				`ports = [`,
				`80`,
				`443`,
				`8080`,
			},
		},
		{
			name: "map values",
			data: map[string]any{
				"tags": map[string]any{
					"Name":        "my-resource",
					"Environment": "production",
				},
			},
			expectedSnippets: []string{
				`tags = {`,
				`Name = "my-resource"`,
				`Environment = "production"`,
				`}`,
			},
		},
		{
			name: "mixed types",
			data: map[string]any{
				"string_val": "text",
				"int_val":    123,
				"float_val":  45.67,
				"bool_val":   true,
				"list_val":   []any{1, 2, 3},
				"map_val":    map[string]any{"key": "value"},
			},
			expectedSnippets: []string{
				`string_val = "text"`,
				`int_val = 123`,
				`float_val = 45.67`,
				`bool_val = true`,
				`list_val = [`,
				`map_val = {`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
			}

			output, err := ToTfvars(snapshot)
			if err != nil {
				t.Fatalf("ToTfvars failed: %v", err)
			}

			outputStr := string(output)

			// Check that expected snippets are present
			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(outputStr, snippet) {
					t.Errorf("expected output to contain %q", snippet)
					t.Logf("Output:\n%s", outputStr)
				}
			}

			// Check that forbidden snippets are absent
			for _, snippet := range tt.forbiddenSnippets {
				if strings.Contains(outputStr, snippet) {
					t.Errorf("output should NOT contain %q (type confusion)", snippet)
					t.Logf("Output:\n%s", outputStr)
				}
			}
		})
	}
}

// TestToTfvars_NestedStructures tests that deeply nested hierarchies
// (maps within maps, lists within maps, etc.) are correctly serialized
// with proper HCL syntax and indentation.
//
// T025: Nested structures test (objects and arrays)
func TestToTfvars_NestedStructures(t *testing.T) {
	tests := []struct {
		name             string
		data             map[string]any
		expectedSnippets []string
	}{
		{
			name: "nested maps (3 levels)",
			data: map[string]any{
				"vpc": map[string]any{
					"config": map[string]any{
						"cidr": "10.0.0.0/16",
						"tags": map[string]any{
							"Name": "main-vpc",
							"Env":  "prod",
						},
					},
				},
			},
			expectedSnippets: []string{
				`vpc = {`,
				`config = {`,
				`cidr = "10.0.0.0/16"`,
				`tags = {`,
				`Name = "main-vpc"`,
				`Env = "prod"`,
				`}`,
			},
		},
		{
			name: "deeply nested maps (4 levels)",
			data: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"level4": map[string]any{
								"value": "deep",
							},
						},
					},
				},
			},
			expectedSnippets: []string{
				`level1 = {`,
				`level2 = {`,
				`level3 = {`,
				`level4 = {`,
				`value = "deep"`,
			},
		},
		{
			name: "list of maps",
			data: map[string]any{
				"instances": []any{
					map[string]any{
						"type": "t2.micro",
						"zone": "us-west-2a",
					},
					map[string]any{
						"type": "t2.small",
						"zone": "us-west-2b",
					},
				},
			},
			expectedSnippets: []string{
				`instances = [`,
				`{`,
				`type = "t2.micro"`,
				`zone = "us-west-2a"`,
				`}`,
				`type = "t2.small"`,
				`zone = "us-west-2b"`,
			},
		},
		{
			name: "map containing lists",
			data: map[string]any{
				"security_groups": map[string]any{
					"web": []any{80, 443},
					"app": []any{8080, 8443},
				},
			},
			expectedSnippets: []string{
				`security_groups = {`,
				`web = [`,
				`80`,
				`443`,
				`]`,
				`app = [`,
				`8080`,
				`8443`,
			},
		},
		{
			name: "complex nested structure",
			data: map[string]any{
				"infrastructure": map[string]any{
					"region": "us-west-2",
					"vpcs": []any{
						map[string]any{
							"cidr": "10.0.0.0/16",
							"subnets": []any{
								map[string]any{
									"cidr": "10.0.1.0/24",
									"zone": "us-west-2a",
								},
								map[string]any{
									"cidr": "10.0.2.0/24",
									"zone": "us-west-2b",
								},
							},
						},
					},
				},
			},
			expectedSnippets: []string{
				`infrastructure = {`,
				`region = "us-west-2"`,
				`vpcs = [`,
				`{`,
				`cidr = "10.0.0.0/16"`,
				`subnets = [`,
				`cidr = "10.0.1.0/24"`,
				`zone = "us-west-2a"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
			}

			output, err := ToTfvars(snapshot)
			if err != nil {
				t.Fatalf("ToTfvars failed: %v", err)
			}

			outputStr := string(output)

			// Check that all expected snippets are present
			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(outputStr, snippet) {
					t.Errorf("expected output to contain %q", snippet)
				}
			}

			if t.Failed() {
				t.Logf("Full output:\n%s", outputStr)
			}
		})
	}
}

// TestToTfvars_ErrorMessageQuality tests that error messages for unsupported
// types include both the type name and the source location/path where the
// unsupported value was found.
//
// T026a: Error message quality test (unsupported types include type name and source location)
func TestToTfvars_ErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name             string
		data             map[string]any
		expectedTypeName string // Type name that should appear in error
		expectedPath     string // Path that should appear in error (e.g., "data.config.handler")
	}{
		{
			name: "function at top level",
			data: map[string]any{
				"handler": func() {},
			},
			expectedTypeName: "func",
			expectedPath:     "handler",
		},
		{
			name: "channel at top level",
			data: map[string]any{
				"channel": make(chan int),
			},
			expectedTypeName: "chan",
			expectedPath:     "channel",
		},
		{
			name: "complex number at top level",
			data: map[string]any{
				"complex_num": complex(1, 2),
			},
			expectedTypeName: "complex",
			expectedPath:     "complex_num",
		},
		{
			name: "function in nested map",
			data: map[string]any{
				"config": map[string]any{
					"callbacks": map[string]any{
						"onSuccess": func() {},
					},
				},
			},
			expectedTypeName: "func",
			expectedPath:     "config.callbacks.onSuccess",
		},
		{
			name: "channel in nested structure",
			data: map[string]any{
				"settings": map[string]any{
					"messaging": map[string]any{
						"queue": make(chan string),
					},
				},
			},
			expectedTypeName: "chan",
			expectedPath:     "settings.messaging.queue",
		},
		{
			name: "function in list",
			data: map[string]any{
				"handlers": []any{
					"valid",
					func() {},
				},
			},
			expectedTypeName: "func",
			expectedPath:     "handlers[1]", // Or "handlers.1" depending on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := compiler.Snapshot{
				Data: tt.data,
			}

			_, err := ToTfvars(snapshot)
			if err == nil {
				t.Fatal("expected error for unsupported type, got nil")
			}

			errMsg := err.Error()

			// Check that error message contains the type name
			if !strings.Contains(errMsg, tt.expectedTypeName) {
				t.Errorf("error message should contain type name %q, got: %v", tt.expectedTypeName, err)
			}

			// Check that error message contains path information
			// Accept flexible path formats (dots, brackets, etc.)
			pathParts := strings.Split(tt.expectedPath, ".")
			foundAllParts := true
			for _, part := range pathParts {
				// Remove array notation if present (e.g., "[1]" -> "1")
				cleanPart := strings.Trim(part, "[]")
				if !strings.Contains(errMsg, cleanPart) {
					foundAllParts = false
					break
				}
			}

			if !foundAllParts {
				t.Errorf("error message should contain path %q (or similar), got: %v", tt.expectedPath, err)
			}

			// Verify error message contains "unsupported" or similar indicator
			errMsgLower := strings.ToLower(errMsg)
			if !strings.Contains(errMsgLower, "unsupported") &&
				!strings.Contains(errMsgLower, "invalid") &&
				!strings.Contains(errMsgLower, "cannot") {
				t.Errorf("error message should indicate unsupported type, got: %v", err)
			}
		})
	}
}
