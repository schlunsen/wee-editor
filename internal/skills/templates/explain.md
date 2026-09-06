---
name: explain
description: Explain how a piece of code or system works
category: research
context: fork
agent: Explore
allowed-tools: "Read, Grep, Glob"
argument-hint: "[topic-or-file]"
---
Explain how $ARGUMENTS works:

1. Find the relevant source files and entry points
2. Trace the code flow from start to finish
3. Explain the architecture and key design decisions
4. Document important data structures and their relationships
5. Note any non-obvious behavior or gotchas
6. Create a clear summary suitable for onboarding a new developer
