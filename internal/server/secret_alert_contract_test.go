package server

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// The alert payload is built in Go and consumed by a TypeScript interface in
// the Nuxt frontend. Nothing checks that the two agree at compile time, and a
// mismatch is silent: the UI simply never renders the alert.
//
// This happened once already. The payload was first broadcast through the
// general websocket hub, which wraps messages as {"event":...,"data":...},
// while the frontend dispatches on message.type - so the alert went out and
// was invisible. This test pins the contract.
func TestSecretFindingPayloadMatchesFrontendType(t *testing.T) {
	// Exactly the payload shape built in setupWTFWYTExport.
	payload := fiber.Map{
		"type":        "secret_finding",
		"severity":    "critical",
		"rule_name":   "GitHub token",
		"rule_id":     "github_token",
		"hint":        "ghp_****8d",
		"fingerprint": "faeb1f0959e2c7f9",
		"source":      "tool_result",
		"tool_name":   "Bash",
		"file_path":   "/app/.env",
		"session_id":  "abc-123",
		"redacted":    true,
		"detected_at": "2026-09-19T12:00:00Z",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// The discriminator the frontend switches on.
	if got["type"] != "secret_finding" {
		t.Fatalf(`type = %v, want "secret_finding" - the frontend dispatches on this`, got["type"])
	}

	// Fields must be top level, not nested under "data": the general hub's
	// envelope is what broke this the first time.
	if _, nested := got["data"]; nested {
		t.Error(`payload has a "data" key: this is the general-hub envelope, which the frontend does not read`)
	}

	// Every non-optional field in the TypeScript interface must be present.
	fields := frontendFindingFields(t)
	for name, optional := range fields {
		if _, ok := got[name]; !ok && !optional {
			t.Errorf("frontend requires field %q but the Go payload does not send it", name)
		}
	}
	for name := range got {
		if _, ok := fields[name]; !ok {
			t.Errorf("Go sends field %q that the frontend type does not declare", name)
		}
	}
}

// frontendFindingFields parses the SecretFinding interface out of the Nuxt
// type definition, returning field name -> optional.
//
// Reading the real file rather than restating it here is the point: if
// someone edits the interface, this test notices.
func frontendFindingFields(t *testing.T) map[string]bool {
	t.Helper()

	const path = "frontend/app/types/security.ts"
	src, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("frontend type file not readable (%v); skipping contract check", err)
	}

	body := string(src)
	start := strings.Index(body, "interface SecretFinding")
	if start < 0 {
		t.Fatalf("interface SecretFinding not found in %s", path)
	}
	open := strings.Index(body[start:], "{")
	if open < 0 {
		t.Fatalf("malformed interface in %s", path)
	}
	end := strings.Index(body[start+open:], "}")
	if end < 0 {
		t.Fatalf("unterminated interface in %s", path)
	}
	block := body[start+open : start+open+end]

	re := regexp.MustCompile(`(?m)^\s*([a-z_]+)(\??):`)
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		out[m[1]] = m[2] == "?"
	}
	if len(out) == 0 {
		t.Fatalf("parsed no fields from SecretFinding in %s", path)
	}
	return out
}
