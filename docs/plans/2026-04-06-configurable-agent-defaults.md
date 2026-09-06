# Configurable Agent Defaults (Permission Mode, Provider, Model)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make default permission mode, provider, and model configurable via settings.json and environment variables, so wee.cat sandbox images boot with custom provider (qwen3.5) in YOLO mode.

**Architecture:** Add 3 new fields to `AgentSettings` in the server config. Wire env var overrides following the existing `WEE_` prefix pattern. Apply defaults at session creation time (lifecycle.go) when the frontend doesn't provide explicit options. Update the Django provisioner to pass the new env vars to sandbox containers.

**Tech Stack:** Go (server config, session lifecycle), Python (Django provisioner)

---

### Task 1: Add Config Fields and Env Var Support

**Files:**
- Modify: `internal/server/config.go`

**Step 1: Add fields to AgentSettings struct**

In `internal/server/config.go`, add 3 new fields to `AgentSettings`:

```go
// AgentSettings holds agent configuration
type AgentSettings struct {
	Model                 string            `json:"model"`
	MaxConcurrentSessions int               `json:"max_concurrent_sessions"`
	SessionRetentionDays  int               `json:"session_retention_days"`
	CleanupEnabled        bool              `json:"cleanup_enabled"`
	CleanupIntervalHours  int               `json:"cleanup_interval_hours"`
	Worktree              WorktreeSettings  `json:"worktree"`
	DefaultProvider       string            `json:"default_provider,omitempty"`    // Provider ID: "claude", "custom", "deepseek", "glm", "kimi"
	DefaultModel          string            `json:"default_model,omitempty"`       // Model name override (e.g., "qwen3.5")
	PermissionMode        string            `json:"permission_mode,omitempty"`     // "default", "acceptEdits", "bypassPermissions", "yolo"
}
```

**Step 2: Add `isYOLOMode` helper function**

Add after the existing `parseBool` function at the bottom of `config.go`:

```go
// isYOLOMode returns true if the given permission mode string means bypass all permissions.
// Accepts both "yolo" (friendly alias) and "bypassPermissions" (Go constant name).
func isYOLOMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "yolo" || m == "bypasspermissions"
}
```

**Step 3: Add env var overrides in `LoadConfigFromEnvironment`**

Add these lines at the end of `LoadConfigFromEnvironment`, just before the `return cm.LoadOAuthSecretsFromEnv(config)` line:

```go
	// Agent defaults
	if defaultProvider := os.Getenv("WEE_AGENT_DEFAULT_PROVIDER"); defaultProvider != "" {
		config.Agent.DefaultProvider = defaultProvider
	}
	if defaultModel := os.Getenv("WEE_AGENT_DEFAULT_MODEL"); defaultModel != "" {
		config.Agent.DefaultModel = defaultModel
	}
	if permissionMode := os.Getenv("WEE_AGENT_PERMISSION_MODE"); permissionMode != "" {
		config.Agent.PermissionMode = permissionMode
	}
```

**Step 4: Verify defaults in `getDefaultConfig`**

The zero values for strings (`""`) are correct defaults — empty means "no override". No change needed to `getDefaultConfig()`.

**Step 5: Commit**

```bash
git add internal/server/config.go
git commit -m "feat: add default provider, model, and permission mode to AgentSettings config

Adds WEE_AGENT_DEFAULT_PROVIDER, WEE_AGENT_DEFAULT_MODEL, and
WEE_AGENT_PERMISSION_MODE env var support. Accepts both 'yolo' and
'bypassPermissions' as valid permission mode values."
```

---

### Task 2: Apply Config Defaults at Session Creation

**Files:**
- Modify: `internal/server/agents/lifecycle.go`

**Step 1: Add config default injection in `CreateSession`**

In `lifecycle.go`, the `CreateSession` method receives `options SessionOptions` from the frontend. We need to fill in defaults from config when the frontend doesn't provide explicit values.

