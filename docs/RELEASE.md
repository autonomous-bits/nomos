# Nomos Library Release Process

This document describes the process for releasing versioned Go modules from the Nomos monorepo.

## Tag Naming Convention

Each library in the `libs/` directory is versioned independently using **per-module tags**:

```
libs/<module-name>/v<MAJOR>.<MINOR>.<PATCH>
```

### Examples
- `libs/compiler/v0.1.0`
- `libs/parser/v0.1.0`
- `libs/provider-proto/v0.1.0`

### Rationale
- **Independent versioning**: Each library can evolve at its own pace
- **Semantic versioning**: Follows [SemVer 2.0](https://semver.org/spec/v2.0.0.html)
- **Go module compatibility**: Recognized by Go's module proxy and tooling
- **v0.x.x indicates unstable API**: Major version 0 signals that the API may change

## Release Checklist

Before creating and pushing a release tag, ensure:

- [ ] All tests pass: `go test ./...` from the library directory
- [ ] No linting errors: `golangci-lint run` (if configured)
- [ ] CI/CD pipeline is green on the main branch
- [ ] CHANGELOG.md updated with all changes under a new version heading
- [ ] All replace directives in go.mod (if any) are intentional and documented
- [ ] Code review approved and PR merged
- [ ] Local repository is on the main branch and up-to-date: `git checkout main && git pull`

## Creating a Release

### Step 1: Update CHANGELOG

Move all items from `[Unreleased]` to a new version section in the library's `CHANGELOG.md`:

```markdown
## [0.1.0] - 2025-11-02

### Added
- Initial release
- Feature X
- Feature Y

## [Unreleased]
<!-- Future changes go here -->
```

Update the comparison links at the bottom of CHANGELOG.md:

```markdown
[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/<module>/v0.1.0...HEAD
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/<module>/v0.1.0
```

Commit the CHANGELOG changes:

```bash
git add libs/<module>/CHANGELOG.md
git commit -m "docs(libs/<module>): prepare v0.1.0 release"
git push origin main
```

### Step 2: Create Annotated Tag

Use **annotated tags** (not lightweight tags) to include release metadata:

```bash
# Format: git tag -a libs/<module>/v<version> -m "<module> v<version>"
git tag -a libs/compiler/v0.1.0 -m "libs/compiler v0.1.0"
```

To include more detailed release notes in the tag:

```bash
git tag -a libs/compiler/v0.1.0 -m "libs/compiler v0.1.0

Initial stable release of the Nomos compiler library.

Features:
- Compile Nomos scripts
- Provider integration
- Import resolution"
```

### Step 3: Push Tag to Remote

Push the tag to GitHub:

```bash
git push origin libs/compiler/v0.1.0
```

Or push all tags at once (use with caution):

```bash
git push origin --tags
```

### Step 4: Verify on GitHub

Check that the tag appears:
- GitHub UI: https://github.com/autonomous-bits/nomos/tags
- Via git: `git ls-remote --tags origin | grep libs/<module>`

### Step 5: Create GitHub Release (Optional)

For better discoverability, create a GitHub Release:

1. Go to https://github.com/autonomous-bits/nomos/releases/new
2. Select the tag: `libs/<module>/v<version>`
3. Set release title: `libs/<module> v<version>`
4. Copy release notes from CHANGELOG.md
5. Check "Set as a pre-release" if version is 0.x.x
6. Publish release

## Consuming Published Modules

Once tags are pushed, consumers can use the modules via standard Go commands:

### Initial Installation

```bash
go get github.com/autonomous-bits/nomos/libs/compiler@v0.1.0
go get github.com/autonomous-bits/nomos/libs/parser@v0.1.0
go get github.com/autonomous-bits/nomos/libs/provider-proto@v0.1.0
```

### In go.mod

```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
    github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
)
```

### Updating to Latest Version

```bash
go get -u github.com/autonomous-bits/nomos/libs/compiler
go mod tidy
```

### Listing Available Versions

```bash
go list -m -versions github.com/autonomous-bits/nomos/libs/compiler
```

## Permissions and Access

### Required Permissions

To push tags, you need:
- Write access to the `autonomous-bits/nomos` repository
- Permission to push tags (typically granted with write access)

### CI/CD Token Configuration

If automating releases via GitHub Actions, the workflow needs:
- `GITHUB_TOKEN` with `contents: write` permission
- Or a Personal Access Token (PAT) stored in repository secrets

Example workflow permissions:

```yaml
permissions:
  contents: write  # Required to push tags
```

## Troubleshooting

### Tag Already Exists

If a tag already exists locally or remotely:

```bash
# Delete local tag
git tag -d libs/compiler/v0.1.0

# Delete remote tag (use with extreme caution)
git push origin --delete libs/compiler/v0.1.0
```

**Warning:** Deleting published tags can break consumers who depend on that version. Only delete tags that were just created and not yet used.

### Consumer Cannot Fetch Module

If `go get` fails:

1. **Check tag format**: Must be `libs/<module>/vX.Y.Z`
2. **Verify tag exists**: `git ls-remote --tags origin | grep libs/<module>`
3. **Check module path**: Must match path in go.mod
4. **Clear Go cache**: `go clean -modcache` and try again
5. **Check network/proxy**: If using private repo, ensure GOPRIVATE is set

### Module Sum Mismatch

If Go reports checksum errors:

```bash
go clean -modcache
go get github.com/autonomous-bits/nomos/libs/compiler@v0.1.0
```

## Semantic Versioning Guidelines

Follow [SemVer 2.0](https://semver.org/spec/v2.0.0.html):

- **MAJOR** (v1.0.0 → v2.0.0): Incompatible API changes
- **MINOR** (v0.1.0 → v0.2.0): Backward-compatible new functionality
- **PATCH** (v0.1.0 → v0.1.1): Backward-compatible bug fixes

### Pre-v1.0.0 Versions

For v0.x.x releases:
- API is considered unstable
- Breaking changes can occur in MINOR versions (0.1.0 → 0.2.0)
- Consumers should expect API evolution

## Automated Release Workflow

### Using Make Targets

The root `Makefile` provides convenient targets for releasing libraries:

#### Pre-Release Checks

```bash
make release-check LIB=compiler
```

This runs all pre-release checks:
- ✅ On main branch
- ✅ No uncommitted changes
- ✅ Up-to-date with origin/main
- ✅ Tests pass
- ✅ CHANGELOG.md exists

#### Creating a Release Tag

```bash
make release-lib LIB=compiler VERSION=v0.1.0
```

This will:
1. Run all pre-release checks
2. Create an annotated tag `libs/compiler/v0.1.0`
3. Display the push command

Then push the tag:

```bash
git push origin libs/compiler/v0.1.0
```

#### List All Tags

```bash
make list-tags
```

### CI/CD Automation

For CI/CD automation, refer to `.github/workflows/release.yml` (if configured).

## References

- [Go Modules: Multi-module repositories](https://go.dev/doc/modules/managing-source#multi-module-repositories)
- [Go Modules: Publishing](https://go.dev/doc/modules/publishing)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
- [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
- [Monorepo Structure Guide](./architecture/go-monorepo-structure.md)
