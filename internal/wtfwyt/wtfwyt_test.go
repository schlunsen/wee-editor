package wtfwyt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/schlunsen/wee-editor/internal/analytics"
	"github.com/schlunsen/wtfwyt/server/pkg/detect"
	"github.com/schlunsen/wtfwyt/server/pkg/wire"
)

// fakeServer captures what the exporter actually transmits.
type fakeServer struct {
	*httptest.Server
	mu      sync.Mutex
	batches []wire.Batch
	bodies  []string
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "wtk_test",
			"expires_at":   time.Now().Add(time.Hour),
		})
	})
	mux.HandleFunc("/api/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		var buf strings.Builder
		var b wire.Batch
		dec := json.NewDecoder(io_TeeReader(r.Body, &buf))
		_ = dec.Decode(&b)

		f.mu.Lock()
		f.batches = append(f.batches, b)
		f.bodies = append(f.bodies, buf.String())
		f.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(wire.BatchResult{Accepted: len(b.Events)})
	})

	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// everythingSent returns every raw byte the exporter transmitted.
func (f *fakeServer) everythingSent() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.bodies, "\n")
}

func (f *fakeServer) events() []wire.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []wire.Event
	for _, b := range f.batches {
		out = append(out, b.Events...)
	}
	return out
}

func newTestExporter(t *testing.T, srv *fakeServer, exportContent bool) *Exporter {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Endpoint = srv.URL
	cfg.ClientID = "wee_test"
	cfg.ClientSecret = "secret"
	cfg.ExportContent = exportContent
	cfg.FlushInterval = Duration(50 * time.Millisecond)

	e, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e == nil {
		t.Fatal("exporter is nil though enabled")
	}
	return e
}

func drain(t *testing.T, e *Exporter) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { e.Run(ctx); close(done) }()
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done
}

// The property that matters most: a credential in a tool result must never
// appear in anything the exporter transmits.
func TestSecretInToolResultIsNeverTransmitted(t *testing.T) {
	const awsKey = "AKIAIOSFODNN7EXAMPLE"

	for _, exportContent := range []bool{false, true} {
		name := "findings_only"
		if exportContent {
			name = "with_content"
		}
		t.Run(name, func(t *testing.T) {
			srv := newFakeServer(t)
			e := newTestExporter(t, srv, exportContent)

			e.ExportToolExecution(&analytics.ToolExecution{
				ToolID:         "t1",
				ToolName:       "Bash",
				Input:          map[string]any{"command": "cat .env"},
				Result:         "AWS_ACCESS_KEY_ID=" + awsKey + "\nDEBUG=true",
				Success:        true,
				ConversationID: "sess-1",
				ExecutedAt:     time.Now(),
			})
			drain(t, e)

			sent := srv.everythingSent()
			if sent == "" {
				t.Fatal("nothing was transmitted")
			}
			if strings.Contains(sent, awsKey) {
				t.Fatalf("CREDENTIAL TRANSMITTED:\n%s", sent)
			}

			// A finding must still have been reported.
			var gotFinding bool
			for _, ev := range srv.events() {
				if ev.Type == wire.EventSecretFinding && ev.Secret != nil {
					gotFinding = true
					if ev.Secret.RuleID != "aws_access_key_id" {
						t.Errorf("rule = %s", ev.Secret.RuleID)
					}
					if ev.Secret.Fingerprint == "" {
						t.Error("finding has no fingerprint")
					}
				}
			}
			if !gotFinding {
				t.Error("no secret_finding event was sent")
			}
		})
	}
}

// With content export off, no transcript content may be transmitted at all -
// only findings.
func TestFindingsOnlyModeSendsNoContent(t *testing.T) {
	srv := newFakeServer(t)
	e := newTestExporter(t, srv, false)

	e.ExportToolExecution(&analytics.ToolExecution{
		ToolID: "t1", ToolName: "Bash",
		Input:          map[string]any{"command": "cat .env"},
		Result:         "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nSOME_INTERNAL_DETAIL=value",
		ConversationID: "s", ExecutedAt: time.Now(),
	})
	drain(t, e)

	sent := srv.everythingSent()
	for _, leak := range []string{"cat .env", "SOME_INTERNAL_DETAIL", "tool_execution"} {
		if strings.Contains(sent, leak) {
			t.Errorf("content leaked in findings-only mode: %q found in\n%s", leak, sent)
		}
	}
	for _, ev := range srv.events() {
		if ev.Type != wire.EventSecretFinding {
			t.Errorf("unexpected event type %q in findings-only mode", ev.Type)
		}
	}
}

// With content export on, redacted content is sent and the surrounding
// context survives.
func TestContentModeSendsRedactedContent(t *testing.T) {
	srv := newFakeServer(t)
	e := newTestExporter(t, srv, true)

	e.ExportToolExecution(&analytics.ToolExecution{
		ToolID: "t1", ToolName: "Read",
		Input:          map[string]any{"file_path": "/app/.env"},
		Result:         "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nDEBUG=true",
		ConversationID: "s", ExecutedAt: time.Now(),
	})
	drain(t, e)

	var toolEvent *wire.ToolExecution
	for _, ev := range srv.events() {
		if ev.Type == wire.EventToolExecution {
			toolEvent = ev.Tool
		}
	}
	if toolEvent == nil {
		t.Fatal("no tool_execution event sent in content mode")
	}
	if !strings.Contains(toolEvent.Result, "[REDACTED:") {
		t.Errorf("result not redacted: %q", toolEvent.Result)
	}
	if !strings.Contains(toolEvent.Result, "DEBUG=true") {
		t.Errorf("redaction destroyed surrounding context: %q", toolEvent.Result)
	}
}

