package engine

import (
	"encoding/json"
	"fmt"
	"time"
)

// ConflictResolver implements Last-Write-Wins (LWW) conflict resolution.
type ConflictResolver struct{}

// NewConflictResolver creates a new LWW conflict resolver.
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// ShouldApply determines whether an incoming remote event should be
// applied to the local database. It compares the updated_at timestamp
// from the remote event payload against the local row's updated_at.
//
// Returns true if the event should be applied, false if it should be dropped.
func (cr *ConflictResolver) ShouldApply(localUpdatedAt *time.Time, remoteEvent SyncEvent) (bool, error) {
	// DELETEs always apply — can't conflict with a non-existent row.
	if remoteEvent.Action == "DELETE" {
		return true, nil
	}

	// INSERTs: apply if the row doesn't exist locally (localUpdatedAt is nil).
	if remoteEvent.Action == "INSERT" {
		if localUpdatedAt == nil {
			return true, nil
		}
		// Row already exists — treat as a conflict, use LWW.
		remoteTime, err := extractUpdatedAt(remoteEvent.Payload)
		if err != nil {
			// If no timestamp, apply to be safe.
			return true, nil
		}
		return !remoteTime.Before(*localUpdatedAt), nil
	}

	// UPDATEs: LWW based on updated_at.
	if localUpdatedAt == nil {
		// Row doesn't exist locally — skip the update.
		return false, nil
	}

	remoteTime, err := extractUpdatedAt(remoteEvent.Payload)
	if err != nil {
		// If no timestamp in payload, apply to be safe.
		return true, nil
	}

	// Apply if remote is newer or equal (deterministic tiebreaker: remote wins).
	return !remoteTime.Before(*localUpdatedAt), nil
}

// extractUpdatedAt parses the updated_at field from a JSON payload.
func extractUpdatedAt(payload json.RawMessage) (time.Time, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return time.Time{}, err
	}

	val, ok := data["updated_at"]
	if !ok {
		return time.Time{}, fmt.Errorf("no updated_at field")
	}

	str, ok := val.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("updated_at is not a string")
	}

	// Try common formats.
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, str)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse updated_at: %s", str)
}
