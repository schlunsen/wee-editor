package packs

// eccPack returns the Everything Claude Code (ECC) pack — a curated selection
// of the best skills and hooks from the ECC project (github.com/affaan-m/everything-claude-code).
//
// ECC is a performance optimization system for AI agent harnesses with 228+ skills,
// 60+ agents, and 75+ commands. This pack distills the highest-value, most universal
// content into a single installable pack for Wee users.
//
// License: MIT — https://github.com/affaan-m/everything-claude-code/blob/main/LICENSE
func eccPack() *Pack {
	return &Pack{
		Name:        "ecc",
		DisplayName: "Everything Claude Code",
		Description: "Curated skills and hooks from the Everything Claude Code project — research-first workflows, agentic engineering patterns, architecture tools, security hooks, and autonomous loop strategies.",
		Version:     "2.0.0",
		Category:    "workflow",
		Icon:        "brain",
		Skills: []PackSkill{
			eccSearchFirst(),
			eccAgenticEngineering(),
			eccBlueprint(),
			eccADR(),
			eccAutonomousLoops(),
			eccBenchmark(),
			eccCodingStandards(),
			eccCodeReview(),
			eccAgentEval(),
			eccCodeTour(),
		},
		Hooks: []PackHook{
			eccConfigProtectionHook(),
			eccSecretScanHook(),
			eccDestructiveCommandHook(),
			eccSensitiveFileHook(),
			eccSessionSummaryHook(),
		},
	}
}

// --- Skills ---

func eccSearchFirst() PackSkill {
	return PackSkill{
		Name:        "search-first",
		Description: "Research-before-coding workflow — search for existing tools, libraries, and patterns before writing custom code",
		Body: `# /search-first — Research Before You Code

Systematizes the "search for existing solutions before implementing" workflow.

## When to Use

- Starting a new feature that likely has existing solutions
- Adding a dependency or integration
- The user asks "add X functionality" and you're about to write code
- Before creating a new utility, helper, or abstraction

## Workflow

1. **TOOL AVAILABILITY PREFLIGHT** — Check search channels before relying on them; report skipped channels honestly.

2. **NEED ANALYSIS** — Define what functionality is needed. Identify language/framework constraints.

3. **PARALLEL SEARCH** — Search across multiple channels simultaneously:
   - Package registries (npm, PyPI, pkg.go.dev, crates.io)
   - Existing codebase (Grep/Glob for similar patterns)
   - GitHub / Web search for established solutions

4. **EVALUATE** — Score candidates on:
   - Functionality match (does it solve the actual problem?)
   - Maintenance status (recent commits, active maintainers)
   - Community adoption (stars, downloads, dependents)
   - Documentation quality
   - License compatibility
   - Dependency footprint

5. **DECIDE** — Choose one of three paths:
   - **Adopt as-is**: Install the package and configure it
   - **Extend/Wrap**: Use it as a base with a thin wrapper
   - **Build Custom**: Only when no suitable solution exists — document WHY

6. **IMPLEMENT** — Install package / Configure integration / Write minimal custom code.

## Key Principle

> Always search before you build. The best code is code you don't write.

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[what to search for]",
		Effort:       "medium",
	}
}

func eccAgenticEngineering() PackSkill {
	return PackSkill{
		Name:        "agentic-engineering",
		Description: "Operate as an agentic engineer — eval-first execution, task decomposition, and cost-aware model routing",
		Body: `# Agentic Engineering

Use this skill for engineering workflows where AI agents perform most implementation work and humans enforce quality and risk controls.

## Operating Principles

1. Define completion criteria before execution.
2. Decompose work into agent-sized units.
3. Route model tiers by task complexity.
4. Measure with evals and regression checks.

## Eval-First Loop

1. Define capability eval and regression eval.
2. Run baseline and capture failure signatures.
3. Execute implementation.
4. Re-run evals and compare deltas.

## Task Decomposition

Apply the 15-minute unit rule:
- Each unit should be independently verifiable
- Each unit should have a single dominant risk
- Each unit should expose a clear done condition

## Model Routing

- **Haiku**: classification, boilerplate transforms, narrow edits
- **Sonnet**: implementation and refactors
- **Opus**: architecture, root-cause analysis, multi-file invariants

## Session Strategy

- Continue session for closely-coupled units.
- Start fresh session after major phase transitions.
- Compact after milestone completion, not during active debugging.

## Review Focus for AI-Generated Code

Prioritize:
- Invariants and edge cases
- Error boundaries
- Security and auth assumptions
- Hidden coupling and rollout risk

## Anti-Patterns to Avoid

- Running without evals (flying blind)
- Using the strongest model for every task (cost waste)
- Compacting mid-debug (losing critical context)
- Skipping regression checks after changes

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		Effort: "medium",
	}
}

