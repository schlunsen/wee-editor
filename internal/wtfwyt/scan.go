package wtfwyt

import (
	"time"

	"github.com/schlunsen/wee-editor/internal/analytics"
	"github.com/schlunsen/wtfwyt/server/pkg/detect"
	"github.com/schlunsen/wtfwyt/server/pkg/wire"
)

// ExportToolExecution scans a completed tool execution and exports what it
// finds.
//
// Tool results are the highest-yield source of credential exposure in
// practice. Nobody types their AWS key into a chat; they run `cat .env`,
// `env | grep`, `git config --list`, or `kubectl get secret -o yaml`, and the
// output lands in the transcript - and from there in the model provider's
// logs. This is the function that catches that.
//
// The ToolExecution is not mutated: its data is local to this machine, where
// the secret already exists, so redacting it locally would degrade wee-editor
// without improving anyone's security. Redaction applies to what is
// transmitted.
func (e *Exporter) ExportToolExecution(t *analytics.ToolExecution) {
	if e == nil || t == nil {
		return
	}

	findings := detect.Scan(t.Result)

	// Tool arguments carry credentials too: a curl with an -H header, a psql
	// connection string, an API call with a token parameter.
	for _, v := range t.Input {
		if sv, ok := v.(string); ok && sv != "" {
			findings = append(findings, detect.Scan(sv)...)
		}
	}

	ts := t.ExecutedAt
	if ts.IsZero() {
		ts = time.Now()
	}

	for _, f := range findings {
		e.reportFinding(f, t.ConversationID, ts, wire.SourceToolResult, t.ToolName, filePathOf(t))
	}

	if !e.cfg.ExportContent {
		return
	}

	// Content export: send the tool execution with every credential replaced
	// by a placeholder.
	redactedResult, _ := detect.Redact(t.Result)
	input := make(map[string]any, len(t.Input))
	for k, v := range t.Input {
		if sv, ok := v.(string); ok {
			clean, _ := detect.Redact(sv)
			input[k] = clean
			continue
		}
		input[k] = v
	}

	e.send(wire.Event{
		Type:      wire.EventToolExecution,
		ID:        newEventID(),
		SessionID: t.ConversationID,
		Timestamp: ts,
		Tool: &wire.ToolExecution{
			ToolID:           t.ToolID,
			ToolName:         t.ToolName,
			Input:            input,
			Result:           redactedResult,
			Success:          t.Success,
			WorkingDirectory: t.WorkingDirectory,
			GitBranch:        t.GitBranch,
		},
	})
}

// ExportMessage scans a conversation turn and exports what it finds.
//
// Thinking blocks are scanned deliberately: a model reasoning about a key it
// has just read will quote it back.
func (e *Exporter) ExportMessage(sessionID, role, text, thinking, gitBranch, cwd string, ts time.Time) {
	if e == nil {
		return
	}
	if ts.IsZero() {
		ts = time.Now()
	}

	source := wire.SourceAssistantMessage
	if role == "user" {
		source = wire.SourceUserMessage
	}

	for _, part := range []string{text, thinking} {
		for _, f := range detect.Scan(part) {
			e.reportFinding(f, sessionID, ts, source, "", "")
		}
	}

	if !e.cfg.ExportContent {
		return
	}

	cleanText, _ := detect.Redact(text)
	cleanThinking, _ := detect.Redact(thinking)

	var content []wire.ContentBlock
	if cleanText != "" {
		content = append(content, wire.ContentBlock{Type: "text", Text: cleanText})
	}
	if cleanThinking != "" {
		content = append(content, wire.ContentBlock{Type: "thinking", Thinking: cleanThinking})
	}
	if len(content) == 0 {
		return
	}

	e.send(wire.Event{
		Type:      wire.EventMessage,
		ID:        newEventID(),
		SessionID: sessionID,
		Timestamp: ts,
		Message: &wire.Message{
			UUID:      newEventID(),
			Role:      wire.Role(role),
			Content:   content,
			CWD:       cwd,
			GitBranch: gitBranch,
		},
	})
}

// ExportLog scans a log line and exports what it finds.
func (e *Exporter) ExportLog(sessionID, level, message, file string, ts time.Time) {
	if e == nil {
		return
	}
	if ts.IsZero() {
		ts = time.Now()
	}

	for _, f := range detect.Scan(message) {
		e.reportFinding(f, sessionID, ts, wire.SourceToolResult, "", file)
	}

	if !e.cfg.ExportContent {
		return
	}
	clean, _ := detect.Redact(message)
	e.send(wire.Event{
		Type:      wire.EventLog,
		ID:        newEventID(),
		SessionID: sessionID,
		Timestamp: ts,
		Log:       &wire.LogEntry{Level: level, Message: clean, File: file},
	})
}

// reportFinding queues a finding and fires the local alert.
//
// The alert runs first and synchronously: someone who has just pasted a live
// credential needs to know now, not when the batch flushes.
func (e *Exporter) reportFinding(f detect.Finding, sessionID string, ts time.Time, source wire.SecretSource, tool, file string) {
	e.mu.Lock()
	alert := e.alert
	e.mu.Unlock()

	if alert != nil && (e.cfg.AlertOnCritical || f.Severity != detect.SeverityCritical) {
		alert(f, AlertContext{
			Source:    string(source),
			ToolName:  tool,
			FilePath:  file,
			SessionID: sessionID,
		})
	}

	wf := wire.FindingFromDetect(f, source)
	wf.ToolName = tool
	wf.FilePath = file

	e.send(wire.SecretEvent(newEventID(), sessionID, ts, wf))
}

func filePathOf(t *analytics.ToolExecution) string {
	if p, ok := t.Input["file_path"].(string); ok {
		return p
	}
	return ""
}