// A secret the model quotes back in its reasoning must be caught too.
func TestSecretInThinkingIsCaught(t *testing.T) {
	srv := newFakeServer(t)
	e := newTestExporter(t, srv, true)

	const ghToken = "ghp_u8jzPde0IgxLd6GncfBAepfJBd0Kh8oOOL8d"
	e.ExportMessage("s", "assistant", "I'll use the token.",
		"the token I just read is "+ghToken, "main", "/repo", time.Now())
	drain(t, e)

	sent := srv.everythingSent()
	if strings.Contains(sent, ghToken) {
		t.Fatalf("token from thinking block transmitted:\n%s", sent)
	}
	var found bool
	for _, ev := range srv.events() {
		if ev.Type == wire.EventSecretFinding && ev.Secret.RuleID == "github_token" {
			found = true
			if ev.Secret.Source != wire.SourceAssistantMessage {
				t.Errorf("source = %s, want assistant_message", ev.Secret.Source)
			}
		}
	}
	if !found {
		t.Error("no finding for the token in the thinking block")
	}
}

// The local alert must fire before anything is transmitted.
func TestLocalAlertFires(t *testing.T) {
	srv := newFakeServer(t)
	e := newTestExporter(t, srv, false)

	var mu sync.Mutex
	var alerts []detect.Finding
	e.SetAlert(func(f detect.Finding, where string) {
		mu.Lock()
		alerts = append(alerts, f)
		mu.Unlock()
	})

	e.ExportToolExecution(&analytics.ToolExecution{
		ToolID: "t", ToolName: "Bash",
		Result:         "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
		ConversationID: "s", ExecutedAt: time.Now(),
	})

	mu.Lock()
	defer mu.Unlock()
	if len(alerts) != 1 {
		t.Fatalf("want 1 alert, got %d", len(alerts))
	}
	if alerts[0].Severity != detect.SeverityCritical {
		t.Errorf("severity = %s", alerts[0].Severity)
	}
}

// Clean content must produce no traffic at all.
func TestCleanContentSendsNothing(t *testing.T) {
	srv := newFakeServer(t)
	e := newTestExporter(t, srv, false)

	e.ExportToolExecution(&analytics.ToolExecution{
		ToolID: "t", ToolName: "Edit",
		Input:          map[string]any{"file_path": "main.go"},
		Result:         "Applied 3 edits to main.go",
		ConversationID: "s", ExecutedAt: time.Now(),
	})
	drain(t, e)

	if got := srv.everythingSent(); got != "" {
		t.Errorf("clean tool execution produced traffic:\n%s", got)
	}
}

// Disabled config must produce a nil exporter whose methods are all safe.
func TestDisabledExporterIsSafe(t *testing.T) {
	e, err := New(Config{Enabled: false}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e != nil {
		t.Fatal("expected nil exporter when disabled")
	}

	// Every method must tolerate a nil receiver, so callers need no guards.
	e.ExportToolExecution(&analytics.ToolExecution{ToolName: "Bash", Result: "AKIAIOSFODNN7EXAMPLE"})
	e.ExportMessage("s", "user", "AKIAIOSFODNN7EXAMPLE", "", "", "", time.Now())
	e.ExportLog("s", "info", "AKIAIOSFODNN7EXAMPLE", "", time.Now())
	e.SetProject("p", "n", "/path")
	e.SetAlert(nil)
	if e.Enabled() {
		t.Error("nil exporter reports enabled")
	}
	if e.Dropped() != 0 {
		t.Error("nil exporter reports drops")
	}
	e.Run(context.Background()) // must return immediately
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"disabled is always valid", func(c *Config) { c.Enabled = false }, false},
		{"valid", func(c *Config) {}, false},
		{"no client id", func(c *Config) { c.ClientID = "" }, true},
		{"no secret", func(c *Config) { c.ClientSecret = "" }, true},
		{"no endpoint", func(c *Config) { c.Endpoint = "" }, true},
		{"plaintext remote endpoint", func(c *Config) { c.Endpoint = "http://wtfwyt.com" }, true},
		{"localhost plaintext allowed", func(c *Config) { c.Endpoint = "http://localhost:8090" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultConfig()
			c.Enabled = true
			c.ClientID = "id"
			c.ClientSecret = "sec"
			tt.mutate(&c)

			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigNeverSerialisesSecret(t *testing.T) {
	c := DefaultConfig()
	c.ClientID = "wee_public"
	c.ClientSecret = "super-secret-value"

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "super-secret-value") {
		t.Fatalf("client secret serialised into config JSON: %s", b)
	}
	if !strings.Contains(string(b), "wee_public") {
		t.Error("client id missing from config JSON")
	}
}

func TestDurationRoundTrip(t *testing.T) {
	c := DefaultConfig()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var back Config
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.FlushInterval != c.FlushInterval {
		t.Errorf("flush interval = %v, want %v",
			time.Duration(back.FlushInterval), time.Duration(c.FlushInterval))
	}
}
