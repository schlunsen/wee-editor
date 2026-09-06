# ADR 012: AI-Powered Session Handoff Mechanism

## Status

Accepted

## Date

2024-01-01

## Context

AI agent sessions accumulate significant context over time: files read, decisions made, approaches tried and abandoned, and domain understanding built up. When a session needs to be continued later (due to context window limits, task switching, or team collaboration), starting fresh loses all this accumulated knowledge.

Manual context transfer (copy-pasting summaries) is error-prone and time-consuming. We needed an automated mechanism to preserve and transfer session context.

## Decision

We implemented an **AI-powered session handoff system** that uses the agent itself to create structured context summaries for session continuation.

**Handoff flow:**

1. User initiates handoff from an active session
2. The current agent generates a structured context summary including:
   - Task description and current status
   - Key decisions made and their rationale
   - Files modified and their purposes
   - Open questions and next steps
   - Important constraints or requirements discovered
3. The summary is stored with a handoff token (configurable expiry)
4. A new session can accept the handoff, receiving the summary as initial context
5. The new session continues with full awareness of prior work

**Implementation:**

- `internal/agents/` handles handoff creation and acceptance
- MCP tools (`create_handover`, `accept_handover`) enable handoff from any MCP client
- Handoff data persisted in SQLite for durability
- Token-based access control for security
- Skill creation from handoffs (`create_skill_from_handover`) for reusable workflows

**Integration with Skills system:**

Handoffs can be converted into reusable Skills, capturing not just the context but the workflow pattern for future automation.

## Consequences

### Positive

- Context preservation eliminates the "cold start" problem for continued sessions
- AI-generated summaries are more comprehensive than manual notes
- Token-based access enables secure sharing across team members
- Skill creation from handoffs captures institutional knowledge
- Works across all clients (CLI, web, desktop, iOS) via MCP
- Reduces wasted tokens re-establishing context in new sessions
- Enables long-running tasks that span multiple sessions

### Negative

- AI-generated summaries may miss nuances that the user considers important
- Summary quality depends on the model's ability to identify what matters
- Additional token cost for generating the handoff summary
- Handoff token management adds security surface area
- Large context summaries may consume significant portions of the new session's context window
- No guarantee that the receiving session interprets context the same way
- Handoff data in SQLite grows over time without aggressive cleanup policies