func eccBlueprint() PackSkill {
	return PackSkill{
		Name:        "blueprint",
		Description: "Turn a one-line objective into a step-by-step construction plan for multi-session, multi-agent projects",
		Body: `# Blueprint — Construction Plan Generator

Turn a one-line objective into a step-by-step construction plan that any coding agent can execute cold.

## When to Use

- Breaking a large feature into multiple PRs with clear dependency order
- Planning a refactor or migration that spans multiple sessions
- Coordinating parallel workstreams across sub-agents
- Any task where context loss between sessions would cause rework

**Do not use** for tasks completable in a single PR or fewer than 3 tool calls.

## Pipeline

### Phase 1: Research
- Pre-flight checks (git status, branch, remote)
- Read project structure, existing plans, memory files

### Phase 2: Design
- Break the objective into one-PR-sized steps (3-12 typical)
- Assign dependency edges and parallel/serial ordering
- Assign model tier (strongest for architecture, default for implementation)
- Define rollback strategy per step

### Phase 3: Draft
Write a self-contained Markdown plan. Every step includes:
- **Context brief** — so a fresh agent can execute without reading prior steps
- **Task list** — concrete actions to take
- **Verification commands** — how to know it worked
- **Exit criteria** — definition of done

### Phase 4: Review
- Adversarial review against anti-pattern checklist:
  - Missing dependency edges
  - Steps that modify the same files (merge conflict risk)
  - Steps without verification commands
  - Overly large steps (should be split)
  - Missing rollback strategy for risky steps

### Phase 5: Register
- Save plan to project docs
- Present step count, parallelism summary, and estimated effort

## Key Features

- **Cold-start execution** — Every step includes full context. No prior knowledge needed.
- **Parallel step detection** — Dependency graph identifies independent steps.
- **Plan mutation** — Steps can be split, inserted, skipped, or reordered with audit trail.

## Output Format

` + "```" + `markdown
# Plan: [objective]

## Step 1: [title]
**Depends on**: none
**Branch**: feature/step-1-[slug]
**Model tier**: default

### Context
[What a fresh agent needs to know to execute this step]

### Tasks
- [ ] Task 1
- [ ] Task 2

### Verification
` + "```" + `bash
[commands to verify this step]
` + "```" + `

### Exit Criteria
- [concrete measurable condition]
` + "```" + `

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code), inspired by antbotlab/blueprint`,
		ArgumentHint: "[project-name] [objective description]",
		Agent:        "Plan",
		Effort:       "high",
	}
}

