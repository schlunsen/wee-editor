# Feature: Integrated Terminal for Agent Sessions

## 🎯 Overview

Add an integrated terminal view within agent sessions that allows users to switch between the chat interface and a fully functional terminal, with the working directory (`pwd`) set to the session's project path.

This feature enables developers to:
- Run commands directly in the project directory without leaving the Wee interface
- Toggle seamlessly between chatting with Claude and executing shell commands
- Maintain context by keeping the terminal tied to the specific agent session's working directory
- View command output in real-time with full terminal emulation support

## 📋 User Stories

### Primary Use Case
**As a developer using Wee**, I want to switch the chat area to a terminal view so that I can run shell commands in my project directory without leaving the interface.

### Additional Use Cases
- **As a power user**, I want to quickly toggle between Claude chat and terminal using a keyboard shortcut
- **As a developer**, I want my terminal sessions to persist when I refresh the page
- **As a security-conscious user**, I want terminal access to be authenticated and session-scoped
- **As a mobile user** (future), I want to view terminal output on iOS (read-only initially)

## 🏗️ Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (Nuxt 4/Vue 3)                │
├─────────────────────────────────────────────────────────────┤
│  • SessionTerminal.vue (xterm.js integration)               │
│  • useTerminalWebSocket.ts (WebSocket composable)           │
│  • Chat/Terminal Toggle (SessionViewToggle.vue)             │
│  • Local storage for view persistence                       │
└──────────────────┬──────────────────────────────────────────┘
                   │ WebSocket (/api/agent/sessions/:id/terminal/ws)
                   │
┌──────────────────▼──────────────────────────────────────────┐
│                   Backend (Go + Fiber)                      │
├─────────────────────────────────────────────────────────────┤
│  internal/server/terminal/                                  │
│    • terminal_handler.go (WebSocket handler)                │
│    • pty_manager.go (PTY lifecycle)                         │
│    • terminal_session.go (Session management)               │
│                                                             │
│  API Endpoints:                                             │
│    POST   /api/agent/sessions/:id/terminal/start           │
│    DELETE /api/agent/sessions/:id/terminal/stop            │
│    WS     /api/agent/sessions/:id/terminal/ws              │
│    GET    /api/agent/sessions/:id/terminal/status          │
└──────────────────┬──────────────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────────────┐
│                   Database (SQLite)                         │
├─────────────────────────────────────────────────────────────┤
│  • terminal_sessions (session metadata)                     │
│  • terminal_history (command history)                       │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 Technical Implementation

### 1. Backend Implementation (Go)

#### New Package: `internal/server/terminal/`

**File: `pty_manager.go`**
```go
type PTYManager struct {
    sessions map[string]*TerminalSession
    mutex    sync.RWMutex
}

type TerminalSession struct {
    ID              string
    AgentSessionID  string
    PTY             *os.File
    WorkingDir      string
    Shell           string
    Env             []string      // Must include TERM=xterm-256color
    CreatedAt       time.Time
    LastActivityAt  time.Time
    Rows            int
    Cols            int
}

func (m *PTYManager) SpawnTerminal(agentSessionID, workingDir, shell string) (*TerminalSession, error)
func (m *PTYManager) ResizeTerminal(terminalID string, rows, cols int) error
func (m *PTYManager) KillTerminal(terminalID string) error
func (m *PTYManager) CleanupIdleTerminals(timeout time.Duration) int
```

**File: `terminal_handler.go`**
```go
func HandleTerminalWebSocket(c *websocket.Conn, agentSessionID string) error
func streamPTYOutput(pty *os.File, conn *websocket.Conn) error
func handleTerminalInput(conn *websocket.Conn, pty *os.File) error
```

**File: `terminal_session.go`**
```go
type TerminalSessionStore interface {
    SaveTerminalSession(session *TerminalSession) error
    GetTerminalSession(id string) (*TerminalSession, error)
    ListTerminalsByAgentSession(agentSessionID string) ([]*TerminalSession, error)
    RecordCommand(terminalID, command string) error
}
```

#### API Endpoints

```go
// Server routes registration
api := app.Group("/api/agent/sessions/:id/terminal")
api.Post("/start", authMiddleware, startTerminalHandler)
api.Delete("/stop", authMiddleware, stopTerminalHandler)
api.Get("/status", authMiddleware, terminalStatusHandler)
api.Get("/ws", websocket.New(terminalWebSocketHandler))
```

