// Package llmclient implements a streaming OpenAI-compatible API client with
// full tool/function calling support. This enables non-Claude models (qwen,
// deepseek, gpt, ollama, etc.) to use tools via native function calling.
//
// The client follows the OpenAI Chat Completions API spec, which is supported by:
// - OpenAI (GPT-4, GPT-3.5)
// - DeepSeek
// - Qwen (via OpenAI-compatible endpoint)
// - Ollama (via OpenAI-compatible endpoint)
// - vLLM, LiteLLM, and other OpenAI-compatible servers
//
// This package has NO dependencies on mcp, agents, or providers to avoid import cycles.
package llmclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a streaming OpenAI-compatible API client.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new OpenAI-compatible client.
func NewClient(baseURL, apiKey, model string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// === Request Types ===

// ChatCompletionRequest represents an OpenAI chat completion request.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	ToolChoice  interface{}   `json:"tool_choice,omitempty"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    interface{} `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// Tool represents a tool definition in OpenAI format.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition.
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall represents a tool call from the assistant.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
	Index    *int         `json:"index,omitempty"` // Used in streaming deltas
}

// FunctionCall represents the function details in a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// === Response Types ===

// ChatCompletionResponse represents a non-streaming response.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatCompletionChunk represents a streaming chunk.
type ChatCompletionChunk struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []DeltaChoice `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

// DeltaChoice represents a delta in streaming.
type DeltaChoice struct {
	Index        int          `json:"index"`
	Delta        DeltaContent `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// DeltaContent represents the content delta in streaming.
type DeltaContent struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// === Client Methods ===

// ChatCompletion sends a non-streaming chat completion request.
func (c *Client) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Model = c.model
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llmclient: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llmclient: failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llmclient: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("llmclient: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("llmclient: failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// ChatCompletionStream sends a streaming chat completion request.
// Returns a channel of StreamEvents. The channel is closed when the stream ends.
func (c *Client) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan StreamEvent, error) {
	req.Model = c.model
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llmclient: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llmclient: failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llmclient: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("llmclient: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	events := make(chan StreamEvent, 32)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				select {
				case events <- StreamEvent{Type: StreamEventError, Error: ctx.Err()}:
				default:
				}
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				select {
				case events <- StreamEvent{Type: StreamEventDone}:
				case <-ctx.Done():
				}
				return
			}

			var chunk ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			finishReason := chunk.Choices[0].FinishReason

			if delta.Content != "" {
				select {
				case events <- StreamEvent{Type: StreamEventTextDelta, Text: delta.Content}:
				case <-ctx.Done():
					return
				}
			}

			if len(delta.ToolCalls) > 0 {
				select {
				case events <- StreamEvent{Type: StreamEventToolCallDelta, ToolCalls: delta.ToolCalls}:
				case <-ctx.Done():
					return
				}
			}

			if finishReason != nil && *finishReason != "" {
				select {
				case events <- StreamEvent{
					Type:         StreamEventDone,
					FinishReason: *finishReason,
					Usage:        chunk.Usage,
				}:
				case <-ctx.Done():
				}
				// Don't return here — some providers (e.g. Qwen) send finish_reason
				// before "data: [DONE]". Continue reading so we don't miss chunks
				// and so the body gets fully consumed / connection reused.
			}
		}

		if err := scanner.Err(); err != nil {
			// Do not race this send against ctx.Done(): when the read fails
			// *because* the context was cancelled, both cases are ready and
			// select picks at random, silently dropping the error for roughly
			// half of all cancelled streams. events is buffered and this is the
			// final send before close, so a non-blocking send delivers the
			// error without any risk of blocking the goroutine.
			select {
			case events <- StreamEvent{Type: StreamEventError, Error: err}:
			default:
			}
		}
	}()

	return events, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// === Stream Event Types ===

// StreamEvent represents different events during streaming.
type StreamEvent struct {
	Type         StreamEventType
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        *Usage
	Error        error
}

// StreamEventType enumerates the types of stream events.
type StreamEventType int

const (
	StreamEventTextDelta     StreamEventType = iota
	StreamEventToolCallDelta
	StreamEventDone
	StreamEventError
)

// === Utility Functions ===

// ParseToolCallArguments parses the JSON arguments string from a tool call.
func ParseToolCallArguments(argsJSON string) (map[string]interface{}, error) {
	if argsJSON == "" || argsJSON == "{}" {
		return map[string]interface{}{}, nil
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, err
	}
	return args, nil
}
