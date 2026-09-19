package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"

	"github.com/schlunsen/wee-editor/internal/analytics"
	"github.com/schlunsen/wee-editor/internal/wtfwyt"
	"github.com/schlunsen/wtfwyt/server/pkg/detect"
)

// setupAnalyticsComponents initializes analytics-related components
func (s *Server) setupAnalyticsComponents() {
	s.conversationAnalyzer = analytics.NewConversationAnalyzer(s.claudeDir)
	s.conversationParser = analytics.NewConversationParser(s.repo)
	s.stateCalculator = analytics.NewStateCalculator()
	s.processDetector = analytics.NewProcessDetector()
	s.shellDetector = analytics.NewShellDetector()
	s.resetTracker = analytics.NewResetTracker(s.claudeDir)
	s.modelProviderLookup = analytics.NewModelProviderLookup()

	s.setupWTFWYTExport()
}

// setupWTFWYTExport wires credential-exposure export, if configured.
//
// Disabled by default. When disabled, New returns a nil exporter and
// SetExportHook is never called, so the parser behaves exactly as before.
func (s *Server) setupWTFWYTExport() {
	if s.config == nil {
		return
	}

	exporter, err := wtfwyt.New(s.config.WTFWYT, func(format string, args ...any) {
		pterm.Warning.Printfln(format, args...)
	})
	if err != nil {
		// Misconfiguration must not stop the editor from starting; say so
		// clearly and carry on without export.
		pterm.Warning.Printfln("wtfwyt export disabled: %v", err)
		return
	}
	if exporter == nil {
		return
	}

	// Surface a detection immediately.
	//
	// wee runs primarily as a web app, so the browser is the only place an
	// alert is actually seen - a terminal line scrolls past unread in a window
	// nobody is looking at.
	//
	// This goes out on the AGENT websocket (/agent/ws), not the general hub.
	// The general hub wraps payloads as {"event":...,"data":...}, and nothing
	// in the frontend reads that envelope; the agent socket delivers
	// {"type":...} messages, which is what useAgentWebSocket dispatches on.
	// Sending to the wrong one would have been silently invisible.
	//
	// The terminal line stays as a fallback for headless runs.
	exporter.SetAlert(func(f detect.Finding, actx wtfwyt.AlertContext) {
		if s.agentHandler != nil {
			// Only the masked hint and the fingerprint travel. The credential
			// is never put on the wire, including to the local browser.
			s.agentHandler.BroadcastGlobal(fiber.Map{
				"type":        "secret_finding",
				"severity":    string(f.Severity),
				"rule_name":   f.Name,
				"rule_id":     f.RuleID,
				"hint":        f.Hint,
				"fingerprint": f.Fingerprint,
				"source":      actx.Source,
				"tool_name":   actx.ToolName,
				"file_path":   actx.FilePath,
				"session_id":  actx.SessionID,
				"redacted":    true,
				"detected_at": time.Now().UTC().Format(time.RFC3339),
			})
		}

		if s.quiet {
			return
		}
		where := actx.Source
		if actx.ToolName != "" {
			where += " (" + actx.ToolName + ")"
		}
		if f.Severity == detect.SeverityCritical {
			pterm.Error.Printfln("credential exposed: %s (%s) in %s - rotate it", f.Name, f.Hint, where)
			return
		}
		pterm.Warning.Printfln("possible credential: %s (%s) in %s", f.Name, f.Hint, where)
	})

	s.wtfwytExporter = exporter
	s.conversationParser.SetExportHook(exporter)

	go exporter.Run(context.Background())

	mode := "findings only"
	if s.config.WTFWYT.ExportContent {
		mode = "findings and redacted content"
	}
	pterm.Info.Printfln("wtfwyt export enabled (%s) -> %s", mode, s.config.WTFWYT.Endpoint)
}
