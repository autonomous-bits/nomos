package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/initcmd"
)

// runInit executes the init subcommand.
func runInit(args []string) error {
	// Create flagset for init command
	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	// Define flags
	var fromFlags arrayFlags
	dryRun := fs.Bool("dry-run", false, "Preview actions without executing")
	force := fs.Bool("force", false, "Overwrite existing providers/lockfile")
	osFlag := fs.String("os", "", "Override target OS (default: runtime OS)")
	archFlag := fs.String("arch", "", "Override target architecture (default: runtime arch)")
	upgrade := fs.Bool("upgrade", false, "Force upgrade to latest versions")

	fs.Var(&fromFlags, "from", "Local provider path (alias=path, can be repeated)")

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Get remaining args as paths
	paths := fs.Args()
	if len(paths) == 0 {
		// If no explicit paths, scan current directory for .csl files
		// For now, we require explicit paths
		return fmt.Errorf("no .csl files specified")
	}

	// Parse --from flags into map
	fromPaths := make(map[string]string)
	for _, f := range fromFlags {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --from format (expected alias=path): %s", f)
		}
		fromPaths[parts[0]] = parts[1]
	}

	// Build options
	opts := initcmd.Options{
		Paths:     paths,
		FromPaths: fromPaths,
		DryRun:    *dryRun,
		Force:     *force,
		OS:        *osFlag,
		Arch:      *archFlag,
		Upgrade:   *upgrade,
	}

	// Run init
	return initcmd.Run(opts)
}

// arrayFlags implements flag.Value for repeated string flags.
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}
