package parser_test

import (
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// TestExamplesCompatibility verifies all example configs parse successfully (backward compatibility)
func TestExamplesCompatibility(t *testing.T) {
	files := []string{
		"../../examples/config/config.csl",
		"../../examples/config/config2.csl",
		"../../examples/config/config-from-remote-state.csl",
		"../../examples/config/test-deeply-nested.csl",
		"../../examples/config/test-final.csl",
		"../../examples/config/test-no-provider.csl",
		"../../examples/config/test-provider.csl",
		"../../examples/config/test-scalars.csl",
		"../../examples/config/test-simple.csl",
		"../../examples/config/test-source.csl",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			_, err := parser.ParseFile(file)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", file, err)
			}
		})
	}
}
