---
name: error-handling
description: Improve error handling patterns in code
category: quality
argument-hint: "[file-or-module]"
---
Improve error handling in $0:

1. Read the code and identify error handling gaps:
   - Unhandled promise rejections
   - Swallowed errors (empty catch blocks)
   - Missing null/undefined checks
   - Panics or unchecked type assertions
2. Add proper error propagation and wrapping
3. Ensure error messages are descriptive and actionable
4. Add error recovery where appropriate
5. Follow the project's existing error handling patterns
