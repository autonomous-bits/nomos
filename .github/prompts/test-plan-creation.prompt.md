---
mode: qa-story-reviewer
description: "Create per-story QA test plans aligned with PRD sub-issues, with mandatory integration tests and BDD coverage."
---

# Input
- ${input:issueId:placeholder}: This github PRD issue.

# QA Test Plan Creation

 - Review the provided requirements in the GitHub PRD Issue, ALL GitHub PRD issue COMMENTS and each sub-issue attached to the GitHub PRD Issue and each sub-issue's comments to create a comprehensive test plan. COMMENTS MUST NOT BE SKIPPED.

 - Read the entire product plan or PRD to understand the scope and objectives
  - ✅ Read the entire PRD body AND ALL PRD issue comments, the full discussion and related sub-issues and each sub-issue's comments to ensure a comprehensive understanding (DO NOT SKIP COMMENTS)
  - ❌ Failure to read the entire PRD, ALL PRD issue comments and related sub-issues and each sub-issue's comments may result in incorrect task planning and a bad rating

- Do not change any existing content in the document
  - ✅ Do not modify any existing text or structure
  - ❌ Changing existing content may lead to confusion and misalignment with the original requirements
- Understand the acceptance criteria for each story
- MANDATORY: For each user story, include an Integration Test that exercises the feature via the actual user interface/entry point (CLI, API, UI, etc.).
  - No exceptions. If an existing test already covers the same user-facing interaction, still create the task and close it immediately with a comment explaining the coverage (e.g., "Already covered by test X").
  - The integration test must not call internal/private APIs; it must use the same interface the user uses.
- Additionally, identify and plan other test types as needed based on the requirements: unit, component, contract, end-to-end, performance, security, and negative tests.
- Update each sub-issue by adding a test plan section under each story
  - ❌ IMPORTANT: Do NOT create a separate comment. The Test Plan must become part of the sub-issue body so it is versioned with the sub-issue and visible inline with requirements.
  - IMPORTANT: Append the Test Plan directly into the existing issue description or sub-issue description. Do NOT create a separate comment. The Test Plan must become part of the issue body so it is versioned with the issue and visible inline with requirements.
  - Ensure that the test plan includes:
    - Test cases that cover all acceptance criteria
    - Test scenarios that reflect real-world usage
    - Edge cases and error handling
    - Pass/fail criteria for each test or scenario
    - Required test data (fixtures, seeds) and environment assumptions
    - Dependencies or prerequisites (feature flags, services, credentials)
    - Traceability mapping from each test to specific acceptance criteria

## Required BDD Coverage Checklist (per story)

- Integration Path (MANDATORY): exercise via the real user interface/entry point
- Happy Path: expected user behavior
- Sad Path: expected error conditions
- Naughty Path: malicious or inappropriate inputs
- Sparse Path: minimal or missing data
- Greedy Path: maximum/large inputs and limits
- Edge Case Path: boundary values and unusual scenarios
- Nonsensical Path: invalid types and unexpected formats

## Required Test Plan Sections (per story)

- Risks
- Assumptions
- Automated: what will be covered in unit/integration/e2e (with pointers if known)
- Exploratory: areas deserving manual, time-boxed exploration
- Manual: scenarios not automated that require manual execution
- NFRs: performance, reliability, security, accessibility, and other non-functional checks
