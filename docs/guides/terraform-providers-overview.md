# Terraform Providers: How They Work

Last updated: 2025-10-29

This document summarizes how Terraform providers work based on HashiCorp’s official documentation, with links for deeper dives. It focuses on the provider side of Terraform’s plugin architecture and the points where providers interact with the Terraform CLI lifecycle.

## Big picture

Terraform is split into two parts:
- Terraform Core: the `terraform` CLI (a statically compiled Go binary) that parses configuration, manages state, builds the dependency graph, creates the plan, and executes it.
- Terraform Plugins: separate binaries (also written in Go) that implement providers and provisioners. Terraform Core starts them as separate processes and communicates with them over an RPC interface.

Providers are plugins that integrate Terraform with external platforms (clouds, SaaS, on‑prem APIs). They define the resources and data sources practitioners use in configuration. Terraform Core discovers the needed provider binaries and starts them as subprocesses of the `terraform` CLI, coordinating all interactions.

Reference: How Terraform Works With Plugins (HashiCorp)

## Provider responsibilities

Provider plugins are responsible for:
- Initializing SDKs/clients they need to talk to the target platform’s API.
- Authenticating with the platform (tokens, credentials, endpoints, regions).
- Defining resources (managed lifecycle objects) and data sources (read‑only lookups) exposed to Terraform configurations.
- Optionally defining helper functions that simplify configuration logic.

Provisioner plugins are different: they execute commands or scripts on a resource after creation or before destruction. Most modern workflows avoid provisioners in favor of cloud‑native capabilities.

## How Core and providers communicate

- Execution model: Terraform Core launches each required plugin as its own process (subprocess of the main `terraform` CLI execution).
- Transport: Core communicates with plugins via RPC using the Terraform Plugin Protocol (modern versions are gRPC‑based). As a provider author, the framework/SDK abstracts the low‑level protocol details.
- Contract: Providers expose schemas and CRUD/read functions for resources and data sources. Core calls these during plan/apply/destroy.

## Discovery, installation, and versioning

When you run `terraform init`, Core:
1) Determines which providers are required by scanning configuration.
2) Locates already-installed binaries in standard locations.
3) Selects a version that satisfies version constraints in configuration.
4) If needed, downloads the newest acceptable version from the Terraform Registry into `.terraform/providers/` for providers published there.
5) Writes/updates a dependency lock file so subsequent runs use the same versions until you run `init` again.

Notes:
- If multiple acceptable versions are installed locally, Core picks the newest installed version that matches the constraint, even if the registry has a newer one.
- `terraform init -upgrade` rechecks the registry and upgrades to newer acceptable versions, but only for providers whose acceptable versions are managed under `.terraform/providers/`.
- For providers not published in the Registry, installation is manual (or via custom discovery), and `init` will fail if an acceptable version isn’t found.

See: Provider installation and development overrides; Provider version constraints.

## Role during the Terraform lifecycle

- init: Providers are discovered/installed and configured (credentials, endpoints, aliases, etc.).
- plan: Core queries providers to read current state from the platform and to compute the diff for resources/data sources defined in the configuration.
- apply: Core invokes provider operations to create, update, or delete resources. Providers make the actual API calls and report results/diagnostics back to Core.
- destroy: Apply in the “delete” direction; providers implement the deletion semantics.

## Key concepts

- Resources vs data sources: Resources manage lifecycle (create/read/update/delete). Data sources are read‑only and used to look up existing information.
- Schemas: Providers declare the schema for each resource/data source (attributes, types, required/optional/computed) so Core can validate configs and construct plans.
- Authentication/config: Providers typically accept configuration blocks (e.g., credentials, region, custom endpoints). Configuration can be overridden or aliased to support multiple instances.
- Diagnostics: Providers must surface clear errors and warnings; Core renders them consistently.

## Developing providers (quick pointers)

