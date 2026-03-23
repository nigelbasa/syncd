package engine

import (
	"encoding/json"
	"time"
)

// SyncEvent represents a single change captured by a CDC trigger.
type SyncEvent struct {
	ID        int64           `json:"id"`
	TableName string          `json:"table_name"`
	RowID     string          `json:"row_id"`
	Action    string          `json:"action"` // INSERT, UPDATE, DELETE
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// SyncCursor tracks the last synced position for pull operations.
type SyncCursor struct {
	LastEventID int64     `json:"last_event_id"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PushPayload is the request body for POST /api/v1/sync/push.
type PushPayload struct {
	ClientID string      `json:"client_id"`
	Events   []SyncEvent `json:"events"`
}

// PullResponse is the response body for GET /api/v1/sync/pull.
type PullResponse struct {
	Events []SyncEvent `json:"events"`
	Cursor int64       `json:"cursor"`
}
