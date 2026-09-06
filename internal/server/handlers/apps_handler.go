package handlers

import (
	"database/sql"
	"fmt"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// SandboxApp represents a registered app served on a sub-subdomain
type SandboxApp struct {
	Name string `json:"name"` // e.g. "app" → app.xyz2.wee.cat
	Port int    `json:"port"` // internal port the app listens on
}

// AppsHandler manages sandbox app sub-subdomains
type AppsHandler struct {
	subdomain string // this sandbox's subdomain (e.g. "xyz2")
	apps      map[string]*SandboxApp
	mu        sync.RWMutex
	sqlDB     *sql.DB // optional: if set, apps are persisted to SQLite
}

// NewAppsHandler creates a new apps handler
func NewAppsHandler() *AppsHandler {
	subdomain := os.Getenv("WEE_SUBDOMAIN")
	return &AppsHandler{
		subdomain: subdomain,
		apps:      make(map[string]*SandboxApp),
	}
}

// SetDatabase sets the database for app persistence and loads existing apps.
// Must be called after database initialization.
func (h *AppsHandler) SetDatabase(sqlDB *sql.DB) {
	h.sqlDB = sqlDB
	if err := h.loadAppsFromDB(); err != nil {
		logging.Error("Failed to load sandbox apps from database: %v", err)
	}
}

// loadAppsFromDB loads all persisted apps from the database into memory
func (h *AppsHandler) loadAppsFromDB() error {
	if h.sqlDB == nil {
		return nil
	}

	rows, err := h.sqlDB.Query("SELECT name, port FROM sandbox_apps")
	if err != nil {
		return fmt.Errorf("query sandbox_apps: %w", err)
	}
	defer rows.Close()

	h.mu.Lock()
	defer h.mu.Unlock()

	count := 0
	for rows.Next() {
		var app SandboxApp
		if err := rows.Scan(&app.Name, &app.Port); err != nil {
			logging.Error("Failed to scan sandbox app row: %v", err)
			continue
		}
		h.apps[app.Name] = &app
		count++
	}

	if count > 0 {
		logging.Info("Loaded %d sandbox app(s) from database", count)
	}

	return rows.Err()
}

// persistApp saves an app to the database (upsert)
func (h *AppsHandler) persistApp(app *SandboxApp) {
	if h.sqlDB == nil {
		return
	}
	_, err := h.sqlDB.Exec(
		"INSERT OR REPLACE INTO sandbox_apps (name, port) VALUES (?, ?)",
		app.Name, app.Port,
	)
	if err != nil {
		logging.Error("Failed to persist sandbox app %q: %v", app.Name, err)
	}
}

// removeAppFromDB removes an app from the database
func (h *AppsHandler) removeAppFromDB(name string) {
	if h.sqlDB == nil {
		return
	}
	_, err := h.sqlDB.Exec("DELETE FROM sandbox_apps WHERE name = ?", name)
	if err != nil {
		logging.Error("Failed to remove sandbox app %q from database: %v", name, err)
	}
}

// validAppName checks that an app name is safe for use as a subdomain
var validAppName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// RegisterApp registers an app programmatically (used by MCP tools)
func (h *AppsHandler) RegisterApp(app *SandboxApp) {
	h.mu.Lock()
	h.apps[app.Name] = app
	h.mu.Unlock()
	h.persistApp(app)
}

// ListApps returns all registered apps (used by MCP tools)
func (h *AppsHandler) ListApps() []*SandboxApp {
	h.mu.RLock()
	defer h.mu.RUnlock()
	apps := make([]*SandboxApp, 0, len(h.apps))
	for _, app := range h.apps {
		apps = append(apps, app)
	}
	return apps
}

// RemoveApp removes an app by name (used by MCP tools). Returns false if not found.
func (h *AppsHandler) RemoveApp(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.apps[name]; !exists {
		return false
	}
	delete(h.apps, name)
	h.removeAppFromDB(name)
	return true
}

// HandleRegisterApp registers a new app subdomain
// POST /api/apps
func (h *AppsHandler) HandleRegisterApp(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if !validAppName.MatchString(body.Name) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid app name: must be lowercase alphanumeric with optional hyphens"})
	}

	if body.Port < 1024 || body.Port > 65535 {
		return c.Status(400).JSON(fiber.Map{"error": "port must be between 1024 and 65535"})
	}

	app := &SandboxApp{Name: body.Name, Port: body.Port}

	h.mu.Lock()
	h.apps[body.Name] = app
	h.mu.Unlock()

	h.persistApp(app)

	appURL := ""
	if h.subdomain != "" {
		appURL = fmt.Sprintf("https://%s.%s.wee.cat", body.Name, h.subdomain)
	}

	return c.JSON(fiber.Map{
		"app": app,
		"url": appURL,
	})
}

// HandleListApps lists all registered apps
// GET /api/apps
func (h *AppsHandler) HandleListApps(c *fiber.Ctx) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	apps := make([]*SandboxApp, 0, len(h.apps))
	for _, app := range h.apps {
		apps = append(apps, app)
	}

	return c.JSON(fiber.Map{
		"apps":      apps,
		"subdomain": h.subdomain,
	})
}

// HandleDeleteApp removes a registered app
// DELETE /api/apps/:name
func (h *AppsHandler) HandleDeleteApp(c *fiber.Ctx) error {
	name := c.Params("name")

	h.mu.Lock()
	_, exists := h.apps[name]
	if !exists {
		h.mu.Unlock()
		return c.Status(404).JSON(fiber.Map{"error": "app not found"})
	}
	delete(h.apps, name)
	h.mu.Unlock()

	h.removeAppFromDB(name)

	return c.JSON(fiber.Map{"message": "app removed"})
}

// AppProxyMiddleware returns Fiber middleware that intercepts requests to
// app sub-subdomains (e.g. app.xyz2.wee.cat) and proxies them to the
// registered internal port.
func (h *AppsHandler) AppProxyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if h.subdomain == "" {
			return c.Next()
		}

		host := strings.ToLower(c.Hostname())

		// Check if this is a sub-subdomain: {app}.{subdomain}.wee.cat
		suffix := "." + h.subdomain + ".wee.cat"
		if !strings.HasSuffix(host, suffix) {
			return c.Next()
		}

		// Extract the app name
		appName := strings.TrimSuffix(host, suffix)
		if appName == "" || strings.Contains(appName, ".") {
			return c.Next()
		}

		h.mu.RLock()
		app, exists := h.apps[appName]
		h.mu.RUnlock()

		if !exists {
			// No app registered for this subdomain — fall through to the
			// normal wee server so the dashboard is still accessible via
			// any *.{subdomain}.wee.cat URL
			return c.Next()
		}

		// Reverse proxy to the app's internal port
		target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", app.Port))
		proxy := httputil.NewSingleHostReverseProxy(target)

		// Use fasthttpadaptor to bridge Fiber ↔ net/http
		handler := fasthttpadaptor.NewFastHTTPHandler(proxy)
		handler(c.Context())
		return nil
	}
}
