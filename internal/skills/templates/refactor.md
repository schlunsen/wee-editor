---
name: refactor
description: Refactor code for better readability and maintainability
category: refactoring
argument-hint: "[file-or-function]"
---
Refactor $0:

1. Read and understand the current implementation
2. Identify code smells:
   - Duplicated logic
   - Overly long functions
   - Deep nesting
   - Poor naming
3. Apply targeted improvements while preserving behavior
4. Extract reusable helpers where appropriate
5. Run existing tests to verify nothing is broken
6. Summarize what changed and why
