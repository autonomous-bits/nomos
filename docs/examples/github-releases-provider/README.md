# GitHub Releases Provider Example

This example demonstrates how to install and use a provider from GitHub Releases.

## Scenario

You want to use a published provider from GitHub Releases. Nomos will automatically download the correct binary for your platform, verify checksums, and install it.

## Prerequisites

- Nomos CLI installed
- Internet connection (to download from GitHub)

## Step-by-Step Guide

### 1. Project Structure

```
github-releases-provider/
├── README.md           # This file
├── config.csl          # Nomos configuration
├── data/               # Sample data files
│   └── database/
│       ├── prod.csl
│       └── staging.csl
└── .nomos/             # Created by nomos init
    ├── providers/      # Downloaded providers
    └── providers.lock.json
```

### 2. Create Nomos Configuration

Create `config.csl`:

```csl
// Declare the file provider from GitHub Releases
source file as configs {
  version = "1.0.0"
  
  // Optional: Specify GitHub source explicitly
  source = {
    github = {
      owner = "autonomous-bits"
      repo = "nomos-provider-file"
    }
  }
  
  config = {
    directory = "./data"
  }
}

// Import multiple configurations
prod_db = import configs["database"]["prod"]
staging_db = import configs["database"]["staging"]

// Compose final configuration
config = {
  environments = {
    production = {
      database = prod_db
    }
    staging = {
      database = staging_db
    }
  }
  app_name = "example-app"
}
```

### 3. Create Sample Data Files

Create data files that the provider will read:

```bash
mkdir -p data/database

# Production database config
cat > data/database/prod.csl << 'EOF'
config = {
  host = "prod-db.example.com"
  port = 5432
  database = "production"
  username = "prod_user"
  max_connections = 100
  ssl_mode = "require"
}
EOF

# Staging database config
cat > data/database/staging.csl << 'EOF'
config = {
  host = "staging-db.example.com"
  port = 5432
  database = "staging"
  username = "staging_user"
  max_connections = 50
  ssl_mode = "prefer"
}
EOF
```

### 4. Initialize Providers

Run `nomos init` to download and install the provider:

```bash
nomos init config.csl
```

**What happens**:
1. Nomos parses `config.csl` and identifies required providers
2. Resolves the provider source (GitHub owner/repo from config or defaults)
3. Determines your OS and architecture (e.g., darwin/arm64)
4. Downloads the matching binary from GitHub Releases:
   - `https://github.com/autonomous-bits/nomos-provider-file/releases/download/v1.0.0/nomos-provider-file-1.0.0-darwin-arm64`
5. Downloads and verifies checksums from `SHA256SUMS`
6. Installs to `.nomos/providers/file/1.0.0/darwin-arm64/provider`
7. Writes `.nomos/providers.lock.json`

**Expected output**:

```
Initializing Nomos providers...
  → Resolving provider 'configs' (file v1.0.0)
  → Downloading from GitHub: autonomous-bits/nomos-provider-file
  ✓ Downloaded nomos-provider-file-1.0.0-darwin-arm64 (2.1 MB)
  ✓ Verified checksum
  ✓ Installed to .nomos/providers/file/1.0.0/darwin-arm64/provider
Provider lock file written to .nomos/providers.lock.json

Providers initialized successfully!
```

### 5. Inspect Lock File

Check the generated lock file:

```bash
cat .nomos/providers.lock.json
```

**Example content**:

```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "1.0.0",
      "os": "darwin",
      "arch": "arm64",
      "source": {
        "github": {
          "owner": "autonomous-bits",
          "repo": "nomos-provider-file",
          "asset": "nomos-provider-file-1.0.0-darwin-arm64"
        }
      },
      "checksum": "sha256:a1b2c3d4e5f6789...",
      "path": ".nomos/providers/file/1.0.0/darwin-arm64/provider"
    }
  ]
}
```

### 6. Build Configuration

Run `nomos build` to compile the configuration:

```bash
nomos build config.csl
```

**What happens**:
1. Nomos reads the lock file to locate provider binaries
2. Starts the file provider subprocess
3. Calls `Init` with config `{directory: "./data"}`
4. Calls `Fetch` for `["database", "prod"]`
5. Calls `Fetch` for `["database", "staging"]`
6. Merges fetched data into the compilation
7. Shuts down provider subprocess

