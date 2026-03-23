package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nyanhewe/syncd/internal/config"
	"github.com/nyanhewe/syncd/internal/db"
)

// Engine is the main sync daemon orchestrating watch→push and pull→apply cycles.
type Engine struct {
	cfg         *config.Config
	db          db.Database
	watcher     *Watcher
	transporter *Transporter
	applicator  *Applicator
	cursor      int64 // Last synced cursor for pull operations
}

// New creates a new Engine with all dependencies wired up.
func New(cfg *config.Config, database db.Database) *Engine {
	resolver := NewConflictResolver()
	clientID := generateClientID()

	return &Engine{
		cfg:         cfg,
		db:          database,
		watcher:     NewWatcher(database, 100),
		transporter: NewTransporter(cfg.App.RemoteURL, cfg.Sync.APIKey, clientID),
		applicator:  NewApplicator(database, resolver),
		cursor:      0,
	}
}

// NewServerEngine creates an engine suitable for server mode
// (no transporter, receives push requests via web API instead).
func NewServerEngine(cfg *config.Config, database db.Database) *Engine {
	resolver := NewConflictResolver()

	return &Engine{
		cfg:        cfg,
		db:         database,
		watcher:    NewWatcher(database, 100),
		applicator: NewApplicator(database, resolver),
	}
}

// Start runs the background sync loop until the context is cancelled.
// In client mode: watches outbox → pushes to server, then pulls from server → applies locally.
// In server mode: only watches outbox for events pushed by clients via the API.
func (e *Engine) Start(ctx context.Context) error {
	log.Printf("[engine] Starting sync daemon (mode=%s, poll=%s)",
		e.cfg.App.Mode, e.cfg.Sync.PollInterval)

	ticker := time.NewTicker(e.cfg.Sync.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[engine] Shutting down sync daemon...")
			return nil
		case <-ticker.C:
			if err := e.tick(ctx); err != nil {
				log.Printf("[engine] Tick error: %v", err)
			}
		}
	}
}

// tick runs one sync cycle.
func (e *Engine) tick(ctx context.Context) error {
	if e.cfg.App.Mode == "client" {
		return e.clientTick(ctx)
	}
	return e.serverTick(ctx)
}

// clientTick: push local changes, then pull remote changes.
func (e *Engine) clientTick(ctx context.Context) error {
	// 1. Push local outbox to server.
	if err := e.PushLocalChanges(ctx); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	// 2. Pull new remote events from server.
	if err := e.PullAndApplyRemoteChanges(ctx); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	return nil
}

// serverTick: read the outbox for events queued by API pushes.
// On the server, the outbox is populated by CDC triggers on the
// PostgreSQL side, which clients then pull via the web API.
func (e *Engine) serverTick(ctx context.Context) error {
	// Server mode is mostly passive — the web handlers queue events.
	// This tick can be used for cleanup or monitoring in the future.
	return nil
}

// PushLocalChanges reads unsynced local outbox events and sends
// them to the remote server.
func (e *Engine) PushLocalChanges(ctx context.Context) error {
	events, err := e.watcher.Poll(ctx)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	if err := e.transporter.Push(ctx, events); err != nil {
		return err
	}

	// Mark events as synced after successful push.
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return e.watcher.MarkSynced(ctx, ids)
}

// PullAndApplyRemoteChanges fetches new events from the server
// and safely applies them using replay transactions.
func (e *Engine) PullAndApplyRemoteChanges(ctx context.Context) error {
	resp, err := e.transporter.Pull(ctx, e.cursor)
	if err != nil {
		return err
	}

	if len(resp.Events) == 0 {
		return nil
	}

	if err := e.applicator.ApplyEvents(ctx, resp.Events); err != nil {
		return err
	}

	// Update cursor for next pull.
	e.cursor = resp.Cursor
	return nil
}

// ApplyIncomingEvents is used by the web server handler to apply
// events pushed by a remote client (server mode).
func (e *Engine) ApplyIncomingEvents(ctx context.Context, events []SyncEvent) error {
	return e.applicator.ApplyEvents(ctx, events)
}

// GetEventsSince retrieves sync events since the given cursor
// (used by the pull handler in server mode).
func (e *Engine) GetEventsSince(ctx context.Context, cursor int64) ([]SyncEvent, int64, error) {
	var query string
	switch e.db.Driver() {
	case "sqlite":
		query = `
			SELECT id, table_name, row_id, action, payload, created_at
			FROM syncd_logs.sync_events WHERE id > ? ORDER BY id ASC LIMIT 100`
	case "postgres":
		query = `
			SELECT id, table_name, row_id, action, payload, created_at
			FROM syncd.sync_events WHERE id > $1 ORDER BY id ASC LIMIT 100`
	}

	rows, err := e.db.QueryContext(ctx, query, cursor)
	if err != nil {
		return nil, cursor, err
	}
	defer rows.Close()

	var events []SyncEvent
	var maxID int64
	for rows.Next() {
		var ev SyncEvent
		var payload, createdAt string
		if err := rows.Scan(&ev.ID, &ev.TableName, &ev.RowID, &ev.Action, &payload, &createdAt); err != nil {
			return nil, cursor, err
		}
		ev.Payload = []byte(payload)
		events = append(events, ev)
		if ev.ID > maxID {
			maxID = ev.ID
		}
	}

	if maxID == 0 {
		maxID = cursor
	}

	return events, maxID, rows.Err()
}

// generateClientID creates a simple unique client identifier.
func generateClientID() string {
	hostname, _ := os.Hostname()
	dir, _ := os.Getwd()
	return fmt.Sprintf("%s-%s", hostname, filepath.Base(dir))
}