#### WebSocket Protocol

**Client → Server Messages:**
```json
{
  "type": "terminal.input",
  "data": "ls -la\n"
}

{
  "type": "terminal.resize",
  "cols": 80,
  "rows": 24
}
```

**Server → Client Messages:**
```json
{
  "type": "terminal.output",
  "data": "\x1b[32mfile1.txt\x1b[0m\n"
}

{
  "type": "terminal.started",
  "terminal_id": "uuid",
  "shell": "/bin/zsh",
  "cwd": "/path/to/project",
  "rows": 24,
  "cols": 80
}

{
  "type": "terminal.exited",
  "terminal_id": "uuid",
  "exit_code": 0
}

{
  "type": "terminal.error",
  "error": "PTY spawn failed: permission denied"
}
```

### 2. Database Schema

```sql
-- Terminal sessions table
CREATE TABLE terminal_sessions (
    id TEXT PRIMARY KEY,
    agent_session_id TEXT NOT NULL,
    shell TEXT NOT NULL,
    working_directory TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    exit_code INTEGER,
    rows INTEGER DEFAULT 24,
    cols INTEGER DEFAULT 80,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE
);

-- Terminal command history table
CREATE TABLE terminal_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    terminal_session_id TEXT NOT NULL,
    command TEXT NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (terminal_session_id) REFERENCES terminal_sessions(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_terminal_sessions_agent_id
    ON terminal_sessions(agent_session_id);

CREATE INDEX idx_terminal_sessions_active
    ON terminal_sessions(agent_session_id)
    WHERE ended_at IS NULL;

CREATE INDEX idx_terminal_history_session
    ON terminal_history(terminal_session_id, executed_at DESC);
```

### 3. Frontend Implementation (Nuxt 4/Vue 3)

#### Dependencies

```json
{
  "xterm": "^5.3.0",
  "xterm-addon-fit": "^0.8.0",
  "xterm-addon-web-links": "^0.9.0",
  "xterm-addon-webgl": "^0.16.0"
}
```

**Note:** The `webgl` addon provides GPU-accelerated rendering, which significantly improves performance for TUI applications like vim, htop, and tmux.

#### Component: `SessionTerminal.vue`

```vue
<template>
  <div class="terminal-container" ref="terminalContainer">
    <div ref="terminalElement"></div>
  </div>
</template>

<script setup lang="ts">
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'

const props = defineProps<{
  sessionId: string
}>()

const terminalElement = ref<HTMLElement>()
const terminal = ref<Terminal>()
const fitAddon = ref<FitAddon>()
const { connect, send, disconnect } = useTerminalWebSocket(props.sessionId)

onMounted(() => {
  terminal.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, Courier New, monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4'
    },
    // Critical for vim support
    scrollback: 10000,
    allowTransparency: false,
    convertEol: true
  })

  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)
  terminal.value.loadAddon(new WebLinksAddon())

  // WebGL addon for better performance with vim/htop/tmux
  try {
    const { WebglAddon } = await import('xterm-addon-webgl')
    terminal.value.loadAddon(new WebglAddon())
  } catch (e) {
    console.warn('WebGL addon failed to load, falling back to canvas renderer')
  }

  terminal.value.open(terminalElement.value!)
  fitAddon.value.fit()

  // Handle terminal input
  terminal.value.onData((data) => {
    send({ type: 'terminal.input', data })
  })

  // Handle resize
  terminal.value.onResize(({ cols, rows }) => {
    send({ type: 'terminal.resize', cols, rows })
  })

  // Connect WebSocket
  connect((message) => {
    if (message.type === 'terminal.output') {
      terminal.value?.write(message.data)
    }
  })
})

onUnmounted(() => {
  disconnect()
  terminal.value?.dispose()
})
</script>

<style scoped>
.terminal-container {
  height: 100%;
  width: 100%;
  padding: 1rem;
  background: #1e1e1e;
}
</style>
```

#### Composable: `useTerminalWebSocket.ts`

