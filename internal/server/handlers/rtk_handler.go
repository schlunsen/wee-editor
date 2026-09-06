// Package handlers contains HTTP request handlers for the Wee server.
// This file implements RTK (Rust Token Killer) stats endpoints that
// surface compression savings from rtk's local tracking database and
// the `rtk gain --format json` command.
package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schlunsen/claude-agent-sdk-go/middleware/rtk"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// RTKHandler exposes endpoints that surface rtk compression savings.
//
// Aggregate stats are read via the SDK's rtk.Gain() wrapper around
// `rtk gain --format json`. Per-session stats are read directly from
// rtk's SQLite tracking DB so we can filter by session start time
// (which rtk gain does not expose as a flag).
type RTKHandler struct {
	sessionManager *agents.SessionManager
	gainTimeout    time.Duration
}

// NewRTKHandler constructs an RTKHandler. sessionManager is used to
// look up sessions by ID for the per-session endpoint.
func NewRTKHandler(sessionManager *agents.SessionManager) *RTKHandler {
	return &RTKHandler{
		sessionManager: sessionManager,
		gainTimeout:    10 * time.Second,
	}
}

// RTKStatusResponse is returned by GET /api/rtk/status.
type RTKStatusResponse struct {
	Installed      bool   `json:"installed"`
	TrackingDB     string `json:"tracking_db"`
	TrackingDBOK   bool   `json:"tracking_db_ok"`
	TrackingDBSize int64  `json:"tracking_db_size_bytes,omitempty"`
}

// HandleGetStatus reports whether rtk is installed on PATH and whether
// its tracking DB exists. The UI uses this to hide/gate RTK widgets
// without first failing a stats call.
func (h *RTKHandler) HandleGetStatus(c *fiber.Ctx) error {
	installed := rtk.IsInstalled("")
	dbPath, _ := rtk.TrackingDBPath()

	resp := RTKStatusResponse{
		Installed:  installed,
		TrackingDB: dbPath,
	}
	if dbPath != "" {
		if info, err := os.Stat(dbPath); err == nil && !info.IsDir() {
			resp.TrackingDBOK = true
			resp.TrackingDBSize = info.Size()
		}
	}
	return c.JSON(resp)
}

// HandleGetGain proxies `rtk gain --format json` and returns the
// upstream payload verbatim (Summary + optional Daily/Weekly/Monthly
// breakdowns). Query params:
//   - breakdown=all|daily|weekly|monthly (default: none — summary only)
//   - project=<path> (optional; rtk's --project filter)
func (h *RTKHandler) HandleGetGain(c *fiber.Ctx) error {
	if !rtk.IsInstalled("") {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":     "rtk binary not installed",
			"installed": false,
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), h.gainTimeout)
	defer cancel()

	opts := []rtk.GainOption{}
	switch strings.ToLower(c.Query("breakdown")) {
	case "all":
		opts = append(opts, rtk.WithAll())
	case "daily":
		opts = append(opts, rtk.WithDaily())
	case "weekly":
		opts = append(opts, rtk.WithWeekly())
	case "monthly":
		opts = append(opts, rtk.WithMonthly())
	case "":
		// summary only
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "breakdown must be one of: all, daily, weekly, monthly",
		})
	}
	if project := strings.TrimSpace(c.Query("project")); project != "" {
		opts = append(opts, rtk.WithProject(project))
	}

	export, err := rtk.Gain(ctx, opts...)
	if err != nil {
		if errors.Is(err, rtk.ErrRTKNotInstalled) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":     "rtk binary not installed",
				"installed": false,
			})
		}
		logging.Warning("rtk gain failed: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(export)
}

// SessionStatsResponse is returned by GET /api/rtk/session-stats.
// It aggregates rows from rtk's tracking.db filtered to rows created
// at or after the session's CreatedAt. The project filter is a prefix
// match on the session's worktree path (or cwd if no worktree).
type SessionStatsResponse struct {
	SessionID     uuid.UUID        `json:"session_id"`
	SessionStart  time.Time        `json:"session_start"`
	Commands      int64            `json:"commands"`
	InputTokens   int64            `json:"input_tokens"`
	OutputTokens  int64            `json:"output_tokens"`
	SavedTokens   int64            `json:"saved_tokens"`
	SavingsPct    float64          `json:"savings_pct"`
	TotalTimeMs   int64            `json:"total_time_ms"`
	ProjectFilter string           `json:"project_filter,omitempty"`
	Recent        []RecentRTKCmd   `json:"recent,omitempty"`
	TrackingDB    string           `json:"tracking_db"`
	TrackingDBOK  bool             `json:"tracking_db_ok"`
}

// RecentRTKCmd is one row from the `commands` table. Only the most
// recent N rows are returned; original_cmd is truncated to keep the
// payload small.
type RecentRTKCmd struct {
	Timestamp    time.Time `json:"timestamp"`
	OriginalCmd  string    `json:"original_cmd"`
	SavedTokens  int64     `json:"saved_tokens"`
	SavingsPct   float64   `json:"savings_pct"`
	ExecTimeMs   int64     `json:"exec_time_ms"`
}

