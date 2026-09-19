package server

import (
	"context"

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

	// Surface a detection immediately, in the terminal. Whoever just pasted a
	// live credential needs to know now - not when someone opens a dashboard.
	exporter.SetAlert(func(f detect.Finding, where string) {
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
