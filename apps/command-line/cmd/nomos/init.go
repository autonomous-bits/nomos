package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/providercmd"
	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// initFlags holds flags for the init command
var initFlags struct {
	dryRun     bool
	force      bool
	os         string
	arch       string
	upgrade    bool
	jsonOutput bool
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [files...]",
	Short: "Install provider dependencies from .csl files",
	Long: `Init scans .csl configuration files, discovers provider requirements,
and installs the necessary provider binaries from GitHub Releases.

This command:
  - Parses .csl files to extract 'source:' declarations
  - Resolves provider versions from GitHub Releases
  - Downloads and installs provider binaries
  - Creates/updates .nomos/providers.lock.json

Provider binaries are installed to:
  .nomos/providers/{owner}/{repo}/{version}/{os-arch}/provider

Network Operations:
  - Fetches provider binaries from GitHub Releases
  - Respects GITHUB_TOKEN for higher rate limits
  - Use --dry-run to preview without downloading`,
	RunE: initCommand,
}

func init() {
	initCmd.Flags().BoolVar(&initFlags.dryRun, "dry-run", false, "Preview actions without executing")
	initCmd.Flags().BoolVar(&initFlags.force, "force", false, "Overwrite existing providers/lockfile")
	initCmd.Flags().StringVar(&initFlags.os, "os", "", "Override target OS (default: runtime OS)")
	initCmd.Flags().StringVar(&initFlags.arch, "arch", "", "Override target architecture (default: runtime arch)")
	initCmd.Flags().BoolVar(&initFlags.upgrade, "upgrade", false, "Force upgrade to latest versions")
	initCmd.Flags().BoolVar(&initFlags.jsonOutput, "json", false, "Output results as JSON")
}

// initCommand executes the init subcommand.
func initCommand(_ *cobra.Command, args []string) error {
	// Get paths from args
	if len(args) == 0 {
		return fmt.Errorf("no .csl files specified")
	}

	// Build options
	opts := providercmd.Options{
		Paths:   args,
		DryRun:  initFlags.dryRun,
		Force:   initFlags.force,
		OS:      initFlags.os,
		Arch:    initFlags.arch,
		Upgrade: initFlags.upgrade,
	}

	// Show spinner during installation (unless dry-run, quiet, or json output)
	var sp *spinner.Spinner
	if !initFlags.dryRun && !globalFlags.quiet && !initFlags.jsonOutput {
		sp = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sp.Suffix = " Installing providers..."
		sp.Start()
	}

	// Run init
	result, err := providercmd.Run(opts)

	// Stop spinner
	if sp != nil {
		sp.Stop()
	}

	if err != nil {
		return err
	}

	// No providers found
	if len(result.Providers) == 0 {
		if !globalFlags.quiet {
			fmt.Println("No providers found in source files.")
		}
		return nil
	}

	// JSON output
	if initFlags.jsonOutput {
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Quiet mode - just exit
	if globalFlags.quiet {
		return nil
	}

	// Table output for results
	if initFlags.dryRun {
		fmt.Println("Dry run mode - would install:")
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Alias", "Type", "Version", "OS", "Arch")
		for _, p := range result.Providers {
			if err := table.Append(p.Alias, p.Type, p.Version, p.OS, p.Arch); err != nil {
				return fmt.Errorf("failed to append table row: %w", err)
			}
		}
		if err := table.Render(); err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}
	} else {
		// Show installation results
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Alias", "Type", "Version", "Status", "Size")
		for _, p := range result.Providers {
			size := "-"
			if p.Size > 0 {
				size = formatSize(p.Size)
			}
			status := string(p.Status)
			if p.Error != nil {
				status = fmt.Sprintf("%s: %v", status, p.Error)
			}
			if err := table.Append(p.Alias, p.Type, p.Version, status, size); err != nil {
				return fmt.Errorf("failed to append table row: %w", err)
			}
		}
		if err := table.Render(); err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}

		fmt.Println()
		if result.Skipped > 0 {
			fmt.Printf("Successfully installed %d provider(s), skipped %d already installed\n",
				result.Installed, result.Skipped)
		} else {
			fmt.Printf("Successfully installed %d provider(s)\n", result.Installed)
		}
	}

	return nil
}

// formatSize formats a byte size into a human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
