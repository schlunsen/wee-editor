// Package handlers contains HTTP request handlers for the Wee server.
// This file implements user settings management endpoints.
package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	ws "github.com/schlunsen/wee-editor/internal/websocket"
)

// SettingsHandler handles user settings management endpoints
type SettingsHandler struct {
	repo  *database.Repository
	wsHub *ws.Hub
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(repo *database.Repository, wsHub *ws.Hub) *SettingsHandler {
	return &SettingsHandler{
		repo:  repo,
		wsHub: wsHub,
	}
}

// HandleGetAllSettings returns all user settings
func (h *SettingsHandler) HandleGetAllSettings(c *fiber.Ctx) error {
	settings, err := h.repo.GetAllUserSettings()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"settings": settings,
		"count":    len(settings),
	})
}

// HandleGetSetting returns a specific user setting by key
func (h *SettingsHandler) HandleGetSetting(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "setting key is required",
		})
	}

	setting, err := h.repo.GetUserSetting(key)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("setting not found: %v", err),
		})
	}

	return c.JSON(setting)
}

// HandleUpdateSetting updates or creates a user setting
func (h *SettingsHandler) HandleUpdateSetting(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "setting key is required",
		})
	}

	type UpdateSettingRequest struct {
		Value       string `json:"value"`
		ValueType   string `json:"value_type,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var req UpdateSettingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Default to string type if not specified
	if req.ValueType == "" {
		req.ValueType = "string"
	}

	setting := &database.UserSetting{
		Key:         key,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Description: req.Description,
	}

	if err := h.repo.SetUserSetting(setting); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to update setting: %v", err),
		})
	}

	// Broadcast update to WebSocket clients
	h.wsHub.BroadcastData("setting_updated", fiber.Map{
		"key":   key,
		"value": req.Value,
	})

	return c.JSON(fiber.Map{
		"status":  "updated",
		"setting": setting,
	})
}

// HandleDeleteSetting deletes a user setting
func (h *SettingsHandler) HandleDeleteSetting(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "setting key is required",
		})
	}

	if err := h.repo.DeleteUserSetting(key); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to delete setting: %v", err),
		})
	}

	// Broadcast update to WebSocket clients
	h.wsHub.BroadcastData("setting_deleted", fiber.Map{
		"key": key,
	})

	return c.JSON(fiber.Map{
		"status": "deleted",
		"key":    key,
	})
}
