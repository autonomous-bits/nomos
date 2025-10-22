---
mode: story-planner
description: "Creates story issues from a feature description."
---

# Input
- ${input:featureId:placeholder}: This github feature issue.

# Create user stories
- Read and understand the feature description - ALWAYS read the entire GitHub issue AND ALL issue comments to ensure a comprehensive understanding (COMMENTS MUST NOT BE SKIPPED)
  - ✅ Read the entire issue body AND ALL issue comments to ensure a comprehensive understanding
  - ❌ Failure to read the entire issue or any of the comments may result in incorrect task planning and a bad rating
- If you have not also been given the original product description or requirements, in addition to the feature description, ask for it
  - Hint to the user that they can attach a file to the context in order to provide
- If the GitHub issue includes any relevant links or references, be sure to follow them for additional context. ALWAYS inspect comments on the issue and discussion for clarifications, decisions, or hidden requirements.
- If the Github feature issue id `${input:featureId:placeholder}` has not been provided, prompt the user to provide it.

# Output
- Create a GitHub Issues for each story in required to complete the feature and tag them as `user-story`.
  - Each user story title should start with the PRD id and a unique number (e.g. "FEATURE-1") followed by the user story title.
  - Make sure to add a link back to the PRD issue in the user story description (i.e. "See PRD #<PRD_ID> for details.")

IMPORTANT: Do not create, edit, or commit Markdown (.md) files in the repository. Instead, produce the PRD and user stories as Markdown-formatted text to be used in GitHub Issue bodies, and create or update GitHub Issues/Discussions as described above. Agents must not write files into the codebase; use the GitHub API or issue creation tools to publish content.