**Expected output**:

```
Building config.csl...
  ✓ Starting provider 'configs' (file v1.0.0)
  ✓ Provider initialized successfully
  ✓ Fetching data from provider (2 requests)
  ✓ Configuration compiled successfully

Output written to: config.json
```

### 7. Verify Output

Check the compiled output:

```bash
cat config.json
```

**Expected output**:

```json
{
  "environments": {
    "production": {
      "database": {
        "host": "prod-db.example.com",
        "port": 5432,
        "database": "production",
        "username": "prod_user",
        "max_connections": 100,
        "ssl_mode": "require"
      }
    },
    "staging": {
      "database": {
        "host": "staging-db.example.com",
        "port": 5432,
        "database": "staging",
        "username": "staging_user",
        "max_connections": 50,
        "ssl_mode": "prefer"
      }
    }
  },
  "app_name": "example-app"
}
```

### 8. Upgrade Provider

To upgrade to a newer version:

```bash
# Update version in config.csl
sed -i '' 's/version = "1.0.0"/version = "1.1.0"/' config.csl

# Re-run init with --upgrade flag
nomos init --upgrade config.csl
```

**What happens**:
- Downloads the new version (v1.1.0)
- Updates lock file with new checksums
- Keeps old version installed (doesn't delete it)

**Lock file will now have**:

```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "1.1.0",
      ...
    }
  ]
}
```

## Troubleshooting

### Download Fails

**Error**: `failed to download provider from GitHub`

**Solutions**:
- Check internet connection
- Verify GitHub repo exists: https://github.com/autonomous-bits/nomos-provider-file
- Check if release exists for the specified version
- Ensure asset name matches convention (or is correct in source config)

### Checksum Mismatch

**Error**: `checksum verification failed`

**Solutions**:
- The download may be corrupted; try re-running `nomos init --force`
- Check if checksums file exists in the GitHub Release
- Report issue to provider maintainer if checksums are incorrect

### Platform Not Supported

**Error**: `no binary found for darwin/arm64`

**Solutions**:
- Check provider's GitHub Releases for available platforms
- Contact provider maintainer to request support for your platform
- Build provider from source for your platform

### Rate Limit

**Error**: `GitHub API rate limit exceeded`

**Solutions**:
- Wait an hour (unauthenticated requests have low limits)
- Set `GITHUB_TOKEN` environment variable with a personal access token:
  ```bash
  export GITHUB_TOKEN="ghp_your_token_here"
  nomos init config.csl
  ```

### macOS Quarantine

**Error**: macOS blocks downloaded provider from running

**Solutions**:
- macOS may quarantine downloaded executables
- Remove quarantine attribute:
  ```bash
  xattr -d com.apple.quarantine .nomos/providers/file/1.0.0/darwin-arm64/provider
  ```
- Or allow in System Settings → Privacy & Security

## Version Constraints

You can use semantic version constraints:

```csl
source file as configs {
  version = "^1.0.0"  // Any 1.x.x version
  // or
  version = "~1.2.0"  // 1.2.x only
  // or
  version = ">=1.0.0 <2.0.0"  // Range
  
  config = { directory = "./data" }
}
```

## Offline Usage

Once providers are installed, you can work offline:

1. **Initial install** (requires internet):
   ```bash
   nomos init config.csl
   ```

2. **Subsequent builds** (offline):
   ```bash
   nomos build config.csl
   # Uses already-downloaded binaries from .nomos/providers/
   ```

3. **Share with team** (optional):
   - Commit `.nomos/providers.lock.json` to version control
   - Team members run `nomos init` to get identical provider versions
   - Or commit `.nomos/providers/` directory for fully offline setup

## Key Takeaways

- `nomos init` downloads providers from GitHub Releases automatically
- Checksums are verified for security and integrity
- Lock file ensures reproducible builds across machines
- Providers are cached locally and reused for subsequent builds
- Version constraints allow flexible dependency management
- Offline usage is supported after initial download

## Next Steps

- Read the [Provider Authoring Guide](../../guides/provider-authoring-guide.md) to publish your own provider
- See [External Providers Migration Guide](../../guides/external-providers-migration.md) for migrating projects
- Try the [Local Provider](../local-provider/) example for development workflows

---

**Last updated**: 2025-10-31
