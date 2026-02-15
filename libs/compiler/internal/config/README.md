# Provider Configuration: Lockfile and Manifest

This document describes the lockfile and manifest formats used by the Nomos external providers feature.

## Overview

Nomos uses two configuration files to manage external provider binaries:

1. **Lockfile** (`.nomos/providers.lock.json`): Records exact provider binaries with versions, paths, and checksums for reproducible builds
2. **Manifest** (`.nomos/providers.yaml`): Provides declarative provider configuration with source hints

## Lockfile Format

The lockfile is a JSON file located at `.nomos/providers.lock.json` that records the exact provider binaries installed for a project.

### Schema

```json
{
  "providers": [
    {
      "alias": "string (required)",
      "type": "string (required)",
      "version": "string (required)",
      "os": "string (required)",
      "arch": "string (required)",
      "source": {
        "github": {
          "owner": "string",
          "repo": "string",
          "asset": "string"
        },
        "local": {
          "path": "string"
        }
      },
      "checksum": "string (sha256:...)",
      "path": "string (required)"
    }
  ]
}
```

### Fields

- **alias**: Provider alias used in `.csl` source declarations (must be unique)
- **type**: Provider implementation type (e.g., `file`, `http`)
- **version**: Semantic version of the provider binary
- **os**: Operating system (e.g., `darwin`, `linux`, `windows`)
- **arch**: Architecture (e.g., `amd64`, `arm64`)
- **source**: Where the provider was obtained from (GitHub or local)
  - **github.owner**: GitHub repository owner
  - **github.repo**: GitHub repository name
  - **github.asset**: Downloaded release asset filename
  - **local.path**: Original local filesystem path
- **checksum**: SHA256 checksum of the binary for verification
- **path**: Relative path to the installed provider binary

### Example

```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "0.2.0",
      "os": "darwin",
      "arch": "arm64",
      "source": {
        "github": {
          "owner": "autonomous-bits",
          "repo": "nomos-provider-file",
          "asset": "nomos-provider-file-0.2.0-darwin-arm64"
        }
      },
      "checksum": "sha256:1234567890abcdef...",
      "path": ".nomos/providers/file/0.2.0/darwin-arm64/provider"
    }
  ]
}
```

### Standard Installation Path

Provider binaries are installed following this convention:
```
.nomos/providers/{type}/{version}/{os}-{arch}/provider
```

## Manifest Format

The manifest is a YAML file located at `.nomos/providers.yaml` that provides declarative provider configuration.

### Schema

```yaml
providers:
  - alias: string (required)
    type: string (required)
    source:
      github:
        owner: string
        repo: string
        asset: string  # Template with {version}, {os}, {arch} placeholders
      local:
        path: string
    config:
      # Provider-specific configuration keys
      key: value
```

### Fields

- **alias**: Provider alias used in `.csl` source declarations (must be unique)
- **type**: Provider implementation type
- **source**: Hints for where to obtain the provider binary
  - **github.owner**: GitHub repository owner
  - **github.repo**: GitHub repository name
  - **github.asset**: Asset name template (supports `{version}`, `{os}`, `{arch}`)
  - **local.path**: Local filesystem path to provider binary
- **config**: Optional default configuration passed to provider `Init`
  - Keys are provider-specific (e.g., `directory` for file provider)

### Example

```yaml
providers:
  - alias: configs
    type: file
    source:
      github:
        owner: autonomous-bits
        repo: nomos-provider-file
        asset: nomos-provider-file-{version}-{os}-{arch}
    config:
      directory: ./apps/command-line/testdata/configs

  - alias: http-provider
    type: http
    source:
      local:
        path: /usr/local/bin/nomos-provider-http
```

## Version Precedence Rules

**IMPORTANT**: Versions are authoritative in `.csl` source declarations, NOT in the manifest.

The manifest provides source hints and default configuration, but:
- Versions MUST be specified in `.csl` files
- The lockfile records the installed version
- The manifest MUST NOT override versions

Example `.csl` source declaration:
```csl
source file as configs {
  version = "0.2.0"  # Authoritative version
  directory = "./configs"
}
```

## Resolution Precedence

The resolver combines information from both files with these precedence rules:

1. **Lockfile is authoritative** for installed binaries:
   - Version, OS, Architecture
   - Binary path and checksum
   - Source provenance

