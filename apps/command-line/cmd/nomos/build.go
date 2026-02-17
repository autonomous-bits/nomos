// Package main implements the build command for the Nomos CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/diagnostics"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/options"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/providercmd"
	"github.com/autonomous-bits/nomos/apps/command-line/internal/serialize"
	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/pkg/encryption"
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
	includeMetadata        bool
	encryptionKey          string
}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile Nomos .csl files into configuration snapshots",
	Long: `Build compiles Nomos .csl configuration scripts into versioned snapshots.

The build command discovers .csl files in the specified path (file or directory),
compiles them using the Nomos compiler, and produces deterministic output in
JSON, YAML, or Terraform .tfvars format.

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

Output Formats:
  json   - Canonical JSON with sorted keys (default)
  yaml   - YAML 1.2 format for Kubernetes, Ansible, Docker Compose
  tfvars - Terraform .tfvars format (HCL syntax)

Metadata Control:
  By default, output contains only configuration data (clean, minimal).
  Use --include-metadata to add compilation metadata for debugging:
    - Compiler version and timestamps
    - Source file list
    - Per-key provenance (which file defined each key)
    - Provider aliases used

  Default (no metadata):
    {"app": "example", "env": "prod"}

  With --include-metadata:
    {"data": {"app": "example", "env": "prod"}, "metadata": {...}}

Examples:
  # Compile to JSON (default)
  nomos build -p config.csl -o output.json

  # Compile to YAML for Kubernetes
  nomos build -p config.csl --format yaml -o deployment.yaml
  kubectl apply -f deployment.yaml

  # Compile to Terraform .tfvars
  nomos build -p config.csl --format tfvars -o terraform.tfvars
  terraform apply -var-file=terraform.tfvars

  # Automatic extension handling
  nomos build -p config.csl --format yaml -o config
  # Creates: config.yaml

  # Multiple environments
  nomos build -p envs/prod.csl --format tfvars -o prod.auto.tfvars
  nomos build -p k8s/app.csl --format yaml -o deployment.yaml

Format Validation:
  - YAML: Keys cannot contain null bytes (\x00)
  - Tfvars: Keys must match HCL identifier pattern [a-zA-Z_][a-zA-Z0-9_-]*
  - Invalid keys cause compilation errors with clear messages

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
	buildCmd.Flags().StringVarP(&buildFlags.format, "format", "f", "json", "Output format: json, yaml, or tfvars")
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

	// Output flags
	buildCmd.Flags().BoolVar(&buildFlags.includeMetadata, "include-metadata", false, "Include compilation metadata in output (timestamps, source files, provenance)")

	// Debug flags
	// Debug flags
	buildCmd.Flags().BoolVarP(&buildFlags.verbose, "verbose", "v", false, "Enable verbose output")

	// Encryption flags
	buildCmd.Flags().StringVar(&buildFlags.encryptionKey, "encryption-key", "", "Path to encryption key file (generated by 'nomos keys generate')")
}

// buildCommand executes the build subcommand.
func buildCommand(_ *cobra.Command, _ []string) error {
	// Validate flags
	if buildFlags.maxConcurrentProviders < 0 {
		return fmt.Errorf("max-concurrent-providers must be non-negative (got %d)", buildFlags.maxConcurrentProviders)
	}

	// Load encryption key if provided
	var encryptionKey []byte
	if buildFlags.encryptionKey != "" {
		var err error
		encryptionKey, err = encryption.LoadKey(buildFlags.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to load encryption key: %w", err)
		}
	}

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
		EncryptionKey:          encryptionKey,
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
	output, err := serializeSnapshot(snapshot, buildFlags.format, buildFlags.includeMetadata)
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	// Write output
	if buildFlags.out != "" {
		// Resolve output path with extension handling
		resolvedPath, err := resolveOutputPath(buildFlags.out, serialize.OutputFormat(strings.ToLower(buildFlags.format)))
		if err != nil {
			return fmt.Errorf("invalid output path: %w", err)
		}

		// Ensure output directory exists
		dir := filepath.Dir(resolvedPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0750); err != nil {
				return fmt.Errorf("cannot create output directory: %w", err)
			}
		}

		// Write file using resolved path
		if err := os.WriteFile(resolvedPath, output, 0600); err != nil {
			return fmt.Errorf("cannot write output file: %w", err)
		}

		if !globalFlags.quiet {
			fmt.Fprintf(os.Stderr, "Output written to %s\n", resolvedPath)
		}
	} else {
		// Write to stdout
		fmt.Println(string(output))
	}

	return nil
}

// serializeSnapshot serializes a snapshot to the requested format.
// Supported formats: json, yaml, tfvars
func serializeSnapshot(snapshot compiler.Snapshot, format string, includeMetadata bool) ([]byte, error) {
	// Normalize format to lowercase for case-insensitive matching
	normalizedFormat := strings.ToLower(format)

	switch serialize.OutputFormat(normalizedFormat) {
	case serialize.FormatJSON:
		return serialize.ToJSON(snapshot, includeMetadata)
	case serialize.FormatYAML:
		return serialize.ToYAML(snapshot, includeMetadata)
	case serialize.FormatTfvars:
		return serialize.ToTfvars(snapshot, includeMetadata)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, yaml, tfvars)", format)
	}
}

// resolveOutputPath resolves and validates the output file path.
// It automatically appends the format's default extension if the path
// has no extension. If the path already has an extension, it is preserved
// to respect the user's explicit choice.
//
// The function also validates that:
//   - The path is not empty or whitespace-only
//   - The path is not just an extension (e.g., ".json")
//   - The path does not end with a path separator (must be a file)
//   - The format is valid
//
// Extension Detection:
// Only recognized file extensions are treated as actual extensions.
// Unrecognized suffixes (like .prod, .v2) are treated as part of the
// filename and the format extension is appended.
//
// Parameters:
//   - outputPath: User-provided output path (may or may not have extension)
//   - format: Output format (json, yaml, tfvars)
//
// Returns:
//   - Fully resolved path with appropriate extension
//   - Error if path is invalid or format is unsupported
//
// Examples:
//   - resolveOutputPath("output", FormatYAML) → "output.yaml", nil
//   - resolveOutputPath("config.yml", FormatYAML) → "config.yml", nil
//   - resolveOutputPath("config.prod", FormatJSON) → "config.prod.json", nil
//   - resolveOutputPath("", FormatJSON) → "", error
func resolveOutputPath(outputPath string, format serialize.OutputFormat) (string, error) {
	// Validate path is not empty or whitespace-only
	trimmedPath := strings.TrimSpace(outputPath)
	if trimmedPath == "" {
		return "", fmt.Errorf("output path must not be empty")
	}

	// Check if original path ends with separator before cleaning
	// (filepath.Clean removes trailing slashes, so we check first)
	if strings.HasSuffix(trimmedPath, string(filepath.Separator)) || strings.HasSuffix(trimmedPath, "/") {
		return "", fmt.Errorf("output path must be a file, not a directory")
	}

	// Clean the path to normalize it
	cleanedPath := filepath.Clean(trimmedPath)

	// Get the base filename to check special cases
	base := filepath.Base(cleanedPath)

	// Check if it's just an extension (e.g., ".json", ".yaml", ".tfvars")
	// These are invalid because there's no actual filename
	if base == ".json" || base == ".yaml" || base == ".yml" || base == ".tfvars" {
		return "", fmt.Errorf("invalid output path: path cannot be just an extension")
	}

	// Validate format is supported
	if err := format.Validate(); err != nil {
		return "", err
	}

	// Check for special multi-part extension .auto.tfvars
	if strings.HasSuffix(cleanedPath, ".auto.tfvars") {
		return cleanedPath, nil
	}

	// Get the extension
	ext := filepath.Ext(cleanedPath)

	// Check if path has a recognized file extension
	// Standard file extensions that should always be preserved
	recognizedExtensions := map[string]bool{
		".json":   true,
		".yaml":   true,
		".yml":    true,
		".tfvars": true,
		".txt":    true,
		".conf":   true,
		".hcl":    true,
		".data":   true,
		".xml":    true,
		".backup": true,
	}

	// Extensions that are only preserved when they're the sole dot in the filename
	// (e.g., "app.config" preserves .config, but "app.v2.config" does not)
	// However, hidden files like ".config" (where base == ".config") are NOT treated
	// as having an extension - they should get the format extension appended.
	contextualExtensions := map[string]bool{
		".config": true,
	}

	// If we have a recognized extension, preserve it
	if ext != "" && recognizedExtensions[ext] {
		return cleanedPath, nil
	}

	// For contextual extensions, only preserve if:
	// 1. It's the only dot in the basename AND
	// 2. The base is not just the extension (i.e., not a hidden file like ".config")
	if ext != "" && contextualExtensions[ext] {
		dotCount := strings.Count(base, ".")
		if dotCount == 1 && base != ext {
			return cleanedPath, nil
		}
	}

	// No recognized extension - append format's default extension
	return cleanedPath + format.Extension(), nil
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