Add a new method to `SessionManager` and call it at the top of `CreateSession`. Insert this helper method before `CreateSession`:

```go
// applyConfigDefaults fills in SessionOptions from server config when not explicitly set by the client.
func (sm *SessionManager) applyConfigDefaults(options *SessionOptions) {
	cfg := sm.config

	// Default provider
	if options.Provider == nil && cfg.Agent.DefaultProvider != "" {
		options.Provider = &cfg.Agent.DefaultProvider
	}

	// Default model
	if options.Model == nil && cfg.Agent.DefaultModel != "" {
		options.Model = &cfg.Agent.DefaultModel
	}

	// Default permission mode (YOLO)
	if options.DangerouslySkipPermissions == nil && isYOLOMode(cfg.Agent.PermissionMode) {
		t := true
		options.DangerouslySkipPermissions = &t
		options.AllowDangerouslySkipPermissions = &t
	}
}
```

Note: `isYOLOMode` is in the `server` package but we need it in the `agents` package. We have two options:
- Move it to a shared location
- Duplicate the small helper in the agents package

Since it's a 3-line function, duplicate it in `lifecycle.go`:

```go
// isYOLOMode returns true if the permission mode means bypass all permissions.
func isYOLOMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "yolo" || m == "bypasspermissions"
}
```

**Step 2: Call `applyConfigDefaults` in `CreateSession`**

In the `CreateSession` function, add the call right after the debug log line:

```go
func (sm *SessionManager) CreateSession(sessionID uuid.UUID, options SessionOptions) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	logging.Debug("CreateSession called for session: %s", sessionID)

	// Apply server config defaults for any options not explicitly set by the client
	sm.applyConfigDefaults(&options)

	// Check if session already exists in memory ...
```

**Step 3: Ensure `strings` is imported**

Check that `"strings"` is in the import block of `lifecycle.go`. It's already imported — confirmed.

**Step 4: The `sm.config` field — verify access**

The `SessionManager` already has access to config via `sm.config` (type `*Config` from `internal/agents/config.go`). However, the `agents.Config` struct doesn't have the new `Agent.PermissionMode` etc. fields — those are on the `server.Config.Agent` (type `AgentSettings`).

We need to check how `SessionManager` accesses config. Let's look at what `sm.config` actually is:

The `SessionManager` uses `sm.config` which is `*agents.Config` (from `internal/agents/config.go`). This is a different struct than `server.Config`. We need to either:

**Option A (chosen — minimal change):** Add the 3 new fields to `agents.Config` and populate them from env vars in `DefaultConfig()`.

In `internal/agents/config.go`, add to the `Config` struct:

```go
type Config struct {
	Host                  string
	Port                  int
	LogLevel              string
	Model                 string
	APIKey                string
	MaxConcurrentSessions int
	ServerDir             string
	PIDFile               string
	LogFile               string
	DefaultProvider       string
	DefaultModel          string
	PermissionMode        string
}
```

And in `DefaultConfig()`, populate them:

```go
	return &Config{
		// ... existing fields ...
		DefaultProvider: getEnvOrDefault("WEE_AGENT_DEFAULT_PROVIDER", ""),
		DefaultModel:    getEnvOrDefault("WEE_AGENT_DEFAULT_MODEL", ""),
		PermissionMode:  getEnvOrDefault("WEE_AGENT_PERMISSION_MODE", "default"),
	}
```

Then `applyConfigDefaults` in `lifecycle.go` references `sm.config.DefaultProvider`, `sm.config.DefaultModel`, `sm.config.PermissionMode` directly (no nested `Agent` field).

Updated `applyConfigDefaults`:

