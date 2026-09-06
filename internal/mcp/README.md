# Wee MCP Server

Model Context Protocol (MCP) integration for Wee agent handover operations.

## Overview

The Wee server now includes built-in MCP support, allowing Claude Code agents to discover their session IDs and perform handover operations using native MCP tools.

## Available Tools

### 1. `get_current_session`

Discover your own agent session ID by matching working directory and git branch.

**Input**: None

**Output**:
```json
{
  "session_id": "uuid",
  "status": "processing",
  "git_branch": "feature/my-feature",
  "working_directory": "/path/to/project",
  "created_at": "2025-10-27T23:45:10Z",
  "message_count": 42,
  "model_name": "sonnet",
  "match_reason": "exact_match"
}
```

### 2. `list_sessions`

List all agent sessions with optional filtering.

**Input**:
```json
{
  "status": "processing",  // Optional: filter by status
  "limit": 50              // Optional: limit results (default: 50)
}
```

**Output**:
```json
{
  "total": 5,
  "sessions": [
    {
      "id": "uuid",
      "status": "processing",
      "git_branch": "main",
      "working_directory": "/path/to/project",
      "created_at": "2025-10-27T23:45:10Z",
      "message_count": 42,
      "model_name": "sonnet",
      "cost_usd": 0.15
    }
  ]
}
```

### 3. `create_handover`

Create a handover package from an agent session.

**Input**:
```json
{
  "session_id": "uuid",
  "message_limit": 100,           // Optional: default 100
  "expiration_minutes": 60,       // Optional: default 60, max 1440
  "handover_note": "...",         // Optional: context note
  "preserve_yolo_mode": true      // Optional: default true
}
```

**Output**:
```json
{
  "success": true,
  "handover_token": "abc123...",
  "source_session_id": "uuid",
  "created_at": "2025-10-27T23:45:10Z",
  "expires_at": "2025-10-28T00:45:10Z",
  "context": {
    "working_directory": "/path/to/project",
    "git_branch": "feature/my-feature",
    "message_count": 100
  },
  "instructions": "To accept this handover, use the accept_handover tool with token: abc123..."
}
```

### 4. `accept_handover`

Accept a handover token to create a new session with inherited context.

**Input**:
```json
{
  "handover_token": "abc123...",
  "auto_start": false,           // Optional: default false
  "initial_prompt": "..."        // Optional: requires auto_start=true
}
```

**Output**:
```json
{
  "success": true,
  "session_id": "new-uuid",
  "handover_applied": true,
  "messages_imported": 100,
  "context_restored": {
    "working_directory": "/path/to/project",
    "git_branch": "feature/my-feature",
    "provider": "claude"
  },
  "session_status": "idle",
  "parent_session_id": "source-uuid",
  "yolo_mode_enabled": true
}
```

## Claude Desktop Configuration

Add this to your Claude Desktop MCP configuration (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "wee-tools": {
      "command": "/absolute/path/to/wee-editor/scripts/mcp-bridge"
    }
  }
}
```

**Important**: Replace `/absolute/path/to/wee-editor` with the actual absolute path to your Wee repository.

Example:
```json
{
  "mcpServers": {
    "wee-tools": {
      "command": "/path/to/wee-editor/scripts/mcp-bridge"
    }
  }
}
```

The `mcp-bridge` script handles:
- Reading MCP JSON-RPC requests from stdin
- Adding API key authentication
- Forwarding requests to Wee server (https://localhost:3333/mcp)
- Returning responses to stdout

## Usage Examples

### Example 1: Find Your Session ID

```javascript
// In Claude Code with MCP enabled
const result = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "get_current_session",
  arguments: {}
});

console.log(result.session_id); // Your session ID
```

### Example 2: Create and Accept Handover

```javascript
// Agent 1: Create handover
const handover = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "create_handover",
  arguments: {
    session_id: "my-session-id",
    handover_note: "Working on feature X, need help with Y",
    message_limit: 50
  }
});

const token = handover.handover_token;

// Agent 2: Accept handover
const result = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "accept_handover",
  arguments: {
    handover_token: token,
    auto_start: false
  }
});

console.log(result.session_id); // New session ID
console.log(result.messages_imported); // 50
```

### Example 3: List Active Sessions

```javascript
const sessions = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "list_sessions",
  arguments: {
    status: "processing",
    limit: 10
  }
});

console.log(sessions.sessions); // Array of active sessions
```

## Endpoint

**URL**: `https://localhost:3333/mcp`

**Method**: `POST`

**Authentication**: API key required (Bearer token)

**Content-Type**: `application/json`

**Protocol**: JSON-RPC 2.0 (MCP specification)

## Security

- API key authentication required (same as other Wee endpoints)
- Self-signed certificate support (use `-k` flag with curl)
- CORS enabled for localhost origins
- Rate limiting applied

## Error Handling

All tools return structured error responses:

```json
{
  "error": "Error message",
  "tool": "tool_name",
  "arguments": {...}
}
```

Common error codes (JSON-RPC 2.0):
- `-32700`: Parse error (malformed JSON)
- `-32601`: Method not found
- `-32602`: Invalid params
- Application-specific errors returned with `isError: true`

## Testing

Test the MCP endpoint manually:

```bash
# Initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  curl -sk -X POST https://localhost:3333/mcp \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $(cat ~/.claude/analytics/.secret)" \
    -d @-

# List tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | \
  curl -sk -X POST https://localhost:3333/mcp \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $(cat ~/.claude/analytics/.secret)" \
    -d @-

# Call tool
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_current_session","arguments":{}}}' | \
  curl -sk -X POST https://localhost:3333/mcp \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $(cat ~/.claude/analytics/.secret)" \
    -d @-
```

## Benefits Over CLI Scripts

1. **Native Integration**: Works directly in Claude Code with no external commands
2. **Auto-Discovery**: Automatically finds your session ID based on context
3. **Type-Safe**: Structured inputs and outputs with JSON schemas
4. **Error Handling**: Consistent error responses
5. **No Shell Required**: Pure API calls, no bash/curl needed

## Architecture

```
Claude Code (MCP Client)
    ↓
MCP Protocol (JSON-RPC 2.0)
    ↓
Wee Server (/mcp endpoint)
    ↓
MCP Server (internal/mcp/server.go)
    ↓
Session Manager (agent operations)
    ↓
Database (persistent storage)
```

## Related

- HTTP API: `/api/agent/sessions/:id/handover`, `/api/agent/handover/accept`
- CLI Scripts: `scripts/handover-create`, `scripts/handover-accept`
- WebSocket: Real-time session updates
