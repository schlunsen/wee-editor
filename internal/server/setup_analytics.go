package server

import (
	"github.com/schlunsen/wee-editor/internal/analytics"
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
}
