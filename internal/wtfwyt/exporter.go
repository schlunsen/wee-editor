package wtfwyt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wtfwyt/server/pkg/detect"
	"github.com/schlunsen/wtfwyt/server/pkg/wire"
)

// AlertFunc is called when a credential is detected locally, before anything
// is transmitted. It is how the user finds out immediately rather than when
// someone next opens a dashboard.
type AlertFunc func(finding detect.Finding, where string)

// Exporter batches session events and ships them to a wtfwyt server.
//
// The zero value is unusable; call New. A nil *Exporter is safe to call every
// method on, which lets callers skip nil checks when export is disabled.
type Exporter struct {
	cfg  Config
	http *http.Client
	in   chan wire.Event

	mu      sync.Mutex
	token   string
	expires time.Time
	project *wire.Project

	alert AlertFunc

	// dropped counts events discarded because the buffer was full, so the
	// condition is visible rather than silent.
	dropped uint64

	logf func(format string, args ...any)
}

// New creates an exporter. It returns nil when export is disabled, so callers
// can assign the result unconditionally.
func New(cfg Config, logf func(string, ...any)) (*Exporter, error) {
	cfg.LoadSecret()
	if !cfg.Enabled {
		return nil, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.withDefaults()

	if logf == nil {
		logf = func(string, ...any) {}
	}

	return &Exporter{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
		in:   make(chan wire.Event, cfg.BufferSize),
		logf: logf,
	}, nil
}

// SetAlert installs the local alert callback.
func (e *Exporter) SetAlert(fn AlertFunc) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.alert = fn
	e.mu.Unlock()
}

// SetProject attaches project metadata to subsequent batches.
func (e *Exporter) SetProject(id, name, path string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.project = &wire.Project{ID: id, Name: name, Path: path, IsActive: true}
	e.mu.Unlock()
}

// Enabled reports whether this exporter will send anything.
func (e *Exporter) Enabled() bool { return e != nil }

// Dropped returns how many events were discarded due to a full buffer.
func (e *Exporter) Dropped() uint64 {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dropped
}

// send queues an event without blocking.
//
// When the buffer is full the OLDEST event is dropped. This matches the
// behaviour of wee-editor's own LogBroadcaster: recent context is worth more
// than a complete but stale backlog, and blocking here would stall whatever
// part of the editor produced the event.
func (e *Exporter) send(ev wire.Event) {
	if e == nil {
		return
	}
	select {
	case e.in <- ev:
		return
	default:
	}

	select {
	case <-e.in:
		e.mu.Lock()
		e.dropped++
		e.mu.Unlock()
	default:
	}
	select {
	case e.in <- ev:
	default:
	}
}

// Run ships batches until ctx is cancelled, then makes a final flush attempt
// so the tail of a session is not lost on shutdown.
func (e *Exporter) Run(ctx context.Context) {
	if e == nil {
		return
	}

	ticker := time.NewTicker(time.Duration(e.cfg.FlushInterval))
	defer ticker.Stop()

	pending := make([]wire.Event, 0, e.cfg.FlushSize)

	flush := func(ctx context.Context) {
		if len(pending) == 0 {
			return
		}
		if err := e.post(ctx, pending); err != nil {
			e.logf("wtfwyt: export failed: %v (%d events)", err, len(pending))
		}
		pending = pending[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// A fresh context: the cancelled one cannot carry the final send.
			shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			flush(shutdown)
			cancel()
			if d := e.Dropped(); d > 0 {
				e.logf("wtfwyt: %d events dropped (buffer full)", d)
			}
			return

		case ev := <-e.in:
			pending = append(pending, ev)
			if len(pending) >= e.cfg.FlushSize {
				flush(ctx)
			}

		case <-ticker.C:
			flush(ctx)
		}
	}
}

