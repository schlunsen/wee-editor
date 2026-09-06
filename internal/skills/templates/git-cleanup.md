---
name: git-cleanup
description: Clean up git history and branches
category: git
disable-model-invocation: true
allowed-tools: "Bash, Read"
---
Clean up the git repository:

1. List merged branches that can be safely deleted
2. Show stale remote-tracking branches
3. Identify large files in git history if applicable
4. Suggest cleanup commands (do NOT execute destructive commands without confirmation)
5. Report the current state: branch count, remote status, working tree cleanliness
