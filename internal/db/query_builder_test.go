package db

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildInsertQuery(t *testing.T) {
	payload := json.RawMessage(`{"id": "abc-123", "name": "Alice", "email": "alice@example.com"}`)

	query, args, err := BuildInsertQuery("users", payload)
	if err != nil {
		t.Fatalf("BuildInsertQuery error: %v", err)
	}

	if !strings.HasPrefix(query, "INSERT INTO users") {
		t.Errorf("Query should start with INSERT INTO users, got: %s", query)
	}
	if !strings.Contains(query, "ON CONFLICT DO NOTHING") {
		t.Error("Insert should have ON CONFLICT DO NOTHING")
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

func TestBuildInsertQuery_EmptyPayload(t *testing.T) {
	payload := json.RawMessage(`{}`)
	_, _, err := BuildInsertQuery("users", payload)
	if err == nil {
		t.Error("Expected error for empty payload")
	}
}

func TestBuildUpdateQuery(t *testing.T) {
	payload := json.RawMessage(`{"id": "abc-123", "name": "Bob", "email": "bob@example.com"}`)

	query, args, err := BuildUpdateQuery("users", "id", payload)
	if err != nil {
		t.Fatalf("BuildUpdateQuery error: %v", err)
	}

	if !strings.HasPrefix(query, "UPDATE users SET") {
		t.Errorf("Query should start with UPDATE users SET, got: %s", query)
	}
	if !strings.Contains(query, "WHERE id =") {
		t.Error("Update should have WHERE clause on primary key")
	}
	// 2 SET columns + 1 PK value = 3 args.
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

func TestBuildUpdateQuery_MissingPK(t *testing.T) {
	payload := json.RawMessage(`{"name": "Bob"}`)
	_, _, err := BuildUpdateQuery("users", "id", payload)
	if err == nil {
		t.Error("Expected error for missing primary key")
	}
}

func TestBuildDeleteQuery(t *testing.T) {
	query, args, err := BuildDeleteQuery("users", "id", "abc-123")
	if err != nil {
		t.Fatalf("BuildDeleteQuery error: %v", err)
	}

	if query != "DELETE FROM users WHERE id = $1" {
		t.Errorf("Unexpected query: %s", query)
	}
	if len(args) != 1 || args[0] != "abc-123" {
		t.Errorf("Expected [abc-123], got %v", args)
	}
}

func TestBuildInsertQuery_DeterministicOrder(t *testing.T) {
	payload := json.RawMessage(`{"z_col": 3, "a_col": 1, "m_col": 2}`)

	q1, _, _ := BuildInsertQuery("test", payload)
	q2, _, _ := BuildInsertQuery("test", payload)

	if q1 != q2 {
		t.Error("Insert queries should be deterministic regardless of JSON key order")
	}

	// Keys should be sorted alphabetically.
	if !strings.Contains(q1, "a_col, m_col, z_col") {
		t.Errorf("Columns should be sorted, got: %s", q1)
	}
}
