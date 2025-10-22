---
mode: feature-planner
description: "Creates a PRD for a feature."
---

# Input
- ${input:discussionId:placeholder}: This github discussion.

# Create a PRD
- Read and understand the feature description - ALWAYS read the entire GitHub discussion AND ALL discussion comments to ensure a comprehensive understanding (COMMENTS MUST NOT BE SKIPPED)
  - ✅ Read the entire discussion thread AND ALL discussion comments to ensure a comprehensive understanding
  - ❌ Failure to read the entire issue, discussion, or any of their comments may result in incorrect task planning and a bad rating
- If you have not also been given the original product description or requirements, in addition to the feature description, ask for it
  - Hint to the user that they can attach a file to the context in order to provide
- Ask clarifying questions to ensure you understand the feature requirements
- If the GitHub discussion or issue includes any relevant links or references, be sure to follow them for additional context. ALWAYS inspect comments on the issue and discussion for clarifications, decisions, or hidden requirements.
- If the Github discussion id `${input:discussionId:placeholder}` has not been provided, prompt the user to provide it.

# Output
- Create a GitHub Issue for the PRD labeled `feature` containing the full PRD as the issue description.
  - The PRD should have a unique identifier (ID) for tracking purposes in the GitHub issue title.
- For each Agile user story, create a sub-issue linked to the PRD issue and label each sub-issue `story`.
  - Each user story title should start with the PRD id and a unique number (e.g. "FEATURE-1") followed by the user story title.
  - Make sure to add a link back to the PRD issue in the user story description (i.e. "See PRD #<PRD_ID> for details.")

IMPORTANT: Do not create, edit, or commit Markdown (.md) files in the repository. Instead, produce the PRD and user stories as Markdown-formatted text to be used in GitHub Issue bodies, and create or update GitHub Issues/Discussions as described above. Agents must not write files into the codebase; use the GitHub API or issue creation tools to publish content.