func eccADR() PackSkill {
	return PackSkill{
		Name:        "adr",
		Description: "Capture architectural decisions as structured Architecture Decision Records (ADRs)",
		Body: `# Architecture Decision Records

Capture architectural decisions as they happen during coding sessions. Instead of decisions living only in chat history or someone's memory, this skill produces structured ADR documents that live alongside the code.

## When to Activate

- User explicitly says "let's record this decision" or "ADR this"
- User chooses between significant alternatives (framework, library, pattern, database, API design)
- User says "we decided to..." or "the reason we're doing X instead of Y is..."
- User asks "why did we choose X?" (read existing ADRs)

## ADR Format

Use the lightweight Nygard format:

` + "```" + `markdown
# ADR-NNNN: [Decision Title]

**Date**: YYYY-MM-DD
**Status**: proposed | accepted | deprecated | superseded by ADR-NNNN
**Deciders**: [who was involved]

## Context
What is the issue that we're seeing that is motivating this decision or change?

## Decision
What is the change that we're proposing and/or doing?

## Alternatives Considered

### Alternative 1: [Name]
- **Pros**: [benefits]
- **Cons**: [drawbacks]
- **Why not**: [specific reason this was rejected]

## Consequences

### Positive
- [benefit 1]

### Negative
- [trade-off 1]

### Risks
- [risk and mitigation]
` + "```" + `

## Workflow

### Capturing a New ADR
1. **Initialize** (first time) — Create docs/adr/ directory with README.md index and template.md
2. **Identify** the core architectural choice being made
3. **Gather context** — what problem prompted this? What constraints exist?
4. **Document alternatives** — what was considered? Why rejected?
5. **State consequences** — what are the trade-offs?
6. **Assign a number** — scan existing ADRs and increment
7. **Write** to docs/adr/NNNN-decision-title.md
8. **Update the index** — append to docs/adr/README.md

### Reading Existing ADRs
When asked "why did we choose X?":
1. Check if docs/adr/ exists
2. Scan the README.md index for relevant entries
3. Read matching ADR files and present Context and Decision sections

## Categories Worth Recording

| Category | Examples |
|----------|---------|
| Technology choices | Framework, language, database, cloud provider |
| Architecture patterns | Monolith vs microservices, event-driven, CQRS |
| API design | REST vs GraphQL, versioning, auth mechanism |
| Data modeling | Schema design, normalization, caching strategy |
| Infrastructure | Deployment model, CI/CD, monitoring stack |
| Security | Auth strategy, encryption, secret management |

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[decision to record or question to look up]",
		Effort:       "medium",
	}
}

func eccAutonomousLoops() PackSkill {
	return PackSkill{
		Name:        "autonomous-loops",
		Description: "Patterns and architectures for autonomous agent loops — from sequential pipelines to multi-agent DAG orchestration",
		Body: `# Autonomous Loops

Patterns, architectures, and reference implementations for running coding agents autonomously in loops.

## Loop Pattern Spectrum

From simplest to most sophisticated:

| Pattern | Complexity | Best For |
|---------|-----------|----------|
| Sequential Pipeline | Low | Daily dev steps, scripted workflows |
| Infinite Agentic Loop | Medium | Parallel content generation, spec-driven work |
| Continuous PR Loop | Medium | Multi-day iterative projects with CI gates |
| De-Sloppify Pattern | Add-on | Quality cleanup after any implementation step |
| RFC-Driven DAG | High | Large features, multi-unit parallel work with merge queue |

## 1. Sequential Pipeline

Break development into a sequence of non-interactive calls. Each call is a focused step:

` + "```" + `bash
#!/bin/bash
set -e
# Step 1: Implement
claude -p "Read the spec in docs/spec.md. Implement the feature. Write tests first (TDD)."
# Step 2: Cleanup
claude -p "Review all changes. Remove unnecessary checks and testing of language features."
# Step 3: Verify
claude -p "Run full build, lint, type check, and test suite. Fix any failures."
# Step 4: Commit
claude -p "Create a conventional commit for all staged changes."
` + "```" + `

### Key Principles
- Each step is isolated (fresh context window)
- Order matters (sequential filesystem state)
- Use separate cleanup steps instead of negative instructions

### Model Routing Variation
` + "```" + `bash
claude -p --model opus "Analyze architecture and write a plan..."    # Deep reasoning
claude -p "Implement according to the plan..."                       # Fast implementation
claude -p --model opus "Review all changes for security issues..."   # Thorough review
` + "```" + `

## 2. De-Sloppify Pattern

Don't constrain the implementer with "don't do X." Instead, add a separate cleanup pass:

` + "```" + `bash
# Let the implementer be thorough
claude -p "Implement the feature with full TDD. Be thorough with tests."

# Then clean up in a separate context
claude -p "Review all changes. Remove:
- Tests that verify language/framework behavior rather than business logic
- Redundant type checks the type system already enforces
- Over-defensive error handling for impossible states
- Console.log statements and commented-out code
Run the test suite after cleanup."
` + "```" + `

> Two focused agents outperform one constrained agent.

## 3. Continuous PR Loop

A shell script that runs in a continuous loop: create branch, implement, create PR, wait for CI, auto-fix failures, merge, repeat.

Key features:
- **Exit conditions**: --max-runs, --max-cost, --max-duration, or completion signal
- **Context bridge**: SHARED_TASK_NOTES.md persists across iterations
- **CI failure recovery**: Auto-fetches failed run logs and spawns fix agent
- **Parallel via worktrees**: Multiple loops in separate git worktrees

## 4. Choosing the Right Pattern

` + "```" + `
Single focused change?
├─ Yes → Sequential Pipeline
└─ No → Written spec/RFC exists?
         ├─ Yes → Need parallel implementation?
         │        ├─ Yes → RFC-Driven DAG orchestration
         │        └─ No → Continuous PR Loop
         └─ No → Need many variations?
                  ├─ Yes → Infinite Agentic Loop
                  └─ No → Sequential Pipeline + De-Sloppify
` + "```" + `

## Anti-Patterns

1. **Infinite loops without exit conditions** — Always have max-runs, max-cost, or max-duration
2. **No context bridge** — Use SHARED_TASK_NOTES.md or filesystem state between iterations
3. **Retrying same failure** — Capture error context and feed to next attempt
4. **Negative instructions instead of cleanup passes** — Don't say "don't do X"; add a pass that removes X
5. **All agents in one context window** — Separate concerns into different agent processes
6. **Ignoring file overlap in parallel work** — Need a merge strategy when agents touch same files

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)
Patterns by: @disler (Infinite Loop), @AnandChowdhary (Continuous Claude), @enitrat (Ralphinho DAG)`,
		Effort: "medium",
	}
}

func eccBenchmark() PackSkill {
	return PackSkill{
		Name:        "benchmark",
		Description: "Measure performance baselines, detect regressions before/after changes, and compare alternatives",
		Body: `# Benchmark — Performance Baseline & Regression Detection

## When to Use

- Before and after a PR to measure performance impact
- Setting up performance baselines for a project
- When users report "it feels slow"
- Before a launch to ensure performance targets are met
- Comparing stack alternatives

## Modes

### Mode 1: Page Performance
Measure Core Web Vitals and resource metrics:
- **LCP** (Largest Contentful Paint) — target < 2.5s
- **CLS** (Cumulative Layout Shift) — target < 0.1
- **INP** (Interaction to Next Paint) — target < 200ms
- **FCP** (First Contentful Paint) — target < 1.8s
- **TTFB** (Time to First Byte) — target < 800ms
- Total page weight (target < 1MB), JS bundle size (target < 200KB gzipped)

### Mode 2: API Performance
- Hit each endpoint 100 times
- Measure p50, p95, p99 latency
- Test under load: 10 concurrent requests
- Compare against SLA targets

### Mode 3: Build Performance
- Cold build time, hot reload time (HMR)
- Test suite duration, TypeScript check time
- Lint time, Docker build time

### Mode 4: Before/After Comparison
` + "```" + `bash
/benchmark baseline    # saves current metrics
# ... make changes ...
/benchmark compare     # compares against baseline
` + "```" + `

Output format:
| Metric | Before | After | Delta | Verdict |
|--------|--------|-------|-------|---------|
| LCP | 1.2s | 1.4s | +200ms | WARNING |
| Bundle | 180KB | 175KB | -5KB | BETTER |

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[baseline|compare|page|api|build]",
		Effort:       "medium",
	}
}