```typescript
export function useTerminalWebSocket(sessionId: string) {
  const socket = ref<WebSocket>()
  const connected = ref(false)

  function connect(onMessage: (msg: any) => void) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/agent/sessions/${sessionId}/terminal/ws`

    socket.value = new WebSocket(wsUrl)

    socket.value.onopen = () => {
      connected.value = true
      send({ type: 'terminal.start' })
    }

    socket.value.onmessage = (event) => {
      const data = JSON.parse(event.data)
      onMessage(data)
    }

    socket.value.onerror = (error) => {
      console.error('Terminal WebSocket error:', error)
    }

    socket.value.onclose = () => {
      connected.value = false
    }
  }

  function send(message: any) {
    if (socket.value?.readyState === WebSocket.OPEN) {
      socket.value.send(JSON.stringify(message))
    }
  }

  function disconnect() {
    socket.value?.close()
  }

  return { connect, send, disconnect, connected }
}
```

#### Toggle UI in `agents.vue`

```vue
<template>
  <div class="session-view">
    <!-- View Toggle -->
    <div class="view-toggle">
      <button
        :class="{ active: currentView === 'chat' }"
        @click="currentView = 'chat'"
      >
        💬 Chat
      </button>
      <button
        :class="{ active: currentView === 'terminal' }"
        @click="currentView = 'terminal'"
      >
        💻 Terminal
      </button>
    </div>

    <!-- Content Area -->
    <div v-show="currentView === 'chat'" class="chat-view">
      <SessionChat :session-id="sessionId" />
    </div>

    <div v-show="currentView === 'terminal'" class="terminal-view">
      <SessionTerminal :session-id="sessionId" />
    </div>
  </div>
</template>

<script setup lang="ts">
const currentView = ref<'chat' | 'terminal'>('chat')

// Persist view preference
watch(currentView, (view) => {
  localStorage.setItem(`session-${sessionId}-view`, view)
})

onMounted(() => {
  const savedView = localStorage.getItem(`session-${sessionId}-view`)
  if (savedView === 'terminal') {
    currentView.value = 'terminal'
  }
})
</script>
```

### 4. Configuration

Add to `~/.claude/analytics/config.json`:

```json
{
  "terminal": {
    "enabled": true,
    "default_shell": "/bin/zsh",
    "idle_timeout_minutes": 30,
    "max_concurrent_terminals": 5,
    "allowed_shells": ["/bin/bash", "/bin/zsh", "/bin/sh"],
    "restricted_mode": false,
    "command_history_enabled": true,
    "max_output_buffer_kb": 1024,
    "environment": {
      "TERM": "xterm-256color",
      "COLORTERM": "truecolor",
      "LANG": "en_US.UTF-8"
    }
  }
}
```

**Critical:** The `TERM=xterm-256color` environment variable is required for vim, tmux, htop, and other TUI applications to function correctly.

## 🎮 TUI Application Support

One of the key requirements for this terminal integration is **full support for TUI (Text User Interface) applications** like vim, nano, htop, tmux, and others.

### Why TUI Support Matters

Developers need to run interactive editors and tools in their project directory:
- **vim/neovim** - Edit files directly in the terminal
- **nano** - Simple text editing
- **htop** - Monitor system resources
- **tmux/screen** - Terminal multiplexing
- **git** interactive commands (git add -p, git rebase -i)
- **docker** interactive commands (docker exec -it)
- **Node.js REPL**, **Python REPL**, **psql** - Interactive interpreters

### Technical Requirements for TUI Support

#### 1. Proper TERM Environment Variable
```go
// PTY spawning must set TERM correctly
env := []string{
    "TERM=xterm-256color",   // Critical for vim color schemes
    "COLORTERM=truecolor",   // Enable true color support
    "LANG=en_US.UTF-8",      // UTF-8 support for Unicode chars
}
```

#### 2. PTY with Proper Dimensions
```go
// Create PTY with correct initial size
pty, err := pty.StartWithSize(cmd, &pty.Winsize{
    Rows: 24,
    Cols: 80,
})

// Handle resize events from frontend
func (m *PTYManager) ResizeTerminal(terminalID string, rows, cols int) error {
    return unix.IoctlSetWinsize(int(pty.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
        Row: uint16(rows),
        Col: uint16(cols),
    })
}
```

#### 3. xterm.js Configuration
```typescript
// Frontend terminal must support xterm-256color capabilities
const terminal = new Terminal({
    scrollback: 10000,         // Vim needs scrollback for history
    cursorBlink: true,          // Vim cursor visibility
    cursorStyle: 'block',       // Vim-style block cursor
    allowTransparency: false,   // Solid background for vim
    convertEol: true,           // Handle different line endings
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#ffffff',
        selection: '#264f78'
    }
})

