---
mode: architecture-prd-reviewer
description: "Architect pre-implementation review and requirements"
---

# Input
- ${input:discussionId:placeholder}: This github discussion.
- ${input:issueId:placeholder}: This github PRD issue.

# Architecture Pre-Implementation Review

- Review the provided PRD GitHub issue, the GitHub discussion, and any related sub-issues on the PRD.
    - ✅ Read the entire PRD GitHub issue, including all comments, sub-issues, and discussions, to ensure a comprehensive understanding
    - ❌ Failure to read the entire PRD GitHub issue, including all comments, sub-issues, and discussions, may result in incorrect task planning and a bad rating

- If you have not also been given the original product description or requirements, in addition to the plan, ask for it

  - Hint to the user that they provide the PRD Github Issue Id and GitHub discussion Id.
  - If you do ask for the original product description or requirements, read it thoroughly to understand the context and
    objectives

- Add your review to the "Architecture Review" section of the PRD GitHub issue
    - ❌ Do not remove any Acceptance Criteria or Test Plan items

- The "Architecture Review" section should be appended to the end of PRD GitHub issue.
    - ❌ Do not add the review in a separate comment.


**Note:** Do not create additional tasks only review the provided plan and add your comments to the "Architecture Review" section.