func eccCodingStandards() PackSkill {
	return PackSkill{
		Name:        "coding-standards",
		Description: "Cross-project coding conventions — naming, readability, immutability, KISS, DRY, and YAGNI enforcement",
		Body: `# Coding Standards & Best Practices

Baseline coding conventions applicable across projects and languages.

## When to Activate

- Starting a new project or module
- Reviewing code for quality and maintainability
- Refactoring existing code to follow conventions
- Onboarding new contributors to coding conventions

## Core Principles

### 1. Readability First
- Code is read more than written
- Clear variable and function names — descriptive over concise
- Self-documenting code preferred over comments
- Consistent formatting throughout

### 2. KISS (Keep It Simple)
- Simplest solution that works
- Avoid over-engineering and premature optimization
- Easy to understand > clever code

### 3. DRY (Don't Repeat Yourself)
- Extract common logic into functions
- Create reusable components
- Share utilities across modules

### 4. YAGNI (You Aren't Gonna Need It)
- Don't build features before they're needed
- Avoid speculative generality
- Start simple, refactor when needed

## Naming Conventions

- **Variables**: descriptive, context-clear (marketSearchQuery not q, isUserAuthenticated not flag)
- **Functions**: verb-first, describes action (getUserById, calculateTotalRevenue)
- **Booleans**: is/has/can/should prefix (isLoading, hasPermission)
- **Constants**: UPPER_SNAKE_CASE for true constants
- **Files**: match the primary export or concept

## Error Handling

- Handle errors at the appropriate level — not too early, not too late
- Use typed errors with context (wrap with additional info)
- Never swallow errors silently
- Fail fast on unrecoverable errors
- Log errors with structured context (request ID, user, operation)

## Code Smells to Watch For

- Functions longer than 30 lines
- More than 3 parameters (use an options object/struct)
- Deeply nested conditionals (use early returns / guard clauses)
- Comments explaining "what" instead of "why"
- Copy-pasted code blocks
- God objects / classes doing too many things
- Magic numbers without named constants

## Immutability Defaults

- Prefer const/let over var, final over var, val over var
- Return new collections instead of mutating in place
- Use immutable data structures where the language supports them
- Mutation is acceptable for performance-critical paths — document why

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		Effort: "low",
	}
}

func eccCodeReview() PackSkill {
	return PackSkill{
		Name:        "ecc-code-review",
		Description: "Comprehensive multi-dimensional code review covering correctness, security, performance, and style",
		Body: `# Code Review — Comprehensive PR Review

Perform a thorough, multi-dimensional code review.

## Review Dimensions

Check each of the following categories:

### 1. Correctness
- Logic errors and edge cases
- Off-by-one errors
- Null/nil/undefined handling
- Race conditions and concurrency issues
- Error propagation paths

### 2. Security
- SQL injection, XSS, CSRF vulnerabilities
- Hardcoded secrets or API keys
- Authentication/authorization bypasses
- Input validation gaps
- Dependency vulnerabilities

### 3. Performance
- N+1 queries
- Unnecessary allocations or copies
- Missing database indices
- Unbounded collections
- Missing pagination

### 4. Code Style
- Naming conventions consistency
- Code organization and structure
- DRY violations (duplicated logic)
- Dead code and unused imports
- Function/method length

### 5. Testing
- Are new features tested?
- Are edge cases covered?
- Are error paths tested?
- Test quality (not just coverage numbers)

### 6. Documentation
- Are public APIs documented?
- Are complex algorithms explained?
- Are breaking changes noted?

### 7. Error Handling
- Are errors properly caught, logged, and surfaced?
- Are error messages helpful for debugging?
- Is there appropriate retry/fallback logic?

## Output Format

For each issue found, provide:
- **Severity**: critical / warning / suggestion / nit
- **File and line**: exact location
- **Issue**: what the problem is
- **Fix**: suggested fix with code

## Summary

End with an overall assessment:
- **Verdict**: approve / request changes / comment
- **Critical issues** that must be fixed
- **Suggestions** for improvement
- **Positive callouts** for good patterns

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[PR number or file path]",
		Context:      "fork",
		Effort:       "high",
	}
}

