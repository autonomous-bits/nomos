---
name: changelog-maintenance
description: Ensures consistent CHANGELOG.md updates following Keep a Changelog format with SemVer categorization. Use this when updating the changelog after merging PRs, creating releases, or when reviewing changelog entries for correctness.
---

# Changelog Maintenance for Nomos

This skill ensures consistent `CHANGELOG.md` updates following Keep a Changelog format and Semantic Versioning standards for the Nomos monorepo.

## When to Use This Skill

- After merging a PR that affects user-visible behavior
- When creating or preparing a release
- Reviewing changelog entries for correctness
- Converting PR descriptions to changelog entries
- Updating compare links after releases

## Standards Overview

- **Format:** Keep a Changelog (https://keepachangelog.com/en/1.1.0/)
- **Versioning:** Semantic Versioning (https://semver.org/)
- **File:** Root-level `CHANGELOG.md` (single source of truth)
- **Order:** Newest release first; always maintain `[Unreleased]` section at top
- **Date format:** `YYYY-MM-DD` (UTC)

## Entry Categories (In Order)

Use only these categories in this exact order:

1. **Added** - New features
2. **Changed** - Changes in existing functionality
3. **Deprecated** - Soon-to-be removed features
4. **Removed** - Removed features
5. **Fixed** - Bug fixes
6. **Security** - Security fixes
7. **Performance** - Performance improvements

## Adding Entries to Unreleased

### Step 1: Map PR to Category

**Label → Category mapping:**
- `enhancement`/`feature` → Added
- `bug`/`bugfix` → Fixed
- `perf`/`performance` → Performance
- `security` → Security
- `refactor`/`chore` → Changed (only if user-visible)
- `deprecate` → Deprecated
- `remove` → Removed
- `docs` → Changed (only if affects public API/behavior)

**If multiple labels:** Pick most user-relevant category

### Step 2: Determine Scope

Derive scope from file paths:

**Examples:**
- `apps/command-line/**` → `[CLI]`
- `libs/compiler/**` → `[Compiler]`
- `libs/parser/**` → `[Parser]`
- `libs/provider-downloader/**` → `[Provider Downloader]`
- `libs/provider-proto/**` → `[Provider Proto]`
- `.github/**` or `docs/**` → `[Docs]` or omit if internal-only

**Optional sub-scope:**
- `[CLI][Init]` - for init command
- `[Compiler][References]` - for reference resolution
- `[Parser][Errors]` - for error handling

### Step 3: Write Entry

**Format:**
```
- [Scope] Imperative description (#PR-number)
- [Scope][SubScope] BREAKING: Description (closes #issue)
```

**Guidelines:**
- Use imperative mood ("Add", "Fix", "Improve", not "Added", "Fixed", "Improved")
- Be concise and user-facing
- Reference PR: `(#123)`
- Reference issue: `(closes #456)` or `(fixes #789)`
- Prefix breaking changes: `BREAKING:`
- No trailing periods
- Wrap code identifiers in backticks: `Options.AllowMissingProvider`

**Examples:**
```markdown
### Added
- [CLI] Add `--allow-missing-provider` flag for graceful provider failures (#145)
- [Compiler] Add provider caching for deterministic builds (#152)

### Changed
- [Parser] BREAKING: Remove top-level `reference:` statements (closes #130)
- [Compiler] Improve error messages for unresolved references (#148)

### Fixed
- [CLI] Fix exit code for strict mode validation failures (#150)
- [Provider Downloader] Correct checksum validation for cross-platform binaries (#155)

### Performance
- [Compiler] Reduce reference resolution time by ~40% with cache (#160)
```

### Step 4: Insert Entry

1. **Ensure Unreleased exists:**
   ```markdown
   ## [Unreleased]
   ```

2. **Find or create category:**
   ```markdown
   ## [Unreleased]
   
   ### Added
   ```

3. **Add entry at top of category** (newest first):
   ```markdown
   ### Added
   - [CLI] Add new feature (#NEW)  ← New entry here
   - [Compiler] Previous feature (#OLD)
   ```

4. **Maintain category order:**
   - Added
   - Changed
   - Deprecated
   - Removed
   - Fixed
   - Security
   - Performance

## Creating a Release

### Step 1: Determine Version Bump

**SemVer rules:**
- **MAJOR** (X.0.0) - Breaking changes (API incompatibility)
- **MINOR** (0.X.0) - New features (backward-compatible)
- **PATCH** (0.0.X) - Bug fixes (backward-compatible)

**Look for:**
- Any `BREAKING:` entries → MAJOR bump
- Any `Added` entries without breaking → MINOR bump
- Only `Fixed`, `Performance`, `Security` → PATCH bump

**Nomos-specific versioning:**
- CLI: `apps/command-line/v1.x.x` (stable, MAJOR version)
- Compiler: `libs/compiler/v0.x.x` (pre-1.0, MINOR breaking changes allowed)
- Parser: `libs/parser/v0.x.x` (pre-1.0)

### Step 2: Create Version Section

1. **Add new section below Unreleased:**
   ```markdown
   ## [Unreleased]
   
   ## [1.2.0] - 2025-12-29
   ```

2. **Move entries from Unreleased:**
   ```markdown
   ## [Unreleased]
   
   ## [1.2.0] - 2025-12-29
   
   ### Added
   - [CLI] Add `--allow-missing-provider` flag (#145)
   
   ### Fixed
   - [CLI] Fix exit code for strict mode (#150)
   ```

3. **Keep Unreleased empty:**
   ```markdown
   ## [Unreleased]
   
   ## [1.2.0] - 2025-12-29
   ```

### Step 3: Update Compare Links

At bottom of `CHANGELOG.md`:

**Before release:**
```markdown
[Unreleased]: https://github.com/autonomous-bits/nomos/compare/v1.1.0...HEAD
```

**After release v1.2.0:**
```markdown
[Unreleased]: https://github.com/autonomous-bits/nomos/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/autonomous-bits/nomos/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/autonomous-bits/nomos/compare/v1.0.0...v1.1.0
```

**Pattern:**
```markdown
[Unreleased]: https://github.com/autonomous-bits/nomos/compare/v{LATEST}...HEAD
[{LATEST}]: https://github.com/autonomous-bits/nomos/compare/v{PREVIOUS}...v{LATEST}
```

## Common Scenarios

### Scenario 1: Adding Entry After PR Merge

**Given:** PR #145 merged adding `--allow-missing-provider` flag to CLI

**Steps:**
1. Identify category: Added (new feature)
2. Determine scope: [CLI]
3. Write entry: `[CLI] Add \`--allow-missing-provider\` flag for graceful provider failures (#145)`
4. Add to Unreleased → Added section

**Result:**
```markdown
## [Unreleased]

### Added
- [CLI] Add `--allow-missing-provider` flag for graceful provider failures (#145)
```

### Scenario 2: Breaking Change

**Given:** PR #130 removes deprecated top-level `reference:` statements

**Steps:**
1. Identify category: Removed (removing feature)
2. Determine scope: [Parser]
3. Write entry: `[Parser] BREAKING: Remove top-level \`reference:\` statements (closes #130)`
4. Add to Unreleased → Removed section

**Result:**
```markdown
## [Unreleased]

### Removed
- [Parser] BREAKING: Remove top-level `reference:` statements (closes #130)
```

### Scenario 3: Bug Fix

**Given:** PR #150 fixes CLI exit code bug

**Steps:**
1. Identify category: Fixed (bug fix)
2. Determine scope: [CLI]
3. Write entry: `[CLI] Fix exit code for strict mode validation failures (#150)`
4. Add to Unreleased → Fixed section

**Result:**
```markdown
## [Unreleased]

### Fixed
- [CLI] Fix exit code for strict mode validation failures (#150)
```

### Scenario 4: Preparing Release

**Given:** 
- Current version: v1.1.0
- Unreleased has 2 Added, 1 Fixed
- No breaking changes

**Steps:**
1. Version bump: MINOR (1.2.0) - has Added entries
2. Create section: `## [1.2.0] - 2025-12-29`
3. Move all entries from Unreleased
4. Update compare links

**Result:**
```markdown
## [Unreleased]

## [1.2.0] - 2025-12-29

### Added
- [CLI] Add `--allow-missing-provider` flag (#145)
- [Compiler] Add provider caching (#152)

### Fixed
- [CLI] Fix exit code for strict mode (#150)

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/autonomous-bits/nomos/compare/v1.1.0...v1.2.0
```

## Validation Checklist

Before committing changelog updates:

- [ ] Unreleased section exists at top
- [ ] Entry in correct category (Added/Changed/Fixed/etc)
- [ ] Categories in correct order
- [ ] Scope in square brackets `[CLI]`
- [ ] Imperative mood ("Add" not "Added")
- [ ] PR reference included `(#123)`
- [ ] Issue reference if applicable `(closes #456)`
- [ ] Breaking changes prefixed with `BREAKING:`
- [ ] Code identifiers in backticks
- [ ] No trailing periods
- [ ] No secrets or sensitive information
- [ ] User-facing language (not internal jargon)
- [ ] Compare links updated for releases

## What NOT to Include

**Exclude these from changelog:**

1. **Internal-only changes:**
   - CI configuration updates
   - Test-only refactoring
   - Internal tool updates
   - Build script changes (unless affects users)

2. **Non-user-visible changes:**
   - Code refactoring (unless API changed)
   - Internal documentation
   - Development dependencies
   - Logging improvements

3. **Sensitive information:**
   - Secrets or API keys
   - Internal URLs or endpoints
   - Security vulnerability details (until fixed)
   - Personal information

4. **Trivial updates:**
   - Typo fixes in comments
   - Whitespace changes
   - Code formatting

## Multi-Module Releases

Nomos is a monorepo with independent module versioning:

**Separate entries for separate modules:**
```markdown
### Added
- [CLI] Add `--timeout` flag (#145)
- [Compiler] Add reference caching (#152)
```

**NOT grouped by module:**
```markdown
### CLI
- Add `--timeout` flag (#145)

### Compiler  
- Add reference caching (#152)
```

## Reference Documentation

For complete changelog rules, see:
- [.github/instructions/changelog.instructions.md](../../.github/instructions/changelog.instructions.md)
- Keep a Changelog: https://keepachangelog.com/en/1.1.0/
- Semantic Versioning: https://semver.org/
