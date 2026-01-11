// Package main implements the providers command for the Nomos CLI.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autonomous-bits/nomos/apps/command-line/internal/providercmd"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// providersCmd represents the providers command group
var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage installed providers",
	Long:  `View and manage installed Nomos providers`,
}

// providersListCmd represents the providers list command
var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed providers",
	Long:  `List all providers installed in the .nomos/providers directory`,
	RunE:  providersListCommand,
}

var providersListFlags struct {
	jsonOutput bool
}

func init() {
	providersCmd.AddCommand(providersListCmd)
	providersListCmd.Flags().BoolVar(&providersListFlags.jsonOutput, "json", false, "Output as JSON")
}

// providersListCommand executes the providers list subcommand.
func providersListCommand(_ *cobra.Command, _ []string) error {
	// Read lockfile
	lockPath := filepath.Join(".nomos", "providers.lock.json")

	//nolint:gosec // G304: Path is hardcoded to .nomos/providers.lock.json, safe
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			if !globalFlags.quiet {
				fmt.Println("No providers installed. Run 'nomos init' to install providers.")
			}
			return nil
		}
		return fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lock providercmd.LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("failed to parse lockfile: %w", err)
	}

	if len(lock.Providers) == 0 {
		if !globalFlags.quiet {
			fmt.Println("No providers installed.")
		}
		return nil
	}

	// JSON output
	if providersListFlags.jsonOutput {
		output, err := json.MarshalIndent(lock.Providers, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Table output
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Alias", "Type", "Version", "OS", "Arch", "Path")

	for _, p := range lock.Providers {
		if err := table.Append(p.Alias, p.Type, p.Version, p.OS, p.Arch, p.Path); err != nil {
			return fmt.Errorf("failed to append table row: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	if !globalFlags.quiet {
		fmt.Printf("\nTotal: %d provider(s)\n", len(lock.Providers))
	}

	return nil
}
