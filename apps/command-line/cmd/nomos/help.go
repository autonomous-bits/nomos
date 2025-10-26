// Package main implements help text and usage for the Nomos CLI.
package main

import (
	"fmt"
)

// printHelp prints the main CLI help text.
func printHelp() {
	fmt.Println(`Nomos CLI - Compile Nomos scripts into configuration snapshots

Usage:
  nomos <command> [options]

Commands:
  build    Compile Nomos .csl files into a configuration snapshot

Global Options:
  -h, --help    Show help

Use 'nomos build --help' for more information about the build command.

Examples:
  # Compile a single file to JSON
  nomos build -p config.csl

  # Compile a directory with custom format
  nomos build -p ./configs -f yaml -o snapshot.yaml

  # Compile with variable substitution
  nomos build -p ./envs/dev --var region=us-west --var env=dev`)
}

// printBuildHelp prints detailed help for the build command.
func printBuildHelp() {
	fmt.Println(`Compile Nomos .csl files into a configuration snapshot

Usage:
  nomos build [options]

Options:
  -p, --path <path>                  Path to .csl file or directory (required)
  -f, --format <format>              Output format: json, yaml, or hcl (default: json)
  -o, --out <file>                   Write output to file (default: stdout)
  --var <key=value>                  Variable substitution (repeatable)
  --strict                           Treat warnings as errors
  --allow-missing-provider           Allow missing provider fetches
  --timeout-per-provider <duration>  Timeout for each provider fetch (e.g., 5s, 1m)
  --max-concurrent-providers <int>   Maximum concurrent provider fetches
  --verbose                          Enable verbose logging
  -h, --help                         Show this help

Exit Codes:
  0   Successful compilation (or warnings only without --strict)
  1   Compilation failed with errors, or warnings in --strict mode
  2   Invalid usage or bad arguments

Diagnostics:
  Errors and warnings are printed to stderr with file:line:col information.
  Use --strict to treat warnings as errors (exit code 1).

Network and Safety:
  The CLI does NOT make network calls by default (offline-first behavior).
  Provider fetches only occur when provider types are explicitly configured
  and required by your .csl scripts. This ensures safe, reproducible builds
  in CI environments.

File Discovery:
  When --path is a directory, .csl files are discovered recursively and processed
  in UTF-8 lexicographic order of their full paths. This ordering is deterministic
  and affects compilation due to last-wins merge semantics.

  Example ordering:
    configs/1-base.csl
    configs/2-network.csl
    configs/subdir/3-app.csl

  Use numeric prefixes (e.g., 1-, 2-) or alphabetic names to control the order.

Examples:
  # Compile a single file
  nomos build -p config.csl

  # Compile directory to YAML
  nomos build -p ./configs -f yaml -o snapshot.yaml

  # Compile with variables
  nomos build -p ./envs/prod --var region=eu-west --var env=production

  # Strict mode (warnings fail)
  nomos build -p ./configs --strict

  # Allow provider failures
  nomos build -p config.csl --allow-missing-provider`)
}
