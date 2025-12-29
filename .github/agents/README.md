# Agent System Architecture

## Overview

The agent system uses **generic specialists with context injection** rather than component-specific agents. This architecture separates generic knowledge (Go best practices, testing strategies) from project-specific knowledge (APIs, conventions, patterns).

## Generic Specialist Agents

### Core Agents

1. **orchestrator.agent.md**
   - Coordinates all specialists through iterative phase-gated workflow
   - Reads AGENTS.md files from affected modules
   - Extracts relevant context based on task
   - Briefs specialists with structured input containing task and context
   - Receives structured output from specialists
   - Validates outputs against project patterns
   - **Iteratively resolves problems**: When a phase reports issues, analyzes the problem and delegates to the appropriate specialist for resolution
   - Continues iteration until task is fully completed

2. **go-specialist.agent.md**
   - Generic Go implementation expert
   - Knows Go idioms, patterns, best practices
   - Receives project-specific context from orchestrator
   - Implements following project conventions from AGENTS.md

3. **go-tester.agent.md**
   - Generic Go testing expert  
   - Table-driven tests, mocking, benchmarks, coverage
   - Receives testing context from AGENTS.md
   - Follows project-specific test patterns

4. **architecture-specialist.agent.md**
   - Generic architecture design expert
   - Trade-off evaluation, ADRs, system design
   - Receives architectural constraints from AGENTS.md
   - Works with domain specialists as needed

5. **security-reviewer.agent.md**
   - Generic security review expert
   - Input validation, secrets management, vulnerabilities
   - Receives security boundaries from AGENTS.md

6. **documentation-specialist.agent.md**
   - Generic documentation expert
   - Godoc, README, architecture docs, migration guides
   - Receives docs structure from AGENTS.md

## Project-Specific Knowledge (AGENTS.md Files)

Each module has an `AGENTS.md` file containing project-specific patterns:

```
apps/command-line/AGENTS.md        # CLI patterns, command structure, exit codes
libs/compiler/AGENTS.md             # Compilation pipeline, provider lifecycle
libs/parser/AGENTS.md               # AST design, scanner patterns, golden tests
libs/provider-proto/AGENTS.md       # gRPC protocol patterns
libs/provider-downloader/AGENTS.md  # Download, caching, checksums
```

### AGENTS.md Content Structure

Each AGENTS.md file contains:
- **Project-specific patterns**: How this module does things
- **API usage examples**: How to use module APIs correctly
- **Testing conventions**: Test organization, fixtures, benchmarks
- **Error handling**: Sentinel errors, wrapping patterns
- **Integration points**: How this module integrates with others

## Iterative Workflow

The orchestrator manages an **iterative problem-resolution workflow**. When any phase reports problems, the orchestrator analyzes the issue and delegates to the appropriate specialist for resolution. This continues until the task is fully completed.

```
┌─────────────────┐
│  User Request   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│  Orchestrator                           │
│  1. Identify affected modules           │
│  2. Read AGENTS.md files                │
│  3. Extract relevant sections           │
│  4. Determine current phase             │
└────────┬────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│  Brief Specialist (Structured Input)    │
│  {                                      │
│    "task": "...",                      │
│    "phase": "architecture|impl|test",  │
│    "context": { AGENTS.md excerpts },   │
│    "previous_output": { ... },         │
│    "issues_to_resolve": [ ... ]        │
│  }                                      │
└────────┬────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│  Specialist Processes                   │
│  - Reads structured input               │
│  - Performs phase-specific work         │
│  - Generates structured output          │
└────────┬────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│  Specialist Output (Structured)         │
│  {                                      │
│    "status": "success|problem|blocked", │
│    "artifacts": { files, docs, ... },  │
│    "problems": [ ... ],                │
│    "recommendations": [ ... ]          │
│  }                                      │
└────────┬────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│  Orchestrator Validates & Decides       │
│  - Validates against AGENTS.md patterns │
│  - If problems: analyze & delegate      │
│  - If success: proceed to next phase    │
│  - If blocked: escalate or pivot        │
└────────┬────────────────────────────────┘
         │
         ├─── Problem? ───┐
         │                │
         ▼                ▼
    Next Phase      Resolve Problem
    or Complete     (Loop back to
                     appropriate
                     specialist)
```

### Problem Resolution Loop