func eccAgentEval() PackSkill {
	return PackSkill{
		Name:        "agent-eval",
		Description: "Compare coding agents head-to-head on custom tasks with pass rate, cost, time, and consistency metrics",
		Body: `# Agent Eval — Head-to-Head Agent Comparison

A framework for comparing coding agents on reproducible tasks. Every "which agent is best?" comparison should be data-driven, not vibes.

## When to Use

- Comparing coding agents on your own codebase
- Measuring agent performance before adopting a new tool or model
- Running regression checks when an agent updates
- Producing data-backed agent selection decisions

## Core Concepts

### YAML Task Definitions

Define tasks declaratively:

` + "```" + `yaml
name: add-retry-logic
description: Add exponential backoff retry to the HTTP client
repo: ./my-project
files:
  - src/http_client.py
prompt: |
  Add retry logic with exponential backoff to all HTTP requests.
  Max 3 retries. Initial delay 1s, max delay 30s.
judge:
  - type: test
    command: pytest tests/test_http_client.py -v
  - type: grep
    pattern: "exponential_backoff|retry"
    files: src/http_client.py
` + "```" + `

### Metrics Collected

| Metric | What It Measures |
|--------|-----------------|
| Pass rate | Did the agent produce code that passes the judge? |
| Cost | API spend per task (when available) |
| Time | Wall-clock seconds to completion |
| Consistency | Pass rate across repeated runs (e.g., 3/3 = 100%) |

### Workflow

1. **Define Tasks** — Create YAML task definitions with acceptance criteria
2. **Run Agents** — Execute each agent against tasks in isolated git worktrees
3. **Judge Results** — Automated judging via tests, grep, or custom scripts
4. **Compare** — Side-by-side metrics table
5. **Report** — Markdown report with recommendations

### Judging

Support multiple judge types:
- **test**: Run test suite, pass if exit code 0
- **grep**: Check for required patterns in output files
- **build**: Verify the project builds successfully
- **custom**: Run any script that exits 0 for pass, non-0 for fail

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[task-file or task-dir]",
		AllowedTools: "Read,Write,Edit,Bash,Grep,Glob",
		Effort:       "high",
	}
}

func eccCodeTour() PackSkill {
	return PackSkill{
		Name:        "code-tour",
		Description: "Generate an interactive guided tour of a codebase for onboarding new developers",
		Body: `# Code Tour — Codebase Walkthrough Generator

Generate a guided tour of a codebase to help new developers understand the architecture, key patterns, and important files.

## When to Use

- Onboarding new team members
- Documenting a codebase you're handing off
- Creating "start here" guides for open source projects
- When someone asks "how does this codebase work?"

## Tour Generation Process

### 1. Discovery
- Read the project root: README, config files, directory structure
- Identify the tech stack and framework
- Find the entry point(s) (main, index, app)
- Map the module/package structure

### 2. Architecture Overview
- Draw the high-level architecture (components, data flow)
- Identify the key patterns used (MVC, hexagonal, event-driven, etc.)
- Map dependencies between modules
- Note the database/storage layer

### 3. Key Paths
Trace the most important code paths:
- **Request lifecycle**: How does a request flow through the system?
- **Data flow**: How does data get created, read, updated, deleted?
- **Auth flow**: How does authentication/authorization work?
- **Error handling**: How are errors propagated and reported?

### 4. Important Files
Highlight the files every developer should read first:
- Configuration and environment setup
- Core domain models/types
- Main business logic
- Test examples and patterns
- Build and deployment scripts

### 5. Conventions & Patterns
Document the project-specific conventions:
- File naming and organization
- Testing patterns and fixtures
- Error handling conventions
- Logging and observability patterns

## Output

Generate a structured markdown document with:
- Table of contents
- Architecture diagram (text-based)
- Numbered tour stops with file paths and explanations
- "Next steps" for going deeper

Credit: Everything Claude Code (github.com/affaan-m/everything-claude-code)`,
		ArgumentHint: "[directory or repo path]",
		Effort: "high",
	}
}

// --- Hooks ---

func eccConfigProtectionHook() PackHook {
	return PackHook{
		EventName:   "PreToolUse",
		Matcher:     "Write|Edit",
		Type:        "command",
		Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); if printf "%s" "$FILE" | grep -qEi "(\.eslintrc|\.prettierrc|\.flake8|ruff\.toml|\.golangci|biome\.json|tsconfig\.json|\.editorconfig|rustfmt\.toml|clippy\.toml)(\..*)?$"; then echo "{\"decision\":\"block\",\"reason\":\"ECC: Blocked modification to linter/formatter config. Fix the code instead of weakening the config.\"}"; else echo "{\"decision\":\"allow\"}"; fi'`,
		Timeout:     5,
		Description: "Block modifications to linter/formatter config files — steers agents to fix code instead of weakening configs",
	}
}

