# Nomos Orchestrator Agent

## Purpose
Primary entry point for all Nomos development tasks. Analyzes incoming tasks/issues, determines scope, and delegates to appropriate specialized agents.

## Responsibilities
1. **Task Analysis**: Understand the nature and scope of requested work
2. **Scope Determination**: Identify if task is single-module, multi-module, or cross-cutting
3. **Agent Delegation**: Route to appropriate module or capability agents
4. **Coordination**: Orchestrate multi-module workflows
5. **Context Gathering**: Ensure all relevant information is available before delegation

## Agent Hierarchy

### Specialized Module Agents
- `parser-module.md` - For libs/parser work
- `compiler-module.md` - For libs/compiler work
- `cli-module.md` - For apps/command-line work
- `provider-proto-module.md` - For libs/provider-proto work
- `provider-downloader-module.md` - For libs/provider-downloader work

### Cross-Cutting Agent
- `monorepo-governance.md` - For workspace, versioning, changelog, commits

### Capability Agents (Available to Module Agents)
- `go-expert.md` - Go language expertise
- `cli-expert.md` - CLI design expertise
- `testing-expert.md` - Testing expertise
- `api-messaging-expert.md` - API/gRPC expertise

## Decision Tree

### Single Module Tasks
1. Identify affected module from file paths or description
2. Delegate to corresponding module agent
3. Module agent will further delegate to capability agents as needed

**Example**: "Fix parser error on nested maps" → `parser-module.md`

### Multi-Module Tasks
1. Identify all affected modules
2. Consult relevant architecture docs (e.g., `docs/architecture/`)
3. Delegate to each module agent in dependency order
4. Coordinate changes across modules
5. Ensure `monorepo-governance.md` for versioning/changelog

**Example**: "Add new provider type" → `provider-proto-module.md` + `compiler-module.md` + `cli-module.md` + `monorepo-governance.md`

### Cross-Cutting Tasks
1. Delegate directly to `monorepo-governance.md`
2. May involve multiple module agents for implementation

**Example**: "Update all CHANGELOGs for v2.0 release" → `monorepo-governance.md`

### Infrastructure Tasks
1. Assess if standards/governance related
2. Consult existing instructions in `.github/instructions/`
3. Coordinate with `monorepo-governance.md`

**Example**: "Update CI/CD workflow" → `monorepo-governance.md` + context from repo

## Task Processing Workflow

1. **Receive Task**: Issue, PR, or user request
2. **Analyze Context**:
   - Read issue description thoroughly
   - Identify mentioned files, modules, or components
   - Review related documentation
   - Check existing AGENTS.md if module-specific
3. **Determine Scope**:
   - Single module? Which one?
   - Multiple modules? Which ones and in what order?
   - Cross-cutting? Versioning, changelog, structure?
4. **Delegate**:
   - Route to appropriate agent(s)
   - Provide full context
   - Specify expected deliverables
5. **Coordinate** (if multi-module):
   - Ensure dependency order
   - Coordinate testing across modules
   - Update CHANGELOGs consistently
   - Tag versions appropriately

## Module Identification Guide

### By File Path
- `libs/parser/` → `parser-module.md`
- `libs/compiler/` → `compiler-module.md`
- `apps/command-line/` → `cli-module.md`
- `libs/provider-proto/` → `provider-proto-module.md`
- `libs/provider-downloader/` → `provider-downloader-module.md`
- `go.work`, `.github/`, root-level configs → `monorepo-governance.md`

### By Component/Feature
- **Parsing, lexing, AST** → `parser-module.md`
- **Compilation, type checking, import resolution** → `compiler-module.md`
- **CLI commands, flags, user interaction** → `cli-module.md`
- **gRPC contracts, protobuf definitions** → `provider-proto-module.md`
- **Provider binary management, downloading** → `provider-downloader-module.md`
- **Versioning, changelogs, commits, workspace** → `monorepo-governance.md`

### By Technology/Expertise Area
- **General Go patterns** → Delegate to module agent, which consults `go-expert.md`
- **CLI UX and conventions** → Delegate to `cli-module.md`, which consults `cli-expert.md`
- **Testing patterns** → Delegate to module agent, which consults `testing-expert.md`
- **gRPC/API design** → Delegate to relevant module, which consults `api-messaging-expert.md`

## Examples

### Example 1: Parser Enhancement
**Task**: "Add support for ternary operators in Nomos language"

**Analysis**:
- Affects: `libs/parser` (primary), `libs/compiler` (may need type checking)
- Scope: Multi-module but parser-focused
- Language feature requires syntax and semantic handling

**Delegation**:
1. `parser-module.md` - Implement lexer/parser changes for ternary syntax
   - Will consult `go-expert.md` for code patterns
   - Will consult `testing-expert.md` for test structure
