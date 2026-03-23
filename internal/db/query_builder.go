package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BuildInsertQuery constructs an INSERT statement from a JSON payload.
// Returns the query string and argument slice.
func BuildInsertQuery(table string, payload json.RawMessage) (string, []any, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", nil, fmt.Errorf("syncd: unmarshal insert payload: %w", err)
	}

	if len(data) == 0 {
		return "", nil, fmt.Errorf("syncd: empty insert payload for table %s", table)
	}

	// Sort keys for deterministic query generation.
	keys := sortedKeys(data)
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))

	for i, k := range keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = data[k]
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		table,
		strings.Join(keys, ", "),
		strings.Join(placeholders, ", "),
	)

	return query, args, nil
}

// BuildUpdateQuery constructs an UPDATE statement from a JSON payload.
// The pkColumn is used in the WHERE clause.
func BuildUpdateQuery(table, pkColumn string, payload json.RawMessage) (string, []any, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", nil, fmt.Errorf("syncd: unmarshal update payload: %w", err)
	}

	pkValue, ok := data[pkColumn]
	if !ok {
		return "", nil, fmt.Errorf("syncd: primary key %s not found in update payload", pkColumn)
	}

	// Build SET clause excluding the PK.
	keys := sortedKeys(data)
	setClauses := make([]string, 0, len(keys)-1)
	args := make([]any, 0, len(keys))
	argIdx := 1

	for _, k := range keys {
		if k == pkColumn {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, data[k])
		argIdx++
	}

	// PK value is the last argument.
	args = append(args, pkValue)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		table,
		strings.Join(setClauses, ", "),
		pkColumn,
		argIdx,
	)

	return query, args, nil
}

// BuildDeleteQuery constructs a DELETE statement using the primary key.
func BuildDeleteQuery(table, pkColumn string, rowID any) (string, []any, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, pkColumn)
	return query, []any{rowID}, nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
