// Package main implements the build command for the Nomos CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/diagnostics"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/options"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/providercmd"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/serialize"
	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/spf13/cobra"
)

// buildFlags holds all flags for the build command
var buildFlags struct {
	path                   string
	format                 string
	out                    string
	vars                   []string
	strict                 bool
	allowMissingProvider   bool
	timeoutPerProvider     string
	maxConcurrentProviders int
	verbose                bool
	forceProviders         bool
	dryRun                 bool
}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Nomos .csl files into configuration snapshots",
	Long: `Build compiles Nomos .csl configuration scripts into versioned snapshots.

The build command discovers .csl files in the specified path (file or directory),
compiles them using the Nomos compiler, and produces deterministic JSON output.

File Discovery:
  - If --path points to a file, only that file is compiled
  - If --path points to a directory, all .csl files are discovered recursively
  - Files are processed in UTF-8 lexicographic order (deterministic builds)

Provider Management:
  - Providers are automatically discovered and downloaded during build
  - Cached providers are reused for speed (SHA256-verified)
  - Use --force-providers to force re-download of all providers
  - Use --dry-run to preview provider operations without executing
  - Use --allow-missing-provider to tolerate missing providers (non-deterministic)

Exit Codes:
  0 - Success
  1 - Compilation errors (or warnings in strict mode)
  2 - Invalid usage or flags`,
	RunE: buildCommand,
}

func init() {
	// Required flags
	buildCmd.Flags().StringVarP(&buildFlags.path, "path", "p", "", "Path to .csl file or directory (required)")
	_ = buildCmd.MarkFlagRequired("path") // Error only occurs if flag doesn't exist

	// Output flags
	buildCmd.Flags().StringVarP(&buildFlags.format, "format", "f", "json", "Output format (only json currently supported)")
	buildCmd.Flags().StringVarP(&buildFlags.out, "out", "o", "", "Output file (default: stdout)")

	// Configuration flags
	buildCmd.Flags().StringSliceVar(&buildFlags.vars, "var", nil, "Set variable: key=value (repeatable)")
	buildCmd.Flags().BoolVar(&buildFlags.strict, "strict", false, "Treat warnings as errors")

	// Provider flags
	buildCmd.Flags().BoolVar(&buildFlags.allowMissingProvider, "allow-missing-provider", false, "Allow compilation with missing providers")
	buildCmd.Flags().StringVar(&buildFlags.timeoutPerProvider, "timeout-per-provider", "30s", "Timeout for provider operations (e.g., 5s, 1m)")
	buildCmd.Flags().IntVar(&buildFlags.maxConcurrentProviders, "max-concurrent-providers", 4, "Max concurrent provider operations")
	buildCmd.Flags().BoolVar(&buildFlags.forceProviders, "force-providers", false, "Force re-download of all providers")
	buildCmd.Flags().BoolVar(&buildFlags.dryRun, "dry-run", false, "Preview provider operations without executing")

	// Debug flags
	buildCmd.Flags().BoolVarP(&buildFlags.verbose, "verbose", "v", false, "Enable verbose output")
}

