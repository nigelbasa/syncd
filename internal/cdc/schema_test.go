package cdc

import (
	"strings"
	"testing"
)

func TestGenerateOutboxSchema_SQLite(t *testing.T) {
	sql := GenerateOutboxSchema("sqlite")

	if !strings.Contains(sql, "syncd_logs.sync_events") {
		t.Error("SQLite schema should reference syncd_logs.sync_events")
	}
	if !strings.Contains(sql, "AUTOINCREMENT") {
		t.Error("SQLite schema should use AUTOINCREMENT")
	}
}

func TestGenerateOutboxSchema_Postgres(t *testing.T) {
	sql := GenerateOutboxSchema("postgres")

	if !strings.Contains(sql, "CREATE SCHEMA IF NOT EXISTS syncd") {
		t.Error("Postgres schema should create syncd schema")
	}
	if !strings.Contains(sql, "syncd.sync_events") {
		t.Error("Postgres schema should reference syncd.sync_events")
	}
	if !strings.Contains(sql, "BIGSERIAL") {
		t.Error("Postgres schema should use BIGSERIAL")
	}
	if !strings.Contains(sql, "JSONB") {
		t.Error("Postgres schema should use JSONB")
	}
}

func TestGenerateOutboxSchema_Unknown(t *testing.T) {
	sql := GenerateOutboxSchema("mysql")
	if sql != "" {
		t.Error("Unknown driver should return empty string")
	}
}
