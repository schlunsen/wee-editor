// Package handlers contains HTTP request handlers for the Wee server.
// This file implements connector management endpoints for external service integrations.
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/connectors"
	"github.com/schlunsen/wee-editor/internal/database"
)

// ConnectorHandler handles connector management endpoints
type ConnectorHandler struct {
	repo          *database.Repository
	encryptionKey []byte
}

// NewConnectorHandler creates a new connector handler
func NewConnectorHandler(repo *database.Repository, encryptionKey []byte) *ConnectorHandler {
	return &ConnectorHandler{
		repo:          repo,
		encryptionKey: encryptionKey,
	}
}

// HandleGetConnectors returns all available connector definitions and user's connections
func (h *ConnectorHandler) HandleGetConnectors(c *fiber.Ctx) error {
	// Get all connector definitions
	definitions, err := h.repo.Connector.GetAllConnectorDefinitions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch connector definitions",
		})
	}

	// Get user's active connections
	connections, err := h.repo.Connector.GetAllConnections()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch connections",
		})
	}

	// Build connection map for quick lookup and mask API keys
	connectionMap := make(map[string]map[string]interface{})
	for _, conn := range connections {
		connData := map[string]interface{}{
			"id":             conn.ID,
			"connector_slug": conn.ConnectorSlug,
			"status":         conn.Status,
			"created_at":     conn.CreatedAt,
			"updated_at":     conn.UpdatedAt,
			"last_used_at":   conn.LastUsedAt,
		}

		// Mask the API key for display
		if conn.APIKey != nil && *conn.APIKey != "" {
			decrypted, err := connectors.Decrypt(*conn.APIKey, h.encryptionKey)
			if err == nil && decrypted != "" {
				connData["api_key_masked"] = connectors.MaskAPIKey(decrypted)
			}
		}

		if conn.StatusMessage != nil {
			connData["status_message"] = *conn.StatusMessage
		}
		if conn.ExternalAccountName != nil {
			connData["external_account_name"] = *conn.ExternalAccountName
		}
		if conn.ExtraConfig != nil {
			connData["extra_config"] = *conn.ExtraConfig
		}

		connectionMap[conn.ConnectorSlug] = connData
	}

	// Build response with definitions and their connection status
	var responseConnectors []map[string]interface{}
	for _, def := range definitions {
		connData := map[string]interface{}{
			"slug":        def.Slug,
			"name":        def.Name,
			"description": def.Description,
			"category":    def.Category,
			"auth_type":   def.AuthType,
			"icon":        def.Icon,
			"bg_class":    def.BgClass,
			"sort_order":  def.SortOrder,
			"is_active":   def.IsActive,
		}

		// Add connection info if connected
		if conn, exists := connectionMap[def.Slug]; exists {
			connData["connection"] = conn
			connData["is_connected"] = true
		} else {
			connData["is_connected"] = false
		}

		responseConnectors = append(responseConnectors, connData)
	}

	return c.JSON(fiber.Map{
		"connectors":       responseConnectors,
		"connection_count": len(connections),
		"total_count":      len(definitions),
	})
}

// ConnectRequest represents a request to connect a service
type ConnectRequest struct {
	APIKey    string `json:"api_key"`
	ExtraConfig string `json:"extra_config,omitempty"`
}

// HandleConnect creates or updates a connection for a connector
func (h *ConnectorHandler) HandleConnect(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Connector slug is required",
		})
	}

	var req ConnectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate connector exists
	def, err := h.repo.Connector.GetConnectorDefinition(slug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Connector not found: " + slug,
		})
	}

	// API key is required for api_key auth type
	if def.AuthType == "api_key" && req.APIKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key is required",
		})
	}

	// Encrypt the API key
	encryptedKey, err := connectors.Encrypt(req.APIKey, h.encryptionKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to encrypt API key",
		})
	}

	// Save the connection
	conn := &database.UserConnection{
		ConnectorSlug: slug,
		APIKey:        &encryptedKey,
		Status:        "active",
	}

	if req.ExtraConfig != "" {
		conn.ExtraConfig = &req.ExtraConfig
	}

	if err := h.repo.Connector.SaveConnection(conn); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save connection: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"message":       def.Name + " connected successfully",
		"connection_id": conn.ID,
	})
}

// HandleDisconnect removes a connection
func (h *ConnectorHandler) HandleDisconnect(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Connector slug is required",
		})
	}

	if err := h.repo.Connector.DeleteConnectionBySlug(slug); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disconnect: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Disconnected successfully",
	})
}

// HandleGetConnectionAPIKey retrieves the decrypted API key for a connector (used internally)
func (h *ConnectorHandler) HandleGetConnectionAPIKey(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Connector slug is required",
		})
	}

	conn, err := h.repo.Connector.GetConnectionBySlug(slug)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get connection",
		})
	}
	if conn == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No connection found for " + slug,
		})
	}

	// Decrypt the API key
	if conn.APIKey == nil || *conn.APIKey == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No API key stored for this connection",
		})
	}

	decrypted, err := connectors.Decrypt(*conn.APIKey, h.encryptionKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decrypt API key",
		})
	}

	// Update last used timestamp
	_ = h.repo.Connector.UpdateConnectionLastUsed(conn.ID)

	return c.JSON(fiber.Map{
		"api_key": decrypted,
	})
}
