---
mode: task-planner
description: "Creates tasks for user stories."
---

# Input
- ${input:featureId:placeholder}: This github feature issue.

# Break down user stories into tasks
- For each user story in the PRD, read and understand the story description - ALWAYS read the entire GitHub issue AND ALL issue comments to ensure a comprehensive understanding (COMMENTS MUST NOT BE SKIPPED)
  - ✅ Read the entire issue body AND ALL issue comments to ensure a comprehensive understanding
  - ❌ Failure to read the entire issue or any of the comments may result in incorrect task planning and a bad rating
- If the GitHub issue includes any relevant links or references, be sure to follow them for additional context. ALWAYS inspect comments on the issue and discussion for clarifications, decisions, or hidden requirements.
- If the Github feature issue id `${input:featureId:placeholder}` has not been provided, prompt the user to provide it.

# Output
- Update each user story with a comprehensive set of tasks required to complete the story, ensuring all acceptance criteria are covered.
  - Each task should be clear, actionable, and focused on a specific aspect of the user story.
  - Ensure that tasks are organized logically and sequenced appropriately to facilitate efficient development.

IMPORTANT: Do not create, edit, or commit Markdown (.md) files in the repository. Instead, produce the PRD and user stories as Markdown-formatted text to be used in GitHub Issue bodies, and create or update GitHub Issues/Discussions as described above. Agents must not write files into the codebase; use the GitHub API or issue creation tools to publish content.
