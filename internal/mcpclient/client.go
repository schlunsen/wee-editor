// Package mcpclient implements a Go-native MCP client that communicates with
// MCP servers via stdio (JSON-RPC 2.0). This enables non-Claude models to
// discover and invoke MCP tools without going through the Claude CLI.
//
// This package is intentionally separate from internal/mcp to avoid import cycles.
// internal/mcp contains the MCP server implementation (which imports agents).
// internal/mcpclient contains the MCP client (which agents imports).
//
// Dependency chain: agents → mcpclient (no cycle)
package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// === MCP Protocol Types (duplicated from mcp/types.go to avoid cycles) ===

// MCPRequest is a JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse is a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError is a JSON-RPC 2.0 error.
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema describes the parameters a tool accepts.
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property describes a single tool parameter.
type Property struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// ToolsListResult is the response from tools/list.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams is the request for tools/call.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult is the response from tools/call.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError *bool          `json:"isError,omitempty"`
}

// ContentBlock is a content block in tool results.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// LogFunc is a logging callback. If nil, stderr from the MCP server is discarded.
type LogFunc func(format string, args ...interface{})

// === MCP Client ===

// Client is a Go-native client for communicating with MCP servers via stdio.
// It manages the subprocess lifecycle and provides typed methods for MCP operations.
//
// Thread Safety: Client is safe for concurrent use. The internal reader goroutine
// demultiplexes responses by JSON-RPC ID, so multiple requests can be in-flight
// (though in practice MCP stdio is sequential).
type Client struct {
	command string
	args    []string
	workDir string // Working directory for the subprocess
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stderr  io.ReadCloser
	logFn   LogFunc

	writeMu   sync.Mutex // Protects stdin writes
	requestID int64
	connected bool
	tools     []Tool // Cached tool list

	// pending tracks in-flight requests by their JSON-RPC ID.
	pending   map[int64]chan *MCPResponse
	pendingMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient creates a new MCP client that will spawn the given command.
// The command is not started until Connect() is called.
func NewClient(command string, args ...string) *Client {
	return &Client{
		command: command,
		args:    args,
		pending: make(map[int64]chan *MCPResponse),
	}
}

// SetWorkDir sets the working directory for the MCP server subprocess.
// Must be called before Connect().
func (c *Client) SetWorkDir(dir string) {
	c.workDir = dir
}

// SetLogFunc sets a logging callback for MCP server stderr output and debug messages.
// Must be called before Connect().
func (c *Client) SetLogFunc(fn LogFunc) {
	c.logFn = fn
}

func (c *Client) logf(format string, args ...interface{}) {
	if c.logFn != nil {
		c.logFn(format, args...)
	}
}

// Connect starts the MCP server subprocess and performs the initialize handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.pendingMu.Lock()
	if c.connected {
		c.pendingMu.Unlock()
		return fmt.Errorf("mcpclient: already connected")
	}
	c.pendingMu.Unlock()

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.cmd = exec.CommandContext(c.ctx, c.command, c.args...)
	if c.workDir != "" {
		c.cmd.Dir = c.workDir
	}

	var err error

	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcpclient: stdin pipe: %w", err)
	}

	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcpclient: stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("mcpclient: stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("mcpclient: start: %w", err)
	}

	// Drain stderr — log at debug level if a log function is configured
	go func() {
		scanner := bufio.NewScanner(c.stderr)
		for scanner.Scan() {
			c.logf("mcpclient stderr: %s", scanner.Text())
		}
	}()

	// Start the response reader goroutine that demuxes by ID
	go c.readLoop(bufio.NewReader(stdoutPipe))

	// MCP initialize handshake
	initReq := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "wee-direct-client",
				"version": "1.0.0",
			},
		},
	}

	resp, err := c.sendRequest(initReq)
	if err != nil {
		c.cmd.Process.Kill()
		return fmt.Errorf("mcpclient: initialize: %w", err)
	}

	if resp.Error != nil {
		c.cmd.Process.Kill()
		return fmt.Errorf("mcpclient: initialize error: %s", resp.Error.Message)
	}

	// MCP spec: client MUST send initialized notification after successful initialize
	if err := c.sendNotification("notifications/initialized", nil); err != nil {
		c.logf("mcpclient: warning: failed to send initialized notification: %v", err)
		// Non-fatal — continue anyway
	}

	c.pendingMu.Lock()
	c.connected = true
	c.pendingMu.Unlock()

	return nil
}

