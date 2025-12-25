// Package main implements the build command for the Nomos CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/diagnostics"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/flags"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/options"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/serialize"
	"github.com/autonomous-bits/nomos/libs/compiler"
)

// buildCommand executes the build subcommand.
func buildCommand(args []string) error {
	// Check for help flag before parsing
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printBuildHelp()
			os.Exit(0)
		}
	}

	// Parse flags
	buildFlags, err := flags.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		printBuildHelp()
		os.Exit(exitUsageErr)
	}

	// Create provider registries (supports external providers via lockfile)
	providerRegistry, providerTypeRegistry := options.NewProviderRegistries()

	// Build compiler options using the new options package
	opts, err := options.BuildOptions(options.BuildParams{
		Path:                   buildFlags.Path,
		Vars:                   buildFlags.Vars,
		TimeoutPerProvider:     buildFlags.TimeoutPerProvider,
		MaxConcurrentProviders: buildFlags.MaxConcurrentProviders,
		AllowMissingProvider:   buildFlags.AllowMissingProvider,
		ProviderRegistry:       providerRegistry,
		ProviderTypeRegistry:   providerTypeRegistry,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		printBuildHelp()
		os.Exit(exitUsageErr)
	}

	// Call compiler
	ctx := context.Background()
	snapshot, compileErr := compiler.Compile(ctx, opts)

	// Create diagnostics formatter (no color for now - can be enhanced later)
	formatter := diagnostics.NewFormatter(false)

	// Handle diagnostics
	hasErrors := len(snapshot.Metadata.Errors) > 0
	hasWarnings := len(snapshot.Metadata.Warnings) > 0

	// Print warnings
	if hasWarnings {
		formatter.PrintWarnings(os.Stderr, snapshot.Metadata.Warnings)
	}

	// Print errors
	if hasErrors {
		formatter.PrintErrors(os.Stderr, snapshot.Metadata.Errors)
	}

	// Check for fatal compile error
	if compileErr != nil {
		return fmt.Errorf("compilation failed: %w", compileErr)
	}

	// If metadata has errors, exit with error code
	if hasErrors {
		return fmt.Errorf("compilation completed with errors")
	}

	// If strict mode and warnings exist, exit with error code
	if buildFlags.Strict && hasWarnings {
		return fmt.Errorf("compilation completed with warnings (strict mode)")
	}

	// Serialize output based on format
	output, err := serializeSnapshot(snapshot, buildFlags.Format)
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	// Write output
	if buildFlags.Out != "" {
		// Ensure output directory exists
		dir := filepath.Dir(buildFlags.Out)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0750); err != nil {
				fmt.Fprintf(os.Stderr, "Error: cannot create output directory: %v\n\n", err)
				printBuildHelp()
				os.Exit(exitUsageErr)
			}
		}

		// Try to write file
		if err := os.WriteFile(buildFlags.Out, output, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot write output file: %v\n\n", err)
			printBuildHelp()
			os.Exit(exitUsageErr)
		}
	} else {
		fmt.Println(string(output))
	}

	return nil
}

// serializeSnapshot serializes a snapshot to the requested format.
func serializeSnapshot(snapshot compiler.Snapshot, format string) ([]byte, error) {
	switch format {
	case "json":
		return serialize.ToJSON(snapshot)
	case "yaml":
		return serialize.ToYAML(snapshot)
	case "hcl":
		return serialize.ToHCL(snapshot)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
