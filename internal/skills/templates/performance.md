---
name: performance
description: Analyze code for performance bottlenecks
category: debugging
context: fork
agent: Explore
argument-hint: "[file-or-area]"
---
Analyze performance of $ARGUMENTS:

1. Read the code and identify hot paths
2. Look for common performance issues:
   - N+1 queries or redundant database calls
   - Unnecessary re-renders or recomputations
   - Missing caching opportunities
   - Large memory allocations in loops
   - Synchronous blocking in async contexts
3. Check algorithmic complexity of key operations
4. Suggest concrete optimizations with expected impact
5. Note any trade-offs (readability vs speed)
