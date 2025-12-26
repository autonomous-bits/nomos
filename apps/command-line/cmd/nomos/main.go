// Package main provides the nomos CLI entry point.
package main

import (
	"fmt"
	"os"
)

func main() {
	// Setup color output based on flags
	setupColorOutput()

	// Execute root command
	if err := Execute(); err != nil {
		// Cobra already prints the error, but we control the exit code
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
