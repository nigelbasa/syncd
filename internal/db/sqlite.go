package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SqliteDB implements Database for SQLite with ATTACH-based outbox.
type SqliteDB struct {
	conn   *sql.DB
	syncDB string // Path to the attached syncd_logs database
}

// NewSqliteAdapter opens the application database and attaches the
// sync-log database as the "syncd_logs" schema.
func NewSqliteAdapter(appDBPath, syncDBPath string) (*SqliteDB, error) {
	conn, err := sql.Open("sqlite", appDBPath)
	if err != nil {
		return nil, fmt.Errorf("syncd: open sqlite %s: %w", appDBPath, err)
	}

	// Enable WAL for concurrent reads.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("syncd: enable WAL: %w", err)
	}

	// Attach the outbox database.
	attachSQL := fmt.Sprintf(`ATTACH DATABASE '%s' AS syncd_logs`, syncDBPath)
	if _, err := conn.Exec(attachSQL); err != nil {
		conn.Close()
		return nil, fmt.Errorf("syncd: attach sync db %s: %w", syncDBPath, err)
	}

	return &SqliteDB{conn: conn, syncDB: syncDBPath}, nil
}

func (s *SqliteDB) Driver() string { return "sqlite" }

func (s *SqliteDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.conn.QueryContext(ctx, query, args...)
}

func (s *SqliteDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.conn.ExecContext(ctx, query, args...)
}

func (s *SqliteDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.conn.QueryRowContext(ctx, query, args...)
}

func (s *SqliteDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.conn.BeginTx(ctx, nil)
}

// BeginReplayTx creates a transaction that sets the _syncd_replaying
// temp table flag so triggers ignore operations within it.
func (s *SqliteDB) BeginReplayTx(ctx context.Context) (ReplayTransaction, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Create the temporary flag table that triggers check for.
	_, err = tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS _syncd_replaying (flag INT)")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("syncd: set replay flag: %w", err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO _syncd_replaying VALUES (1)")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("syncd: insert replay flag: %w", err)
	}

	return &sqliteReplayTx{Tx: tx, ctx: ctx}, nil
}

func (s *SqliteDB) GetTables(ctx context.Context) ([]string, error) {
	rows, err := s.conn.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (s *SqliteDB) GetPrimaryKey(ctx context.Context, table string) (string, error) {
	rows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return "", err
		}
		if pk == 1 {
			return name, nil
		}
	}
	return "id", nil // Default fallback
}

func (s *SqliteDB) GetColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (s *SqliteDB) Close() error {
	return s.conn.Close()
}

// sqliteReplayTx wraps sql.Tx and drops the temp flag on commit/rollback.
type sqliteReplayTx struct {
	*sql.Tx
	ctx context.Context
}

func (t *sqliteReplayTx) Commit() error {
	// Clean up the replay flag before committing.
	t.Tx.ExecContext(t.ctx, "DROP TABLE IF EXISTS _syncd_replaying")
	return t.Tx.Commit()
}

func (t *sqliteReplayTx) Rollback() error {
	t.Tx.ExecContext(t.ctx, "DROP TABLE IF EXISTS _syncd_replaying")
	return t.Tx.Rollback()
}
