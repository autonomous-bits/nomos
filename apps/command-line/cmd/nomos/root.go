// Package main provides the nomos CLI entry point.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	// Version information (set by build system)
	version   = "v2.0.0"
	commit    = "unknown"
	buildDate = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nomos",
	Short: "Nomos configuration scripting language compiler",
	Long: `Nomos CLI - Configuration scripting language compiler

Nomos is a configuration scripting language that reduces configuration toil
by promoting re-use and cascading overrides.

These configuration scripts compile to versioned snapshots that serve as
inputs for infrastructure as code.`,
	SilenceUsage:  true, // Don't show usage on errors
	SilenceErrors: true, // We handle errors ourselves
}

// globalFlags holds flags that apply to all commands
var globalFlags struct {
	color string
	quiet bool
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&globalFlags.color, "color", "auto", "Colorize output: auto, always, never")
	rootCmd.PersistentFlags().BoolVarP(&globalFlags.quiet, "quiet", "q", false, "Suppress non-error output")

	// Add commands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(versionCmd)

	// Optionally add new commands (Phase 2.4)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(providersCmd)

	// Add shell completion commands
	rootCmd.AddCommand(completionCmd)

	// Custom error handler for removed commands
	rootCmd.SetFlagErrorFunc(flagErrorFunc)
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for Nomos CLI.

To load completions:

Bash:
  $ source <(nomos completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ nomos completion bash > /etc/bash_completion.d/nomos
  # macOS:
  $ nomos completion bash > $(brew --prefix)/etc/bash_completion.d/nomos

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ nomos completion zsh > "${fpath[1]}/_nomos"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ nomos completion fish | source

  # To load completions for each session, execute once:
  $ nomos completion fish > ~/.config/fish/completions/nomos.fish

PowerShell:
  PS> nomos completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> nomos completion powershell > nomos.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(_ *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	err := rootCmd.Execute()

	// Check for 'init' command attempt
	if err != nil && isInitCommandAttempt(err) {
		// Print migration message to stderr
		// Since we're about to exit with error, we gracefully handle print errors
		// but don't let them prevent the migration message from being displayed
		_ = printMigrationMessage()
		os.Exit(1)
	}

	return err
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print version information including build metadata`,
	RunE:  runVersion,
}

// runVersion displays version information.
func runVersion(_ *cobra.Command, _ []string) error {
	if globalFlags.quiet {
		fmt.Println(version)
		return nil
	}

	fmt.Printf("Nomos CLI Version: %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Build Date: %s\n", buildDate)

	// Try to get module version info
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("Go Version: %s\n", info.GoVersion)
		if version == "dev" {
			// In development, show the module path
			fmt.Printf("Module: %s\n", info.Main.Path)
			if info.Main.Version != "(devel)" {
				fmt.Printf("Module Version: %s\n", info.Main.Version)
			}
		}
	}

	return nil
}

// setupColorOutput configures color output based on flags and terminal capabilities
func setupColorOutput() {
	// This will be implemented with color support in Phase 2.3
	// For now, just a placeholder
	switch globalFlags.color {
	case "always":
		// Force color on
		_ = os.Setenv("CLICOLOR_FORCE", "1") // Ignore error, not critical
	case "never":
		// Force color off
		_ = os.Setenv("NO_COLOR", "1") // Ignore error, not critical
	case "auto":
		// Let libraries auto-detect
	default:
		// Invalid value, default to auto
	}
}

// isInitCommandAttempt checks if the error is due to attempting to run the 'init' command
func isInitCommandAttempt(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains 'init' as unknown command
	errMsg := err.Error()
	return len(errMsg) > 0 &&
		(errMsg == "unknown command \"init\" for \"nomos\"" ||
			errMsg == `unknown command "init" for "nomos"`)
}

// printMigrationMessage prints the init command removal migration message to stderr.
// Returns error if any print operation fails, but attempts to print all lines.
func printMigrationMessage() error {
	lines := []string{
		"Error: The 'init' command has been removed in v2.0.0.",
		"",
		"Providers are now automatically downloaded during 'nomos build'.",
		"",
		"Migration:",
		"  Old: nomos init && nomos build",
		"  New: nomos build",
		"",
		"For more information, see the migration guide at:",
		"https://github.com/autonomous-bits/nomos/blob/main/docs/guides/migration-v2.md",
	}

	var firstErr error
	for _, line := range lines {
		if _, err := fmt.Fprintln(os.Stderr, line); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// flagErrorFunc is called when there's an error parsing flags or an unknown command
func flagErrorFunc(_ *cobra.Command, err error) error {
	// Let cobra handle the error normally
	return err
}
