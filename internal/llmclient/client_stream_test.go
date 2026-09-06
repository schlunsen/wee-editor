package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// === Streaming Tests with Mock HTTP Server ===

func TestChatCompletionStream_TextOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth = %q, want 'Bearer test-key'", r.Header.Get("Authorization"))
		}

		// Verify request body
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("model = %q, want 'test-model'", req.Model)
		}
		if !req.Stream {
			t.Error("stream should be true")
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")
	ctx := context.Background()

	stream, err := client.ChatCompletionStream(ctx, ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream error: %v", err)
	}

	var textParts []string
	var doneEvents int
	var gotUsage *Usage

	for event := range stream {
		switch event.Type {
		case StreamEventTextDelta:
			textParts = append(textParts, event.Text)
		case StreamEventDone:
			doneEvents++
			if event.Usage != nil {
				gotUsage = event.Usage
			}
		case StreamEventError:
			t.Fatalf("unexpected error event: %v", event.Error)
		}
	}

	fullText := strings.Join(textParts, "")
	if fullText != "Hello world" {
		t.Errorf("text = %q, want %q", fullText, "Hello world")
	}

	if doneEvents == 0 {
		t.Error("expected at least one done event")
	}

	if gotUsage != nil {
		if gotUsage.TotalTokens != 12 {
			t.Errorf("total_tokens = %d, want 12", gotUsage.TotalTokens)
		}
	}
}

func TestChatCompletionStream_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Simulate streaming tool call (split across chunks like real providers do)
		chunks := []string{
			// First chunk: tool call ID and name start
			`{"id":"1","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"gpu_status","arguments":""}}]},"finish_reason":null}]}`,
			// Second chunk: arguments streamed
			`{"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"ver"}}]},"finish_reason":null}]}`,
			// Third chunk: more arguments
			`{"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"bose\":true}"}}]},"finish_reason":null}]}`,
			// Finish
			`{"id":"1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")

	stream, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Check GPU"}},
		Tools:    []Tool{{Type: "function", Function: Function{Name: "gpu_status"}}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var toolCalls []ToolCall
	for event := range stream {
		if event.Type == StreamEventToolCallDelta {
			toolCalls = append(toolCalls, event.ToolCalls...)
		}
	}

	if len(toolCalls) == 0 {
		t.Fatal("expected tool call deltas")
	}

	// Verify the first delta has the ID and name
	if toolCalls[0].ID != "call_abc123" {
		t.Errorf("tool call ID = %q, want %q", toolCalls[0].ID, "call_abc123")
	}
	if toolCalls[0].Function.Name != "gpu_status" {
		t.Errorf("function name = %q, want %q", toolCalls[0].Function.Name, "gpu_status")
	}
}

func TestChatCompletionStream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")

	_, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})

	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want to contain '429'", err.Error())
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %q, want to contain error body", err.Error())
	}
}

func TestChatCompletionStream_ContextCancellation(t *testing.T) {
	// Server that sends data slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		fmt.Fprintf(w, "data: %s\n\n", `{"id":"1","choices":[{"index":0,"delta":{"content":"start"},"finish_reason":null}]}`)
		flusher.Flush()

		// Wait long enough for context to be cancelled
		time.Sleep(1 * time.Second)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	stream, err := client.ChatCompletionStream(ctx, ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("error starting stream: %v", err)
	}

	var gotError bool
	for event := range stream {
		if event.Type == StreamEventError {
			gotError = true
		}
	}

	if !gotError {
		t.Error("expected error event from context cancellation")
	}
}

// === Non-Streaming Tests ===

func TestChatCompletion_NonStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Stream {
			t.Error("stream should be false for non-streaming")
		}

		resp := ChatCompletionResponse{
			ID:    "chatcmpl-123",
			Model: "test-model",
			Choices: []Choice{{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: "Hello!"},
				FinishReason: "stop",
			}},
			Usage: &Usage{
				PromptTokens:     5,
				CompletionTokens: 1,
				TotalTokens:      6,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")

	resp, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(resp.Choices))
	}

	content, ok := resp.Choices[0].Message.Content.(string)
	if !ok {
		t.Fatalf("content type = %T, want string", resp.Choices[0].Message.Content)
	}
	if content != "Hello!" {
		t.Errorf("content = %q, want %q", content, "Hello!")
	}

	if resp.Usage == nil || resp.Usage.TotalTokens != 6 {
		t.Errorf("usage.TotalTokens = %v, want 6", resp.Usage)
	}
}

// === Header Tests ===

func TestSetHeaders_WithAPIKey(t *testing.T) {
	client := NewClient("http://localhost", "sk-test123", "model")

	req, _ := http.NewRequest("POST", "http://localhost", nil)
	client.setHeaders(req)

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "Bearer sk-test123" {
		t.Errorf("Authorization = %q", req.Header.Get("Authorization"))
	}
}

func TestSetHeaders_NoAPIKey(t *testing.T) {
	client := NewClient("http://localhost", "", "model")

	req, _ := http.NewRequest("POST", "http://localhost", nil)
	client.setHeaders(req)

	if req.Header.Get("Authorization") != "" {
		t.Errorf("Authorization should be empty, got %q", req.Header.Get("Authorization"))
	}
}

// === Qwen-style Behavior Tests ===

func TestChatCompletionStream_FinishReasonBeforeDone(t *testing.T) {
	// Some providers (Qwen) send finish_reason in a chunk before "data: [DONE]"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{
			`{"id":"1","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":null}]}`,
			`{"id":"1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			// Extra chunk after finish_reason (some providers do this)
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "qwen-test")
	stream, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var gotText, gotDone bool
	for event := range stream {
		switch event.Type {
		case StreamEventTextDelta:
			gotText = true
		case StreamEventDone:
			gotDone = true
		}
	}

	if !gotText {
		t.Error("expected text delta")
	}
	if !gotDone {
		t.Error("expected done event")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://localhost:11434/v1/", "key", "model")
	if c.baseURL != "http://localhost:11434/v1" {
		t.Errorf("baseURL = %q, want trailing slash trimmed", c.baseURL)
	}
}
