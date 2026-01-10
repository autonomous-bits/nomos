// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// ProviderOptions holds configuration for provider management during build.
type ProviderOptions struct {
	// Paths are the input .csl files to scan for provider declarations
	Paths []string

	// Force overwrites existing providers/lockfile
	Force bool

	// DryRun previews actions without executing
	DryRun bool

	// OS is the target operating system (default: runtime.GOOS)
	OS string

	// Arch is the target architecture (default: runtime.GOARCH)
	Arch string

	// Timeout is the timeout per provider operation
	Timeout time.Duration

	// MaxConcurrent is the maximum number of concurrent provider operations
	MaxConcurrent int

	// AllowMissing allows compilation to continue with missing providers
	AllowMissing bool

	// GitHubToken is the GitHub personal access token for API requests
	GitHubToken string
}

// BuildFlags represents the flags from the build command.
type BuildFlags struct {
	// Path is the input .csl file path
	Path string

	// ForceProviders overwrites existing providers/lockfile
	ForceProviders bool

	// DryRun previews actions without executing
	DryRun bool

	// TimeoutPerProvider is the timeout per provider operation (e.g., "30s", "2m")
	TimeoutPerProvider string

	// MaxConcurrentProviders is the maximum number of concurrent provider operations
	MaxConcurrentProviders int

	// AllowMissingProvider allows compilation to continue with missing providers
	AllowMissingProvider bool
}

// NewProviderOptionsFromBuildFlags creates ProviderOptions from build command flags.
// Returns an error if the timeout duration cannot be parsed.
func NewProviderOptionsFromBuildFlags(flags BuildFlags) (ProviderOptions, error) {
	opts := ProviderOptions{
		Paths:         []string{flags.Path},
		Force:         flags.ForceProviders,
		DryRun:        flags.DryRun,
		MaxConcurrent: flags.MaxConcurrentProviders,
		AllowMissing:  flags.AllowMissingProvider,
	}

	// Set defaults for OS/Arch
	opts.OS = runtime.GOOS
	opts.Arch = runtime.GOARCH

	// Parse timeout duration
	if flags.TimeoutPerProvider != "" {
		timeout, err := time.ParseDuration(flags.TimeoutPerProvider)
		if err != nil {
			return ProviderOptions{}, fmt.Errorf("invalid timeout duration %q: %w", flags.TimeoutPerProvider, err)
		}
		opts.Timeout = timeout
	}

	// Get GitHub token from environment
	opts.GitHubToken = os.Getenv("GITHUB_TOKEN")

	return opts, nil
}
