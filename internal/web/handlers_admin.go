package web

import (
	"github.com/gofiber/fiber/v2"
)

// HandleAdminDashboard returns a JSON summary of sync status.
// A full embedded HTML UI can be added in a future iteration.
func (s *Server) HandleAdminDashboard(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service": "syncd",
		"mode":    s.cfg.App.Mode,
		"status":  "running",
		"message": "Admin UI coming soon. Use the API endpoints for now.",
	})
}
