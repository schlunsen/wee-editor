---
name: migration
description: Migrate code between frameworks or patterns
category: refactoring
argument-hint: "[component] [from] [to]"
---
Migrate $0:

1. Read the existing implementation
2. Understand all current behavior and edge cases
3. Create the new implementation preserving all behavior
4. Update imports and dependencies
5. Run existing tests to verify nothing is broken
6. Update or create tests for the new implementation
