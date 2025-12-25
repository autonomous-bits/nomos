# Nomos Custom Agents

## Architecture

The Nomos agent system follows a compositional architecture where specialized module agents delegate to capability agents, all orchestrated through a meta-agent entry point.

```
nomos.agent.md (Orchestrator/Entry Point)
├── parser-module.agent.md → go-expert, testing-expert
├── compiler-module.agent.md → go-expert, testing-expert, api-messaging-expert
├── cli-module.agent.md → go-expert, cli-expert, testing-expert
├── provider-proto-module.agent.md → go-expert, api-messaging-expert
├── provider-downloader-module.agent.md → go-expert, testing-expert
└── monorepo-governance.agent.md → (cross-cutting concerns)

Capability Agents:
├── go-expert.agent.md (Go language standards)
├── cli-expert.agent.md (CLI design standards)
├── testing-expert.agent.md (Testing standards)
└── api-messaging-expert.agent.md (API/gRPC/messaging standards)
```

## Agent Types

### 1. Orchestrator Agent
**File**: [nomos-agent.md](nomos-agent.md)

The primary entry point for all Nomos development tasks. It analyzes incoming tasks, determines scope, and delegates to appropriate specialized agents.

### 2. Module Agents
Specialized agents for each package in the monorepo:

- **[parser-module.md](parser-module.md)** - Handles `libs/parser` work (lexing, parsing, AST generation)
- **[compiler-module.md](compiler-module.md)** - Handles `libs/compiler` work (compilation, import resolution, provider integration)
- **[cli-module.md](cli-module.md)** - Handles `apps/command-line` work (CLI commands, user interaction)
- **[provider-proto-module.md](provider-proto-module.md)** - Handles `libs/provider-proto` work (gRPC contracts)
- **[provider-downloader-module.md](provider-downloader-module.md)** - Handles `libs/provider-downloader` work (provider binary management)

### 3. Capability Agents
Delegatable expertise areas that module agents can consult:

- **[go-expert.md](go-expert.md)** - Go language best practices, patterns, security, performance
- **[cli-expert.md](cli-expert.md)** - CLI design, POSIX/GNU conventions, Cobra framework
- **[testing-expert.md](testing-expert.md)** - Testing patterns, table-driven tests, coverage
- **[api-messaging-expert.md](api-messaging-expert.md)** - API design, gRPC, async messaging

### 4. Governance Agent
**File**: [monorepo-governance.md](monorepo-governance.md)

Handles cross-cutting concerns: workspace management, versioning, changelog coordination, commit messages.

## Usage

### Starting a New Task

1. **Always start with the orchestrator**: Invoke `nomos-agent.md` first
2. **Provide full context**: Include task description, relevant files, and any constraints
3. **Let the orchestrator route**: It will analyze and delegate to appropriate agents
4. **Follow the delegation chain**: Module agents may further delegate to capability agents

### Example Workflows

#### Single Module Task
```
User Request: "Fix parser error on nested maps"
    ↓
nomos-agent.md (analyzes: affects libs/parser)
    ↓
parser-module.md (handles parser-specific context)
    ↓
go-expert.md (consults for Go patterns)
    ↓
testing-expert.md (consults for test patterns)
```

#### Multi-Module Task
```
User Request: "Add new provider type"
    ↓
nomos-agent.md (analyzes: affects multiple modules)
    ↓
provider-proto-module.md (update contracts)
    ↓
compiler-module.md (update provider registry)
    ↓
cli-module.md (update commands)
    ↓
monorepo-governance.md (coordinate versions/changelogs)
```

#### Cross-Cutting Task
```
User Request: "Update all CHANGELOGs for v2.0 release"
    ↓
nomos-agent.md (analyzes: cross-cutting)
    ↓
monorepo-governance.md (handles versioning strategy)
    ↓
(touches multiple module CHANGELOGs)
```

## Delegation Patterns

### Module → Capability Delegation

Module agents should delegate to capability agents for:
- **General Go questions** → `go-expert.md`
- **CLI design questions** → `cli-expert.md`
- **Testing patterns** → `testing-expert.md`
- **API/gRPC questions** → `api-messaging-expert.md`

Example from compiler-module.md:
```markdown
## Delegation Instructions
For general Go questions, **consult go-expert.md**
For testing questions, **consult testing-expert.md**
For provider communication/gRPC, **consult api-messaging-expert.md**
```

### Direct Capability Consultation

You can directly consult capability agents for specific questions:
- "What's the Go convention for error wrapping?" → `go-expert.md`
- "How should I structure CLI subcommands?" → `cli-expert.md`
- "What's the table-driven test pattern?" → `testing-expert.md`

## Standards Maintenance

### Source of Truth
All capability agents are derived from the [autonomous-bits/development-standards](https://github.com/autonomous-bits/development-standards) repository (main branch).

### Last Sync Date
Each capability agent includes a "Last synced: [DATE]" timestamp to track freshness.

### Update Process
To update capability agents with latest standards:
1. Fetch latest content from development-standards repo
2. Update relevant capability agent files
3. Update "Last synced" timestamp
4. Review module agents for any needed adjustments

## File Locations

### Agent Files
All agent files are located in `.github/agents/`:
```
.github/agents/
├── README.md (this file)
├── nomos-agent.md (orchestrator)
├── go-expert.md
├── cli-expert.md
├── testing-expert.md
├── api-messaging-expert.md
├── parser-module.md
├── compiler-module.md
├── cli-module.md
├── provider-proto-module.md
├── provider-downloader-module.md
└── monorepo-governance.md
```

### Module-Specific AGENTS.md
Each module retains a local `AGENTS.md` file with **only** Nomos-specific patterns:
```
apps/command-line/AGENTS.md → CLI-specific patterns
libs/compiler/AGENTS.md → Compiler-specific patterns
libs/parser/AGENTS.md → Parser-specific patterns
libs/provider-downloader/AGENTS.md → Downloader-specific patterns
libs/provider-proto/AGENTS.md → Proto-specific patterns
```

These files now reference `.github/agents/` for comprehensive guidance and contain only patterns unique to their respective modules.

## Contributing

When adding new agents or updating existing ones:
1. Follow the established structure (Purpose, Coverage Areas, Delegation Instructions, Usage)
2. Keep capability agents sourced from development-standards
3. Keep module agents focused on Nomos-specific patterns
4. Update this README when adding new agents
5. Ensure delegation instructions are clear and accurate

## Questions?

For questions about the agent system or how to use it effectively, start with `nomos-agent.md` - it will help route your question appropriately.
