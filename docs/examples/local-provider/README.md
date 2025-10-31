# Local Provider Installation Example

This example demonstrates how to install and use a provider from a local binary file.

## Scenario

You have a provider binary (either built locally or downloaded manually) that you want to use with Nomos without publishing it to GitHub Releases.

## Prerequisites

- Nomos CLI installed
- Provider binary (we'll use a mock for this example)

## Step-by-Step Guide

### 1. Project Structure

```
local-provider/
├── README.md                      # This file
├── config.csl                     # Nomos configuration
├── provider-binary/               # Provider binaries
│   ├── darwin-arm64/
│   │   └── nomos-provider-file   # Provider executable for macOS ARM64
│   └── linux-amd64/
│       └── nomos-provider-file   # Provider executable for Linux
└── .nomos/                        # Created by nomos init
    ├── providers/                 # Installed providers
    └── providers.lock.json        # Lock file
```

### 2. Obtain Provider Binary

For this example, we'll assume you have the `nomos-provider-file` binary.

**Option A: Download from GitHub Releases**

```bash
# Create directory for binaries
mkdir -p provider-binary/darwin-arm64

# Download provider for your platform
VERSION="1.0.0"
PLATFORM="darwin-arm64"  # or linux-amd64, linux-arm64, etc.

curl -L -o "provider-binary/${PLATFORM}/nomos-provider-file" \
  "https://github.com/autonomous-bits/nomos-provider-file/releases/download/v${VERSION}/nomos-provider-file-${VERSION}-${PLATFORM}"

# Make executable
chmod +x "provider-binary/${PLATFORM}/nomos-provider-file"
```

**Option B: Build Locally**

If you have the provider source code:

```bash
cd /path/to/nomos-provider-file
go build -o nomos-provider-file ./cmd/provider
mv nomos-provider-file /path/to/example/provider-binary/darwin-arm64/
chmod +x /path/to/example/provider-binary/darwin-arm64/nomos-provider-file
```

### 3. Create Nomos Configuration

Create `config.csl`:

```csl
// Declare the file provider
source file as configs {
  version = "1.0.0"
  config = {
    directory = "./data"
  }
}

// Import configuration from the provider
database_config = import configs["database"]["prod"]

// Use the imported configuration
config = {
  database = database_config
  app_name = "example-app"
}
```

### 4. Create Sample Data

Create data files that the file provider will read:

```bash
mkdir -p data/database

cat > data/database/prod.csl << 'EOF'
config = {
  host = "prod-db.example.com"
  port = 5432
  database = "production"
  max_connections = 100
}
EOF
```

### 5. Install Provider Locally

Use `nomos init` with the `--from` flag to specify the local binary path:

```bash
# For macOS ARM64
nomos init --from configs=./provider-binary/darwin-arm64/nomos-provider-file config.csl

# For Linux AMD64
nomos init --from configs=./provider-binary/linux-amd64/nomos-provider-file config.csl
```

**What happens**:
1. Nomos parses `config.csl` and finds the `file` provider with alias `configs`
2. Instead of downloading from GitHub, it copies/links the specified binary to `.nomos/providers/file/1.0.0/{os}-{arch}/provider`
3. Creates `.nomos/providers.lock.json` with the resolved provider information

**Expected output**:

```
Initializing Nomos providers...
  ✓ Provider 'configs' (file v1.0.0) installed from local path
Provider lock file written to .nomos/providers.lock.json

Providers initialized successfully!
```

### 6. Inspect Lock File

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
        "local": {
          "path": "./provider-binary/darwin-arm64/nomos-provider-file"
        }
      },
      "path": ".nomos/providers/file/1.0.0/darwin-arm64/provider",
      "checksum": "sha256:a1b2c3d4e5f6..."
    }
  ]
}
```

### 7. Build Configuration

Now run `nomos build` to compile the configuration:

```bash
nomos build config.csl
```

**What happens**:
1. Nomos reads the lock file to locate the provider binary
2. Starts the provider as a subprocess
3. Calls `Init` with the configuration from `config.csl`
4. Calls `Fetch` to retrieve data at path `["database", "prod"]`
5. Merges the fetched data into the compilation
6. Shuts down the provider subprocess

**Expected output**:

```
Building config.csl...
  ✓ Starting provider 'configs' (file v1.0.0)
  ✓ Provider initialized successfully
  ✓ Fetching data from provider
  ✓ Configuration compiled successfully

Output written to: config.json
```

### 8. Verify Output

Check the compiled output:

```bash
cat config.json
```

**Expected output**:

```json
{
  "database": {
    "host": "prod-db.example.com",
    "port": 5432,
    "database": "production",
    "max_connections": 100
  },
  "app_name": "example-app"
}
```

## Troubleshooting

### Provider Binary Not Executable

**Error**: `permission denied: ./provider-binary/darwin-arm64/nomos-provider-file`

**Solution**:
```bash
chmod +x ./provider-binary/darwin-arm64/nomos-provider-file
```

### Wrong Architecture

**Error**: `cannot execute binary file: Exec format error`

**Solution**: Make sure you're using the correct binary for your system architecture:
```bash
uname -m  # Check your architecture (x86_64 = amd64, arm64 = arm64)
```

### Provider Not Found

**Error**: `failed to find provider binary`

**Solution**: Verify the path to the binary is correct and the file exists:
```bash
ls -l ./provider-binary/darwin-arm64/nomos-provider-file
```

### macOS Quarantine

**Error**: macOS blocks the provider from running

**Solution**: Remove the quarantine attribute:
```bash
xattr -d com.apple.quarantine ./provider-binary/darwin-arm64/nomos-provider-file
```

## Key Takeaways

- `nomos init --from` installs providers from local binaries
- The provider binary is copied/linked to `.nomos/providers/` for consistent location
- Lock file tracks the source and checksum for reproducibility
- Useful for development, testing, and internal/custom providers
- Platform-specific binaries must match your system architecture

## Next Steps

- Try the [GitHub Releases Provider](../github-releases-provider/) example for production workflows
- Read the [Provider Authoring Guide](../../guides/provider-authoring-guide.md) to build your own provider
- See [External Providers Migration Guide](../../guides/external-providers-migration.md) for migrating existing projects

---

**Last updated**: 2025-10-31
