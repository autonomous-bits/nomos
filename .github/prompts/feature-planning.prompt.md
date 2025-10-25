---
mode: feature-planner
description: "Creates a PRD for a feature."
---

# Inputs
- ${input:discussionId:placeholder}

# Task
- Produce a Product Requirements Document (PRD) for the feature based on the provided GitHub discussion and its comments.

# Constraints
- PRD only (no user stories or plan breakdown).
- Do not create, edit, or commit Markdown files in the repository.
- If ${input:discussionId:placeholder} is missing, ask the user to provide it.

# Publishing
- After the user confirms the requirements, create a GitHub Issue labeled `feature` with the PRD as the issue body.
- Include a unique identifier (ID) in the issue title for tracking.

IMPORTANT: Produce the PRD as Markdown-formatted text for a GitHub Issue body; do not write Markdown files to the repo. Use GitHub Issues/Discussions or the GitHub API to publish content.
