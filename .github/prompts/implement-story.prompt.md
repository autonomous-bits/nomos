---
description: 'Kick off implementation using the code-engineer chatmode; collect Feature (PRD) and Story issue identifiers and essential context.'
mode: code-engineer
---

# Implement Story — Kickoff Prompt

Provide the required identifiers so we can start implementation in code-engineer mode.

Required:
- Feature (PRD) issue: <number or URL>
- User story issue: <number or URL>

Optional context (helps speed and accuracy):
- Target module(s): <apps/... or libs/...>
- Target branch (if not main): <branch-name>
- Runtime/OS constraints: <e.g., linux/amd64>
- Performance/security constraints: <brief>
- Any linked designs/specs: <URLs>

What the engineer will do (aligned with code-engineer.chatmode.md):
- Standards & context
  - Fetch the Development Standards Constitution from the GitHub Copilot Space `general-standards` (owner: `pewpewpotato`).
  - Fetch Go standards (general, performance, security, testing) from the same space.
  - Review repo docs: `docs/architecture/go-monorepo-structure.md`.
- Research-first policy
  - When unsure or details are missing, proactively research via internet and available MCP servers (Microsoft Docs, official Go docs, repo docs, GitHub issues/discussions, Copilot Spaces). Summarize findings with brief citations.
- Test-Driven Development (MANDATORY)
  - Follow Red → Green → Refactor for each change.
  - Red: write a failing test for acceptance criteria and at least one edge case.
  - Green: implement minimal code to pass.
  - Refactor: improve design with tests green; repeat as needed.
  - For legacy code, write characterization tests first.
- Execution workflow
  - Confirm acceptance criteria and constraints; create a small todo list and a tiny contract (inputs/outputs, errors, success criteria).
  - Implement in small, cohesive steps; keep module boundaries clean; avoid import cycles.
  - Validate continuously: build, run tests (include `-race` where applicable).
- Quality gates (reported as PASS/FAIL with one‑line rationale)
  - TDD compliance, Build, Lint/Typecheck, Tests, Security, Coverage.
- Documentation & changelog
  - Update READMEs if behavior or public APIs change.
  - Update relevant module CHANGELOG(s) per `.github/instructions/changelog.instructions.md` (Keep a Changelog + SemVer). Create one if missing.
- PR readiness
  - Prepare concise summary of changes, risks, and validation.
  - Use `.github/instructions/commit-messages.instructions.md` and `.github/instructions/pull-request-description.instructions.md` when drafting messages on request.
- Safe operations
  - Never commit/push code directly from this session; prepare changes locally only. Follow module `AGENTS.md` for build/run instructions.

Reply with:
- Feature (PRD) issue: <number or URL>
- User story issue: <number or URL>
- Optional context (module/branch/constraints/links), if any
