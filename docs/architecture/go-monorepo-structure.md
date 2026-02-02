# Go Monorepo Structure: Best Practices and Patterns

**Author:** AI Research  
**Date:** October 21, 2025  
**Status:** Reference Document

## Executive Summary

This document outlines best practices for structuring a Go monorepo that hosts multiple independent Go projects side-by-side. The recommendations are based on analysis of successful large-scale Go monorepos including `golang/go`, `googleapis/google-cloud-go`, `grafana/grafana`, and others.

## Table of Contents

1. [Introduction](#introduction)
2. [Core Principles](#core-principles)
3. [Directory Structure](#directory-structure)
4. [Go Workspaces](#go-workspaces)
5. [Module Organization](#module-organization)
6. [Build and Tooling](#build-and-tooling)
7. [Dependency Management](#dependency-management)
8. [Common Patterns](#common-patterns)
9. [Implementation Guide](#implementation-guide)
10. [References](#references)

## Introduction

A monorepo (monolithic repository) is a software development strategy where code for multiple projects is stored in a single repository. For Go projects, this approach offers several advantages:

- **Atomic changes**: Changes across multiple modules can be made in a single commit
- **Simplified dependency management**: Internal dependencies are co-located
- **Code reuse**: Shared libraries and utilities are easily accessible
- **Consistent tooling**: Build, test, and deployment tools are unified
- **Better collaboration**: Cross-team code reviews and refactoring are easier

## Core Principles

### 1. Module Independence

Each project should be an independent Go module with its own:
- `go.mod` file
- Versioning scheme
- Release cycle
- Dependencies

### 2. Clear Boundaries

Projects should have well-defined boundaries with:
- Distinct purpose and scope
- Minimal coupling between modules
- Clear ownership and responsibility

### 3. Shared Infrastructure

Common elements should be centralized:
- Build scripts and tools
- CI/CD configurations
- Development tooling
- Documentation templates

## Directory Structure

### Recommended Layout

```
.
├── go.work                      # Workspace file (Go 1.18+)
├── README.md
├── .gitignore
├── .github/                     # GitHub-specific files
│   ├── workflows/              # CI/CD workflows
│   └── copilot-instructions.md
├── apps/                        # Application modules
│   └── command-line/
│       ├── go.mod
│       ├── go.sum
│       ├── README.md
│       ├── CHANGELOG.md
│       ├── cmd/
│       │   └── nomos/
│       │       └── main.go
│       └── internal/
│           └── ...
├── libs/                        # Shared library modules
│   ├── compiler/
│   │   ├── go.mod
│   │   ├── go.sum
│   │   ├── README.md
│   │   ├── CHANGELOG.md
│   │   ├── compiler.go         # Public API
│   │   ├── internal/           # Private compiler internals
│   │   │   ├── lexer/
│   │   │   ├── parser/
│   │   │   ├── analyzer/
│   │   │   └── codegen/
│   │   └── test/
│   │       └── ...
│   ├── common/
│   │   ├── go.mod
│   │   ├── go.sum
│   │   ├── README.md
│   │   └── ...
│   └── parser/
│       ├── go.mod
│       ├── go.sum
│       ├── README.md
│       └── ...
├── tools/                       # Development tools and scripts
│   ├── scripts/
│   │   ├── build.sh
│   │   ├── test.sh
│   │   └── release.sh
│   └── generators/
│       └── ...
├── internal/                    # Monorepo-wide shared internal code
│   ├── build/
│   └── testing/
├── docs/                        # Documentation
│   ├── architecture/
│   ├── contributing/
│   └── guides/
└── examples/                    # Example code and demos
    └── ...
```

### Directory Purposes

#### `/apps/`
Contains standalone applications that produce executables. Each app is a complete, deployable unit.

**Guidelines:**
- One application per subdirectory
- Each with its own `go.mod`
- Contains `cmd/` for main packages
- Use `internal/` for app-specific code

#### `/libs/`
Contains reusable library modules that can be imported by apps or other libraries.

**Guidelines:**
- Focused, single-purpose libraries
- Stable APIs with semantic versioning
- Comprehensive documentation
- Minimal dependencies

#### `/tools/`
Contains development tools, build scripts, and generators.

**Guidelines:**
- Build automation scripts
- Code generators
- Testing utilities
- CI/CD helpers

#### `/internal/`
Contains code shared across the monorepo that should not be imported by external projects.

**Guidelines:**
- Truly shared utilities only
- Not versioned independently
- Can be freely refactored
- Used by multiple modules

## Go Workspaces

Go 1.18+ introduced workspaces, which are essential for monorepo management.

### `go.work` File

Create a `go.work` file at the repository root:

```go
go 1.25.6

use (
    ./apps/command-line
    ./libs/compiler
    ./libs/common
    ./libs/parser
)
```

### Benefits

1. **Local Development**: No need for replace directives in individual `go.mod` files
2. **Unified Building**: Build and test multiple modules together
3. **Dependency Resolution**: Automatic resolution of inter-module dependencies
4. **Tool Integration**: IDE support for cross-module navigation

### Commands

```bash
# Initialize workspace
go work init ./apps/command-line

# Add modules
go work use ./libs/compiler
go work use ./libs/common
go work use ./libs/parser

# Sync dependencies
go work sync
```

## Module Organization

### Module Structure

Each module should follow the standard Go project layout:

```
module-name/
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE (optional, can inherit from root)
├── cmd/                    # Command-line applications
│   └── app-name/
│       └── main.go
├── internal/               # Private application logic
│   ├── domain/            # Business logic
│   ├── handlers/          # HTTP handlers, CLI commands
│   └── repository/        # Data access
├── pkg/                    # Public API (optional)
│   └── client/
└── test/                   # Integration tests
    └── ...
```

### Naming Conventions

1. **Module Paths**: Use descriptive names that reflect purpose
   ```
   github.com/autonomous-bits/nomos/apps/command-line
   github.com/autonomous-bits/nomos/libs/compiler
   github.com/autonomous-bits/nomos/libs/parser
   ```

2. **Package Names**: Short, lowercase, no underscores
   ```go
   package parser
   package config
   package httputil
   ```

3. **Directories**: Match package names where possible

### Versioning

Use semantic versioning with module-specific tags:

```bash
# App releases
apps/command-line/v1.0.0

# Library releases
libs/compiler/v1.2.3
libs/parser/v0.5.0
libs/common/v1.0.0
```

## Build and Tooling

### Makefile Pattern

Create a root-level `Makefile` for common operations:

```makefile
.PHONY: help build test lint clean

help:
	@echo "Available targets:"
	@echo "  build    - Build all applications"
	@echo "  test     - Run all tests"
	@echo "  lint     - Run linters"
	@echo "  clean    - Clean build artifacts"

build:
	@echo "Building all applications..."
	go build ./apps/...

build-cli:
	cd apps/command-line && go build -o ../../bin/nomos ./cmd/nomos

test:
	go test -v ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	go clean -cache

.PHONY: work-sync
work-sync:
	go work sync
```

### Task Runners

Consider using modern task runners:

#### [Task](https://taskfile.dev/)
```yaml
# Taskfile.yml
version: '3'

tasks:
  build:all:
    desc: Build all applications
    cmds:
      - task: build:cli

  build:cli:
    dir: apps/command-line
    cmds:
      - go build -o ../../bin/nomos ./cmd/nomos

  test:
    desc: Run all tests
    cmd: go test -v ./...

  lint:
    desc: Run linters
    cmd: golangci-lint run ./...
```

#### [Mage](https://magefile.org/)
```go
// magefile.go
//go:build mage

package main

import (
    "github.com/magefile/mage/sh"
)

// Build all applications
func Build() error {
    return sh.RunV("go", "build", "./apps/...")
}

// Run all tests
func Test() error {
    return sh.RunV("go", "test", "-v", "./...")
}
```

## Dependency Management

### Internal Dependencies

Use workspace features for local development:

```go
// apps/command-line/go.mod
module github.com/autonomous-bits/nomos/apps/command-line

go 1.25.6

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/common v0.1.0
)
```

The command-line app imports and uses the compiler library:

```go
// apps/command-line/cmd/nomos/main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: nomos <input-file>")
        os.Exit(1)
    }
    
    // Use the compiler library
    result, err := compiler.Compile(os.Args[1])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println(result)
}
```

The workspace will automatically resolve these to local paths during development.

### External Dependencies

1. **Centralize Where Possible**: Use a shared `libs/common` for frequently used external dependencies
2. **Version Consistency**: Strive for consistent versions across modules
3. **Minimal Dependencies**: Keep dependency trees shallow
4. **Vendoring Consideration**: Consider vendoring for critical dependencies

### Dependency Updates

Create a script to update all modules:

```bash
#!/bin/bash
# tools/scripts/update-deps.sh

set -e

echo "Updating dependencies for all modules..."

for mod in apps/*/go.mod libs/*/go.mod; do
    dir=$(dirname "$mod")
    echo "Updating $dir..."
    (cd "$dir" && go get -u ./... && go mod tidy)
done

echo "Syncing workspace..."
go work sync

echo "Done!"
```

## Common Patterns

### Shared Internal Code

For code shared across the monorepo but not intended for external use:

```
internal/
├── build/           # Build utilities
│   ├── version.go
│   └── metadata.go
├── testing/         # Test helpers
│   ├── fixtures/
│   └── mocks/
└── config/          # Configuration utilities
    └── loader.go
```

### Code Generation

Place generators in a dedicated location:

```
tools/
├── generators/
│   ├── go.mod
│   └── cmd/
│       ├── genstruct/
│       └── genapi/
└── scripts/
    └── generate.sh
```

### Proto/API Definitions

If using Protocol Buffers or OpenAPI:

```
api/
├── proto/
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   └── v1/
│       └── service.proto
└── openapi/
    └── spec.yaml
```

### Documentation Structure

```
docs/
├── architecture/        # Architecture decisions and designs
│   ├── README.md
│   └── adr/            # Architecture Decision Records
│       └── 001-monorepo-structure.md
├── contributing/        # Contribution guidelines
│   ├── README.md
│   └── code-review.md
├── guides/             # How-to guides
│   ├── development.md
│   └── testing.md
└── api/                # API documentation
    └── README.md
```

## Implementation Guide

### Step 1: Initialize Repository Structure

```bash
# Create directory structure
mkdir -p apps/command-line
mkdir -p libs/{compiler,common,parser}
mkdir -p tools/{scripts,generators}
mkdir -p internal/{build,testing}
mkdir -p docs/{architecture,contributing,guides}

# Create workspace
go work init

# Initialize modules
cd apps/command-line && go mod init github.com/autonomous-bits/nomos/apps/command-line
cd ../../libs/compiler && go mod init github.com/autonomous-bits/nomos/libs/compiler
cd ../common && go mod init github.com/autonomous-bits/nomos/libs/common
cd ../parser && go mod init github.com/autonomous-bits/nomos/libs/parser

# Add to workspace
cd ../../
go work use ./apps/command-line
go work use ./libs/compiler
go work use ./libs/common
go work use ./libs/parser
```

### Step 2: Set Up Build System

Create root `Makefile` or `Taskfile.yml` as shown above.

### Step 3: Configure CI/CD

Example GitHub Actions workflow:

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.6'
          
      - name: Test
        run: go test -v -race -coverprofile=coverage.txt ./...
        
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.txt

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.6'
          
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
```

### Step 4: Document Standards

Create contributing guidelines:

```markdown
# Contributing

## Development Setup

1. Install Go 1.25.6 or later
2. Clone the repository
3. Run `go work sync`

## Making Changes

1. Create a feature branch
2. Make your changes
3. Run tests: `make test`
4. Run linters: `make lint`
5. Submit a pull request

## Module Guidelines

- Each module must have its own `go.mod`
- Keep dependencies minimal
- Write tests for new functionality
- Update documentation
```

### Step 5: Establish Versioning Strategy

Document versioning approach:

```markdown
# Versioning

We follow semantic versioning (semver) for all modules.

## Tagging Releases

Each module is versioned independently:

```bash
# Release CLI app v1.0.0
git tag apps/command-line/v1.0.0
git push origin apps/command-line/v1.0.0

# Release compiler library v1.2.3
git tag libs/compiler/v1.2.3
git push origin libs/compiler/v1.2.3

# Release parser library v0.5.0
git tag libs/parser/v0.5.0
git push origin libs/parser/v0.5.0
```

## Changelog Management

Each module maintains its own CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/).
```

## Best Practices Summary

### Do's ✅

1. **Use Go Workspaces** (`go.work`) for local development
2. **Separate concerns** - apps, libs, tools, internal
3. **Version independently** - each module has its own lifecycle
4. **Centralize tooling** - build scripts, CI/CD at root
5. **Document everything** - README, CHANGELOG for each module
6. **Keep modules focused** - single responsibility principle
7. **Test comprehensively** - unit and integration tests
8. **Automate processes** - building, testing, releasing
9. **Use conventional commits** - clear commit message format
10. **Review code rigorously** - enforce quality standards

### Don'ts ❌

1. **Don't create circular dependencies** between modules
2. **Don't version-lock modules unnecessarily** - allow flexibility
3. **Don't skip documentation** - future you will thank current you
4. **Don't ignore test coverage** - maintain high standards
5. **Don't tightly couple modules** - maintain independence
6. **Don't commit vendor directories** unless absolutely necessary
7. **Don't mix concerns** - keep apps, libs, and tools separate
8. **Don't use replace directives** in committed `go.mod` files
9. **Don't ignore CI failures** - fix immediately
10. **Don't bypass code reviews** - quality over speed

## Advanced Patterns

### Module Generation

Create a generator for new modules:

```bash
#!/bin/bash
# tools/scripts/new-module.sh

MODULE_TYPE=$1  # app or lib
MODULE_NAME=$2

if [ -z "$MODULE_TYPE" ] || [ -z "$MODULE_NAME" ]; then
    echo "Usage: $0 <app|lib> <name>"
    exit 1
fi

BASE_DIR="${MODULE_TYPE}s/${MODULE_NAME}"
mkdir -p "$BASE_DIR"

cd "$BASE_DIR"

# Initialize module
go mod init "github.com/autonomous-bits/nomos/${MODULE_TYPE}s/${MODULE_NAME}"

# Create structure
mkdir -p cmd internal pkg
mkdir -p test

# Create files
cat > README.md << EOF
# ${MODULE_NAME}

## Overview

TODO: Add description

## Installation

\`\`\`bash
go get github.com/autonomous-bits/nomos/${MODULE_TYPE}s/${MODULE_NAME}
\`\`\`

## Usage

TODO: Add usage examples
EOF

cat > CHANGELOG.md << EOF
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
EOF

# Add to workspace
cd ../..
go work use "./${MODULE_TYPE}s/${MODULE_NAME}"

echo "Created module at ${BASE_DIR}"
```

### Build Tags for Module Selection

Use build tags to conditionally include/exclude modules:

```go
//go:build tools
// +build tools

package tools

import (
    _ "golang.org/x/tools/cmd/goimports"
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
```

### Cross-Module Testing

Create integration tests that span multiple modules:

```
test/
├── integration/
│   ├── compiler_cli_test.go
│   └── parser_integration_test.go
└── e2e/
    └── full_workflow_test.go
```

## Troubleshooting

### Common Issues

1. **Import cycle errors**
   - Solution: Restructure dependencies, use interfaces, extract common code

2. **Version conflicts**
   - Solution: Use `go work sync`, update dependencies consistently

3. **Build failures in CI**
   - Solution: Ensure workspace file is not committed, use proper build commands

4. **IDE doesn't recognize imports**
   - Solution: Ensure IDE supports Go workspaces (VSCode, GoLand do)

## References

### Official Documentation
- [Go Modules Reference](https://go.dev/ref/mod)
- [Go Workspaces](https://go.dev/doc/tutorial/workspaces)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

### Example Repositories
- [golang/go](https://github.com/golang/go) - Go source code (internal monorepo structure)
- [googleapis/google-cloud-go](https://github.com/googleapis/google-cloud-go) - Multi-module monorepo
- [grafana/grafana](https://github.com/grafana/grafana) - Large-scale Go monorepo
- [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) - Complex monorepo structure

### Articles and Guides
- [Monorepos in Go by Uber](https://eng.uber.com/go-monorepo-bazel/)
- [Managing Multiple Go Modules](https://go.dev/doc/modules/managing-dependencies)
- [Go Workspaces Tutorial](https://go.dev/blog/get-familiar-with-workspaces)

### Tools
- [Bazel](https://bazel.build/) - Build and test tool (for very large monorepos)
- [Task](https://taskfile.dev/) - Task runner/build tool
- [Mage](https://magefile.org/) - Make-like build tool using Go
- [golangci-lint](https://golangci-lint.run/) - Fast Go linters runner
- [goreleaser](https://goreleaser.com/) - Release automation

## Conclusion

A well-structured Go monorepo provides significant benefits in terms of code reuse, atomic changes, and simplified tooling. By following the patterns outlined in this document, you can create a maintainable and scalable monorepo structure that supports multiple independent Go projects while maximizing developer productivity.

Key takeaways:
1. Use Go workspaces for seamless local development
2. Maintain clear boundaries between applications and libraries
3. Automate common tasks with unified tooling
4. Version modules independently but coordinate releases
5. Document extensively at both repository and module levels

As your monorepo grows, continuously refine your structure and processes based on team feedback and evolving needs.
