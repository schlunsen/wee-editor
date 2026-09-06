# SDK YOLO Mode Implementation Plan

## Overview

Add support for `--dangerously-skip-permissions` and `--allow-dangerously-skip-permissions` CLI flags to the `claude-agent-sdk-go` SDK.

**Project Location**: `~/projects/claude-agent-sdk-go`

**Purpose**: Enable YOLO Mode (permission bypass) for agent sessions by adding the necessary flags to the Claude CLI subprocess invocation.

## Background

The Claude CLI supports two flags for bypassing permissions:
- `--allow-dangerously-skip-permissions`: Enables permission bypass as an option
- `--dangerously-skip-permissions`: Actually bypasses all permissions (requires the allow flag to be set first)

These flags need to be added to the SDK so that Wee can use them via the `buildSDKOptions` method.

## Files to Modify

### 1. `types/options.go`

**Location**: `~/projects/claude-agent-sdk-go/types/options.go`

**Changes Needed**:

#### A. Add Fields to ClaudeAgentOptions Struct

Find the `ClaudeAgentOptions` struct (around line 78) and add these fields:

```go
type ClaudeAgentOptions struct {
	// ... existing fields ...

	// Permission bypass configuration (use with caution - only for sandboxed environments)
	// These flags disable ALL permission checks, allowing Claude to execute any tool without approval.
	//
	// DangerouslySkipPermissions: Actually bypass all permissions (requires AllowDangerouslySkipPermissions)
	// AllowDangerouslySkipPermissions: Enable permission bypass as an option
	//
	// Security Warning: Only use in isolated environments with no internet access.
	DangerouslySkipPermissions      bool `json:"dangerously_skip_permissions,omitempty"`
	AllowDangerouslySkipPermissions bool `json:"allow_dangerously_skip_permissions,omitempty"`
}
```

**Location Hint**: Add these fields after the existing permission-related fields (like `PermissionMode`, `CanUseTool`, etc.)

#### B. Add Builder Methods

Find the builder methods section (around line 356) and add:

```go
// WithDangerouslySkipPermissions bypasses all permission checks.
// This is DANGEROUS and should only be used in sandboxed environments.
// Requires AllowDangerouslySkipPermissions to be enabled first.
//
// Security Warning: This disables ALL safety checks. Only use in isolated environments
// with no internet access and no sensitive data.
func (o *ClaudeAgentOptions) WithDangerouslySkipPermissions(skip bool) *ClaudeAgentOptions {
	o.DangerouslySkipPermissions = skip
	return o
}

// WithAllowDangerouslySkipPermissions enables the option to bypass permissions.
// This must be set to true before DangerouslySkipPermissions can be used.
//
// This is the "safety switch" that allows the dangerous flag to work.
func (o *ClaudeAgentOptions) WithAllowDangerouslySkipPermissions(allow bool) *ClaudeAgentOptions {
	o.AllowDangerouslySkipPermissions = allow
	return o
}
```

**Location Hint**: Add these methods alongside other `With*` builder methods like `WithModel()`, `WithPermissionMode()`, etc.

### 2. `internal/transport/subprocess_cli.go`

**Location**: `~/projects/claude-agent-sdk-go/internal/transport/subprocess_cli.go`

**Changes Needed**:

#### Update CLI Args Building in `Connect` Method

Find the `Connect` method where CLI arguments are constructed (look for `args := []string{}`).

**Current Code Structure** (approximate):
```go
func (t *SubprocessCLITransport) Connect(ctx context.Context) error {
	// ... existing code ...

	args := []string{}

	// Add --resume flag if set
	if t.options != nil && t.options.Resume != nil {
		args = append(args, "--resume", *t.options.Resume)
	}

	// Add --model flag
	if t.options != nil && t.options.Model != "" {
		args = append(args, "--model", t.options.Model)
	}

	// ... other flags ...

	// Start subprocess
	// ...
}
```

**Add This Code** (after existing flag handling, before subprocess start):

```go
// Add permission bypass flags if enabled
if t.options != nil {
	// Must set allow flag first (acts as safety switch)
	if t.options.AllowDangerouslySkipPermissions {
		args = append(args, "--allow-dangerously-skip-permissions")

		// Only add skip flag if allow flag is also set
		if t.options.DangerouslySkipPermissions {
			args = append(args, "--dangerously-skip-permissions")
		}
	}
}
```

**Location Hint**: Add this code block:
1. After the `--resume` flag handling
2. After the `--model` flag handling
3. Before other permission-related flags (if any)
4. Before the subprocess execution (`cmd.Start()`)

**Important**: The order matters! `--allow-dangerously-skip-permissions` must be added before `--dangerously-skip-permissions`.

