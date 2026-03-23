package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nyanhewe/syncd/internal/db"
)

// Watcher polls the local outbox for unsynced events.
type Watcher struct {
	db        db.Database
	batchSize int
}

// NewWatcher creates a new outbox watcher.
func NewWatcher(database db.Database, batchSize int) *Watcher {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Watcher{db: database, batchSize: batchSize}
}

// Poll fetches up to batchSize unsynced events from the outbox.
func (w *Watcher) Poll(ctx context.Context) ([]SyncEvent, error) {
	var query string
	switch w.db.Driver() {
	case "sqlite":
		query = `
			SELECT id, table_name, row_id, action, payload, created_at
			FROM syncd_logs.sync_events
			WHERE synced = 0
			ORDER BY id ASC
			LIMIT ?`
	case "postgres":
		query = `
			SELECT id, table_name, row_id, action, payload, created_at
			FROM syncd.sync_events
			WHERE synced = FALSE
			ORDER BY id ASC
			LIMIT $1`
	default:
		return nil, fmt.Errorf("syncd: unsupported driver %q", w.db.Driver())
	}

	rows, err := w.db.QueryContext(ctx, query, w.batchSize)
	if err != nil {
		return nil, fmt.Errorf("syncd: poll outbox: %w", err)
	}
	defer rows.Close()

	var events []SyncEvent
	for rows.Next() {
		var e SyncEvent
		var payload string
		var createdAt string

		if err := rows.Scan(&e.ID, &e.TableName, &e.RowID, &e.Action, &payload, &createdAt); err != nil {
			return nil, fmt.Errorf("syncd: scan outbox event: %w", err)
		}

		e.Payload = json.RawMessage(payload)

		// Parse created_at.
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			// Try SQLite datetime format.
			t, err = time.Parse("2006-01-02 15:04:05", createdAt)
			if err != nil {
				t = time.Now()
			}
		}
		e.CreatedAt = t
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(events) > 0 {
		log.Printf("[watcher] Polled %d unsynced event(s)", len(events))
	}

	return events, nil
}

// MarkSynced marks the given event IDs as synced in the outbox.
func (w *Watcher) MarkSynced(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		var query string
		switch w.db.Driver() {
		case "sqlite":
			query = "UPDATE syncd_logs.sync_events SET synced = 1 WHERE id = ?"
		case "postgres":
			query = "UPDATE syncd.sync_events SET synced = TRUE WHERE id = $1"
		}

		if _, err := w.db.ExecContext(ctx, query, id); err != nil {
			return fmt.Errorf("syncd: mark synced id=%d: %w", id, err)
		}
	}

	return nil
}
