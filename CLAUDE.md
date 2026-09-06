# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**Wee** is a high-performance Go backend that provides analytics dashboards, real-time agent session monitoring, and site generation capabilities with superior performance and easy deployment.

### Key Features
- 🎮 **Control Center**: Comprehensive management for Claude Code environments
- 🤖 **Agent Server**: Go-based WebSocket server for real-time Claude agent sessions using claude-agent-sdk-go
- 🐳 **Docker Support**: Containerize Claude environments with one command
- 📊 **Analytics Dashboard**: Real-time conversation monitoring with WebSocket support
- 📱 **iOS App**: Native SwiftUI companion app for mobile session monitoring and control
- ⚡ **Performance**: 10-50x faster startup, 3-5x lower memory vs Node.js
- 📦 **Single Binary**: No dependencies, just one executable
- 🌐 **Web Server**: Fiber-based REST API with real-time updates

## Technology Stack

### Core Technologies
- **Language**: Go 1.23+ (using go1.24.8 toolchain)
- **Web Framework**: [Fiber v2](https://github.com/gofiber/fiber) - Express-like HTTP framework
- **WebSocket**: [Gorilla WebSocket](https://github.com/gorilla/websocket) + Fiber WebSocket
- **Agent SDK**: [claude-agent-sdk-go](https://github.com/schlunsen/claude-agent-sdk-go) - Claude agent conversation SDK
- **File Watching**: [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file notifications
- **System Info**: [gopsutil](https://github.com/shirou/gopsutil) - Process detection

### Project Structure

```text
wee-editor/
├── cmd/wee/                    # CLI entry point
│   └── main.go                 # Application bootstrap
├── ios/                        # iOS companion app
│   └── wee/                    # SwiftUI iOS app (Xcode project)
├── internal/                   # Private application code
│   ├── server/                 # Web server & agent functionality
│   │   ├── server.go          # Fiber HTTP/HTTPS server
│   │   ├── config.go          # Configuration management
│   │   ├── tls.go             # TLS certificate generation
│   │   ├── auth.go            # API key authentication
│   │   ├── agents/            # Agent session implementation
│   │   │   ├── agent_handler.go         # WebSocket agent handler
│   │   │   ├── session_manager.go       # Agent session management
│   │   │   ├── messages.go              # Message types
│   │   │   ├── config.go                # Agent configuration
│   │   │   ├── background_agents.go     # Background agent tracking & lifecycle
│   │   │   ├── background_agents_test.go# Background agent tests
│   │   │   ├── streaming.go             # WebSocket streaming utilities
│   │   │   └── message_handler.go       # Message handling & routing
│   │   ├── static.go          # Embedded static files
│   │   └── frontend/          # Nuxt 4 SPA frontend
│   │       ├── app/           # Nuxt app directory (IMPORTANT!)
│   │       │   ├── app.vue    # Root app component
│   │       │   ├── pages/     # Vue pages (index.vue, agents.vue)
│   │       │   └── composables/ # Vue composables (useAgentWebSocket.ts)
│   │       ├── components/    # Vue components (SessionMetrics.vue)
│   │       ├── types/         # TypeScript types
│   │       ├── nuxt.config.ts # Nuxt configuration
│   │       └── package.json   # Frontend dependencies
│   ├── analytics/              # Analytics backend modules
│   │   ├── state_calculator.go       # Conversation state logic
│   │   ├── process_detector.go       # Process monitoring
│   │   ├── conversation_analyzer.go  # JSONL parsing
│   │   └── file_watcher.go          # Real-time file watching
│   ├── database/               # Database layer (SQLite)
│   │   ├── database.go        # Database initialization & connection management
│   │   ├── schema.sql         # Complete unified database schema (embedded)
│   │   ├── models.go          # Data model structs
│   │   ├── repository.go      # Data access layer (CRUD operations)
│   │   └── git_utils.go       # Git metadata extraction helpers
│   ├── docker/                 # Docker support
│   │   ├── docker.go          # Docker operations
│   │   ├── dockerfile_generator.go  # Dockerfile generation
│   │   └── compose_generator.go     # docker-compose generation
│   ├── fileops/                # File operations
│   │   ├── github.go          # GitHub API downloads
│   │   ├── template.go        # Template processing
│   │   └── utils.go           # File utilities
│   ├── sitegenerator/          # Site generation system
│   │   ├── generator.go       # Main orchestration service
│   │   ├── session_creator.go # Agent session creation & handover integration
│   │   ├── workspace_manager.go # Project workspace management
│   │   ├── specialists/       # Specialist implementations
│   │   ├── prompts/           # System prompts for specialists
│   │   └── *_test.go          # Unit tests
│   └── websocket/              # Real-time updates
│       └── websocket.go       # WebSocket hub
├── Makefile                    # Make build automation
├── justfile                    # Just task runner
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # User documentation
```

## Database Architecture

### Overview

Wee uses **SQLite** as its embedded database for persistent storage of command history, user messages, provider configurations, agent sessions, and more. The database provides a unified data layer shared between the analytics server and agent handler.

**Database Location**: `~/.claude/wee/wee.db`

**Database Engine**: SQLite 3 with WAL mode enabled for concurrent access

### Key Features

- ✅ **Single Database File**: All data consolidated in one location
- ✅ **WAL Mode**: Write-Ahead Logging for concurrent read/write access
- ✅ **Automatic Migrations**: Schema evolves automatically on startup
- ✅ **Singleton Pattern**: Single database connection shared across components
- ✅ **Repository Pattern**: Clean data access layer with type-safe operations
- ✅ **Embedded Schema**: Schema definition compiled into binary via `//go:embed`
- ✅ **Secure Permissions**: Database file has 0600 permissions (user read/write only)

### Database Tables

The database schema includes 11 core tables organized by functionality:

#### Command History Tables
- **`shell_commands`**: Records of all Bash tool executions
- **`claude_commands`**: Records of all Claude Code tool invocations
- **`command_stats`**: Aggregated command statistics

#### Conversation & Session Tables
- **`conversations`**: Metadata for Claude Code conversation sessions
- **`user_messages`**: User input messages intercepted by hooks
- **`notifications`**: Permission requests and idle alerts

#### Agent Session Tables
- **`agent_sessions`**: Agent conversation sessions from unified server
- **`agent_messages`**: Individual messages within agent sessions

#### Configuration Tables
- **`providers`**: AI provider configurations (Anthropic, OpenRouter, etc.)
- **`user_settings`**: User preferences and application settings

### Database Operations

```go
// Get repository instance
repo := database.NewRepository(db)

// Record a shell command
cmd := &database.ShellCommand{
    ConversationID:   "conv-123",
    Command:          "git status",
    WorkingDirectory: "/path/to/project",
    GitBranch:        "main",
    ExitCode:         &exitCode,
    ExecutedAt:       time.Now(),
}
repo.RecordShellCommand(cmd)

// Provider management
provider := &database.ProviderConfig{
    ProviderID: "anthropic",
    APIKey:     "sk-ant-...",
    ModelName:  "claude-sonnet-4-20250514",
}
repo.SaveProvider(provider) // Auto-sets as current, unsets others
```

## Site Generator

### Overview

The Site Generator is a multi-agent orchestration system that generates fully functional static sites from natural language descriptions. It uses a **Nuxt UI-based workflow** with a streamlined **3-specialist pipeline**.

**Location**: `internal/sitegenerator/`

**Architecture**:
- **SiteGenerator** - Core orchestration service with workspace management
- **Orchestrator Session** - Plans execution flow and coordinates specialists
- **Designer Session** - Creates Nuxt UI theme configuration and design specifications
- **Implementer Session** - Creates Nuxt project and generates static site

### Key Features

- **Nuxt UI Workflow**: Modern static site generation using Nuxt 3 + Nuxt UI
- **Multi-Session Orchestration**: 3 specialized sessions work in sequence with handovers
- **Workspace Management**: Isolated project directories with structured paths
- **Database-Driven**: All progress persisted in SQLite for resume/retry
- **Real-Time Progress**: WebSocket broadcasting shows live execution status
- **Error Recovery**: Automatic retry with exponential backoff strategies
- **Session Handovers**: Context passing between specialists via Wee handover system
- **Multi-Provider Support**: Claude, DeepSeek, GLM, Kimi, and custom providers

### API Endpoints

```bash
# Start generation
POST /api/site/generate
{
  "description": "Your site description",
  "category": "product|portfolio|service|blog|ecommerce",
  "style_preferences": ["modern", "minimal"],
  "provider": "claude|deepseek|glm|kimi|custom",
  "model": "sonnet|haiku|deepseek-chat|glm-4.6|kimi-k2",
  "notes": "Any special requirements"
}

# List projects
GET /api/site/projects?limit=10&offset=0

# Get project details
GET /api/site/projects/:id

# Download generated page
GET /api/site/projects/:id/download

# Retry failed step
POST /api/site/projects/:id/retry
{
  "step_number": 2
}
```

### Session Execution Timeline

Each specialist session runs in sequence with handover-based context passing:

1. **Orchestrator** (1-2 min): Analyzes description, creates execution plan
2. **Designer** (2-4 min): Creates Nuxt UI theme configuration and design specs
3. **Implementer** (5-8 min): Creates Nuxt project and generates static site

**Total: 8-14 minutes** with automatic retries on failure

### Troubleshooting

**Generation timeout:**
- Increase timeout in `internal/sitegenerator/timeouts.go`
- Check specialist logs: `~/.claude/sitegen/projects/<id>/logs/generation.log`
- Verify session is not stuck via API
- Simplify user description

**Specialist fails to complete:**
- Check session error state: `GET /api/agent/sessions/<session-id>`
- Review specialist output validation in `validators.go`
- Check workspace permissions: `~/.claude/sitegen/projects/<id>/`
- Retry failed step: `POST /api/site/projects/<id>/retry {"step_number": N}`

## Development Commands

### Building & Running

```bash
# Build binary (fast - ~2 seconds)
make build
# or
just build

# Run directly
go run ./cmd/wee

# Install globally
go install ./cmd/wee
```

### Analytics Dashboard

The analytics dashboard is a Nuxt 4 SPA frontend with a Go Fiber backend.

```bash
# Launch analytics server (backend)
./wee --analytics
# or
make run-analytics

# Access dashboard (HTTPS by default)
open https://localhost:3333

# API endpoints (use -k to accept self-signed cert)
curl -k https://localhost:3333/api/data
curl -k https://localhost:3333/api/conversations
curl -k https://localhost:3333/api/processes
curl -k https://localhost:3333/api/stats
```

## Unified Server (Analytics + Agents)

The unified server combines analytics dashboard and agent session functionality in a single Go-based Fiber server on port 3333.

### Features
- **Analytics Dashboard**: Real-time conversation monitoring with WebSocket support
- **Agent Sessions**: WebSocket-based real-time Claude agent conversations using claude-agent-sdk-go
- **API Key Authentication**: Unified authentication for all endpoints
- **TLS/HTTPS**: Automatic self-signed certificate generation
- **Session Management**: Multiple concurrent agent sessions
- **Tool Support**: Full agent tool support (Read, Write, Edit, Bash, etc.)

### Quick Start

```bash
# Start unified server (includes analytics + agents)
./wee --analytics
```

### Unified Server Endpoints

**Port**: 3333 (HTTPS by default)

**Key Endpoints**:
- `GET /api/agent/sessions` - List all agent sessions
- `GET /api/agent/sessions/:id/messages` - Get messages for a session
- `GET /api/agent/sessions/:id/context` - Get live working directory and git branch

### Switching Model / Provider Mid-Session

A live session can be moved to a different provider and/or model without
creating a new session. In the dashboard: session sidebar → **Model → change**.
The switch is refused while a turn is in flight (interrupt it or wait).

```bash
# WebSocket
{"type": "change_session_model", "session_id": "...", "provider": "deepseek", "model": "deepseek-chat", "base_url": "optional"}
# → {"type": "session_model_changed", "history_mode": "resumed|replayed|rebuilt|none", ...}

# REST (409 while processing)
curl -k -X PATCH https://localhost:3333/api/agent/sessions/<id>/model \
  -H "Authorization: Bearer $API_KEY" -H "Content-Type: application/json" \
  -d '{"provider": "claude", "model": "opus"}'
```

**How it works** (`internal/server/agents/model_switch.go`)
- `resolveExecutionPath()` classifies a session as `claude-sdk`, `direct`
  (OpenAI-compatible loop) or `codex` — the same routing `sendPromptInternal` uses.
- `ChangeSessionModel()` rewrites `ModelName`/`Provider`/`Options.{Model,Provider,BaseURL}`
  (session-scoped API key/base URL are dropped on a provider change; the registry
  base URL is applied, DB providers table still wins at prompt time), persists,
  closes the cached Claude SDK client so the next prompt spawns a fresh one, and
  records a `🔀 Switched model` system message in the conversation.
- Conversation history per transition:
  - Claude SDK → Claude SDK (incl. Anthropic-compatible providers like DeepSeek/GLM/Kimi):
    `ClaudeSessionID` is kept and the CLI transcript is resumed with `--resume` (**resumed**).
  - anything → direct: history is rebuilt from `agent_messages` on every turn anyway (**rebuilt**).
  - anything → Claude SDK from a non-SDK path, or anything → Codex from a non-Codex path:
    `SessionOptions.HistoryReplayPending` is set (persisted in the options JSON) and the DB
    transcript is prepended to the first prompt on the new provider inside a
    `<conversation_history>` block, capped at 60k chars, oldest first (**replayed**).
    The flag clears once the new provider holds the history natively (Claude session
    ID captured / Codex thread id remembered).
  - Leaving the SDK/Codex path clears `ClaudeSessionID` / `CodexThreadID` so a later
    switch back replays the full DB history instead of resuming a stale transcript.
- Tests: `model_switch_test.go`.

### Agent Session Handovers

Agent session handovers allow one Claude agent to pass context and control to another agent session.

**How It Works:**

1. **Source Session** creates a handover package containing:
   - Session metadata (working directory, git branch, model, etc.)
   - Conversation history (optional, configurable message limit)
   - Context summary/handover note
   - Permission settings (YOLO mode, always-allow rules)

2. **Handover Token** is generated (secure, time-limited, single-use)

3. **Target Session** accepts the handover token and either:
   - Creates a new session with inherited context
   - Applies context to an existing session

**Creating a Handover:**

```bash
API_KEY=$(cat ~/.claude/analytics/.secret)

curl -k -X POST https://localhost:3333/api/agent/handover/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "session_id": "your-session-id-here",
    "options": {
      "include_messages": true,
      "include_context": true,
      "message_limit": 50,
      "handover_note": "Working on feature X, need help with Y",
      "expiration_minutes": 60
    }
  }'
```

**Accepting a Handover:**

```bash
curl -k -X POST https://localhost:3333/api/agent/handover/accept \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "handover_token": "token-from-create-response",
    "auto_start": true,
    "initial_prompt": "Continue working on the task from the previous session"
  }'
```

**Using Just Commands:**

```bash
just handover-create <session-id>
just handover-accept <token> --auto-start --prompt "Continue task"
```

**MCP Tools (Recommended for Claude Code):**

```javascript
// Get your current session ID automatically
const session = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "get_current_session",
  arguments: {}
});

// Create handover
const handover = await use_mcp_tool({
  server_name: "wee-tools",
  tool_name: "create_handover",
  arguments: {
    session_id: "my-session-id",
    handover_note: "Working on feature X",
    message_limit: 50
  }
});
```

### Configuration

The unified server is configured via `~/.claude/analytics/config.json`:

```json
{
  "server": {
    "port": 3333,
    "host": "127.0.0.1"
  },
  "tls": {
    "enabled": true
  },
  "auth": {
    "enabled": true
  },
  "agent": {
    "model": "claude-sonnet-4-5-20250929",
    "max_concurrent_sessions": 10
  }
}
```

### Troubleshooting

**Port 3333 already in use:**
```bash
lsof -i :3333
kill -9 <PID>
```

**Agent functionality not working:**
```bash
echo $ANTHROPIC_API_KEY
export ANTHROPIC_API_KEY=your-api-key-here
```

**WebSocket connection fails:**
```bash
# Check server is running and use -k flag with curl
curl -k https://localhost:3333/api/health
```

### Background Agents

Background agents allow Claude agents to run asynchronously in the background using the Task tool with `run_in_background: true`. The unified server tracks these agents, monitors their progress, and broadcasts real-time updates via WebSocket.

**How It Works:**

1. **Agent Spawning** - Claude Code spawns an agent with `run_in_background: true`
2. **Registration** - The background agent is registered in the `BackgroundAgentManager`
3. **Monitoring** - Real-time progress updates are broadcast via WebSocket
4. **Completion** - When done, the agent is marked as completed or failed

**Message Types:**

```
📊 Messages broadcast in real-time:
- background_agent_started      - Agent starts running
- background_agent_progress     - Progress percentage and output updates
- background_agent_output       - Incremental output lines
- background_agent_completed    - Agent finishes successfully
- background_agent_failed       - Agent encounters an error
- background_agents_list        - List of agents for a session
```

**Frontend UI:**

The `BackgroundAgentPanel.vue` component displays:
- Agent name and type (e.g., "general-purpose", "code-reviewer")
- Progress bar (0-100%)
- Status badge (running, completed, failed)
- Real-time output streaming
- Error messages
- Execution timeline

**Backend Implementation:**

Located in `internal/server/agents/`:
- `background_agents.go` - Manager for tracking and lifecycle
- `background_agents_test.go` - Comprehensive unit tests
- Integration with `agent_handler.go` for WebSocket messaging

**API Endpoints:**

```bash
# List background agents for a session
{
  "type": "list_background_agents",
  "session_id": "session-uuid"
}

# Response receives agents array with:
# - agent_id, parent_session_id, subagent_type, description
# - status (running/completed/failed), progress (0.0-1.0)
# - output_lines, created_at, updated_at, completed_at, error_message
```

**Example Workflow:**

```javascript
// In Claude Code, spawn a background agent
const result = await use_mcp_tool({
  server_name: "wee",
  tool_name: "Task",
  arguments: {
    description: "Code review",
    prompt: "Review the code",
    subagent_type: "code-reviewer",
    run_in_background: true  // Key for background execution
  }
});

// Frontend receives WebSocket messages:
// 1. background_agent_started - UI shows agent is running
// 2. background_agent_progress - Progress bar updates
// 3. background_agent_output - Output streaming in real-time
// 4. background_agent_completed - Final result shown
```

**Testing:**

Comprehensive test suite (`background_agents_test.go`) covers:
- Agent registration and lifecycle
- Progress tracking and output handling
- Concurrent operations (multiple agents running simultaneously)
- Message broadcasting
- Session filtering and cleanup
- Edge cases (non-existent agents, empty output, etc.)

Run tests:
```bash
go test ./internal/server/agents -v
```

**Performance Notes:**

- **Concurrent Agents**: Supports hundreds of concurrent background agents
- **Memory**: Each agent uses ~1KB for metadata
- **Cleanup**: Completed/failed agents older than 24 hours are auto-removed
- **WebSocket**: Updates streamed in real-time with minimal latency

## OpenAI Codex Provider

Sessions can run on the **OpenAI Codex** agent instead of the Claude CLI. Select
provider `codex` (🟢 "OpenAI Codex") in *Create New Session*; models such as
`gpt-6-astra` are then executed by the `codex` CLI through
[codex-sdk-go](https://github.com/schlunsen/codex-sdk-go).

**Requirements**
- `npm install -g @openai/codex` (or `brew install codex`)
- Auth: `codex login` (ChatGPT subscription) **or** an API key in Settings → Providers
  (`codex`, falling back to `openai`), or `CODEX_API_KEY` / `OPENAI_API_KEY`, or
  `WEE_CODEX_API_KEY` at startup (auto-seeds the provider).

**How it works** (`internal/server/agents/codex_provider.go`)
- Routing: `isCodexSession()` is checked in `sendPromptInternal` and
  `SendPromptWithContent` *before* the Claude/direct-provider decision — an explicit
  `provider: codex` wins, otherwise any model name containing `codex`.
- Each prompt is one Codex turn (`Thread.RunStreamedInputs`). The thread id is stored in
  `SessionOptions.CodexThreadID` (JSON in `agent_sessions.options`, no schema change) and
  resumed on the next prompt, so history lives in `~/.codex/sessions`, not in wee.
- Codex JSONL items are mapped onto the existing SDK messages: `agent_message` → text,
  `reasoning` → thinking block, `command_execution` → `Bash` tool_use/tool_result,
  `file_change` → `FileChange`, `mcp_tool_call` → `mcp__<server>__<tool>`,
  `web_search` → `WebSearch`, `todo_list` → `TodoWrite`. Persistence and WebSocket
  delivery reuse the direct-provider helpers (`sendDirectTextMessage`, …).
- Permission mode → sandbox: `yolo`/`bypassPermissions`/`allow-all` → `danger-full-access`,
  `read-only` → `read-only`, otherwise `workspace-write`. Approval policy is always
  `never` (headless). Effort level → `model_reasoning_effort`.
- The wee MCP server is exposed to Codex via `--config mcp_servers.wee-tools.*`.
- Interrupt: cancelling `session.ctx` kills the `codex` subprocess.
- Provider row: `internal/providers/providers.json`, `internal/database/providers_seed.json`,
  migration 38. Tests: `codex_provider_test.go`, `migration_codex_test.go`.

## GPU Sidecar

The GPU sidecar lets users launch on-demand RunPod GPU pods from their sandbox for ML training, fine-tuning, and inference.

### GPU Tiers & Storage

| Tier | GPU | Cost | Default Volume | Default Disk |
|------|-----|------|---------------|-------------|
| starter | RTX 4090 24GB | ~$0.39/hr | 20 GB | 20 GB |
| pro | A100 80GB | ~$1.64/hr | 50 GB | 40 GB |
| beast | H100 80GB | ~$3.49/hr | 100 GB | 50 GB |

Users can customize storage via `--volume` and `--disk` CLI flags, the TUI storage screen, or `volume_gb`/`container_disk_gb` MCP tool params. Values below the tier default are automatically raised to the minimum.

### GPU Commands

```bash
wee gpu                                          # Interactive TUI picker
wee gpu launch --tier pro --template abc123       # Launch with defaults
wee gpu launch --tier starter --template abc123 \
  --volume 100 --disk 50                         # Custom storage
wee gpu status                                    # Check status & cost
wee gpu shell                                     # SSH into pod
wee gpu push / pull                               # Sync files
wee gpu stop / resume / kill                      # Lifecycle management
```

### GPU MCP Tools

Available as MCP tools when running in a wee.cat sandbox:
- `gpu_launch` — Launch pod (tier, template_id, optional volume_gb, container_disk_gb)
- `gpu_status` — Session status with uptime and cost
- `gpu_run` — Execute command on pod via SSH
- `gpu_push` / `gpu_pull` — Sync files to/from pod
- `gpu_stop` / `gpu_resume` / `gpu_terminate` — Lifecycle management
- `gpu_templates` — List available templates

## Frontend Development

```bash
# Navigate to frontend directory
cd internal/server/frontend

# Install dependencies
npm install

# Run Nuxt dev server (development)
npm run dev

# Build for production
npm run build

# Generate static files
npm run generate
```

**Important**: Nuxt 4 requires all application code (pages, composables, etc.) to be inside the `app/` directory.

## iOS App Development

The iOS app (`ios/wee/`) is a native SwiftUI companion application for real-time monitoring and control of agent sessions.

### Technology Stack
- **Language**: Swift 5.9+ with SwiftUI
- **Deployment Target**: iOS 15.0+
- **Architecture**: MVVM with async/await concurrency
- **WebSocket**: URLSessionWebSocketTask with Network framework
- **Storage**: Keychain for secure credential storage

### Development Commands

```bash
# Open Xcode project
open ios/wee/wee.xcodeproj

# Build app
xcodebuild -project ios/wee/wee.xcodeproj -scheme wee -configuration Debug

# Run tests
xcodebuild -project ios/wee/wee.xcodeproj -scheme wee -configuration Debug test
```

### Key Features
- ✅ **Real-time Sessions**: WebSocket-based live updates from backend
- ✅ **Production WebSocket Service**: Ping/pong keep-alive, network monitoring, auto-reconnection
- ✅ **Dashboard**: View active sessions, projects, and statistics
- ✅ **Session Monitor**: Real-time message streaming with chat interface
- ✅ **Secure Authentication**: API key stored in Keychain
- ✅ **TLS/HTTPS Support**: Self-signed certificate handling
- ✅ **Offline Resilience**: Automatic reconnection with exponential backoff

## Security Features

### TLS/HTTPS Encryption

Wee supports two types of TLS certificates:

**1. mkcert Certificates (Recommended)**
- Locally-trusted development certificates
- No browser security warnings
- Perfect for PWA installation

**2. Self-Signed Certificates (Fallback)**
- Auto-generated on first run
- Works for basic HTTPS

**Setup mkcert:**

```bash
# Install mkcert (one-time setup)
brew install mkcert
mkcert -install

# Generate trusted certificates for Wee
wee cert --regenerate --mkcert
```

### API Key Authentication

**Automatic Setup:**
- API key automatically generated on first run
- Stored in `~/.claude/analytics/.secret`
- Required for all POST/PUT/DELETE/PATCH requests
- GET requests allowed without authentication

**Viewing Your API Key:**
```bash
cat ~/.claude/analytics/.secret
```

### User Authentication (Enabled by Default)

**Wee requires user authentication by default for enhanced security.**

**First Run Setup:**

On first startup, you'll be automatically prompted to create an admin account:

1. Navigate to `https://localhost:3333`
2. You'll be redirected to the setup page automatically
3. Create your admin username and password (minimum 8 characters)
4. You'll be automatically logged in and redirected to the dashboard

**Disabling Authentication:**

If you need to disable user authentication:

- **Environment Variable**: `export WEE_AUTH_USER_AUTH_ENABLED=false`
- **Config File**: Edit `~/.claude/wee/config.json` and set `"user_auth_enabled": false`

**Managing Users:**

After setup, you can manage users via:
- **Dashboard**: Navigate to Settings → User Management (admin only)
- **API**: Use `/api/auth/users` endpoints (admin only)

### Security Best Practices

1. **Keep API Key Secret:**
   - Never commit `.secret` file to version control
   - Regenerate if compromised: `rm ~/.claude/analytics/.secret && ./wee --analytics`

2. **Certificate Management:**
   - Self-signed certs are secure for localhost
   - For remote access, use proper CA-signed certificates

3. **Network Security:**
   - Server binds to `127.0.0.1` by default (localhost-only)
   - For remote access, use SSH tunneling: `ssh -L 3333:localhost:3333 user@remote-host`

## Development Workflow

```bash
# Format code
make fmt
just fmt

# Run tests
make test
just test

# Test with coverage
make test-coverage

# Cross-platform builds
make build-all
just build-all

# Clean build artifacts
make clean
just clean
```

## Code Style & Best Practices

### Go Idioms

1. **Error Handling**: Always check and handle errors explicitly
2. **Struct Initialization**: Use composite literals
3. **Goroutines**: Use for concurrent operations
4. **Channels**: For communication between goroutines

### Project Conventions

1. **Package Organization**:
   - `internal/` for private code (main application)
   - `pkg/` for public libraries (reusable code)
   - `cmd/` for executable entry points

2. **Naming**:
   - Packages: lowercase, single word (`analytics`, `server`)
   - Structs: PascalCase (`ConversationAnalyzer`, `ProcessDetector`)
   - Functions: camelCase for private, PascalCase for exported

3. **File Naming**:
   - Use snake_case for Go files (`state_calculator.go`)
   - Group related functions in same file
   - Keep files focused on single responsibility

## Architecture & Design Patterns

### Analytics Backend

The analytics system is modular and follows the Single Responsibility Principle:

1. **StateCalculator**: Determines conversation state based on timestamps and messages
2. **ProcessDetector**: Monitors running Claude CLI processes
3. **ConversationAnalyzer**: Parses JSONL conversation files
4. **FileWatcher**: Monitors file changes for real-time updates

### Concurrent Patterns

```go
// Hub pattern for WebSocket connections
type Hub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan []byte
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
    mutex      sync.RWMutex
}

// Run hub in goroutine
go hub.Run()
```

### Server Architecture

The Fiber server follows middleware patterns with modular route handlers.

## Common Tasks

### Adding a New API Endpoint

1. Add handler method to `internal/server/server.go`:
   ```go
   func (s *Server) handleNewEndpoint(c *fiber.Ctx) error {
       data := s.getData()
       return c.JSON(fiber.Map{
           "result": data,
       })
   }
   ```

2. Register route in `setupRoutes()`:
   ```go
   api.Get("/new-endpoint", s.handleNewEndpoint)
   ```

### Embedding Static Files

```go
//go:embed static/file.html
var fileHTML []byte

func ServeFile(c *fiber.Ctx) error {
    c.Set("Content-Type", "text/html")
    return c.Send(fileHTML)
}
```

## Performance Considerations

### Benchmarks (vs Node.js version)

| Metric | Node.js | Go | Improvement |
|--------|---------|-----|-------------|
| Build Time | npm install (minutes) | 2-5 seconds | 50-100x faster |
| Binary Size | 50MB+ (node_modules) | ~15MB | 3x smaller |
| Startup Time | ~500ms | <10ms | 50x faster |
| Memory Usage | ~80MB baseline | ~15MB | 5x lower |

## Debugging & Troubleshooting

### Enable Verbose Logging

```bash
./wee --analytics --verbose
```

### Check Build Issues

```bash
# Verify Go version
go version  # Should be 1.23+

# Check dependencies
go mod verify
go mod tidy

# Clear cache
go clean -cache -modcache -i -r
```

### Common Issues

1. **Port 3333 in use**:
   ```bash
   lsof -i :3333
   kill -9 <PID>
   ```

2. **WebSocket connection fails**:
   - Check firewall settings
   - Verify CORS configuration
   - Test with `wscat -c wss://localhost:3333/ws --no-check`

3. **Database locked errors**:
   ```bash
   lsof ~/.claude/wee/wee.db
   sqlite3 ~/.claude/wee/wee.db "PRAGMA wal_checkpoint(TRUNCATE);"
   ```

## Git Workflow

### Branch Protection Policy

⚠️ **CRITICAL FOR CLAUDE CODE**: Never commit directly to the `main` branch unless it's during a release process!

**CLAUDE CODE RULE**: ALWAYS create a feature branch BEFORE making ANY code changes.

**Rules**:
- All development work MUST be done on feature branches
- Feature branches should follow naming conventions:
  - `feature/` - New features (e.g., `feature/agent-session-manager`)
  - `fix/` - Bug fixes (e.g., `fix/websocket-connection-error`)
  - `docs/` - Documentation updates (e.g., `docs/update-readme`)
  - `refactor/` - Code refactoring (e.g., `refactor/analytics-module`)
  - `test/` - Test additions/updates (e.g., `test/add-agent-tests`)
  - `chore/` - Maintenance tasks (e.g., `chore/update-dependencies`)

**Workflow**:
1. **ALWAYS create a feature branch before making changes**
2. Make commits on the feature branch
3. Create a pull request to merge into `main`
4. Only during release processes should commits be made to `main`

### Commit Message Format

```text
<type>: <subject>

<body>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

**Important**: Never use "made with clade" in messages. Never use "co-authored" thing.

### Creating Pull Requests

```bash
# ALWAYS create a feature branch first
git checkout -b feature/new-feature

# Make changes and commit
git add .
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/new-feature
gh pr create --title "Add new feature" --body "Description"
```

## Logs

You can find logs in ~/.claude/wee/logs

## Deployment

### Building for Production

```bash
# Build optimized binary
go build -ldflags="-s -w" -o wee ./cmd/wee

# Cross-compile for all platforms
make build-all

# Outputs:
# - dist/wee-linux-amd64
# - dist/wee-linux-arm64
# - dist/wee-darwin-amd64
# - dist/wee-darwin-arm64
# - dist/wee-windows-amd64.exe
```

### Installation Methods

```bash
# Direct binary
curl -L https://github.com/schlunsen/wee-editor/releases/latest/download/wee-<platform> -o wee
chmod +x wee
sudo mv wee /usr/local/bin/

# From source
git clone https://github.com/schlunsen/wee-editor
cd wee-editor
make install
```

## Resources

### Documentation
- [Fiber Framework](https://docs.gofiber.io/)
- [fsnotify](https://github.com/fsnotify/fsnotify)


### Go Resources
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)

## License

MIT License - See LICENSE file for details

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write tests
5. Run `make fmt && make test`
6. Submit a pull request

---