// readLoop reads JSON-RPC messages from stdout and routes responses to waiting callers.
// Notifications (messages without an ID) are logged and discarded.
func (c *Client) readLoop(reader *bufio.Reader) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// EOF or pipe closed — signal all pending requests
			c.pendingMu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.pendingMu.Unlock()
			return
		}

		// Try to parse as a JSON-RPC message
		var raw struct {
			ID     interface{} `json:"id"`
			Method string      `json:"method"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			c.logf("mcpclient: ignoring unparseable line: %s", string(line))
			continue
		}

		// Notifications have a method but no ID — log and skip
		if raw.ID == nil {
			c.logf("mcpclient: notification: %s", raw.Method)
			continue
		}

		// Parse as full response
		var resp MCPResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			c.logf("mcpclient: failed to parse response: %v", err)
			continue
		}

		// Match response to pending request by ID
		var reqID int64
		switch v := resp.ID.(type) {
		case float64:
			reqID = int64(v)
		case json.Number:
			n, _ := v.Int64()
			reqID = n
		default:
			c.logf("mcpclient: unexpected response ID type: %T", resp.ID)
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[reqID]
		if ok {
			delete(c.pending, reqID)
		}
		c.pendingMu.Unlock()

		if ok {
			ch <- &resp
		} else {
			c.logf("mcpclient: unexpected response for ID %d", reqID)
		}
	}
}

// ListTools calls tools/list and returns available tools. Cached after first call.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.pendingMu.Lock()
	if !c.connected {
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("mcpclient: not connected")
	}
	if c.tools != nil {
		tools := c.tools
		c.pendingMu.Unlock()
		return tools, nil
	}
	c.pendingMu.Unlock()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("mcpclient: tools/list: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcpclient: tools/list error: %s", resp.Error.Message)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var toolsResult ToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		return nil, fmt.Errorf("mcpclient: parse tools: %w", err)
	}

	c.pendingMu.Lock()
	c.tools = toolsResult.Tools
	c.pendingMu.Unlock()

	return toolsResult.Tools, nil
}

// CallTool invokes a tool by name with the given arguments.
// A per-call timeout of 2 minutes is enforced to prevent hung tools from blocking the loop.
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*CallToolResult, error) {
	c.pendingMu.Lock()
	if !c.connected {
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("mcpclient: not connected")
	}
	c.pendingMu.Unlock()

	// Enforce per-call timeout
	callCtx, callCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer callCancel()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}

	resp, err := c.sendRequestWithContext(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("mcpclient: call %s: %w", toolName, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcpclient: call %s error: %s", toolName, resp.Error.Message)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		return nil, fmt.Errorf("mcpclient: parse call result: %w", err)
	}

	return &callResult, nil
}

// IsConnected returns whether the client has an active connection.
func (c *Client) IsConnected() bool {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	return c.connected
}

// Close terminates the subprocess.
func (c *Client) Close() error {
	c.pendingMu.Lock()
	if !c.connected {
		c.pendingMu.Unlock()
		return nil
	}
	c.connected = false
	c.pendingMu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.stdin != nil {
		c.stdin.Close()
	}

	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()

	select {
	case <-time.After(5 * time.Second):
		if c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}
		<-done
	case <-done:
	}
	return nil
}

// sendNotification writes a JSON-RPC notification (no ID, no response expected).
func (c *Client) sendNotification(method string, params interface{}) error {
	notif := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_, err = c.stdin.Write(data)
	return err
}

// sendRequest writes a JSON-RPC request and waits for the matching response.
// Uses the parent context from Connect().
func (c *Client) sendRequest(req MCPRequest) (*MCPResponse, error) {
	return c.sendRequestWithContext(c.ctx, req)
}

// sendRequestWithContext writes a JSON-RPC request and waits for the matching response,
// respecting the given context for cancellation/timeout.
func (c *Client) sendRequestWithContext(ctx context.Context, req MCPRequest) (*MCPResponse, error) {
	// Extract the request ID
	reqID, ok := req.ID.(int64)
	if !ok {
		return nil, fmt.Errorf("mcpclient: request ID must be int64")
	}

	// Register pending response channel before writing
	ch := make(chan *MCPResponse, 1)
	c.pendingMu.Lock()
	c.pending[reqID] = ch
	c.pendingMu.Unlock()

	// Clean up on failure
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, reqID)
		c.pendingMu.Unlock()
	}()

	// Marshal and write
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	c.writeMu.Lock()
	_, err = c.stdin.Write(data)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	// Wait for response with context
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("connection closed while waiting for response")
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) nextID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}