func eccSecretScanHook() PackHook {
	return PackHook{
		EventName:   "PostToolUse",
		Matcher:     "Write|Edit",
		Type:        "command",
		Command:     `bash -c 'INPUT=$(cat); CONTENT=$(printf "%s" "$INPUT" | jq -r ".tool_result.content // empty"); if printf "%s" "$CONTENT" | grep -qEi "(sk|pk|api|token|key|secret|password|bearer)[-_]?[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{36}|xox[bpas]-[A-Za-z0-9-]+|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}"; then echo "ECC WARNING: Possible secret or API key detected in written content. Please review and remove before committing." >&2; fi'`,
		Timeout:     5,
		Description: "Scan written files for accidentally leaked secrets, API keys, and JWT tokens",
	}
}

func eccDestructiveCommandHook() PackHook {
	return PackHook{
		EventName:   "PreToolUse",
		Matcher:     "Bash",
		Type:        "command",
		Command:     `bash -c 'INPUT=$(cat); CMD=$(printf "%s" "$INPUT" | jq -r ".tool_input.command // empty"); if printf "%s" "$CMD" | grep -qEi "rm\s+-rf\s+/[^.]|rm\s+-rf\s+~|git\s+push\s+--force\s+(origin\s+)?(main|master)|DROP\s+TABLE|DROP\s+DATABASE|truncate\s+table|:(){ :|:& };:|mkfs\.|dd\s+if=.+of=/dev/|kubectl\s+delete\s+namespace|terraform\s+destroy\s+--auto-approve|docker\s+system\s+prune\s+-a\s+--force"; then echo "{\"decision\":\"block\",\"reason\":\"ECC: Blocked dangerous command — potential destructive operation detected\"}"; else echo "{\"decision\":\"allow\"}"; fi'`,
		Timeout:     5,
		Description: "Block dangerous shell commands (rm -rf /, force push to main, DROP TABLE, terraform destroy, etc.)",
	}
}

