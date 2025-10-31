# Nomos Provider Examples

This directory contains examples demonstrating how to use Nomos with external providers.

## Examples

### [Local Provider Installation](./local-provider/)

Demonstrates installing and using a provider from a local binary file using `nomos init --from`.

**Use case**: Testing providers during development or using custom/internal providers not published to GitHub.

### [GitHub Releases Provider](./github-releases-provider/)

Demonstrates installing a provider from GitHub Releases and building with it.

**Use case**: Production usage with published providers from the Nomos ecosystem or third parties.

## Quick Start

Each example directory contains:
- `README.md` - Step-by-step instructions
- `config.csl` - Example Nomos configuration
- `.nomos/` - Provider configuration (created by `nomos init`)

## Prerequisites

- Nomos CLI installed (`nomos` command available)
- Go 1.22+ (for building example providers)
- Basic understanding of Nomos configuration syntax

## Running the Examples

Navigate to an example directory and follow its README:

```bash
cd local-provider
cat README.md
```

## Additional Resources

- [Provider Authoring Guide](../guides/provider-authoring-guide.md) - Build your own provider
- [External Providers Migration Guide](../guides/external-providers-migration.md) - Migrate from in-process providers
- [Terraform Providers Overview](../guides/terraform-providers-overview.md) - Comparison with Terraform

---

**Last updated**: 2025-10-31
