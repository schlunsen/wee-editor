package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// === JSON-RPC Serialization Tests ===

func TestMCPRequestSerialization(t *testing.T) {
	tests := []struct {
		name string
		req  MCPRequest
		want map[string]interface{}
	}{
		{
			name: "initialize request",
			req: MCPRequest{
				JSONRPC: "2.0",
				ID:      int64(1),
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
				},
			},
			want: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      float64(1), // JSON numbers decode as float64
				"method":  "initialize",
			},
		},
		{
			name: "tools/list request (no params)",
			req: MCPRequest{
				JSONRPC: "2.0",
				ID:      int64(2),
				Method:  "tools/list",
			},
			want: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      float64(2),
				"method":  "tools/list",
			},
		},
		{
			name: "tools/call request",
			req: MCPRequest{
				JSONRPC: "2.0",
				ID:      int64(3),
				Method:  "tools/call",
				Params: CallToolParams{
					Name:      "gpu_status",
					Arguments: map[string]interface{}{"verbose": true},
				},
			},
			want: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      float64(3),
				"method":  "tools/call",
			},
		},
		{
			name: "notification (no ID)",
			req: MCPRequest{
				JSONRPC: "2.0",
				Method:  "notifications/initialized",
			},
			want: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "notifications/initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got map[string]interface{}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("missing key %q in serialized request", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("key %q = %v (%T), want %v (%T)", key, gotVal, gotVal, wantVal, wantVal)
				}
			}
		})
	}
}

func TestMCPResponseDeserialization(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantID    float64
		wantError bool
		wantErrMsg string
	}{
		{
			name:   "success response",
			json:   `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}`,
			wantID: 1,
		},
		{
			name:       "error response",
			json:       `{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"method not found"}}`,
			wantID:     2,
			wantError:  true,
			wantErrMsg: "method not found",
		},
		{
			name:   "tools/list result",
			json:   `{"jsonrpc":"2.0","id":3,"result":{"tools":[{"name":"test_tool","description":"A test tool","inputSchema":{"type":"object"}}]}}`,
			wantID: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp MCPResponse
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			// Check ID (JSON numbers are float64)
			id, ok := resp.ID.(float64)
			if !ok {
				t.Fatalf("ID type = %T, want float64", resp.ID)
			}
			if id != tt.wantID {
				t.Errorf("ID = %v, want %v", id, tt.wantID)
			}

			if tt.wantError {
				if resp.Error == nil {
					t.Fatal("expected error, got nil")
				}
				if resp.Error.Message != tt.wantErrMsg {
					t.Errorf("error message = %q, want %q", resp.Error.Message, tt.wantErrMsg)
				}
			} else {
				if resp.Error != nil {
					t.Errorf("unexpected error: %v", resp.Error)
				}
			}
		})
	}
}

func TestToolsListResultDeserialization(t *testing.T) {
	raw := `{
		"tools": [
			{
				"name": "gpu_status",
				"description": "Get GPU status",
				"inputSchema": {
					"type": "object",
					"properties": {
						"verbose": {
							"type": "boolean",
							"description": "Show detailed info"
						}
					},
					"required": ["verbose"]
				}
			},
			{
				"name": "deploy_app",
				"description": "Deploy an application",
				"inputSchema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string",
							"description": "App name"
						},
						"env": {
							"type": "string",
							"description": "Target environment",
							"enum": ["staging", "production"]
						}
					}
				}
			}
		]
	}`

	var result ToolsListResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(result.Tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(result.Tools))
	}

	// Check first tool
	tool := result.Tools[0]
	if tool.Name != "gpu_status" {
		t.Errorf("tool[0].Name = %q, want %q", tool.Name, "gpu_status")
	}
	if tool.InputSchema.Type != "object" {
		t.Errorf("tool[0].InputSchema.Type = %q, want %q", tool.InputSchema.Type, "object")
	}
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "verbose" {
		t.Errorf("tool[0].InputSchema.Required = %v, want [verbose]", tool.InputSchema.Required)
	}
	if prop, ok := tool.InputSchema.Properties["verbose"]; !ok {
		t.Error("tool[0] missing 'verbose' property")
	} else if prop.Type != "boolean" {
		t.Errorf("verbose.Type = %q, want %q", prop.Type, "boolean")
	}

	// Check second tool's enum
	tool2 := result.Tools[1]
	if envProp, ok := tool2.InputSchema.Properties["env"]; !ok {
		t.Error("tool[1] missing 'env' property")
	} else if len(envProp.Enum) != 2 {
		t.Errorf("env.Enum = %v, want 2 values", envProp.Enum)
	}
}

func TestCallToolResultDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantText string
		wantErr  bool
	}{
		{
			name:     "text result",
			json:     `{"content":[{"type":"text","text":"GPU 0: NVIDIA A100, 80GB, 45% utilized"}]}`,
			wantText: "GPU 0: NVIDIA A100, 80GB, 45% utilized",
		},
		{
			name:     "multi-block result",
			json:     `{"content":[{"type":"text","text":"line1"},{"type":"text","text":"line2"}]}`,
			wantText: "line1",
		},
		{
			name:    "error result",
			json:    `{"content":[{"type":"text","text":"command failed"}],"isError":true}`,
			wantErr: true,
		},
		{
			name:     "empty content",
			json:     `{"content":[]}`,
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result CallToolResult
			if err := json.Unmarshal([]byte(tt.json), &result); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if tt.wantText != "" {
				if len(result.Content) == 0 {
					t.Fatal("expected content, got empty")
				}
				if result.Content[0].Text != tt.wantText {
					t.Errorf("text = %q, want %q", result.Content[0].Text, tt.wantText)
				}
			}

			isErr := result.IsError != nil && *result.IsError
			if isErr != tt.wantErr {
				t.Errorf("isError = %v, want %v", isErr, tt.wantErr)
			}
		})
	}
}

// === readLoop Response Routing Tests ===

// mockPipeReadWriter creates a connected reader/writer pair for testing readLoop.
func mockPipeReadWriter() (io.Writer, *bufio.Reader) {
	r, w := io.Pipe()
	return w, bufio.NewReader(r)
}

func TestReadLoopRoutesResponsesByID(t *testing.T) {
	w, reader := mockPipeReadWriter()

	c := &Client{
		pending: make(map[int64]chan *MCPResponse),
	}

	// Register two pending requests
	ch1 := make(chan *MCPResponse, 1)
	ch2 := make(chan *MCPResponse, 1)
	c.pendingMu.Lock()
	c.pending[1] = ch1
	c.pending[2] = ch2
	c.pendingMu.Unlock()

	// Start readLoop in background
	go c.readLoop(reader)

	// Send responses out of order (ID 2 first, then ID 1)
	resp2 := `{"jsonrpc":"2.0","id":2,"result":{"data":"response-two"}}` + "\n"
	resp1 := `{"jsonrpc":"2.0","id":1,"result":{"data":"response-one"}}` + "\n"

	if _, err := w.Write([]byte(resp2)); err != nil {
		t.Fatalf("write resp2: %v", err)
	}
	if _, err := w.Write([]byte(resp1)); err != nil {
		t.Fatalf("write resp1: %v", err)
	}

	// Verify responses are routed to correct channels
	select {
	case got := <-ch2:
		if got == nil {
			t.Fatal("ch2 got nil response")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response on ch2")
	}

	select {
	case got := <-ch1:
		if got == nil {
			t.Fatal("ch1 got nil response")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response on ch1")
	}
}

func TestReadLoopIgnoresNotifications(t *testing.T) {
	w, reader := mockPipeReadWriter()

	var logMessages []string
	var logMu sync.Mutex

	c := &Client{
		pending: make(map[int64]chan *MCPResponse),
		logFn: func(format string, args ...interface{}) {
			logMu.Lock()
			logMessages = append(logMessages, format)
			logMu.Unlock()
		},
	}

	// Register a pending request
	ch1 := make(chan *MCPResponse, 1)
	c.pendingMu.Lock()
	c.pending[1] = ch1
	c.pendingMu.Unlock()

	go c.readLoop(reader)

	// Send a notification (no ID), then a real response
	notification := `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n"
	response := `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}` + "\n"

	w.Write([]byte(notification))
	w.Write([]byte(response))

	// Should still get the response despite the notification
	select {
	case got := <-ch1:
		if got == nil {
			t.Fatal("got nil response")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	// Verify the notification was logged
	logMu.Lock()
	found := false
	for _, msg := range logMessages {
		if strings.Contains(msg, "notification") {
			found = true
			break
		}
	}
	logMu.Unlock()

	if !found {
		t.Error("notification was not logged")
	}
}

func TestReadLoopClosesChannelsOnEOF(t *testing.T) {
	r, w := io.Pipe()
	reader := bufio.NewReader(r)

	c := &Client{
		pending: make(map[int64]chan *MCPResponse),
	}

	ch := make(chan *MCPResponse, 1)
	c.pendingMu.Lock()
	c.pending[1] = ch
	c.pendingMu.Unlock()

	done := make(chan struct{})
	go func() {
		c.readLoop(reader)
		close(done)
	}()

	// Close the writer to trigger EOF
	w.Close()

	// readLoop should close the pending channel and exit
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for channel close")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for readLoop to exit")
	}
}

func TestReadLoopSkipsUnparseableLines(t *testing.T) {
	w, reader := mockPipeReadWriter()

	c := &Client{
		pending: make(map[int64]chan *MCPResponse),
		logFn:   func(format string, args ...interface{}) {},
	}

	ch := make(chan *MCPResponse, 1)
	c.pendingMu.Lock()
	c.pending[1] = ch
	c.pendingMu.Unlock()

	go c.readLoop(reader)

	// Send garbage, then a real response
	w.Write([]byte("this is not json\n"))
	w.Write([]byte("{malformed json\n"))
	w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}` + "\n"))

	select {
	case got := <-ch:
		if got == nil {
			t.Fatal("got nil response")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response after garbage lines")
	}
}

