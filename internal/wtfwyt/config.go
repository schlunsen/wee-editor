// Package wtfwyt exports session activity to a wtfwyt server for credential
// exposure monitoring.
//
// The package answers one question: did a secret end up in a chat transcript?
//
// # Secrets never leave the machine
//
// Detection runs here, in the client, before anything is transmitted. Content
// is scanned, every credential found is replaced with an opaque placeholder,
// and only a finding is sent - rule name, severity, and a truncated SHA-256
// fingerprint of the value.
//
// This ordering is the whole design. Shipping raw transcripts to a server and
// scanning them there would turn that server into a durable central store of
// every secret anyone ever pasted into a chat: a far worse exposure than the
// one being reported. The fingerprint is enough to tell you the same key has
// leaked in six sessions, and useless to anyone who obtains it.
//
// Export is off by default and does nothing until configured.
package wtfwyt

import (
	"errors"
	"os"
	"strings"
	"time"
)

// Config holds the wtfwyt export settings.
//
// Following the convention used by TunnelSettings, the client secret is never
// persisted to the config file. It is read from WTFWYT_CLIENT_SECRET so a
// credential does not end up in a JSON file on disk - which would be an
// unusually poor look for this particular feature.
type Config struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint,omitempty"`
	ClientID string `json:"client_id,omitempty"`

	// ClientSecret is loaded from the environment, never persisted.
	ClientSecret string `json:"-"`

	// ExportContent controls whether redacted message and tool content is
	// sent at all.
	//
	// When false, only credential findings are exported: the fingerprint, the
	// rule, and where it surfaced. No transcript content of any kind leaves
	// the machine. This is the setting for people who want the secret
	// scanning without the session history, and it is the default.
	ExportContent bool `json:"export_content"`

	// FlushInterval bounds how long an event waits before being sent.
	FlushInterval Duration `json:"flush_interval,omitempty"`
	// FlushSize is the number of events that triggers an immediate send.
	FlushSize int `json:"flush_size,omitempty"`
	// BufferSize bounds memory when the server is unreachable.
	BufferSize int `json:"buffer_size,omitempty"`

	// AlertOnCritical surfaces a local warning the moment a critical
	// credential is detected, rather than waiting for someone to open a
	// dashboard. The person who just pasted a live AWS key needs to know now.
	AlertOnCritical bool `json:"alert_on_critical"`
}

// Duration is a time.Duration that marshals as a string ("5s") in JSON.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// DefaultConfig returns settings with export disabled.
//
// The defaults are deliberately conservative: findings only, no transcript
// content. Turning on content export is an explicit decision.
func DefaultConfig() Config {
	return Config{
		Enabled:         false,
		Endpoint:        "https://wtfwyt.com",
		ExportContent:   false,
		FlushInterval:   Duration(5 * time.Second),
		FlushSize:       200,
		BufferSize:      10000,
		AlertOnCritical: true,
	}
}

// LoadSecret reads the client secret from the environment.
func (c *Config) LoadSecret() {
	if c.ClientSecret == "" {
		c.ClientSecret = os.Getenv("WTFWYT_CLIENT_SECRET")
	}
}

// Validate reports whether the configuration is usable.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Endpoint == "" {
		return errors.New("wtfwyt: endpoint is required when export is enabled")
	}
	if !strings.HasPrefix(c.Endpoint, "https://") &&
		!strings.HasPrefix(c.Endpoint, "http://localhost") &&
		!strings.HasPrefix(c.Endpoint, "http://127.0.0.1") {
		// Findings carry no secrets, but they do reveal which credentials
		// exist and where - not something to send over plaintext to a remote
		// host. Local endpoints are allowed for development.
		return errors.New("wtfwyt: endpoint must be https (or a localhost address)")
	}
	if c.ClientID == "" {
		return errors.New("wtfwyt: client_id is required when export is enabled")
	}
	if c.ClientSecret == "" {
		return errors.New("wtfwyt: client secret missing; set WTFWYT_CLIENT_SECRET")
	}
	return nil
}

func (c *Config) withDefaults() {
	d := DefaultConfig()
	if c.FlushInterval == 0 {
		c.FlushInterval = d.FlushInterval
	}
	if c.FlushSize <= 0 {
		c.FlushSize = d.FlushSize
	}
	if c.BufferSize <= 0 {
		c.BufferSize = d.BufferSize
	}
	if c.Endpoint == "" {
		c.Endpoint = d.Endpoint
	}
}
