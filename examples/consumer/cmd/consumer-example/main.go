// Package main provides a simple example showing how to use the Nomos compiler
// library as an external consumer after the modules are published.
//
// This example demonstrates:
// - Importing Nomos libraries using versioned require directives
// - Using the compiler.Compile function to process Nomos configurations
// - Proper error handling and result marshaling
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: consumer-example <config-file>")
		fmt.Println("\nExample consumer demonstrating Nomos library usage")
		os.Exit(1)
	}

	configPath := os.Args[1]

	fmt.Printf("Compiling configuration: %s\n", configPath)

	// Create compilation options
	// Note: In production, you would typically configure a provider registry
	opts := compiler.Options{
		Path:                 configPath,
		ProviderRegistry:     nil, // No providers for this simple example
		ProviderTypeRegistry: nil,
		Timeouts: compiler.OptionsTimeouts{
			PerProviderFetch:       30 * time.Second,
			MaxConcurrentProviders: 5,
		},
	}

	// Compile the configuration
	ctx := context.Background()
	result := compiler.Compile(ctx, opts)
	if result.HasErrors() {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", result.Error())
		os.Exit(1)
	}

	snapshot := result.Snapshot

	// Marshal the result to JSON for display
	output, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal result: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nCompilation successful!")
	fmt.Println(string(output))
}
