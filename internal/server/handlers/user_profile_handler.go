// Package handlers contains HTTP request handlers for the Wee server.
// This file implements user profile management endpoints.
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// UserProfileHandler handles user profile management endpoints
type UserProfileHandler struct {
	repo      *database.Repository
	userStore UserStoreInterface
}

// UserStoreInterface defines the interface for user store operations
type UserStoreInterface interface {
	GetUser(username string) (UserInterface, bool)
}

// UserInterface defines the interface for user objects
type UserInterface interface {
	SetAvatarID(avatarID *int64)
}

// UserProfileResponse represents the user profile response
type UserProfileResponse struct {
	Username   string `json:"username"`
	Email      string `json:"email,omitempty"`
	AvatarID   *int64 `json:"avatar_id,omitempty"`
	AuthMethod string `json:"auth_method"`
	IsAdmin    bool   `json:"is_admin"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// UpdateUserProfileRequest represents a request to update user profile
type UpdateUserProfileRequest struct {
	AvatarID *int64 `json:"avatar_id,omitempty"`
}

// NewUserProfileHandler creates a new user profile handler
func NewUserProfileHandler(repo *database.Repository, userStore UserStoreInterface) *UserProfileHandler {
	return &UserProfileHandler{
		repo:      repo,
		userStore: userStore,
	}
}

// HandleGetUserProfile retrieves the current user's profile
// GET /api/user/profile
func (h *UserProfileHandler) HandleGetUserProfile(c *fiber.Ctx) error {
	// Get username from context (set by auth middleware)
	username, ok := c.Locals("username").(string)
	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	// Get user from repository
	dbUser, err := h.repo.GetUser(username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user profile",
		})
	}

	if dbUser == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Convert to response format
	email := ""
	if dbUser.Email.Valid {
		email = dbUser.Email.String
	}

	var avatarID *int64
	if dbUser.AvatarID.Valid {
		avatarID = &dbUser.AvatarID.Int64
	}

	response := UserProfileResponse{
		Username:   dbUser.Username,
		Email:      email,
		AvatarID:   avatarID,
		AuthMethod: dbUser.AuthMethod,
		IsAdmin:    dbUser.IsAdmin,
		CreatedAt:  dbUser.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  dbUser.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(response)
}

// HandleUpdateUserProfile updates the current user's profile
// PUT /api/user/profile
func (h *UserProfileHandler) HandleUpdateUserProfile(c *fiber.Ctx) error {
	// Get username from context (set by auth middleware)
	username, ok := c.Locals("username").(string)
	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	// Parse request body
	var req UpdateUserProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate avatar ID if provided
	if req.AvatarID != nil {
		avatar, err := h.repo.GetAvatarByID(*req.AvatarID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to validate avatar",
			})
		}
		if avatar == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Avatar not found",
			})
		}
	}

	// Update user avatar in database
	err := h.repo.UpdateUserAvatar(username, req.AvatarID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user profile",
		})
	}

	// Update the in-memory user store as well if it exists
	if h.userStore != nil {
		if user, ok := h.userStore.GetUser(username); ok {
			user.SetAvatarID(req.AvatarID)
		}
	}

	// Return updated profile
	return h.HandleGetUserProfile(c)
}