// === Client State Tests ===

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("some-binary", "arg1", "arg2")

	if c.command != "some-binary" {
		t.Errorf("command = %q, want %q", c.command, "some-binary")
	}
	if len(c.args) != 2 || c.args[0] != "arg1" || c.args[1] != "arg2" {
		t.Errorf("args = %v, want [arg1, arg2]", c.args)
	}
	if c.connected {
		t.Error("new client should not be connected")
	}
	if c.IsConnected() {
		t.Error("IsConnected() should return false for new client")
	}
	if c.pending == nil {
		t.Error("pending map should be initialized")
	}
}

func TestSetWorkDir(t *testing.T) {
	c := NewClient("bin")
	c.SetWorkDir("/tmp/test")
	if c.workDir != "/tmp/test" {
		t.Errorf("workDir = %q, want %q", c.workDir, "/tmp/test")
	}
}

func TestSetLogFunc(t *testing.T) {
	c := NewClient("bin")

	called := false
	c.SetLogFunc(func(format string, args ...interface{}) {
		called = true
	})

	c.logf("test %s", "message")
	if !called {
		t.Error("log function was not called")
	}
}

func TestLogfNilSafe(t *testing.T) {
	c := NewClient("bin")
	// Should not panic with nil logFn
	c.logf("test %s", "message")
}

func TestCloseNotConnected(t *testing.T) {
	c := NewClient("bin")
	// Close on a not-connected client should be a no-op
	if err := c.Close(); err != nil {
		t.Errorf("Close() on not-connected client returned error: %v", err)
	}
}

func TestListToolsNotConnected(t *testing.T) {
	c := NewClient("bin")
	_, err := c.ListTools(context.Background())
	if err == nil {
		t.Error("ListTools on not-connected client should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %q, want to contain 'not connected'", err.Error())
	}
}

func TestCallToolNotConnected(t *testing.T) {
	c := NewClient("bin")
	_, err := c.CallTool(context.Background(), "test", nil)
	if err == nil {
		t.Error("CallTool on not-connected client should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %q, want to contain 'not connected'", err.Error())
	}
}

func TestNextIDIsAtomic(t *testing.T) {
	c := NewClient("bin")

	var wg sync.WaitGroup
	ids := make(chan int64, 100)

	// Generate IDs from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				ids <- c.nextID()
			}
		}()
	}

	wg.Wait()
	close(ids)

	// Verify all IDs are unique
	seen := make(map[int64]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate ID: %d", id)
		}
		seen[id] = true
	}

	if len(seen) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(seen))
	}
}

// === CallToolParams Serialization ===

func TestCallToolParamsSerialization(t *testing.T) {
	params := CallToolParams{
		Name: "run_command",
		Arguments: map[string]interface{}{
			"command": "nvidia-smi",
			"timeout": float64(30),
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got["name"] != "run_command" {
		t.Errorf("name = %v, want %q", got["name"], "run_command")
	}

	args, ok := got["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("arguments is not a map")
	}
	if args["command"] != "nvidia-smi" {
		t.Errorf("arguments.command = %v, want %q", args["command"], "nvidia-smi")
	}
}

func TestCallToolParamsOmitsEmptyArguments(t *testing.T) {
	params := CallToolParams{
		Name: "list_tools",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got map[string]interface{}
	json.Unmarshal(data, &got)

	if _, ok := got["arguments"]; ok {
		t.Error("empty arguments should be omitted from JSON")
	}
}
