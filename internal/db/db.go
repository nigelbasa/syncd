package db

import (
	"context"
	"database/sql"
)

// Database is the unified interface for syncd's database operations
// across both SQLite and PostgreSQL.
type Database interface {
	// Standard query execution.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	// BeginReplayTx starts a transaction that sets the "Ignorer" flag
	// so that CDC triggers will not fire for operations within this tx.
	BeginReplayTx(ctx context.Context) (ReplayTransaction, error)

	// BeginTx starts a standard transaction (triggers WILL fire).
	BeginTx(ctx context.Context) (*sql.Tx, error)

	// Schema introspection.
	GetTables(ctx context.Context) ([]string, error)
	GetPrimaryKey(ctx context.Context, table string) (string, error)
	GetColumns(ctx context.Context, table string) ([]string, error)

	// Driver returns "sqlite" or "postgres".
	Driver() string

	// Close cleans up the database connection.
	Close() error
}

// ReplayTransaction is a transaction with the "Ignorer" flag set,
// meaning CDC triggers will not log events created within it.
type ReplayTransaction interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Commit() error
	Rollback() error
}
