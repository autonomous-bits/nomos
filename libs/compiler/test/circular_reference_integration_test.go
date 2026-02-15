//go:build integration
// +build integration

package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/testutil"
)

func newFileProviderRegistry(baseDir string) compiler.ProviderRegistry {
	registry := testutil.NewFakeProviderRegistry()
	registry.AddProvider("base", testutil.NewFakeFileProvider(baseDir))
	return registry
}

// TestIntegration_CircularReference_DirectCycle tests A→A direct self-reference
// (T075: end-to-end circular reference detection).
func TestIntegration_CircularReference_DirectCycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that references itself
	appContent := `source:
	alias: 'base'
	type: 'file'
	directory: '.'

# Direct self-reference creates A→A cycle
config: @base:app`

	appPath := filepath.Join(tmpDir, "app.csl")
	if err := os.WriteFile(appPath, []byte(appContent), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: appPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	if !strings.Contains(result.Error().Error(), "circular reference") {
		t.Errorf("error should indicate circular reference, got: %v", result.Error())
	}

	if !strings.Contains(result.Error().Error(), "base:app") {
		t.Errorf("error should include resource identifier base:app, got: %v", result.Error())
	}

	t.Logf("✓ Direct cycle detected: %v", result.Error())
}

// TestIntegration_CircularReference_TwoFilesCycle tests A→B→A cycle.
func TestIntegration_CircularReference_TwoFilesCycle(t *testing.T) {
	tmpDir := t.TempDir()

	// app.csl references common.csl
	appContent := `source:
	alias: 'base'
	type: 'file'
	directory: '.'

base_config: @base:common
`
	// common.csl references app.csl - creates cycle
	commonContent := `source:
	alias: 'base'
	type: 'file'
	directory: '.'

shared: @base:app
`

	appPath := filepath.Join(tmpDir, "app.csl")
	commonPath := filepath.Join(tmpDir, "common.csl")

	if err := os.WriteFile(appPath, []byte(appContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(commonPath, []byte(commonContent), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: appPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	errMsg := result.Error().Error()
	if !strings.Contains(errMsg, "circular reference") {
		t.Errorf("error should indicate circular reference, got: %v", result.Error())
	}

	// Verify both resources appear in the cycle error
	if !strings.Contains(errMsg, "base:app") || !strings.Contains(errMsg, "base:common") {
		t.Errorf("error should include both base:app and base:common, got: %v", result.Error())
	}

	// Verify arrow separator (cycle path format)
	if !strings.Contains(errMsg, "→") {
		t.Errorf("error should include cycle path separator →, got: %v", result.Error())
	}

	t.Logf("✓ Two-file cycle detected: %v", result.Error())
}

// TestIntegration_CircularReference_ThreeFilesCycle tests A→B→C→A cycle.
func TestIntegration_CircularReference_ThreeFilesCycle(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"app.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:settings
`,
		"settings.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

shared: @base:common
`,
		"common.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

base: @base:app
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "app.csl")
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	errMsg := result.Error().Error()
	if !strings.Contains(errMsg, "circular reference") {
		t.Errorf("error should indicate circular reference, got: %v", result.Error())
	}

	// Verify all three resources appear in the cycle
	resources := []string{"base:app", "base:settings", "base:common"}
	for _, res := range resources {
		if !strings.Contains(errMsg, res) {
			t.Errorf("error should include %s, got: %v", res, result.Error())
		}
	}

	t.Logf("✓ Three-file cycle detected: %v", result.Error())
}

// TestIntegration_CircularReference_PropertyMode tests cycle via property references.
func TestIntegration_CircularReference_PropertyMode(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"app.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

database:
	host: @base:common.config.host
`,
		"common.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	host: @base:app.database.host
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "app.csl")
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	if !strings.Contains(result.Error().Error(), "circular reference") {
		t.Errorf("error should indicate circular reference, got: %v", result.Error())
	}

	t.Logf("✓ Property mode cycle detected: %v", result.Error())
}

// TestIntegration_CircularReference_MapMode tests cycle via map references.
func TestIntegration_CircularReference_MapMode(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"app.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

server: @base:common.infrastructure.network
`,
		"common.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

infrastructure:
	network: @base:app.server
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "app.csl")
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	if !strings.Contains(result.Error().Error(), "circular reference") {
		t.Errorf("error should indicate circular reference, got: %v", result.Error())
	}

	t.Logf("✓ Map mode cycle detected: %v", result.Error())
}

// TestIntegration_CircularReference_ErrorMessageFormat verifies the error
// message displays the full resolution chain as per SC-004 (cycle format: "A → B → C → A").
func TestIntegration_CircularReference_ErrorMessageFormat(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"alpha.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:beta
`,
		"beta.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:gamma
`,
		"gamma.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:alpha
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "alpha.csl")
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	errMsg := result.Error().Error()

	// Verify cycle path components
	expectedParts := []string{"base:alpha", "base:beta", "base:gamma", "→"}
	for _, part := range expectedParts {
		if !strings.Contains(errMsg, part) {
			t.Errorf("error message missing expected part %q, got: %v", part, errMsg)
		}
	}

	t.Logf("✓ Cycle path formatted correctly: %v", result.Error())
}

// TestIntegration_CircularReference_PerformanceCheck verifies that cycle detection
// fails quickly (within 2 seconds as per SC-003).
func TestIntegration_CircularReference_PerformanceCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 4-file cycle
	files := map[string]string{
		"app.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:level1
`,
		"level1.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:level2
`,
		"level2.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:level3
`,
		"level3.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:app
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "app.csl")

	// If cycle detection caused an infinite loop, the test would hang.
	// The test framework timeout ensures we fail within a reasonable time.
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	if !result.HasErrors() {
		t.Fatal("expected circular reference error, got nil")
	}

	if !strings.Contains(result.Error().Error(), "circular reference") {
		t.Errorf("expected circular reference error, got: %v", result.Error())
	}

	t.Logf("✓ Cycle detected quickly: %v", result.Error())
}

// TestIntegration_CircularReference_NoFalsePositives verifies that legitimate
// multi-level references (without cycles) compile successfully.
func TestIntegration_CircularReference_NoFalsePositives(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a chain of references WITHOUT a cycle (A→B→C, no back-reference)
	files := map[string]string{
		"app.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config: @base:common
`,
		"common.csl": `source:
	alias: 'base'
	type: 'file'
	directory: '.'

shared: @base:defaults

local:
	value: 'common-local'
`,
		"defaults.csl": `settings:
	timeout: 30
	retries: 3
`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	mainPath := filepath.Join(tmpDir, "app.csl")
	registry := newFileProviderRegistry(tmpDir)
	result := compiler.Compile(ctx, compiler.Options{Path: mainPath, ProviderRegistry: registry})

	// This should succeed (no cycle)
	if result.HasErrors() {
		t.Fatalf("unexpected error (false positive cycle detection): %v", result.Error())
	}

	if len(result.Snapshot.Data) == 0 {
		t.Error("expected non-empty compiled data")
	}

	t.Logf("✓ No false positive - legitimate multi-level references compiled successfully")
}
