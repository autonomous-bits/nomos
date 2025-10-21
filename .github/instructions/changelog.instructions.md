---
applyTo: '**/*/CHANGELOG.md'
description: 'Authoritative rules for updating the root CHANGELOG.md consistently across this mono-repo (Keep a Changelog + SemVer).'
---

# CHANGELOG Maintenance Instructions

These rules define exactly how agents must update the root `CHANGELOG.md` for this repository. Follow them strictly for every change.

## Standards
- Format: Keep a Changelog (https://keepachangelog.com/en/1.1.0/)
- Versioning: Semantic Versioning (https://semver.org/) using `MAJOR.MINOR.PATCH`
- File: Root-level `CHANGELOG.md` only (single source of truth)
- Order: Newest release first; always keep an `Unreleased` section at the top
- Date format: `YYYY-MM-DD` (UTC)

## Sections and Allowed Categories
Each section must be in this order and include only these categories when they have entries:
1) Unreleased
2) [x.y.z] - YYYY-MM-DD

Categories under each section (use only those that apply, in this order):
- Added
- Changed
- Deprecated
- Removed
- Fixed
- Security
- Performance

## Entry Style (Bullets)
- Use short, imperative, user-facing summaries.
- Include a scope in square brackets that reflects the impacted area (derived from paths). Examples:
  - [Api] for `apps/<MainProject>/src/<MainProject>.Api/**`
  - [Common] for `apps/<MainProject>/src/<MainProject>.Common/**`
  - [<Module>.Application] or [<Module>.Domain] for module projects
- Optionally include a brief feature/area tag after the scope when helpful: `[Api][Auth]`.
- Reference PR number as `(#123)` and issues as `closes #456` when applicable.
- If the change is breaking, prefix the bullet with `BREAKING:` and place it under the most relevant category (usually Changed or Removed).

Examples:
- [Api] Add v1 health endpoint (#123)
- [Orders.Domain] BREAKING: Rename `OrderId` to `OrderKey` (closes #456)
- [Common] Improve JSON serialization performance by ~15% (#789)

## Mapping from PR Labels to Categories
When generating an entry from a PR, map labels to categories:
- enhancement/feature -> Added
- bug/bugfix -> Fixed
- perf/performance -> Performance
- security -> Security
- refactor, chore -> Changed (only if user-visible behavior or API shape changed; otherwise omit)
- deprecate -> Deprecated
- remove -> Removed
- docs -> Changed (only if it affects public API docs or user-facing behavior; otherwise omit)

If multiple labels apply, pick the most user-relevant single category. Avoid duplicating the same change across categories.

## Grouping by Module/Area
- Prefer one bullet per user-visible change, scoped as described above.
- Do not create nested lists by project; the square-bracket scope is sufficient.
- If a change spans multiple scopes, prefer the most user-visible one, or write separate bullets per scope if it improves clarity.

## Unreleased Workflow (per PR)
For every merged PR that affects user-visible behavior:
1) Ensure an `## [Unreleased]` section exists at the top. If missing, create it.
2) Insert the bullet under the correct category within Unreleased, respecting the order of categories listed above.
3) Keep bullets sorted newest-first within each category.
4) Do not include internal-only details (build, CI-only, non-user-visible refactors).
5) Never include secrets, connection strings, or sensitive data.

## Releasing (promoting Unreleased to a version)
Only perform a release when requested or when a release tag is being prepared.
1) Determine version bump:
   - MAJOR for breaking changes
   - MINOR for backward-compatible features
   - PATCH for backward-compatible fixes/perf tweaks/docs
2) Create a new section below Unreleased:
   - `## [x.y.z] - YYYY-MM-DD`
   - Move the categorized bullets from Unreleased into this new section, preserving category order.
   - If a category is empty, omit it.
3) Leave `## [Unreleased]` in place, empty (ready for new entries).
4) Update compare links at the bottom (see next section).
5) Keep the newest release at the top below Unreleased.

## Compare Links
At the bottom of `CHANGELOG.md`, maintain GitHub compare links as reference-style links.

Required links:
- `[Unreleased]`: compare from latest tag to `HEAD`
- For each version `[x.y.z]`: compare from previous tag to this tag

Template (replace placeholders with actual values):
```
[Unreleased]: https://github.com/<owner>/<repo>/compare/vX.Y.Z...HEAD
[X.Y.Z]: https://github.com/<owner>/<repo>/compare/vA.B.C...vX.Y.Z
```

Rules:
- Use the literal `v` prefix in tags when constructing links (e.g., `v1.2.3`).
- On the first release (no previous tag), point `[X.Y.Z]` to the commit where the project started or omit the compare link if unknown.
- Always update `[Unreleased]` to start from the newest released tag.

## Formatting Rules
- Headings use `##` level for sections: `## [Unreleased]`, `## [x.y.z] - YYYY-MM-DD`
- Category headings use `###` level
- Bullets use `- `; no trailing periods
- Wrap code identifiers in backticks
- Keep lines ≤120 chars where practical

## Quick Checklist for Agents
- [ ] Unreleased exists and is at the top
- [ ] Entry uses an allowed category and correct order
- [ ] Bullet is concise, imperative, scoped `[Area]`
- [ ] PR/issue references added: `(#NNN)` / `closes #NNN`
- [ ] Breaking changes prefixed with `BREAKING:`
- [ ] New release sections include date in `YYYY-MM-DD`
- [ ] Compare links updated at bottom
- [ ] No secrets or sensitive info

## Minimal Example
```
# Changelog
All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project adheres to Semantic Versioning.

## [Unreleased]

### Added
- [Api] Add v1 health endpoint (#123)

### Fixed
- [Orders.Domain] Correct discount rounding for edge cases (#130)

## [1.2.0] - 2025-08-22

### Changed
- [Common] BREAKING: Rename `ConfigKey` to `SettingKey` (closes #120)

### Performance
- [Common] Improve JSON serialization throughput by ~15% (#118)

[Unreleased]: https://github.com/<owner>/<repo>/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/<owner>/<repo>/compare/v1.1.3...v1.2.0
```

## Security & Compliance Notes
- Do not include secrets, internal endpoints, stack traces, or PII.
- Summaries must be user-facing; internal-only tasks are excluded unless they change behavior or public API/docs.

## Operational Guidance (Optional for Release Managers)
- After cutting a release, create a Git tag `vX.Y.Z` matching the CHANGELOG section.
- Ensure CI/CD release notes, if generated, mirror the CHANGELOG formatting and content.