When a specialist reports problems:
1. **Orchestrator analyzes** the problem type and severity
2. **Determines which specialist** can resolve it (may be different from reporter)
3. **Briefs the resolver** with:
   - Original task context
   - Problem description
   - Previous attempts (if any)
   - Specific areas to address
4. **Specialist resolves** and returns updated output
5. **Orchestrator validates** resolution
6. **Continues** until problem is resolved or escalation needed

## Standard Input/Output Formats

All specialists use structured formats for communication with the orchestrator. This ensures consistency, traceability, and enables the iterative problem-resolution workflow.

### Specialist Input Format

Every specialist receives a structured brief:

```json
{
  "task": {
    "id": "unique-task-id",
    "description": "Human-readable task description",
    "type": "architecture|implementation|testing|security|documentation"
  },
  "phase": "current-phase-name",
  "context": {
    "modules": ["libs/compiler", "apps/command-line"],
    "patterns": {
      "libs/compiler": {
        "api_usage": "...",
        "error_handling": "...",
        "testing_conventions": "..."
      }
    },
    "constraints": ["Must maintain backward compatibility"],
    "integration_points": ["CLI uses compiler.Parse()"]
  },
  "previous_output": {
    "phase": "architecture",
    "artifacts": { /* previous phase deliverables */ },
    "decisions": ["Use 3-stage pipeline"]
  },
  "issues_to_resolve": [
    {
      "id": "issue-1",
      "severity": "high|medium|low",
      "description": "Detailed problem description",
      "reported_by": "go-tester",
      "context": { /* relevant context */ }
    }
  ]
}
```

### Specialist Output Format

Every specialist returns structured output:

```json
{
  "status": "success|problem|blocked",
  "phase": "completed-phase-name",
  "artifacts": {
    "files": [
      {
        "path": "libs/compiler/validator.go",
        "action": "created|modified|deleted",
        "summary": "Added validation logic"
      }
    ],
    "documentation": [
      {
        "type": "godoc|readme|adr",
        "location": "...",
        "summary": "..."
      }
    ],
    "tests": [
      {
        "path": "libs/compiler/validator_test.go",
        "coverage": "95%",
        "summary": "Table-driven tests for validation"
      }
    ]
  },
  "problems": [
    {
      "id": "new-problem-id",
      "severity": "high|medium|low",
      "description": "Problem description",
      "suggested_resolver": "architecture-specialist|go-specialist|...",
      "context": { /* relevant context */ }
    }
  ],
  "recommendations": [
    "Consider adding integration test",
    "May want to benchmark performance"
  ],
  "validation_results": {
    "patterns_followed": ["3-stage pipeline", "error collection"],
    "conventions_adhered": ["Table-driven tests", "godoc comments"],
    "deviations": [
      {
        "pattern": "...",
        "reason": "...",
        "approved_by": "orchestrator|user"
      }
    ]
  },
  "next_phase_ready": true
}
```

### Format Benefits

1. **Traceability**: Clear record of decisions and changes
2. **Iteration**: Previous outputs feed into next iteration
3. **Problem Resolution**: Structured problems enable targeted delegation
4. **Validation**: Orchestrator can validate against patterns systematically
5. **Debugging**: Easy to identify where issues occurred
6. **Handoff**: Clean phase transitions with complete context

### Agent Awareness

All agents (orchestrator and specialists) are aware of these formats:
- **Orchestrator** constructs input briefs and parses output
- **Specialists** expect input in this format and produce conformant output
- **Format is extensible**: Can add new fields as needed
- **Validation**: Orchestrator validates format compliance

## Example: Adding a Feature

**User Request**: "Add validation command to CLI"

### Phase 1: Context Gathering

Orchestrator identifies affected modules:
- `apps/command-line` (new command)
- `libs/compiler` (validation mode)

Orchestrator reads:
- `apps/command-line/AGENTS.md`
  - Command structure (switch-based routing)
  - Exit codes (0, 1, 2)
  - Output formats (JSON with deterministic serialization)
- `libs/compiler/AGENTS.md`
  - 3-stage pipeline (parse → resolve → merge)
  - Can skip merge for validation
  - Error collection pattern

### Phase 2: Architecture

Brief architecture specialist:
```
Task: Design validation command

Context from apps/command-line/AGENTS.md:
- Simple switch-based routing (not Cobra)
- Exit codes: 0 (valid), 1 (invalid), 2 (error)
- JSON output with sorted keys

Context from libs/compiler/AGENTS.md:
- Run parse + resolve stages only
- Skip merge stage for validation
- Collect all errors, don't stop at first
```

