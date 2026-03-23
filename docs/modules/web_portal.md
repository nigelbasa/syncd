# Module: Web Portal & API (`internal/web`)

## Description
The `web` module handles all external HTTP traffic for the Syncd daemon. Operating primarily when `syncd` is running in "Server" mode (usually attached to the PostgreSQL VPS), this module provides both the REST/WebSocket API for desktop clients to sync data, and an internal secure administrative GUI to monitor sync status, generate client API keys, and view logs.

## Key APIs & Functions

### Client Sync API (Protected by `Authorization: Bearer <API_KEY>`)
- `POST /api/v1/sync/push`: Desktop client sends its local "Outbox" events to the server.
- `GET /api/v1/sync/pull?cursor=<ID>`: Desktop client requests any new remote events since its last known cursor.

### `web.NewServer(port int, db Database, config Config) *Server`
Initializes the Fiber/Gin router, applies CORS and authentication middleware, and mounts API and Admin routes.

### `middleware.AuthGuard()`
Validates incoming JWTs or API Tokens for all protected routes.

## Admin Portal Concepts
The Admin UI will likely be embedded directly into the Go binary (using `go:embed`) to maintain the single-binary requirement.
Pages include:
- **Dashboard**: Current sync rate, active clients, and connection status.
- **Clients/Keys**: Generate new API tokens for new desktop installations.
- **Logs**: View recent CDC events or errors.

## Structure
```
internal/web/
├── server.go       (HTTP lifecycle and routing setup)
├── handlers_sync.go (Push/Pull event handlers)
├── handlers_admin.go (GUI backend endpoints)
├── auth.go         (JWT & API Key validation)
└── middleware.go   (CORS, Logging, Recovery)
```

## Snippets
### Push Handler Outline
```go
func (s *Server) HandleSyncPush(c *fiber.Ctx) error {
    var payload []types.SyncEvent
    if err := c.BodyParser(&payload); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
    }

    // Call the engine to apply these incoming remote events
    err := s.engine.ApplyIncomingEvents(c.Context(), payload)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.Status(200).JSON(fiber.Map{"status": "applied_successfully"})
}
```