```go
func (sm *SessionManager) applyConfigDefaults(options *SessionOptions) {
	// Default provider
	if options.Provider == nil && sm.config.DefaultProvider != "" {
		options.Provider = &sm.config.DefaultProvider
	}

	// Default model
	if options.Model == nil && sm.config.DefaultModel != "" {
		options.Model = &sm.config.DefaultModel
	}

	// Default permission mode (YOLO)
	if options.DangerouslySkipPermissions == nil && isYOLOMode(sm.config.PermissionMode) {
		t := true
		options.DangerouslySkipPermissions = &t
		options.AllowDangerouslySkipPermissions = &t
	}
}
```

**Step 5: Commit**

```bash
git add internal/agents/config.go internal/server/agents/lifecycle.go
git commit -m "feat: apply config defaults for provider, model, permission at session creation

Sessions inherit default provider, model, and permission mode from
server config/env vars when not explicitly set by the frontend."
```

---

### Task 3: Set Custom Provider as Current When Default Provider Configured

**Files:**
- Modify: `internal/server/setup_database.go`

**Step 1: Update `seedCustomProviderFromEnv` to respect `WEE_AGENT_DEFAULT_PROVIDER`**

When `WEE_AGENT_DEFAULT_PROVIDER=custom` is set, the seeded custom provider should be marked as the current/active provider so it's used immediately.

In `setup_database.go`, update the `seedCustomProviderFromEnv` function. Change the `IsCurrent` line:

```go
	// If WEE_AGENT_DEFAULT_PROVIDER is set to "custom", mark this as the active provider
	defaultProvider := os.Getenv("WEE_AGENT_DEFAULT_PROVIDER")
	isCurrent := strings.EqualFold(defaultProvider, "custom")

	provider := &database.ProviderConfig{
		ProviderID: "custom",
		Name:       customName,
		APIKey:     &customKey,
		CustomURL:  &customURL,
		ModelName:  &customModel,
		IsCurrent:  isCurrent,
	}
```

Also add `"strings"` to the import block if not already there.

**Step 2: Commit**

```bash
git add internal/server/setup_database.go
git commit -m "feat: auto-activate custom provider when WEE_AGENT_DEFAULT_PROVIDER=custom

When the default provider env var points to 'custom', the seeded
provider is set as current so agent sessions use it immediately."
```

---

### Task 4: Update Django Provisioner to Pass New Env Vars

**Files:**
- Modify: `wee.cat/backend/sandboxes/provisioner.py`

**Step 1: Add new env var builder function**

Add a new helper function after `_custom_provider_env_args()`:

```python
def _agent_defaults_env_args() -> list[str]:
    """Build docker -e flags for agent default settings from environment.

    Reads:
      WEE_AGENT_DEFAULT_PROVIDER   - Default provider ID (e.g. "custom")
      WEE_AGENT_DEFAULT_MODEL      - Default model name (e.g. "qwen3.5")
      WEE_AGENT_PERMISSION_MODE    - Default permission mode (e.g. "yolo")

    Returns a list of docker -e flags, only for variables that are set.
    """
    args = []
    for var in ("WEE_AGENT_DEFAULT_PROVIDER", "WEE_AGENT_DEFAULT_MODEL", "WEE_AGENT_PERMISSION_MODE"):
        val = os.environ.get(var, "")
        if val:
            args.extend(["-e", f"{var}={val}"])
    return args
```

**Step 2: Splice into docker run commands**

In `provision_sandbox()` (line ~101), add `*_agent_defaults_env_args()` after `*_custom_provider_env_args()`:

```python
        _run([
            "docker", "run", "-d",
            "--name", container_name,
            "--restart", "unless-stopped",
            "-p", f"127.0.0.1:{port}:3333",
            "-e", f"WEE_SUBDOMAIN={sandbox.subdomain}",
            *_custom_provider_env_args(),
            *_agent_defaults_env_args(),
            WEE_IMAGE,
        ])
```

Do the same in `start_sandbox_container()` (line ~165):

```python
        _run([
            "docker", "run", "-d",
            "--name", container_name,
            "--restart", "unless-stopped",
            "-p", f"127.0.0.1:{port}:3333",
            "-e", f"WEE_SUBDOMAIN={sandbox.subdomain}",
            *_custom_provider_env_args(),
            *_agent_defaults_env_args(),
            WEE_IMAGE,
        ])
```

