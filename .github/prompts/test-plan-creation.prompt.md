---
mode: qa-story-reviewer
description: "QA review of implementation code against the test plan and best practices."
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
- Identify the types of tests needed (unit, integration, end-to-end, etc.) based on the requirements
- Update each sub-issue by adding a test plan section under each story
 - Update each sub-issue by adding a test plan section under each story
  - ❌ IMPORTANT: Do NOT create a separate comment. The Test Plan must become part of the sub-issue body so it is versioned with the sub-issue and visible inline with requirements.
  - IMPORTANT: Append the Test Plan directly into the existing issue description or sub-issue description. Do NOT create a separate comment. The Test Plan must become part of the issue body so it is versioned with the issue and visible inline with requirements.
  - Ensure that the test plan includes:
    - Test cases that cover all acceptance criteria
    - Test scenarios that reflect real-world usage
    - Edge cases and error handling
