# ADR 009: Model Context Protocol (MCP) Support

## Status

Accepted

## Date

2024-01-01

## Context

AI agents need access to external tools and data sources to be effective. The Model Context Protocol (MCP) is an emerging open standard for connecting AI models to external capabilities. Supporting MCP allows Wee to expose its features as tools that any MCP-compatible AI client can use, and also allows Wee's agents to consume external MCP servers.

Without MCP, each integration would require custom code for both the tool provider and consumer sides.

## Decision

We implemented **MCP server support** in `internal/mcp/`, allowing Wee to both expose tools via MCP and consume external MCP servers.

**Wee as MCP Server:**

Wee exposes its capabilities as MCP tools that Claude Code and other MCP clients can invoke:

- `Bash` - Execute shell commands in the project context
- `Read` / `Write` / `Edit` - File operations
- `Glob` / `Grep` - File search and content search
- Session management tools (create/list/handover)
- Memory tools (store/recall)
- Skill tools (list/invoke)
- GPU management tools
- Deployment tools

**Wee as MCP Consumer:**

Agent sessions can connect to external MCP servers, giving agents access to third-party tools (databases, APIs, services) defined by MCP server configurations.

**Component system:**

MCP servers can be installed and managed through the Components system (`internal/components/`), allowing users to discover and add new MCP integrations.

## Consequences

### Positive

- Standardized protocol means any MCP-compatible client can use Wee's tools
- Extensible - new capabilities can be added as MCP tools without core changes
- External MCP servers can be added by users without code changes
- Community MCP servers provide instant access to databases, APIs, and services
- Separates tool implementation from AI provider specifics
- Future-proof as MCP ecosystem grows

### Negative

- MCP is still an evolving standard; breaking changes may require updates
- Additional abstraction layer adds some latency to tool invocations
- MCP server lifecycle management (start, health check, restart) adds complexity
- Security surface increases with each connected MCP server
- Debugging tool invocations across MCP boundaries is harder than direct function calls
- Not all AI providers support MCP natively, requiring translation layers
