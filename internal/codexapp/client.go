// Package codexapp implements the bidirectional Codex app-server transport.
// The exec SDK is still used by callers to locate Codex and describe inputs.
package codexapp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type Message struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return fmt.Sprintf("codex app-server (%d): %s", e.Code, e.Message) }

type Client struct {
	ctx     context.Context
	cancel  context.CancelFunc
	stdin   io.WriteCloser
	writeMu sync.Mutex
	mu      sync.Mutex
	next    int
	pending map[string]chan Message
	events  chan Message
	done    chan struct{}
	err     error
}

func Start(ctx context.Context, path string, env []string, dir string) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, path, "app-server")
	cmd.Env, cmd.Dir = env, dir
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		stdin.Close()
		return nil, err
	}
	// Do not copy stderr to the UI: it can include credentials/configuration.
	if err = cmd.Start(); err != nil {
		cancel()
		stdin.Close()
		return nil, err
	}
	c := &Client{ctx: ctx, cancel: cancel, stdin: stdin, pending: make(map[string]chan Message), events: make(chan Message), done: make(chan struct{})}
	// Cancellation must also unblock a pipe inherited by a child process.
	go func() {
		select {
		case <-ctx.Done():
			_ = stdout.Close()
		case <-c.done:
		}
	}()
	raw := make(chan Message)
	go c.deliver(raw)
	go func() {
		defer close(c.done)
		defer close(raw)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 65536), 64*1024*1024)
		for scanner.Scan() {
			var m Message
			if err = json.Unmarshal(scanner.Bytes(), &m); err != nil {
				cancel()
				break
			}
			if m.Method == "" && len(m.ID) > 0 {
				c.mu.Lock()
				ch := c.pending[string(m.ID)]
				c.mu.Unlock()
				if ch != nil {
					select {
					case ch <- m:
					default:
					} // Ignore duplicate responses.
				}
			} else if len(m.ID) > 0 {
				// Never leave server requests hanging or silently grant permissions.
				_ = c.write(map[string]any{"id": m.ID, "error": &RPCError{Code: -32601, Message: "Interactive request unsupported by wee"}})
			} else {
				select {
				case raw <- m:
				case <-ctx.Done():
				}
			}
		}
		if err == nil {
			err = scanner.Err()
		}
		if err != nil {
			cancel()
		}
		waitErr := cmd.Wait()
		if err == nil {
			err = waitErr
		}
		if err == nil {
			err = io.EOF
		}
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
	}()
	return c, nil
}

// Queue notifications separately so a caller awaiting an RPC response cannot
// deadlock behind a burst of tool output.
func (c *Client) deliver(raw <-chan Message) {
	defer close(c.events)
	var queue []Message
	for raw != nil || len(queue) > 0 {
		var out chan Message
		var first Message
		if len(queue) > 0 {
			out = c.events
			first = queue[0]
		}
		select {
		case m, ok := <-raw:
			if !ok {
				raw = nil
			} else {
				queue = append(queue, m)
			}
		case out <- first:
			queue[0] = Message{}
			queue = queue[1:]
		case <-c.ctx.Done():
			return
		}
	}
}
func (c *Client) write(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return json.NewEncoder(c.stdin).Encode(v)
}
func (c *Client) Notify(method string, params any) error {
	return c.write(map[string]any{"method": method, "params": params})
}
func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	c.next++
	id := c.next
	key := fmt.Sprint(id)
	ch := make(chan Message, 1)
	c.pending[key] = ch
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, key); c.mu.Unlock() }()
	if err := c.write(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return m.Error
		}
		if result != nil {
			return json.Unmarshal(m.Result, result)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return c.Err()
	}
}
func (c *Client) Events() <-chan Message { return c.events }
func (c *Client) Err() error             { c.mu.Lock(); defer c.mu.Unlock(); return c.err }
func (c *Client) Close() {
	// EOF lets Codex flush the thread store before we terminate the process.
	_ = c.stdin.Close()
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		c.cancel()
		<-c.done
	}
	c.cancel()
}