## Testing

After implementing the changes, test with this code:

```go
package main

import (
	"context"
	"log"

	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
	ctx := context.Background()

	// Create options with YOLO mode enabled
	opts := types.NewClaudeAgentOptions().
		WithModel("claude-sonnet-4-20250514").
		WithAllowDangerouslySkipPermissions(true).
		WithDangerouslySkipPermissions(true)

	// Create client
	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close(ctx)

	// Test with a command that normally requires permission
	if err := client.Query(ctx, "Read the file /tmp/test.txt"); err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	// Should execute without permission prompt
	messages := client.ReceiveResponse(ctx)
	for msg := range messages {
		log.Printf("Received: %v", msg)
	}
}
```

**Expected Behavior**:
- With flags enabled: No permission prompts appear, tools execute immediately
- With flags disabled: Normal permission prompts appear

## Verification Steps

1. **Code Inspection**:
   - Verify fields added to `ClaudeAgentOptions` struct
   - Verify builder methods return `*ClaudeAgentOptions` (for chaining)
   - Verify CLI args are appended in correct order

2. **Build Test**:
   ```bash
   cd ~/projects/claude-agent-sdk-go
   go build ./...
   ```

3. **Unit Test** (if test files exist):
   ```bash
   go test ./...
   ```

4. **Integration Test**:
   - Create a test program (see Testing section above)
   - Run with flags enabled
   - Verify no permission prompts appear
   - Run with flags disabled
   - Verify permission prompts appear

5. **Wee Integration**:
   - Update Wee's `go.mod` to use the new SDK version
   - Uncomment the YOLO mode code in `session_manager.go`
   - Test YOLO mode toggle in Wee UI
   - Verify session reload works correctly

## Security Considerations

⚠️ **Critical Safety Notes**:

1. **Default Values**: Both flags default to `false` (safe)
2. **Two-Step Enable**: Must set `AllowDangerouslySkipPermissions` before `DangerouslySkipPermissions` works
3. **Documentation**: Add warnings in godoc comments
4. **Flag Order**: `--allow-dangerously-skip-permissions` must come before `--dangerously-skip-permissions`
5. **Use Cases**: Only recommend for:
   - Sandboxed Docker containers
   - Isolated VMs with no network access
   - Automated testing in CI/CD
   - Development environments with no sensitive data

## Success Criteria

✅ Fields added to `ClaudeAgentOptions` struct
✅ Builder methods implemented with correct return types
✅ CLI flags appended in correct order in `subprocess_cli.go`
✅ Code compiles without errors
✅ Test program runs without permission prompts when flags enabled
✅ Permission prompts appear when flags disabled
✅ Wee integration works (after uncommenting code)

## Post-Implementation

After SDK changes are complete:

1. **Tag SDK Release**:
   ```bash
   cd ~/projects/claude-agent-sdk-go
   git tag v0.x.x
   git push origin v0.x.x
   ```

2. **Update Wee**:
   ```bash
   cd ~/projects/wee-editor
   go get github.com/schlunsen/claude-agent-sdk-go@v0.x.x
   ```

3. **Uncomment YOLO Mode Code** in Wee's `session_manager.go`:
   - Remove TODO comments
   - Uncomment SDK method calls
   - Remove temporary workaround (permission mode fallback)

4. **Test Wee**:
   - Build Wee: `go build ./cmd/wee`
   - Start server: `./wee --analytics`
   - Test YOLO mode toggle in UI
   - Verify permissions are actually bypassed

## Questions?

If you encounter issues:

1. **Build Errors**: Check that struct fields and method signatures match exactly
2. **Runtime Errors**: Verify flag order in `subprocess_cli.go`
3. **Flags Not Working**: Check that both `Allow` and `Skip` flags are set
4. **Permission Prompts Still Appear**: Verify flags are being passed to Claude CLI subprocess

## Example Usage (After Implementation)

```go
// Enable YOLO Mode
opts := types.NewClaudeAgentOptions().
    WithModel("claude-sonnet-4-20250514").
    WithAllowDangerouslySkipPermissions(true).  // Safety switch
    WithDangerouslySkipPermissions(true)         // Actual bypass

// Disable YOLO Mode (default)
opts := types.NewClaudeAgentOptions().
    WithModel("claude-sonnet-4-20250514")
    // Flags default to false
```

## Related Files

- Wee Implementation: `~/projects/wee-editor/internal/server/agents/session_manager.go`
- Wee Plan: `~/projects/wee-editor/docs/YOLO_MODE_IMPLEMENTATION_PLAN.md`
- This Plan: `~/projects/wee-editor/docs/SDK_YOLO_MODE_IMPLEMENTATION_PLAN.md`