// post sends one batch, retrying transient failures with exponential backoff
// and jitter.
//
// Retrying is safe because every event carries a stable client-generated ID
// and the server deduplicates on it: a resend after an ambiguous timeout is
// counted as a duplicate rather than stored twice.
func (e *Exporter) post(ctx context.Context, events []wire.Event) error {
	e.mu.Lock()
	project := e.project
	e.mu.Unlock()

	batch := wire.Batch{
		ProtocolVersion: wire.ProtocolVersion,
		Project:         project,
		Events:          events,
	}
	if err := batch.Validate(); err != nil {
		return fmt.Errorf("refusing to send invalid batch: %w", err)
	}

	body, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("encoding batch: %w", err)
	}

	const maxAttempts = 5
	backoff := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		token, err := e.ensureToken(ctx)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			e.cfg.Endpoint+"/api/v1/ingest", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := e.http.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return fmt.Errorf("sending batch: %w", err)
			}
			e.sleep(ctx, &backoff)
			continue
		}

		switch {
		case resp.StatusCode == http.StatusCreated:
			var res wire.BatchResult
			_ = json.NewDecoder(resp.Body).Decode(&res)
			resp.Body.Close()

			if res.Unredacted > 0 {
				// The server had to catch a credential this client missed,
				// which means these detection rules are older than the
				// server's. Worth saying loudly: raw secrets are reaching
				// the wire.
				e.logf("wtfwyt: WARNING - %d credential(s) were not redacted by this client; "+
					"its detection rules are out of date. Upgrade wee-editor.", res.Unredacted)
			}
			if res.Rejected > 0 {
				e.logf("wtfwyt: %d event(s) rejected: %v", res.Rejected, res.Errors)
			}
			return nil

		case resp.StatusCode == http.StatusUnauthorized:
			resp.Body.Close()
			e.invalidateToken()
			if attempt == maxAttempts {
				return fmt.Errorf("unauthorized after %d attempts", attempt)
			}
			continue

		default:
			var apiErr wire.APIError
			_ = json.NewDecoder(resp.Body).Decode(&apiErr)
			resp.Body.Close()

			if !apiErr.Retryable() || attempt == maxAttempts {
				return fmt.Errorf("server rejected batch: %s (%s)", apiErr.Message, apiErr.Code)
			}
			if apiErr.RetryAfter > 0 {
				backoff = time.Duration(apiErr.RetryAfter) * time.Second
			}
			e.sleep(ctx, &backoff)
		}
	}
	return fmt.Errorf("giving up after %d attempts", maxAttempts)
}

func (e *Exporter) sleep(ctx context.Context, backoff *time.Duration) {
	// Jitter stops a fleet of editors reconnecting in lockstep after an outage.
	jitter := time.Duration(rand.Int63n(int64(*backoff/2) + 1))
	select {
	case <-time.After(*backoff + jitter):
	case <-ctx.Done():
	}
	if *backoff < 30*time.Second {
		*backoff *= 2
	}
}

func (e *Exporter) ensureToken(ctx context.Context) (string, error) {
	e.mu.Lock()
	// Refresh a minute early rather than racing expiry mid-batch.
	if e.token != "" && time.Now().Before(e.expires.Add(-time.Minute)) {
		defer e.mu.Unlock()
		return e.token, nil
	}
	e.mu.Unlock()

	payload, _ := json.Marshal(map[string]string{
		"client_id":     e.cfg.ClientID,
		"client_secret": e.cfg.ClientSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.cfg.Endpoint+"/api/v1/auth/token", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var apiErr wire.APIError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return "", fmt.Errorf("token request failed: %s", apiErr.Message)
	}

	var tr struct {
		AccessToken string    `json:"access_token"`
		ExpiresAt   time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	e.mu.Lock()
	e.token, e.expires = tr.AccessToken, tr.ExpiresAt
	e.mu.Unlock()
	return tr.AccessToken, nil
}

func (e *Exporter) invalidateToken() {
	e.mu.Lock()
	e.token, e.expires = "", time.Time{}
	e.mu.Unlock()
}

// newEventID returns a stable per-event identifier.
//
// Generated once, at event creation - never at send time. Generating it per
// attempt would defeat the server's deduplication and turn every retry into
// duplicate rows.
func newEventID() string { return uuid.NewString() }
