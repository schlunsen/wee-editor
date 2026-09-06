---
name: commit-review
description: Review staged changes before committing
category: git
context: fork
allowed-tools: "Read, Bash, Grep"
---
Review the current staged changes:

1. Run `git diff --staged` to see all changes
2. Check for accidental file inclusions (secrets, logs, build artifacts)
3. Verify code quality: no debug statements, no commented-out code
4. Ensure changes are cohesive (single logical change per commit)
5. Suggest a clear, conventional commit message
6. Flag anything that should be split into separate commits
