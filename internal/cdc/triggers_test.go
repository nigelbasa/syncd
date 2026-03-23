package cdc

import (
	"strings"
	"testing"
)

func TestGenerateTriggers_SQLite(t *testing.T) {
	sql := GenerateTriggers("sqlite", "students")

	// Should have all 3 trigger types.
	if !strings.Contains(sql, "syncd_log_students_insert") {
		t.Error("Missing SQLite INSERT trigger")
	}
	if !strings.Contains(sql, "syncd_log_students_update") {
		t.Error("Missing SQLite UPDATE trigger")
	}
	if !strings.Contains(sql, "syncd_log_students_delete") {
		t.Error("Missing SQLite DELETE trigger")
	}

	// Should have the Ignorer guard.
	if !strings.Contains(sql, "_syncd_replaying") {
		t.Error("SQLite triggers should check _syncd_replaying temp table")
	}

	// Should write to outbox.
	if !strings.Contains(sql, "syncd_logs.sync_events") {
		t.Error("SQLite triggers should insert into syncd_logs.sync_events")
	}
}

func TestGenerateTriggers_Postgres(t *testing.T) {
	sql := GenerateTriggers("postgres", "payments")

	// Should have all 3 trigger functions.
	if !strings.Contains(sql, "syncd.log_payments_insert") {
		t.Error("Missing Postgres INSERT trigger function")
	}
	if !strings.Contains(sql, "syncd.log_payments_update") {
		t.Error("Missing Postgres UPDATE trigger function")
	}
	if !strings.Contains(sql, "syncd.log_payments_delete") {
		t.Error("Missing Postgres DELETE trigger function")
	}

	// Should have the Ignorer guard.
	if !strings.Contains(sql, "syncd.is_replaying") {
		t.Error("Postgres triggers should check syncd.is_replaying session var")
	}

	// Should use row_to_json for INSERT/UPDATE.
	if !strings.Contains(sql, "row_to_json(NEW)") {
		t.Error("Postgres INSERT/UPDATE triggers should use row_to_json")
	}
}

func TestGenerateTriggers_Unknown(t *testing.T) {
	sql := GenerateTriggers("mysql", "test")
	if sql != "" {
		t.Error("Unknown driver should return empty string")
	}
}

func TestGenerateTriggers_DifferentTables(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	for _, table := range tables {
		sql := GenerateTriggers("sqlite", table)
		if !strings.Contains(sql, table) {
			t.Errorf("Trigger SQL for %q should contain table name", table)
		}
	}
}