**Step 3: Commit**

```bash
git add wee.cat/backend/sandboxes/provisioner.py
git commit -m "feat: pass agent default env vars to sandbox containers

Forwards WEE_AGENT_DEFAULT_PROVIDER, WEE_AGENT_DEFAULT_MODEL, and
WEE_AGENT_PERMISSION_MODE to sandbox Docker containers at runtime."
```

---

### Task 5: Update `buildSDKOptions` to Handle "yolo" Alias

**Files:**
- Modify: `internal/server/agents/manager.go`

**Step 1: Update permission mode switch in `buildSDKOptions`**

The existing `buildSDKOptions` already handles `DangerouslySkipPermissions` flags (which we set in `applyConfigDefaults`), so YOLO mode from config is already handled. But we should also handle the case where `PermissionMode` is set to `"yolo"` or `"bypassPermissions"` directly in the options (e.g., from a future API call).

In `buildSDKOptions` (around line 799), update the permission mode switch:

```go
	} else if options.PermissionMode != nil {
		// Use configured permission mode
		switch {
		case isYOLOMode(*options.PermissionMode):
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeBypassPermissions).
				WithAllowDangerouslySkipPermissions(true).
				WithDangerouslySkipPermissions(true)
		case *options.PermissionMode == "allow-all":
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeBypassPermissions)
		case *options.PermissionMode == "read-only":
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeDefault)
		default:
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeDefault)
		}
	}
```

**Step 2: Commit**

```bash
git add internal/server/agents/manager.go
git commit -m "feat: handle 'yolo' alias in buildSDKOptions permission mode switch

Allows PermissionMode='yolo' in session options to activate full
permission bypass, same as 'bypassPermissions'."
```

---

### Task 6: Set Env Vars on wee.cat Server

**Step 1: SSH into wee.cat and add env vars**

Add these to `/srv/weecat/.env.production`:

```bash
WEE_AGENT_DEFAULT_PROVIDER=custom
WEE_AGENT_DEFAULT_MODEL=qwen3.5
WEE_AGENT_PERMISSION_MODE=yolo
```

The `WEE_CUSTOM_PROVIDER_URL` and `WEE_CUSTOM_PROVIDER_KEY` should already be set there. Verify they are.

**Step 2: Restart the backend service**

```bash
sudo systemctl restart weecat-backend
```

**Step 3: Verify by creating a test sandbox**

Create a sandbox and check the container env vars:

```bash
docker inspect wee-sandbox-<subdomain> | grep -A 20 "Env"
```

Confirm `WEE_AGENT_DEFAULT_PROVIDER=custom`, `WEE_AGENT_DEFAULT_MODEL=qwen3.5`, and `WEE_AGENT_PERMISSION_MODE=yolo` are present.

---

### Task 7: Build and Test

**Step 1: Build the Go binary**

```bash
make build
```

Expected: Compiles without errors.

**Step 2: Run existing tests**

```bash
make test
```

Expected: All existing tests pass.

**Step 3: Manual smoke test**

Start the server with env vars to simulate sandbox mode:

```bash
WEE_AGENT_DEFAULT_PROVIDER=custom \
WEE_AGENT_DEFAULT_MODEL=qwen3.5 \
WEE_AGENT_PERMISSION_MODE=yolo \
WEE_CUSTOM_PROVIDER_URL=http://localhost:4001/v1/messages \
WEE_CUSTOM_PROVIDER_KEY=test-key \
./wee --analytics
```

Create a session via WebSocket and verify:
- Session starts with YOLO mode active (no permission prompts)
- Custom provider is marked as current in the provider list
- Model defaults to qwen3.5

**Step 4: Final commit (if any fixups needed)**

```bash
git add -A
git commit -m "fix: address any issues found during testing"
```