// WebGL addon for 60 FPS rendering (critical for vim responsiveness)
terminal.loadAddon(new WebglAddon())
```

#### 4. Keyboard Event Handling
```typescript
// Ensure all key combinations reach vim
terminal.attachCustomKeyEventHandler((event: KeyboardEvent) => {
    // Let xterm.js handle all keys (including Ctrl+C, Ctrl+D, etc.)
    return true
})
```

### Tested TUI Applications

The terminal integration **MUST** support these applications out of the box:

| Application | Test Case | Expected Behavior |
|-------------|-----------|-------------------|
| **vim** | Open file, edit, save (`:wq`) | Full editing with syntax highlighting |
| **nano** | Open file, edit, save (Ctrl+X) | Functional editor with menu |
| **htop** | Launch htop | CPU/Memory bars render correctly |
| **tmux** | Create session, split panes | Panes render and resize properly |
| **git add -p** | Interactive staging | Diff view and prompts work |
| **python** | Interactive REPL | Input/output streams correctly |
| **node** | Node.js REPL | JavaScript execution works |
| **less** | Page through large file | Scrolling and search work |
| **man** | View manual pages | Formatted output displays |

### vim-Specific Considerations

**Color Schemes:**
- vim relies on `TERM=xterm-256color` for 256-color support
- xterm.js must properly render ANSI escape codes
- Test with popular color schemes: gruvbox, solarized, monokai

**Input Handling:**
- All vim key combinations must work: `gg`, `dd`, `yy`, `p`, `/`, `:`, etc.
- Arrow keys, Home, End, Page Up/Down
- Ctrl combinations: Ctrl+F, Ctrl+B, Ctrl+U, Ctrl+D
- Visual mode selection with `v`, `V`, Ctrl+V

**Performance:**
- Large file editing (1MB+) should be responsive
- Syntax highlighting should not lag
- Scrolling should be smooth (60 FPS with WebGL addon)

**Exit Handling:**
- Proper handling of `:q`, `:wq`, `:q!`
- Ctrl+C should send interrupt signal, not close terminal

### Testing Checklist for TUI Support

```bash
# Test vim
echo "Testing vim..." > test.txt
vim test.txt
# [Press 'i', type "Hello", press Esc, type ':wq']

# Test nano
nano test.txt
# [Edit, press Ctrl+X, save]

# Test htop
htop
# [Verify CPU bars, press 'q' to quit]

# Test tmux
tmux
# [Ctrl+B then %, verify split, Ctrl+B then d to detach]

# Test interactive git
git add -p
# [Verify diff view, stage hunks]

# Test Python REPL
python3
>>> print("Hello from Python")
>>> exit()

