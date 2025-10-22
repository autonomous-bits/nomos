# Nomos

This is a monorepo for Nomos related tooling and libraries. It aims to provide a cohesive development experience across various Nomos projects. Nomos is a configuration scripting language aimed to reduce configuration toil by promoting re-use and cascading overrides.

These configuration scripts will be compiled producing a versioned snapshot that will be used as inputs for infrastructure as code.

## Projects

- [Nomos CLI](../apps/command-line): A command line interface (CLI) that compiles Nomos scripts into configuration snapshots.
- [Nomos Compiler Library](../libs/compiler): A Go library that provides functionality to parse and compile Nomos scripts.
- [Nomos Parser Library](../libs/parser): A Go library that provides functionality to parse Nomos scripts into an abstract syntax tree (AST).

## Standards and Guidelines

- General project structure and development guidance can be found in the [Go Monorepo Structure documentation](../docs/architecture/go-monorepo-structure.md). Each project contains its own `AGENTS.md` file with module-specific layout and development instructions.
- Coding standards are located in GitHub Spaces. Use the `github/get_copilot_space` tool and fetch the `general-standards` space (owner: `pewpewpotato`).

**IMPORTANT:** Always get the constitution from the `general-standards` GitHub space before starting any code implementation.
**IMPORTANT:** Always refer to the latest coding standards from the `general-standards` GitHub space before starting any code implementation.