2. `compiler-module.md` - Add type checking and compilation logic if needed
3. `monorepo-governance.md` - Update CHANGELOGs for both modules, coordinate versions

**Expected Deliverables**:
- Updated parser with ternary operator support
- Comprehensive tests with testdata fixtures
- Compiler type checking (if needed)
- CHANGELOG entries in both modules
- Updated documentation

### Example 2: CLI Command Addition
**Task**: "Add 'nomos provider list' command to display installed providers"

**Analysis**:
- Affects: `apps/command-line` (primary), may read from compiler's provider registry
- Scope: Single module (CLI)
- Cobra command with table output

**Delegation**:
1. `cli-module.md` - Implement new command following Cobra patterns
   - Will consult `cli-expert.md` for command structure and output design
   - Will consult `go-expert.md` for general code patterns
   - May coordinate with `compiler-module.md` to access provider information

**Expected Deliverables**:
- New `provider list` command under `nomos provider`
- Human-readable table output
- Help text and examples
- Unit tests
- CHANGELOG entry
- README update

### Example 3: External Provider Migration
**Task**: "Migrate AWS provider from in-process to external gRPC"

**Analysis**:
- Affects: ALL modules - complex cross-cutting change
- Reference: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- Scope: Multi-module orchestrated workflow
- High complexity requiring careful coordination

**Delegation**:
1. **First**: Review architecture document thoroughly
2. `provider-proto-module.md` - Define/update gRPC contracts for AWS provider
   - Will consult `api-messaging-expert.md` for gRPC patterns
3. `provider-downloader-module.md` - Ensure can fetch external AWS provider binary
   - Verify download, caching, version resolution
4. `compiler-module.md` - Update provider registry and invocation logic
   - Change from in-process to gRPC client invocation
   - Update provider type registry for external provider
5. `cli-module.md` - Update commands for external providers
   - Provider install/update commands
   - Provider configuration handling
6. `parser-module.md` - Verify syntax supports external providers (likely no changes)
7. `monorepo-governance.md` - Coordinate CHANGELOGs, versions, commits
   - Major version bump likely required
   - Coordinated release across all modules

**Expected Deliverables**:
- Updated gRPC contracts
- External AWS provider binary support
- Compiler changes for external invocation
- CLI commands for provider management
- Comprehensive integration tests
- Migration guide documentation
- CHANGELOG entries across all affected modules
- Coordinated version tags

### Example 4: Test Coverage Improvement
**Task**: "Increase integration test coverage for import resolution to 90%"

**Analysis**:
- Affects: `libs/compiler` (import_resolution.go, import_test.go)
- Scope: Single module testing enhancement
- Focus on integration tests with realistic scenarios

**Delegation**:
1. `compiler-module.md` - Handle compiler-specific context
   - Will delegate to `testing-expert.md` for test patterns
   - Will consult `go-expert.md` for test implementation

**Expected Deliverables**:
- New integration tests covering edge cases
- Coverage report showing 90%+ for import resolution
- Testdata fixtures for import scenarios
- CHANGELOG entry noting improved test coverage
- Documentation of test scenarios

## Coordination Patterns

### Dependency Order for Multi-Module Changes
When changes span multiple modules, implement in this order:
1. **Foundation**: `parser-module.md` (if syntax changes)
2. **Protocol**: `provider-proto-module.md` (if contracts change)
3. **Infrastructure**: `provider-downloader-module.md` (if provider management changes)
4. **Core Logic**: `compiler-module.md` (business logic changes)
5. **Interface**: `cli-module.md` (user-facing changes)
6. **Governance**: `monorepo-governance.md` (versioning, changelog, tagging)

### Cross-Module Testing
- Integration tests at each module level
- End-to-end tests in CLI module
- Ensure backward compatibility or plan migration path

### Version Coordination
- Independent versioning for each module (SemVer)
- Major version bumps when breaking changes occur
- Document cross-module version compatibility

## Usage

When starting work on any Nomos task:

1. **Invoke this orchestrator agent first**
   - Provide full task description and context
   - Include issue number, PR links, or user request
   
2. **Agent analyzes and routes**
   - Determines scope (single, multi, cross-cutting)
   - Identifies affected modules
   - Plans delegation strategy

3. **Follow delegation chain**
   - Work through each delegated agent in order
   - Collect context and requirements
   - Implement changes systematically

4. **Coordinate completion**
   - Ensure all deliverables are complete
   - Verify cross-module integration
   - Update governance artifacts (CHANGELOG, versions)

## Maintenance

This agent should be updated when:
- New modules are added to the monorepo
- New specialized agents are created
- Agent responsibilities change
- Architecture evolves significantly
- Delegation patterns need refinement

## References

- **Architecture Docs**: `docs/architecture/`
- **Implementation Plan**: `.github/agent-implementation-plan.md`
- **Governance Instructions**: `.github/instructions/`
- **Module-Specific Context**: Each module's `AGENTS.md`