2. **Manifest provides additional data**:
   - Source hints (GitHub owner/repo, asset template)
   - Default configuration
   - Type (if not in lockfile)

3. **At least one must exist**:
   - Lockfile only: Can resolve installed providers
   - Manifest only: Can resolve provider types and sources (no path/version)
   - Both: Full resolution with precedence rules

## Usage Examples

### Creating a Lockfile

```go
import "github.com/autonomous-bits/nomos/libs/compiler/internal/config"

lockfile := config.Lockfile{
    Providers: []config.Provider{
        {
            Alias:    "configs",
            Type:     "file",
            Version:  "0.2.0",
            OS:       "darwin",
            Arch:     "arm64",
            Checksum: "sha256:abc123...",
            Path:     ".nomos/providers/file/0.2.0/darwin-arm64/provider",
        },
    },
}

if err := lockfile.Save(".nomos/providers.lock.json"); err != nil {
    log.Fatal(err)
}
```

### Creating a Manifest

```go
import "github.com/autonomous-bits/nomos/libs/compiler/internal/config"

manifest := config.Manifest{
    Providers: []config.ManifestProvider{
        {
            Alias: "configs",
            Type:  "file",
            Source: config.ManifestSource{
                GitHub: &config.ManifestGitHubSource{
                    Owner: "autonomous-bits",
                    Repo:  "nomos-provider-file",
                },
            },
            Config: map[string]any{
                "directory": "./testdata/configs",
            },
        },
    },
}

if err := manifest.Save(".nomos/providers.yaml"); err != nil {
    log.Fatal(err)
}
```

### Using the Resolver

```go
import "github.com/autonomous-bits/nomos/libs/compiler/internal/config"

// Create resolver
resolver, err := config.NewResolver(
    ".nomos/providers.lock.json",
    ".nomos/providers.yaml",
)
if err != nil {
    log.Fatal(err)
}

// Resolve a provider
provider, err := resolver.ResolveProvider("configs")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Provider: %s\n", provider.Alias)
fmt.Printf("Type: %s\n", provider.Type)
fmt.Printf("Version: %s\n", provider.Version)
fmt.Printf("Path: %s\n", provider.Path)

// Get all providers
allProviders := resolver.GetAllProviders()
for _, p := range allProviders {
    fmt.Printf("- %s (%s v%s)\n", p.Alias, p.Type, p.Version)
}
```

## Validation Rules

### Lockfile Validation

- At least one provider required
- Each provider must have: alias, type, version, path
- Aliases must be unique
- OS and architecture should be specified

### Manifest Validation

- At least one provider required
- Each provider must have: alias, type
- Aliases must be unique
- Cannot specify both GitHub and Local sources
- If GitHub source specified: owner and repo required
- If Local source specified: path required

## Error Handling

The resolver and load functions return errors in these cases:

- **File not found**: Both lockfile and manifest missing
- **Parse error**: Invalid JSON or YAML syntax
- **Validation error**: Missing required fields or invalid values
- **Provider not found**: Alias doesn't exist in lockfile or manifest

Example error messages:
```
neither lockfile nor manifest found; run 'nomos build'
provider "unknown" not found in lockfile or manifest
invalid lockfile: provider "configs": version is required
```

## Security Considerations

1. **Checksum verification**: Always validate checksums when installing providers
2. **Path safety**: Only execute binaries under `.nomos/providers/`
3. **Permissions**: Provider binaries should have `0755` permissions
4. **Source provenance**: Record source in lockfile for audit trail

## Integration with provider management in `nomos build`

`nomos build` performs provider management before compilation:

1. Scans `.csl` files for source declarations
2. Merges with manifest (if present) for source hints
3. Resolves and downloads/copies provider binaries
4. Verifies checksums
5. Writes lockfile with exact versions and paths

## Integration with `nomos build`

The `nomos build` command:

1. Loads lockfile and manifest via resolver
2. Resolves required providers by alias
3. Starts provider subprocesses using paths from lockfile
4. Establishes gRPC connections
5. Calls provider `Init` with config from manifest

## See Also

- [External Providers Feature Breakdown](../../docs/architecture/nomos-external-providers-feature-breakdown.md)
- [Provider Process Manager](../providerproc/README.md)
- [Compiler Provider Interface](../../provider.go)