// buildCommand executes the build subcommand.
func buildCommand(_ *cobra.Command, _ []string) error {
	// Phase 0: Provider Management (before compilation)
	// Convert build flags to provider options
	providerFlags := providercmd.BuildFlags{
		Path:                   buildFlags.path,
		ForceProviders:         buildFlags.forceProviders,
		DryRun:                 buildFlags.dryRun,
		TimeoutPerProvider:     buildFlags.timeoutPerProvider,
		MaxConcurrentProviders: buildFlags.maxConcurrentProviders,
		AllowMissingProvider:   buildFlags.allowMissingProvider,
	}

	providerOpts, err := providercmd.NewProviderOptionsFromBuildFlags(providerFlags)
	if err != nil {
		return fmt.Errorf("invalid provider options: %w", err)
	}

	// Ensure providers are available (discover, download, validate)
	providerSummary, err := providercmd.EnsureProviders(providerOpts)
	if err != nil {
		return fmt.Errorf("provider management failed: %w", err)
	}

	// Print provider summary unless quiet
	if !globalFlags.quiet && providerSummary != nil {
		fmt.Fprintf(os.Stderr, "%s\n", providerSummary.String())
	}

	// If dry-run mode, exit successfully after showing provider summary
	if buildFlags.dryRun {
		return nil
	}

	// Validate format
	if buildFlags.format != "json" {
		return fmt.Errorf("invalid format %q, only json is currently supported", buildFlags.format)
	}

	// Create provider registries (supports external providers via lockfile)
	providerRegistry, providerTypeRegistry := options.NewProviderRegistries()

	// Build compiler options
	opts, err := options.BuildOptions(options.BuildParams{
		Path:                   buildFlags.path,
		Vars:                   buildFlags.vars,
		TimeoutPerProvider:     buildFlags.timeoutPerProvider,
		MaxConcurrentProviders: buildFlags.maxConcurrentProviders,
		AllowMissingProvider:   buildFlags.allowMissingProvider,
		ProviderRegistry:       providerRegistry,
		ProviderTypeRegistry:   providerTypeRegistry,
	})
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Call compiler
	ctx := context.Background()
	result := compiler.Compile(ctx, opts)

	snapshot := result.Snapshot
	var compileErr error
	if result.HasErrors() {
		compileErr = result.Error()
	}

	// Create diagnostics formatter
	useColor := shouldUseColor()
	formatter := diagnostics.NewFormatter(useColor)

	// Handle diagnostics
	hasErrors := len(snapshot.Metadata.Errors) > 0
	hasWarnings := len(snapshot.Metadata.Warnings) > 0

	// Print warnings (unless quiet)
	if hasWarnings && !globalFlags.quiet {
		formatter.PrintWarnings(os.Stderr, snapshot.Metadata.Warnings)
	}

	// Print errors (unless quiet)
	if hasErrors && !globalFlags.quiet {
		formatter.PrintErrors(os.Stderr, snapshot.Metadata.Errors)
	}

	// Print validation summary (unless quiet)
	if !globalFlags.quiet && (hasErrors || hasWarnings) {
		fmt.Fprintf(os.Stderr, "\n")
		switch {
		case hasErrors && hasWarnings:
			fmt.Fprintf(os.Stderr, "Compilation failed: %d error(s), %d warning(s)\n",
				len(snapshot.Metadata.Errors), len(snapshot.Metadata.Warnings))
		case hasErrors:
			fmt.Fprintf(os.Stderr, "Compilation failed: %d error(s)\n", len(snapshot.Metadata.Errors))
		case hasWarnings && buildFlags.strict:
			fmt.Fprintf(os.Stderr, "Compilation failed: %d warning(s) (strict mode)\n", len(snapshot.Metadata.Warnings))
		}
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
	if buildFlags.strict && hasWarnings {
		return fmt.Errorf("compilation completed with warnings (strict mode)")
	}

	// Serialize output based on format
	output, err := serializeSnapshot(snapshot, buildFlags.format)
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	// Write output
	if buildFlags.out != "" {
		// Ensure output directory exists
		dir := filepath.Dir(buildFlags.out)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0750); err != nil {
				return fmt.Errorf("cannot create output directory: %w", err)
			}
		}

		// Write file
		if err := os.WriteFile(buildFlags.out, output, 0600); err != nil {
			return fmt.Errorf("cannot write output file: %w", err)
		}

		if !globalFlags.quiet {
			fmt.Fprintf(os.Stderr, "Output written to %s\n", buildFlags.out)
		}
	} else {
		// Write to stdout
		fmt.Println(string(output))
	}

	return nil
}

// serializeSnapshot serializes a snapshot to the requested format.
// Currently only JSON is supported.
func serializeSnapshot(snapshot compiler.Snapshot, format string) ([]byte, error) {
	if format != "json" {
		return nil, fmt.Errorf("unsupported format: %s (only json is currently supported)", format)
	}
	return serialize.ToJSON(snapshot)
}

// shouldUseColor determines whether to colorize output based on flags and terminal
func shouldUseColor() bool {
	switch globalFlags.color {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		// Check if stderr is a terminal
		if fileInfo, err := os.Stderr.Stat(); err == nil {
			return (fileInfo.Mode() & os.ModeCharDevice) != 0
		}
		return false
	default:
		return false
	}
}
