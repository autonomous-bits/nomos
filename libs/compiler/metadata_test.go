package compiler_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestMetadata_JSONSerialization tests that Metadata can be marshaled and unmarshaled as JSON.
func TestMetadata_JSONSerialization(t *testing.T) {
	// Arrange
	startTime := time.Date(2025, 10, 26, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 26, 10, 0, 5, 0, time.UTC)

	metadata := compiler.Metadata{
		InputFiles:      []string{"/path/to/app.csl", "/path/to/config.csl"},
		ProviderAliases: []string{"file", "env"},
		StartTime:       startTime,
		EndTime:         endTime,
		Errors:          []string{"error: unresolved reference"},
		Warnings:        []string{"warning: provider fetch failed"},
		PerKeyProvenance: map[string]compiler.Provenance{
			"app": {
				Source:        "/path/to/app.csl",
				ProviderAlias: "",
			},
			"database": {
				Source:        "/path/to/config.csl",
				ProviderAlias: "env",
			},
		},
	}

	// Act - Marshal to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal Metadata to JSON: %v", err)
	}

	// Assert - JSON should be non-empty
	if len(jsonData) == 0 {
		t.Fatal("Marshaled JSON is empty")
	}

	// Act - Unmarshal back
	var unmarshaled compiler.Metadata
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to Metadata: %v", err)
	}

	// Assert - Values should match
	if len(unmarshaled.InputFiles) != 2 {
		t.Errorf("Expected 2 input files, got %d", len(unmarshaled.InputFiles))
	}
	if unmarshaled.InputFiles[0] != "/path/to/app.csl" {
		t.Errorf("Expected first input file '/path/to/app.csl', got '%s'", unmarshaled.InputFiles[0])
	}

	if len(unmarshaled.ProviderAliases) != 2 {
		t.Errorf("Expected 2 provider aliases, got %d", len(unmarshaled.ProviderAliases))
	}
	if unmarshaled.ProviderAliases[0] != "file" {
		t.Errorf("Expected first provider alias 'file', got '%s'", unmarshaled.ProviderAliases[0])
	}

	if !unmarshaled.StartTime.Equal(startTime) {
		t.Errorf("Expected start time %v, got %v", startTime, unmarshaled.StartTime)
	}

	if !unmarshaled.EndTime.Equal(endTime) {
		t.Errorf("Expected end time %v, got %v", endTime, unmarshaled.EndTime)
	}

	if len(unmarshaled.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(unmarshaled.Errors))
	}

	if len(unmarshaled.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(unmarshaled.Warnings))
	}

	if len(unmarshaled.PerKeyProvenance) != 2 {
		t.Errorf("Expected 2 provenance entries, got %d", len(unmarshaled.PerKeyProvenance))
	}

	appProv, ok := unmarshaled.PerKeyProvenance["app"]
	if !ok {
		t.Error("Expected provenance entry for 'app' key")
	} else {
		if appProv.Source != "/path/to/app.csl" {
			t.Errorf("Expected app source '/path/to/app.csl', got '%s'", appProv.Source)
		}
	}

	dbProv, ok := unmarshaled.PerKeyProvenance["database"]
	if !ok {
		t.Error("Expected provenance entry for 'database' key")
	} else {
		if dbProv.ProviderAlias != "env" {
			t.Errorf("Expected database provider alias 'env', got '%s'", dbProv.ProviderAlias)
		}
	}
}

// TestProvenance_JSONSerialization tests that Provenance can be marshaled and unmarshaled as JSON.
func TestProvenance_JSONSerialization(t *testing.T) {
	// Arrange
	provenance := compiler.Provenance{
		Source:        "/path/to/source.csl",
		ProviderAlias: "git",
	}

	// Act - Marshal to JSON
	jsonData, err := json.Marshal(provenance)
	if err != nil {
		t.Fatalf("Failed to marshal Provenance to JSON: %v", err)
	}

	// Assert - JSON should contain expected fields
	jsonStr := string(jsonData)
	if !contains(jsonStr, "source") && !contains(jsonStr, "Source") {
		t.Errorf("Expected JSON to contain 'source' or 'Source' field, got: %s", jsonStr)
	}

	if !contains(jsonStr, "provider_alias") && !contains(jsonStr, "ProviderAlias") {
		t.Errorf("Expected JSON to contain 'provider_alias' or 'ProviderAlias' field, got: %s", jsonStr)
	}

	// Act - Unmarshal back
	var unmarshaled compiler.Provenance
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to Provenance: %v", err)
	}

	// Assert - Values should match
	if unmarshaled.Source != "/path/to/source.csl" {
		t.Errorf("Expected source '/path/to/source.csl', got '%s'", unmarshaled.Source)
	}

	if unmarshaled.ProviderAlias != "git" {
		t.Errorf("Expected provider alias 'git', got '%s'", unmarshaled.ProviderAlias)
	}
}

// TestMetadata_JSONFieldNames tests that JSON field names follow snake_case convention.
func TestMetadata_JSONFieldNames(t *testing.T) {
	// Arrange
	metadata := compiler.Metadata{
		InputFiles:      []string{"test.csl"},
		ProviderAliases: []string{"file"},
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Errors:          []string{},
		Warnings:        []string{},
		PerKeyProvenance: map[string]compiler.Provenance{
			"key": {Source: "test.csl"},
		},
	}

	// Act
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal Metadata: %v", err)
	}

	jsonStr := string(jsonData)

	// Assert - Check for snake_case field names as per Go JSON conventions
	expectedFields := []string{
		"input_files",
		"provider_aliases",
		"start_time",
		"end_time",
		"per_key_provenance",
		"errors",
		"warnings",
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("Expected JSON to contain field '%s', got: %s", field, jsonStr)
		}
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
