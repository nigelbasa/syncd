package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nyanhewe/syncd/internal/db"
)

// Applicator safely executes incoming remote sync events against
// the local database using replay transactions.
type Applicator struct {
	db               db.Database
	conflictResolver *ConflictResolver
}

// NewApplicator creates a new event applicator.
func NewApplicator(database db.Database, resolver *ConflictResolver) *Applicator {
	return &Applicator{
		db:               database,
		conflictResolver: resolver,
	}
}

// ApplyEvents processes a batch of incoming remote events.
func (a *Applicator) ApplyEvents(ctx context.Context, events []SyncEvent) error {
	for _, event := range events {
		if err := a.applyOne(ctx, event); err != nil {
			log.Printf("[applicator] Error applying event id=%d table=%s action=%s: %v",
				event.ID, event.TableName, event.Action, err)
			// Continue with other events rather than failing the batch.
			continue
		}
	}
	return nil
}

func (a *Applicator) applyOne(ctx context.Context, event SyncEvent) error {
	// 1. Check conflict resolution.
	localUpdatedAt, err := a.getLocalUpdatedAt(ctx, event.TableName, event.RowID)
	if err != nil {
		// If we can't check, proceed with apply (fail-open).
		log.Printf("[applicator] Could not check local state for %s/%s: %v", event.TableName, event.RowID, err)
	}

	shouldApply, err := a.conflictResolver.ShouldApply(localUpdatedAt, event)
	if err != nil {
		return fmt.Errorf("conflict check: %w", err)
	}
	if !shouldApply {
		log.Printf("[applicator] Skipping event id=%d (conflict: local wins)", event.ID)
		return nil
	}

	// 2. Begin a replay transaction (triggers are silenced).
	tx, err := a.db.BeginReplayTx(ctx)
	if err != nil {
		return fmt.Errorf("begin replay tx: %w", err)
	}
	defer tx.Rollback()

	// 3. Build and execute query.
	pk, _ := a.db.GetPrimaryKey(ctx, event.TableName)
	if pk == "" {
		pk = "id"
	}

	var query string
	var args []any

	switch event.Action {
	case "INSERT":
		query, args, err = db.BuildInsertQuery(event.TableName, event.Payload)
	case "UPDATE":
		query, args, err = db.BuildUpdateQuery(event.TableName, pk, event.Payload)
	case "DELETE":
		query, args, err = db.BuildDeleteQuery(event.TableName, pk, event.RowID)
	default:
		return fmt.Errorf("unknown action: %s", event.Action)
	}

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec %s on %s: %w", event.Action, event.TableName, err)
	}

	// 4. Commit.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replay tx: %w", err)
	}

	log.Printf("[applicator] Applied event id=%d: %s on %s (row=%s)",
		event.ID, event.Action, event.TableName, event.RowID)
	return nil
}

// getLocalUpdatedAt retrieves the updated_at of a local row, or nil if not found.
func (a *Applicator) getLocalUpdatedAt(ctx context.Context, table, rowID string) (*time.Time, error) {
	pk, _ := a.db.GetPrimaryKey(ctx, table)
	if pk == "" {
		pk = "id"
	}

	var query string
	switch a.db.Driver() {
	case "sqlite":
		query = fmt.Sprintf("SELECT updated_at FROM %s WHERE %s = ?", table, pk)
	case "postgres":
		query = fmt.Sprintf("SELECT updated_at FROM %s WHERE %s = $1", table, pk)
	}

	var updatedAt string
	err := a.db.QueryRowContext(ctx, query, rowID).Scan(&updatedAt)
	if err != nil {
		return nil, err // Row not found or no updated_at column.
	}

	// Try parsing.
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, updatedAt)
		if err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse local updated_at: %s", updatedAt)
}
