---
name: changelog
description: Generate a changelog from recent git commits
category: documentation
context: fork
allowed-tools: "Read, Bash, Grep, Glob"
---
Generate a changelog from recent changes:

1. Read the git log for recent commits (since last tag or last N commits)
2. Categorize changes into: Features, Bug Fixes, Improvements, Breaking Changes
3. Write clear, user-facing descriptions (not raw commit messages)
4. Group related changes together
5. Format as markdown with version header and date
