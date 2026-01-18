//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos/libs/parser"
)

func main() {
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

	passed := 0
	failed := 0

	for _, file := range files {
		_, err := parser.ParseFile(file)
		basename := filepath.Base(file)
		if err != nil {
			fmt.Printf("❌ FAIL: %s - %v\n", basename, err)
			failed++
		} else {
			fmt.Printf("✅ PASS: %s\n", basename)
			passed++
		}
	}

	fmt.Printf("\n%d passed, %d failed out of %d total\n", passed, failed, len(files))

	if failed > 0 {
		os.Exit(1)
	}
}
