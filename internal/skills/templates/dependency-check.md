---
name: dependency-check
description: Audit project dependencies for updates and vulnerabilities
category: security
context: fork
allowed-tools: "Read, Bash, Grep, Glob"
---
Audit project dependencies:

1. Identify the package manager(s) in use (npm, pip, go mod, cargo, etc.)
2. List outdated dependencies and available updates
3. Check for known security vulnerabilities
4. Identify unused or duplicate dependencies
5. Suggest a prioritized update plan (security fixes first, then majors)
6. Note any breaking changes in major version updates