Architecture specialist designs solution following these patterns.

### Phase 3: Implementation

Brief go-specialist:
```
Task: Implement validation command

Context from apps/command-line/AGENTS.md:
- Add case "validate": return cli.Validate(args)
- Use internal/serialize.ToJSON() for output
- Return exit codes per convention

Context from libs/compiler/AGENTS.md:
- Call compiler.Parse(ctx, file)
- Call compiler.Resolve(ctx, ast)
- Skip compiler.Merge()
- Use errors.Join() for multiple errors
```

Brief go-tester:
```
Task: Write tests

Context from apps/command-line/AGENTS.md:
- Integration tests in test/ directory
- Use testdata/ fixtures
- Table-driven tests with scenarios:
  * valid config → exit 0
  * syntax error → exit 1  
  * missing file → exit 2
```

## Benefits

### 1. Single Source of Truth
- Go best practices defined once in generic agents
- No duplication across component-specific agents
- Update standards in one place

### 2. Dynamic Context
- AGENTS.md is runtime context, not embedded knowledge
- Easy to update project patterns without changing agents
- Specialists receive only relevant context for their task

### 3. Scalability
- Same generic agents work for any Go project
- Just add AGENTS.md files to new modules
- No need to create project-specific agents

### 4. Maintainability
- Generic agents are stable (rarely change)
- Project evolution happens in AGENTS.md files
- Clear separation between generic and specific knowledge

### 5. Flexibility
- Different projects can have different patterns
- Specialists adapt to project conventions via context
- Works for monorepos with multiple modules

## Archived Agents

Component-specific agents were archived on December 29, 2025:
- `nomos-parser-specialist.agent.md`
- `nomos-compiler-specialist.agent.md`
- `nomos-cli-specialist.agent.md`
- `nomos-provider-specialist.agent.md`
- `nomos-testing-specialist.agent.md`

These mixed generic Go knowledge with Nomos-specific patterns, causing:
- Duplication of Go best practices across agents
- Difficulty updating standards (change in 5+ places)
- Tight coupling of generic and specific knowledge

See `archive/README.md` for details.

## Migration for Other Projects

To use this agent system in another Go project:

1. **Keep the generic agents** as-is
   - `orchestrator.agent.md`
   - `go-specialist.agent.md`
   - `go-tester.agent.md`
   - `architecture-specialist.agent.md`
   - `security-reviewer.agent.md`
   - `documentation-specialist.agent.md`

2. **Create AGENTS.md files** for your modules
   - Document your project-specific patterns
   - Include API usage examples
   - Specify testing conventions
   - Describe integration points

3. **Update orchestrator context** (if needed)
   - Modify AGENTS.md discovery pattern for your structure
   - Adjust context extraction for your needs

That's it! The generic agents work with any Go project that has AGENTS.md files.

## Quick Start

### For Orchestrator

When you receive a task:
1. Identify affected modules (e.g., `libs/compiler`, `apps/cli`)
2. Read their AGENTS.md files
3. Extract relevant sections for the task
4. Construct structured input brief (see Standard Input/Output Formats)
5. Brief specialists with focused context
6. Parse specialist structured output
7. Validate implementations against project patterns
8. **If problems reported**: Analyze and delegate to appropriate specialist
9. **Iterate** until task is complete or escalation needed
10. Proceed to next phase when current phase succeeds

### For Specialists

When you receive a brief:
1. **Parse structured input** (see Standard Input/Output Formats)
2. Review the task description and phase
3. Read the context provided from AGENTS.md
4. Review previous phase outputs if available
5. Address any issues_to_resolve from previous iterations
6. Follow the project-specific patterns
7. Use the APIs mentioned in context
8. Adhere to conventions specified
9. **Generate structured output** with status, artifacts, and any problems
10. Report problems clearly with suggested resolvers

## Questions?

- **"How do I update Go best practices?"** → Edit the generic agent files (go-specialist.agent.md, go-tester.agent.md)
- **"How do I update project patterns?"** → Edit the AGENTS.md files in your modules
- **"Can I add new modules?"** → Yes! Just create an AGENTS.md file for the new module
- **"Do I need component-specific agents?"** → No! Generic agents + AGENTS.md context is sufficient

---

*Architecture effective: December 29, 2025*
