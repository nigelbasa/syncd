package engine

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConflictResolver_DeleteAlwaysApplies(t *testing.T) {
	cr := NewConflictResolver()
	now := time.Now()

	event := SyncEvent{Action: "DELETE", RowID: "123"}
	shouldApply, err := cr.ShouldApply(&now, event)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldApply {
		t.Error("DELETE events should always apply")
	}
}

func TestConflictResolver_InsertNewRow(t *testing.T) {
	cr := NewConflictResolver()

	event := SyncEvent{
		Action:  "INSERT",
		Payload: json.RawMessage(`{"id": "1", "name": "Alice"}`),
	}

	shouldApply, err := cr.ShouldApply(nil, event)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldApply {
		t.Error("INSERT on non-existent row should apply")
	}
}

func TestConflictResolver_UpdateRemoteWins(t *testing.T) {
	cr := NewConflictResolver()

	localTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	remoteTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC) // Later

	event := SyncEvent{
		Action:  "UPDATE",
		Payload: json.RawMessage(`{"id": "1", "updated_at": "` + remoteTime.Format(time.RFC3339) + `"}`),
	}

	shouldApply, err := cr.ShouldApply(&localTime, event)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldApply {
		t.Error("Newer remote update should apply (LWW)")
	}
}

func TestConflictResolver_UpdateLocalWins(t *testing.T) {
	cr := NewConflictResolver()

	localTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC) // Later
	remoteTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	event := SyncEvent{
		Action:  "UPDATE",
		Payload: json.RawMessage(`{"id": "1", "updated_at": "` + remoteTime.Format(time.RFC3339) + `"}`),
	}

	shouldApply, err := cr.ShouldApply(&localTime, event)
	if err != nil {
		t.Fatal(err)
	}
	if shouldApply {
		t.Error("Older remote update should be dropped (local wins)")
	}
}

func TestConflictResolver_EqualTimestampRemoteWins(t *testing.T) {
	cr := NewConflictResolver()

	sameTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	event := SyncEvent{
		Action:  "UPDATE",
		Payload: json.RawMessage(`{"id": "1", "updated_at": "` + sameTime.Format(time.RFC3339) + `"}`),
	}

	shouldApply, err := cr.ShouldApply(&sameTime, event)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldApply {
		t.Error("Equal timestamps: remote should win as deterministic tiebreaker")
	}
}

func TestConflictResolver_UpdateMissingLocalRow(t *testing.T) {
	cr := NewConflictResolver()

	event := SyncEvent{
		Action:  "UPDATE",
		Payload: json.RawMessage(`{"id": "1", "name": "Bob"}`),
	}

	shouldApply, err := cr.ShouldApply(nil, event)
	if err != nil {
		t.Fatal(err)
	}
	if shouldApply {
		t.Error("UPDATE on non-existent local row should be skipped")
	}
}
