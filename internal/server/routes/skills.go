package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterSkills configures skill management endpoints
func RegisterSkills(api fiber.Router, h *handlers.SkillsHandler) {
	skills := api.Group("/skills")

	skills.Get("/", h.HandleGetSkills)
	skills.Get("/templates", h.HandleGetSkillTemplates)
	skills.Post("/discover", h.HandleDiscoverSkills)
	skills.Get("/:name", h.HandleGetSkill)
	skills.Post("/", h.HandleCreateSkill)
	skills.Put("/:name", h.HandleUpdateSkill)
	skills.Delete("/:name", h.HandleDeleteSkill)
}