While this document is an overview, the following links are the canonical references for implementation details:
- Plugin Development hub
- Terraform Plugin Protocol
- Terraform Plugin Framework (recommended) and SDKv2
- Testing providers
- Logging and debugging
- Publishing to the Terraform Registry
- Mux (combining/transitioning frameworks)

## Takeaways

- Providers are separate executables that implement the domain‑specific logic to talk to external APIs.
- Terraform Core handles discovery, selection, version pinning, graph planning, and orchestrating RPC calls to providers.
- `terraform init` is where provider installation, version resolution, and lock file updates happen.
- During plan/apply/destroy, Core delegates CRUD/read operations to providers, which perform the real API calls and return results and diagnostics.

---

## Nomos External Providers

Nomos implements a similar external provider model inspired by Terraform's architecture, adapted for configuration compilation rather than infrastructure management.

### Key Similarities

- **Subprocess model**: Nomos compiler starts provider executables as separate processes
- **gRPC communication**: Providers communicate via gRPC (like modern Terraform providers)
- **Discovery and installation**: `nomos init` discovers and installs provider binaries (similar to `terraform init`)
- **Lock file**: `.nomos/providers.lock.json` ensures reproducible builds (similar to `.terraform.lock.hcl`)
- **Version constraints**: Providers are versioned and constraints are enforced

### Key Differences

- **Purpose**: Nomos providers fetch configuration data for compilation, not manage infrastructure lifecycle
- **Operations**: Providers implement `Init`, `Fetch`, `Info`, `Health`, and `Shutdown` (not CRUD operations)
- **No central registry**: Providers are distributed via GitHub Releases or local paths, not a centralized registry
- **Simpler contract**: Nomos providers have a focused contract for data retrieval

### For Nomos Users

If you're using Nomos providers:

1. **Declare providers in `.csl` files**:
   ```csl
   source file as configs {
     version = "1.0.0"
     config = {
       directory = "./configs"
     }
   }
   
   config = import configs["database"]["prod"]
   ```

2. **Install providers with `nomos init`**:
   ```bash
   # From GitHub Releases (default)
   nomos init config.csl
   
   # From local path
   nomos init --from configs=/path/to/provider config.csl
   ```

3. **Build configurations**:
   ```bash
   nomos build config.csl
   ```

**Learn more**: See [External Providers Migration Guide](./external-providers-migration.md) for detailed usage instructions.

### For Provider Authors

If you're building a Nomos provider:

1. **Implement the gRPC contract** defined in `libs/provider-proto`
2. **Follow naming conventions**: `nomos-provider-{type}-{version}-{os}-{arch}`
3. **Publish to GitHub Releases** with checksums
4. **Document configuration** requirements and usage

**Learn more**: See [Provider Authoring Guide](./provider-authoring-guide.md) for complete implementation instructions.

### Comparison Table

| Aspect | Terraform | Nomos |
|--------|-----------|-------|
| **Purpose** | Infrastructure management | Configuration compilation |
| **Communication** | gRPC | gRPC |
| **Process model** | Subprocess per provider | Subprocess per provider alias |
| **Operations** | CRUD + Read | Init + Fetch |
| **Distribution** | Terraform Registry | GitHub Releases + local |
| **Discovery** | `terraform init` | `nomos init` |
| **Lock file** | `.terraform.lock.hcl` | `.nomos/providers.lock.json` |
| **Version constraints** | HCL syntax | Semver in source declarations |

### Additional Resources

- [External Providers Architecture](../architecture/nomos-external-providers-feature-breakdown.md) - Technical specification
- [Provider Authoring Guide](./provider-authoring-guide.md) - Complete guide for building providers
- [External Providers Migration Guide](./external-providers-migration.md) - Migration from in-process providers
- [Provider Proto Documentation](../../libs/provider-proto/README.md) - gRPC contract reference

---

Sources
- How Terraform Works With Plugins (HashiCorp)
- Additional links from HashiCorp docs: Plugin Protocol, Framework/SDKv2, Testing, Logging, Publishing, Mux