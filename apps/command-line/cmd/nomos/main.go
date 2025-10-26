// Package main provides the nomos CLI entry point.
package main

import (
	"fmt"
	"os"
)

const (
	exitSuccess  = 0
	exitError    = 1
	exitUsageErr = 2
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitError)
	}
}

func run(args []string) error {
	// Parse command - for now we only support "build"
	if len(args) == 0 {
		return usageError("no command specified. Use 'nomos build --help' for usage.")
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "build":
		return runBuild(commandArgs)
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return usageError(fmt.Sprintf("unknown command: %s", command))
	}
}

func runBuild(args []string) error {
	return buildCommand(args)
}

func printUsage() {
	printHelp()
}

func usageError(msg string) error {
	fmt.Fprintf(os.Stderr, "%s\n\n", msg)
	printUsage()
	os.Exit(exitUsageErr)
	return nil // unreachable, but satisfies type checker
}
