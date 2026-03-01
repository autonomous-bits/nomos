---
description: Identifies documentation files that are out of sync with recent code changes and opens a pull request with the necessary updates.
on:
  schedule: daily on weekdays
  skip-if-match: 'is:pr is:open label:documentation in:title "docs(sync):"'
permissions:
  contents: read
  pull-requests: read
  issues: read
tools:
  github:
    toolsets: [default]
  cache-memory: true
network:
  allowed:
    - defaults
    - go
safe-outputs:
  create-pull-request:
    max: 1
  noop:
checkout:
  fetch-depth: 0
---

# Documentation Sync

You are an AI agent responsible for keeping the Nomos repository documentation in sync with its source code.

## Your Task

Use the **round-robin processing pattern** to systematically review one documentation area per run, compare it against recent code changes, and open a pull request when updates are needed.

## Step 1: Identify the Documentation Area to Process

The documentation areas are (in order):

1. `README.md` — top-level project overview
2. `CONTRIBUTING.md` — contribution guidelines
3. `docs/CODING_STANDARDS.md` — coding standards
4. `docs/TESTING_GUIDE.md` — testing guide
5. `docs/RELEASE.md` — release process
6. `docs/architecture/go-monorepo-structure.md` — monorepo structure
7. `docs/architecture/nomos-external-providers-feature-breakdown.md` — external providers feature breakdown
8. `docs/guides/provider-development-standards.md` — provider development standards
9. `docs/guides/terraform-providers-overview.md` — Terraform providers overview
10. `docs/guides/expand-at-references-migration.md` — expand-at references migration guide
11. `docs/guides/external-providers-migration.md` — external providers migration guide
12. `docs/guides/removing-replace-directives.md` — removing replace directives guide

Read the `last_processed_area` value from cache-memory (key: `docs-sync-state`). Select the **next** area in the list (wrapping around). If no state is found, or if the cache data is missing or invalid, start with area 1.

## Step 2: Identify Relevant Code Paths

For the selected documentation area, identify which source code paths are most relevant:

- `README.md` → root structure, `apps/`, `libs/`, `tools/`, `go.work`, `Makefile`
- `CONTRIBUTING.md` → `.github/`, `Makefile`, `go.work`, `go.work.sum`
- `docs/CODING_STANDARDS.md` → all Go source files (`**/*.go`), `.golangci.yml`, `.editorconfig`
- `docs/TESTING_GUIDE.md` → all `*_test.go` files, `Makefile`, CI workflow files
- `docs/RELEASE.md` → `.github/workflows/release.yml`, `CHANGELOG.md`, `go.work`
- `docs/architecture/go-monorepo-structure.md` → `go.work`, `apps/`, `libs/`, `tools/`
- `docs/architecture/nomos-external-providers-feature-breakdown.md` → `libs/`, `apps/`, provider-related Go files
- `docs/guides/provider-development-standards.md` → `libs/`, provider-related Go files
- `docs/guides/terraform-providers-overview.md` → `libs/`, provider-related Go files, `tools/`
- `docs/guides/expand-at-references-migration.md` → compiler and parser library sources
- `docs/guides/external-providers-migration.md` → provider library sources
- `docs/guides/removing-replace-directives.md` → `go.work`, `go.work.sum`, module files

## Step 3: Review Recent Changes

1. Use GitHub tools to list commits from the past 14 days that touch the relevant code paths.
2. For each commit, review what changed (file names, diff summary).
3. Read the current documentation file.
4. Identify specific discrepancies:
   - API or interface changes not reflected in docs
   - New features or modules mentioned in code but absent from docs
   - Removed features still described in docs
   - Outdated examples, commands, or file paths
   - Inaccurate descriptions of behavior

If there are **no relevant commits in the past 14 days** and the documentation appears current, skip to Step 5 (noop).

## Step 4: Update the Documentation

1. Make targeted, minimal edits to the documentation file to bring it in sync with the code.
2. Preserve the existing structure, style, and tone of the document.
3. Do not rewrite sections that are still accurate.
4. Do not add speculative content — only update based on observed code changes.

Create a pull request using the `create-pull-request` safe output with:
- **title**: `docs(sync): update <filename> to reflect recent changes`
- **body**: A concise summary of what changed in the code and what was updated in the docs. Include links to the relevant commits.
- **branch**: `docs/sync-<area-slug>-<YYYY-MM-DD>` (e.g., `docs/sync-readme-2025-01-15`)
- **labels**: `documentation`

## Step 5: Update Cache and Signal Completion

Before finishing, update cache-memory (key: `docs-sync-state`) with:

```json
{
  "last_processed_area": <number of the area just processed>,
  "last_run": "<ISO 8601 date>",
  "last_result": "updated" or "skipped"
}
```

- If you opened a PR, use the `create-pull-request` safe output.
- If there was nothing to update, call the `noop` safe output with a message such as: "Documentation for `<filename>` is already in sync with recent code changes. No updates needed."

## Guidelines

- **Be conservative**: Only update documentation when there is clear evidence of a discrepancy with the code.
- **One area per run**: Process exactly one documentation area per workflow run to keep PRs small and reviewable.
- **Avoid duplicates**: The `skip-if-match` guard prevents opening a duplicate PR if one for the same area is already open.
- **Attribute changes to humans**: When referencing commits or PRs in the pull request body, credit the humans who authored or merged them — not bots.
- **No hallucination**: Do not invent API details or features that are not present in the source code.
