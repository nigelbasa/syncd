package web

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/nyanhewe/syncd/internal/engine"
)

// HandleSyncPush receives events from a remote client and applies them.
func (s *Server) HandleSyncPush(c *fiber.Ctx) error {
	var payload engine.PushPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload: " + err.Error(),
		})
	}

	if len(payload.Events) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "no events in payload",
		})
	}

	if err := s.engine.ApplyIncomingEvents(c.Context(), payload.Events); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":         "applied_successfully",
		"events_applied": len(payload.Events),
	})
}

// HandleSyncPull returns new events since the client's cursor.
func (s *Server) HandleSyncPull(c *fiber.Ctx) error {
	cursorStr := c.Query("cursor", "0")
	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid cursor value",
		})
	}

	events, newCursor, err := s.engine.GetEventsSince(c.Context(), cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(engine.PullResponse{
		Events: events,
		Cursor: newCursor,
	})
}
