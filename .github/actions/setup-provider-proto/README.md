# Setup Provider Proto Action

A composite action that installs buf and generates Go code from protobuf definitions for the `provider-proto` library.

## Usage

```yaml
- name: Setup Provider Proto
  uses: ./.github/actions/setup-provider-proto
```

## What This Action Does

1. **Installs buf CLI** - Downloads and installs buf v1.47.2 for protobuf code generation
2. **Generates Go code** - Runs `buf generate` in `libs/provider-proto` to create gRPC stubs
3. **Displays output** - Shows which files were generated for verification

## Why This Action Exists

The `provider-proto` library contains Protocol Buffer definitions that need to be compiled into Go code. The generated files are in `.gitignore` and must be created during the build process.

This action ensures all CI workflows that depend on `provider-proto` (via `libs/compiler` or directly) have the required generated files before building or testing.

## Dependencies

- Go must be installed and available in PATH
- Internet access to download buf and protobuf plugins
- The `libs/provider-proto` directory must exist with valid protobuf definitions

## Generated Output

The action generates files in:
```
libs/provider-proto/gen/go/nomos/provider/v1/
├── provider.pb.go         # Protocol Buffer message definitions
└── provider_grpc.pb.go    # gRPC service stubs
```

## Maintenance

When updating the protobuf schema or buf version:
1. Update the buf version in this action
2. Test locally with `make generate` in `libs/provider-proto`
3. Verify CI passes with the new generated code