# Test less
less /var/log/system.log
# [Scroll, search, quit with 'q']
```

### Troubleshooting TUI Issues

**Problem:** vim shows `E558: Terminal entry not found in terminfo`
- **Solution:** Ensure `TERM=xterm-256color` is set in PTY environment

**Problem:** vim colors are wrong/missing
- **Solution:** Check `COLORTERM=truecolor` environment variable

**Problem:** Arrow keys insert `^[[A` instead of moving cursor
- **Solution:** Verify xterm.js is properly converting key events

**Problem:** vim is slow/laggy with large files
- **Solution:** Enable WebGL addon for GPU-accelerated rendering

**Problem:** Ctrl+C closes terminal instead of sending interrupt
- **Solution:** Ensure `attachCustomKeyEventHandler` returns `true`

## 🔒 Security Considerations

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **Command Injection** | PTY provides raw input/output; no command parsing required |
| **Directory Traversal** | Working directory validated against session's project path |
| **Resource Exhaustion** | Rate limiting, max concurrent terminals, idle timeout |
| **Session Hijacking** | Same authentication as agent sessions (API key + session ID) |
| **Privilege Escalation** | Terminal runs with Wee process permissions (already limited) |
| **Data Exfiltration** | Same file access as Bash tool (already permitted) |
| **Zombie Processes** | Process group cleanup on terminal close |
| **Output Flooding** | Buffer size limits, rate limiting |

### Security Measures

✅ **Authentication:** Terminal endpoints require same auth as agent sessions
✅ **Session Validation:** Verify session ownership before spawning terminal
✅ **Rate Limiting:** Max 5 terminal creations per minute per session
✅ **Idle Timeout:** Auto-kill terminals after 30 minutes of inactivity
✅ **Resource Limits:** Max 5 concurrent terminals per session
✅ **Command Logging:** All commands stored in `terminal_history` for audit trail
✅ **Environment Isolation:** Terminal inherits minimal environment variables

### Optional Security Modes

**Restricted Mode** (future enhancement):
- Whitelist of allowed commands
- Prevent `cd` outside project directory
- Block network commands (curl, wget, etc.)

**Read-Only Mode** (future enhancement):
- View-only terminal (no input)
- Useful for reviewing command output

## 📊 Performance & Scalability

### Resource Consumption

| Component | Per Terminal | 100 Terminals |
|-----------|-------------|---------------|
| Memory | ~15KB | ~1.5MB |
| File Descriptors | 2 (PTY master/slave) | 200 |
| Goroutines | 2-3 | 200-300 |
| WebSocket Connections | 1 | 100 |

### Scalability Limits

- **System `ulimit` on file descriptors:** Typically 1024-4096
- **Maximum recommended concurrent terminals:** 50 per Wee instance
- **Maximum theoretical terminals:** 500-1000 (system dependent)

### Performance Optimizations

- **Lazy Loading:** Terminal only spawned when user toggles to terminal view
- **Connection Pooling:** Reuse WebSocket connections where possible
- **Output Buffering:** Batch terminal output to reduce WebSocket messages
- **Idle Cleanup:** Background goroutine to kill idle terminals every 5 minutes

## 🧪 Testing Strategy

### Unit Tests

```go
// internal/server/terminal/pty_manager_test.go
func TestPTYManager_SpawnTerminal(t *testing.T)
func TestPTYManager_ResizeTerminal(t *testing.T)
func TestPTYManager_KillTerminal(t *testing.T)
func TestPTYManager_CleanupIdleTerminals(t *testing.T)

// internal/server/terminal/terminal_session_test.go
func TestTerminalSession_Lifecycle(t *testing.T)
func TestTerminalSession_InvalidWorkingDir(t *testing.T)
```

### Integration Tests

```go
func TestTerminalWebSocket_EndToEnd(t *testing.T)
func TestTerminalWebSocket_Reconnection(t *testing.T)
func TestTerminalWebSocket_ConcurrentSessions(t *testing.T)
```

### E2E Tests (Playwright/Cypress)

```typescript
test('create session, toggle to terminal, run command, see output', async () => {
  // Create session
  await page.click('[data-test="new-session"]')

  // Toggle to terminal
  await page.click('[data-test="terminal-toggle"]')

  // Wait for terminal to load
  await page.waitForSelector('.xterm')

  // Type command
  await page.keyboard.type('echo "Hello from Wee terminal"')
  await page.keyboard.press('Enter')

  // Verify output
  await expect(page.locator('.xterm')).toContainText('Hello from Wee terminal')
})

test('run vim in terminal', async () => {
  await page.click('[data-test="new-session"]')
  await page.click('[data-test="terminal-toggle"]')
  await page.waitForSelector('.xterm')

  // Launch vim
  await page.keyboard.type('vim test.txt')
  await page.keyboard.press('Enter')

  // Wait for vim to load (look for vim UI elements)
  await page.waitForTimeout(1000)

  // Enter insert mode
  await page.keyboard.press('i')

  // Type text
  await page.keyboard.type('Hello from vim in Wee!')

  // Exit insert mode and save
  await page.keyboard.press('Escape')
  await page.keyboard.type(':wq')
  await page.keyboard.press('Enter')

  // Verify file was created
  await page.keyboard.type('cat test.txt')
  await page.keyboard.press('Enter')
  await expect(page.locator('.xterm')).toContainText('Hello from vim in Wee!')
})

test('run htop in terminal', async () => {
  await page.click('[data-test="new-session"]')
  await page.click('[data-test="terminal-toggle"]')
  await page.waitForSelector('.xterm')

  // Launch htop (if available)
  await page.keyboard.type('htop')
  await page.keyboard.press('Enter')

  // Wait for htop to render
  await page.waitForTimeout(2000)

  // Verify htop UI is visible (CPU/Memory bars)
  await expect(page.locator('.xterm')).toContainText('CPU')

  // Quit htop
  await page.keyboard.press('q')
})
```

### Security Tests

```go
func TestTerminal_SessionHijackingPrevention(t *testing.T)
func TestTerminal_ResourceExhaustionPrevention(t *testing.T)
func TestTerminal_RateLimiting(t *testing.T)
```

## 📱 Mobile Considerations (iOS)

### Phase 1: Desktop Only
- Focus on desktop browsers (Chrome, Firefox, Safari, Edge)
- Terminal UI optimized for keyboard/mouse input

### Phase 2: iOS Read-Only Terminal
- View terminal output on iOS app
- No interactive input (challenging on touch keyboards)
- Scroll and zoom support

### Phase 3: iOS Interactive Terminal
- Custom toolbar with common keys (Tab, Ctrl-C, arrows, etc.)
- Gesture-based navigation
- Clipboard integration

**Recommendation:** Start with desktop, add iOS read-only in Phase 2.

## 📝 Implementation Phases

### Phase 1: Backend Foundation (Est: 1-2 days)

- [ ] Create `internal/server/terminal/` package
- [ ] Implement `pty_manager.go` with `creack/pty`
- [ ] Implement `terminal_session.go` lifecycle management
- [ ] Implement `terminal_handler.go` WebSocket endpoint
- [ ] Add database schema and migrations
- [ ] Add repository methods for terminal sessions
- [ ] Create API routes and handlers
- [ ] Write unit tests for PTY manager
- [ ] Write integration tests for WebSocket communication

**Deliverable:** Functional backend API that can spawn terminals and stream I/O.

### Phase 2: Frontend Integration (Est: 1-2 days)

- [ ] Install xterm.js dependencies (`npm install xterm xterm-addon-fit xterm-addon-web-links`)
- [ ] Create `SessionTerminal.vue` component
- [ ] Implement `useTerminalWebSocket.ts` composable
- [ ] Add chat/terminal toggle UI to `agents.vue`
- [ ] Implement localStorage persistence for view preference
- [ ] Add error handling and reconnection logic
- [ ] Style terminal component to match Wee theme
- [ ] Add keyboard shortcut (Ctrl+` to toggle)

**Deliverable:** Fully functional terminal UI integrated into agents page.

### Phase 3: Polish & Advanced Features (Est: 1-2 days)

- [ ] Terminal resize support (auto-fit on window resize)
- [ ] Command history extraction and analytics
- [ ] Copy/paste support (clipboard integration)
- [ ] Terminal theme customization (match dark/light mode)
- [ ] Session persistence (reconnect to existing terminal on page refresh)
- [ ] Terminal output search functionality
- [ ] Metrics dashboard (terminal usage, popular commands)
- [ ] Documentation and user guide

**Deliverable:** Production-ready feature with polish and docs.

## 🎨 UI/UX Design

### Toggle Button

```
┌──────────────────────────────────────────────┐
│  Session: abc123  [💬 Chat] [💻 Terminal]    │
├──────────────────────────────────────────────┤
│                                              │
│  [Chat/Terminal content area]               │
│                                              │
└──────────────────────────────────────────────┘
```

### Terminal View

```
┌──────────────────────────────────────────────┐
│  Session: abc123  [💬 Chat] [💻 Terminal]    │
├──────────────────────────────────────────────┤
│  user@host:~/project$ ls -la                 │
│  total 48                                    │
│  drwxr-xr-x  12 user  staff   384 Jan 1 12:0│
│  drwxr-xr-x   3 user  staff    96 Jan 1 12:0│
│  -rw-r--r--   1 user  staff  1234 Jan 1 12:0│
│  user@host:~/project$ █                      │
└──────────────────────────────────────────────┘
```

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl + `` | Toggle chat/terminal view |
| `Ctrl + Shift + C` | Copy (in terminal) |
| `Ctrl + Shift + V` | Paste (in terminal) |
| `Ctrl + L` | Clear terminal (native shell) |

## 📈 Success Metrics

- **Adoption:** 30%+ of sessions use terminal within first month
- **Performance:** Terminal spawns in <200ms
- **Reliability:** <1% terminal spawn failures
- **Resource Usage:** <50MB memory with 20 concurrent terminals
- **User Satisfaction:** Positive feedback on feature usefulness

## 🔗 Related Issues / PRs

- None (this is the foundational issue)

## 📚 Technical Dependencies

### Go Libraries

```go
github.com/creack/pty v1.1.21  // PTY creation and management
```

### Frontend Libraries

```json
{
  "xterm": "^5.3.0",           // Terminal emulator
  "xterm-addon-fit": "^0.8.0", // Auto-resize addon
  "xterm-addon-web-links": "^0.9.0", // Clickable links
  "xterm-addon-webgl": "^0.16.0" // GPU-accelerated rendering for vim/htop
}
```

## ❓ Open Questions

1. **Shell Selection:**
   - Auto-detect user's shell from `$SHELL` environment variable?
   - Allow per-session shell override?
   - **Proposed:** Default to `$SHELL`, allow override in config

2. **Terminal Persistence:**
   - Should terminals survive server restarts?
   - **Proposed:** No (too complex for Phase 1), notify user gracefully on reconnection

3. **Command History:**
   - Parse bash history and store in database?
   - **Proposed:** Yes, for analytics and search features

4. **Multiple Terminals:**
   - Support multiple terminal tabs per session?
   - **Proposed:** Phase 3 feature, start with single terminal

5. **Terminal Recording:**
   - Record full terminal session for playback (like asciinema)?
   - **Proposed:** Phase 3 feature

6. **Environment Variables:**
   - Inherit from server process or custom per terminal?
   - **Proposed:** Inherit server env + allow custom via config

## 🏷️ Labels

- `enhancement`
- `feature`
- `frontend`
- `backend`
- `terminal`
- `good-first-issue` (for sub-tasks)

## 📝 Acceptance Criteria

### Must Have (Phase 1-2)

- [ ] User can toggle between chat and terminal view in an agent session
- [ ] Terminal pwd is set to the session's working directory
- [ ] Terminal supports full xterm.js features (colors, cursor movement, etc.)
- [ ] Terminal WebSocket communication is authenticated and secure
- [ ] Terminal sessions are persisted in database
- [ ] Idle terminals are auto-killed after timeout
- [ ] Frontend handles WebSocket disconnection gracefully
- [ ] Error messages are user-friendly
- [ ] Documentation is complete

### Should Have (Phase 3)

- [ ] Terminal resize support
- [ ] Command history is captured and searchable
- [ ] Copy/paste support via clipboard
- [ ] Terminal themes match Wee dark/light mode
- [ ] Session reconnection works after page refresh
- [ ] Keyboard shortcut to toggle view (Ctrl+`)

### Could Have (Future)

- [ ] Multiple terminal tabs per session
- [ ] Terminal output search
- [ ] Terminal session recording and playback
- [ ] iOS read-only terminal view
- [ ] Restricted shell mode (command whitelist)

## 🚀 Getting Started

### For Developers

1. **Backend First:**
   - Start with `internal/server/terminal/pty_manager.go`
   - Implement basic PTY spawning with `creack/pty`
   - Add WebSocket handler in `terminal_handler.go`
   - Test with `wscat` or Postman WebSocket client

2. **Frontend Second:**
   - Install xterm.js: `cd internal/server/frontend && npm install xterm xterm-addon-fit`
   - Create basic `SessionTerminal.vue` component
   - Test terminal rendering with hardcoded output
   - Integrate WebSocket connection

3. **Integration:**
   - Connect frontend WebSocket to backend handler
   - Test end-to-end: spawn terminal, send input, receive output
   - Add error handling and edge cases

### Testing Locally

```bash
# Start Wee server
./wee --analytics

# Open browser
open https://localhost:3333

# Create a new session
# Toggle to terminal view
# Run: echo "Hello from integrated terminal!"
```

## 📞 Questions?

For questions or discussions about this feature, please:
- Comment on this issue
- Join the Wee Discord (if available)
- Reach out to the maintainers

---

**Estimated Total Effort:** 4-6 days (1 developer)
**Priority:** Medium
**Complexity:** High
**Impact:** High
