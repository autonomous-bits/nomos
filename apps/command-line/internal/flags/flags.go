// Package flags provides CLI flag parsing and validation for the Nomos CLI.
package flags

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// BuildFlags holds parsed command-line flags for the build command.
type BuildFlags struct {
	// Path specifies the input file or directory (required).
	Path string

	// Format specifies the output format (json, yaml, or hcl).
	Format string

	// Out specifies the output file path (if empty, writes to stdout).
	Out string

	// Vars holds variable substitutions in key=value form.
	Vars []string

	// Strict treats warnings as errors.
	Strict bool

	// AllowMissingProvider allows missing provider fetches.
	AllowMissingProvider bool

	// TimeoutPerProvider sets timeout for each provider fetch (duration string).
	TimeoutPerProvider string

	// MaxConcurrentProviders limits concurrent provider fetches.
	MaxConcurrentProviders int

	// Verbose enables verbose logging.
	Verbose bool
}

// varFlag implements flag.Value for accumulating multiple --var flags.
type varFlag []string

func (v *varFlag) String() string {
	return strings.Join(*v, ",")
}

func (v *varFlag) Set(value string) error {
	*v = append(*v, value)
	return nil
}

// Parse parses command-line arguments and returns BuildFlags.
func Parse(args []string) (BuildFlags, error) {
	var flags BuildFlags
	var vars varFlag

	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.StringVar(&flags.Path, "path", "", "path to .csl file or directory (required)")
	fs.StringVar(&flags.Path, "p", "", "path to .csl file or directory (shorthand)")
	fs.StringVar(&flags.Format, "format", "json", "output format: json, yaml, or hcl")
	fs.StringVar(&flags.Format, "f", "json", "output format (shorthand)")
	fs.StringVar(&flags.Out, "out", "", "output file path (writes to stdout if empty)")
	fs.StringVar(&flags.Out, "o", "", "output file path (shorthand)")
	fs.Var(&vars, "var", "variable substitution in key=value form (repeatable)")
	fs.BoolVar(&flags.Strict, "strict", false, "treat warnings as errors")
	fs.BoolVar(&flags.AllowMissingProvider, "allow-missing-provider", false, "allow missing provider fetches")
	fs.StringVar(&flags.TimeoutPerProvider, "timeout-per-provider", "", "timeout for each provider fetch (e.g., 5s, 1m)")
	fs.IntVar(&flags.MaxConcurrentProviders, "max-concurrent-providers", 0, "maximum concurrent provider fetches")
	fs.BoolVar(&flags.Verbose, "verbose", false, "enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return BuildFlags{}, err
	}

	flags.Vars = vars

	// Validate required flags
	if flags.Path == "" {
		return BuildFlags{}, errors.New("path is required")
	}

	// Validate format
	validFormats := map[string]bool{"json": true, "yaml": true, "hcl": true}
	if !validFormats[flags.Format] {
		return BuildFlags{}, fmt.Errorf("format must be one of: json, yaml, hcl (got %q)", flags.Format)
	}

	// Validate max-concurrent-providers
	if flags.MaxConcurrentProviders < 0 {
		return BuildFlags{}, fmt.Errorf("max-concurrent-providers must be non-negative (got %d)", flags.MaxConcurrentProviders)
	}

	// Validate timeout-per-provider if provided
	if flags.TimeoutPerProvider != "" {
		if _, err := time.ParseDuration(flags.TimeoutPerProvider); err != nil {
			return BuildFlags{}, fmt.Errorf("timeout-per-provider must be a valid duration (e.g., 5s, 1m): %w", err)
		}
	}

	return flags, nil
}

// ToCompilerOptions converts BuildFlags to compiler.Options.
// Note: Caller must provide ProviderRegistry and ProviderTypeRegistry.
func (f BuildFlags) ToCompilerOptions() (compiler.Options, error) {
	opts := compiler.Options{
		Path:                 f.Path,
		AllowMissingProvider: f.AllowMissingProvider,
		Vars:                 make(map[string]any),
	}

	// Parse vars
	for _, v := range f.Vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return compiler.Options{}, fmt.Errorf("invalid var format %q (expected key=value)", v)
		}
		opts.Vars[parts[0]] = parts[1]
	}

	// Parse timeout
	if f.TimeoutPerProvider != "" {
		duration, err := time.ParseDuration(f.TimeoutPerProvider)
		if err != nil {
			return compiler.Options{}, fmt.Errorf("invalid timeout-per-provider: %w", err)
		}
		opts.Timeouts.PerProviderFetch = duration
	}

	// Set max concurrent providers
	opts.Timeouts.MaxConcurrentProviders = f.MaxConcurrentProviders

	return opts, nil
}
