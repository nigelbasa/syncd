package cdc

import "fmt"

// GenerateOutboxSchema returns the DDL to create the sync_events outbox table.
func GenerateOutboxSchema(driver string) string {
	switch driver {
	case "sqlite":
		return generateSqliteOutboxSchema()
	case "postgres":
		return generatePostgresOutboxSchema()
	default:
		return ""
	}
}

func generateSqliteOutboxSchema() string {
	return `
CREATE TABLE IF NOT EXISTS syncd_logs.sync_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name TEXT    NOT NULL,
    row_id     TEXT    NOT NULL,
    action     TEXT    NOT NULL CHECK(action IN ('INSERT','UPDATE','DELETE')),
    payload    TEXT    NOT NULL,  -- JSON
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    synced     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS syncd_logs.idx_sync_events_unsynced 
    ON sync_events(synced, id) WHERE synced = 0;
`
}

func generatePostgresOutboxSchema() string {
	return fmt.Sprintf(`
CREATE SCHEMA IF NOT EXISTS syncd;

CREATE TABLE IF NOT EXISTS syncd.sync_events (
    id         BIGSERIAL PRIMARY KEY,
    table_name TEXT      NOT NULL,
    row_id     TEXT      NOT NULL,
    action     TEXT      NOT NULL CHECK(action IN ('INSERT','UPDATE','DELETE')),
    payload    JSONB     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced     BOOLEAN   NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_sync_events_unsynced 
    ON syncd.sync_events(synced, id) WHERE synced = FALSE;
`)
}
