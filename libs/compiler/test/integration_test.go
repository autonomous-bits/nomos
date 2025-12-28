//go:build integration
// +build integration

package test

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/testutil"
)

// TestIntegration_SmokeTest verifies Compile works on an empty directory.
func TestIntegration_SmokeTest(t *testing.T) {
	// Create empty temp directory
	tmpDir := t.TempDir()

	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: testutil.NewFakeProviderRegistry(),
	}

	result := compiler.Compile(ctx, opts)
	if result.HasErrors() {
		t.Fatalf("expected no error, got %v", result.Error())
	}

	snapshot := result.Snapshot

	// Verify snapshot structure
	if snapshot.Data == nil {
		t.Error("snapshot.Data should not be nil")
	}

	if snapshot.Metadata.InputFiles == nil {
		t.Error("snapshot.Metadata.InputFiles should not be nil")
	}

	if len(snapshot.Metadata.InputFiles) != 0 {
		t.Errorf("expected 0 input files for empty directory, got %d", len(snapshot.Metadata.InputFiles))
	}

	if snapshot.Metadata.StartTime.IsZero() {
		t.Error("snapshot.Metadata.StartTime should be set")
	}

	if snapshot.Metadata.EndTime.IsZero() {
		t.Error("snapshot.Metadata.EndTime should be set")
	}

	if snapshot.Metadata.PerKeyProvenance == nil {
		t.Error("snapshot.Metadata.PerKeyProvenance should not be nil")
	}
}
