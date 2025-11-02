package main

import (
	"flag"
	"fmt"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/initcmd"
)

// runInit executes the init subcommand.
func runInit(args []string) error {
	// Create flagset for init command
	fs := flag.NewFlagSet("init", flag.ContinueOnError)

	// Define flags
	dryRun := fs.Bool("dry-run", false, "Preview actions without executing")
	force := fs.Bool("force", false, "Overwrite existing providers/lockfile")
	osFlag := fs.String("os", "", "Override target OS (default: runtime OS)")
	archFlag := fs.String("arch", "", "Override target architecture (default: runtime arch)")
	upgrade := fs.Bool("upgrade", false, "Force upgrade to latest versions")

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

	// Build options
	opts := initcmd.Options{
		Paths:   paths,
		DryRun:  *dryRun,
		Force:   *force,
		OS:      *osFlag,
		Arch:    *archFlag,
		Upgrade: *upgrade,
	}

	// Run init
	return initcmd.Run(opts)
}
