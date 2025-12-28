// Package main implements the validate command for the Nomos CLI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/diagnostics"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/options"
	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/spf13/cobra"
)

// validateFlags holds flags for the validate command
var validateFlags struct {
	path    string
	verbose bool
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate .csl files without building",
	Long: `Validate checks .csl configuration files for syntax and semantic errors
without performing a full build.

This is useful for:
  - Pre-commit hooks
  - CI/CD pipelines
  - Quick syntax verification
  - Editor integrations

The validate command performs parsing and type checking but does not:
  - Invoke providers
  - Generate output snapshots
  - Perform provider resolution`,
	RunE: validateCommand,
}

func init() {
	validateCmd.Flags().StringVarP(&validateFlags.path, "path", "p", "", "Path to .csl file or directory (required)")
	_ = validateCmd.MarkFlagRequired("path") // Error only occurs if flag doesn't exist
	validateCmd.Flags().BoolVarP(&validateFlags.verbose, "verbose", "v", false, "Enable verbose output")
}

// validateCommand executes the validate subcommand.
func validateCommand(_ *cobra.Command, _ []string) error {
	// Create provider registries (validation-only, no actual providers needed)
	providerRegistry, providerTypeRegistry := options.NewProviderRegistries()

	// Build compiler options with validation-only mode
	opts, err := options.BuildOptions(options.BuildParams{
		Path:                 validateFlags.path,
		Vars:                 nil,  // No vars needed for validation
		AllowMissingProvider: true, // Don't require providers for validation
		ProviderRegistry:     providerRegistry,
		ProviderTypeRegistry: providerTypeRegistry,
	})
	if err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Call compiler (validation will happen during compilation)
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

	// Print validation summary
	if !globalFlags.quiet {
		fmt.Fprintf(os.Stderr, "\n")
		switch {
		case hasErrors && hasWarnings:
			fmt.Fprintf(os.Stderr, "Validation failed: %d error(s), %d warning(s)\n",
				len(snapshot.Metadata.Errors), len(snapshot.Metadata.Warnings))
		case hasErrors:
			fmt.Fprintf(os.Stderr, "Validation failed: %d error(s)\n", len(snapshot.Metadata.Errors))
		case hasWarnings:
			fmt.Fprintf(os.Stderr, "Validation passed with %d warning(s)\n", len(snapshot.Metadata.Warnings))
		default:
			fmt.Fprintf(os.Stderr, "Validation passed\n")
		}
	}

	// Check for fatal compile error
	if compileErr != nil {
		return fmt.Errorf("validation failed: %w", compileErr)
	}

	// If metadata has errors, exit with error code
	if hasErrors {
		return fmt.Errorf("validation completed with errors")
	}

	return nil
}
