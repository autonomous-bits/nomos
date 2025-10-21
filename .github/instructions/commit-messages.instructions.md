---
description: 'Conventional commit message format with gitmoji'
---

# Commit Message Guidelines

Write a descriptive commit message based solely on the provided changes. Follow these strict rules:

## Conventional Commits
- This repository follows Conventional Commits 1.0.0. See: https://www.conventionalcommits.org/en/v1.0.0/
- Subject format: `<type>[optional scope][!]: <description>`
- Place the gitmoji immediately after the colon and space, before the description, e.g. `feat(api): ✨ add X`
- Breaking changes: use `!` after type/scope (e.g., `feat(api)!: ...`) and/or add a `BREAKING CHANGE:` footer

- Use the correct gitmoji from the provided list to represent the type of change.
- Separate subject from body with a blank line.
- Use the body to explain what and why you have done something. In most cases, you can leave out details about how a change has been made.
- In the body, use bullet point to describe everything.
- Avoid using vague language like "fix" or "update". Instead, be specific about what was changed and why.
- If multiple changes are made, pick the most significant change and describe it in detail.
- Message length is maximum 250 characters.

gitmojis:
- ✨ feat: A new feature
- 🐛 fix: A bug fix
- 📝 docs: Documentation only changes
- 🎨 style: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- ⚡️ perf: A code change that improves performance
- 🔧 chore: Changes to the build process or auxiliary tools and libraries such as documentation generation
- 🚀 deploy: Changes to the deployment process
- 🐳 docker: Changes related to Docker
- 🧪 test: Adding or updating tests
- 🚧 wip: Work in progress
- 🛠️ build: Changes that affect the build system or external dependencies
- ♻️ refactor: A code change that neither fixes a bug nor adds a feature
- 🔒 security: Fix security vulnerabilities
