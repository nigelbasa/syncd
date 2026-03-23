package web

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/nyanhewe/syncd/internal/config"
	"github.com/nyanhewe/syncd/internal/engine"
)

// Server is the HTTP server for syncd.
type Server struct {
	app    *fiber.App
	engine *engine.Engine
	cfg    *config.Config
}

// NewServer initializes the Fiber app with all routes and middleware.
func NewServer(cfg *config.Config, eng *engine.Engine) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Syncd",
		ServerHeader: "Syncd",
	})

	s := &Server{
		app:    app,
		engine: eng,
		cfg:    cfg,
	}

	// Global middleware.
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New())

	// Health check.
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "mode": cfg.App.Mode})
	})

	// Sync API routes (protected).
	api := app.Group("/api/v1/sync", AuthGuard(cfg.Sync.APIKey))
	api.Post("/push", s.HandleSyncPush)
	api.Get("/pull", s.HandleSyncPull)

	// Admin routes (placeholder).
	admin := app.Group("/admin")
	admin.Get("/dashboard", s.HandleAdminDashboard)

	return s
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.App.Port)
	log.Printf("[web] Starting server on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