func eccSensitiveFileHook() PackHook {
	return PackHook{
		EventName:   "PreToolUse",
		Matcher:     "Write|Edit",
		Type:        "command",
		Command:     `bash -c 'INPUT=$(cat); FILE=$(printf "%s" "$INPUT" | jq -r ".tool_input.file_path // .tool_input.path // empty"); if printf "%s" "$FILE" | grep -qEi "\\.env$|\\.env\\.|credentials|secret|private.key$|id_rsa$|id_ed25519$|\\.pem$|\\.p12$|\\.pfx$|token\\.json$|service.account\\.json$"; then printf "{\"decision\":\"block\",\"reason\":\"ECC: Blocked modification to sensitive file (%s). These files may contain secrets.\"}\n" "$FILE"; else echo "{\"decision\":\"allow\"}"; fi'`,
		Timeout:     5,
		Description: "Protect sensitive files (.env, credentials, private keys, service accounts) from modification",
	}
}

func eccSessionSummaryHook() PackHook {
	return PackHook{
		EventName:   "Stop",
		Matcher:     "",
		Type:        "prompt",
		Prompt:      "Summarize all changes made in this session. List: (1) files modified/created/deleted, (2) features added or changed, (3) bugs fixed, (4) any remaining TODOs or known issues. Keep it concise.",
		Description: "Generate a summary of all changes made in the session at completion",
	}
}
