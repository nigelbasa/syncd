package web

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthGuard returns middleware that validates Bearer API tokens.
func AuthGuard(validKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing Authorization header",
			})
		}

		// Expect "Bearer <token>".
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid Authorization format, expected: Bearer <token>",
			})
		}

		if parts[1] != validKey {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}

		return c.Next()
	}
}