// HandleGetSessionStats returns RTK savings aggregated for a single
// session, computed directly from rtk's SQLite tracking database.
// Query params: session_id=<uuid> (required).
func (h *RTKHandler) HandleGetSessionStats(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Query("session_id"))
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id query param is required",
		})
	}
	sid, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is not a valid UUID",
		})
	}
	if h.sessionManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "agent session manager not available",
		})
	}
	sess, err := h.sessionManager.GetSession(sid)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("session %s not found", sid),
		})
	}

	dbPath, err := rtk.TrackingDBPath()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	resp := SessionStatsResponse{
		SessionID:    sid,
		SessionStart: sess.CreatedAt,
		TrackingDB:   dbPath,
	}
	info, err := os.Stat(dbPath)
	if err != nil || info.IsDir() {
		// DB doesn't exist yet — return a zeroed response rather than 404
		// so the frontend can render "no savings yet" without special-casing.
		return c.JSON(resp)
	}
	resp.TrackingDBOK = true

	// Pick a project filter. Worktree path is more precise than the
	// bare project root (sessions in worktrees emit rows with their
	// worktree path, not the project root).
	projectFilter := sess.WorktreePath
	if projectFilter == "" && sess.ProjectID != nil {
		// No worktree — rtk's project_path is the cwd at command time,
		// which for a regular session is the project dir itself. We
		// don't currently have the project path directly on the
		// session, so skip the filter in that case. The session_start
		// time filter still gives a decent approximation.
		projectFilter = ""
	}
	resp.ProjectFilter = projectFilter

	db, err := openReadOnlySQLite(dbPath)
	if err != nil {
		logging.Warning("rtk: open tracking db %q: %v", dbPath, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "failed to open rtk tracking database",
		})
	}
	defer db.Close()

	// Aggregate query. Time column is stored as TEXT (RFC3339 UTC) in rtk;
	// SQLite's string comparison is lexicographic which matches ISO order.
	since := sess.CreatedAt.UTC().Format(time.RFC3339)
	aggArgs := []any{since}
	aggWhere := "timestamp >= ?"
	if projectFilter != "" {
		aggWhere += " AND project_path GLOB ?"
		aggArgs = append(aggArgs, projectFilter+"*")
	}
	aggSQL := `SELECT
		COALESCE(COUNT(*), 0),
		COALESCE(SUM(input_tokens), 0),
		COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(saved_tokens), 0),
		COALESCE(AVG(savings_pct), 0),
		COALESCE(SUM(exec_time_ms), 0)
	FROM commands WHERE ` + aggWhere
	row := db.QueryRowContext(c.Context(), aggSQL, aggArgs...)
	if err := row.Scan(
		&resp.Commands, &resp.InputTokens, &resp.OutputTokens,
		&resp.SavedTokens, &resp.SavingsPct, &resp.TotalTimeMs,
	); err != nil && !errors.Is(err, sql.ErrNoRows) {
		logging.Warning("rtk: aggregate query failed: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "failed to read rtk tracking database",
		})
	}

	// Recent rows (most recent 20). Same filter as the aggregate.
	recentArgs := append([]any{}, aggArgs...)
	recentSQL := `SELECT timestamp, original_cmd, saved_tokens, savings_pct,
		COALESCE(exec_time_ms, 0)
	FROM commands WHERE ` + aggWhere + ` ORDER BY id DESC LIMIT 20`
	rows, err := db.QueryContext(c.Context(), recentSQL, recentArgs...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var (
				tsRaw string
				rc    RecentRTKCmd
			)
			if err := rows.Scan(&tsRaw, &rc.OriginalCmd, &rc.SavedTokens, &rc.SavingsPct, &rc.ExecTimeMs); err != nil {
				continue
			}
			if t, perr := time.Parse(time.RFC3339, tsRaw); perr == nil {
				rc.Timestamp = t
			}
			// Truncate long commands to keep payload bounded.
			if len(rc.OriginalCmd) > 200 {
				rc.OriginalCmd = rc.OriginalCmd[:197] + "..."
			}
			resp.Recent = append(resp.Recent, rc)
		}
	}

	return c.JSON(resp)
}

// openReadOnlySQLite opens rtk's tracking.db in read-only mode with
// immutable=1 so we never attempt schema changes or take write locks
// that could conflict with a running rtk process.
func openReadOnlySQLite(path string) (*sql.DB, error) {
	// Use the file: URI form so we can pass mode=ro and immutable=1.
	// filepath.ToSlash keeps Windows paths valid in the URI.
	uri := "file:" + filepath.ToSlash(path) + "?mode=ro&immutable=1&_query_only=1"
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		return nil, err
	}
	// Cap connections so we don't pool reads against rtk's WAL file.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
