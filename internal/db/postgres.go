package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDB implements Database for PostgreSQL with schema-based outbox.
type PostgresDB struct {
	conn *sql.DB
}

// NewPostgresAdapter opens a PostgreSQL connection using pgx.
func NewPostgresAdapter(dsn string) (*PostgresDB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("syncd: open postgres: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("syncd: ping postgres: %w", err)
	}

	return &PostgresDB{conn: conn}, nil
}

func (p *PostgresDB) Driver() string { return "postgres" }

func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return p.conn.QueryContext(ctx, query, args...)
}

func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.conn.ExecContext(ctx, query, args...)
}

func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.conn.QueryRowContext(ctx, query, args...)
}

func (p *PostgresDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return p.conn.BeginTx(ctx, nil)
}

// BeginReplayTx creates a transaction with syncd.is_replaying session
// variable set so PostgreSQL triggers ignore operations within it.
func (p *PostgresDB) BeginReplayTx(ctx context.Context) (ReplayTransaction, error) {
	tx, err := p.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Set session variable that triggers check.
	_, err = tx.ExecContext(ctx, "SET LOCAL syncd.is_replaying = 'true'")
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("syncd: set replay session var: %w", err)
	}

	return &pgReplayTx{Tx: tx}, nil
}

func (p *PostgresDB) GetTables(ctx context.Context) ([]string, error) {
	rows, err := p.conn.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables 
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)
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

func (p *PostgresDB) GetPrimaryKey(ctx context.Context, table string) (string, error) {
	var pk string
	err := p.conn.QueryRowContext(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass AND i.indisprimary
		LIMIT 1`, table).Scan(&pk)
	if err != nil {
		return "id", nil // Default fallback
	}
	return pk, nil
}

func (p *PostgresDB) GetColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := p.conn.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (p *PostgresDB) Close() error {
	return p.conn.Close()
}

// pgReplayTx wraps sql.Tx for PostgreSQL replay transactions.
// The session variable is automatically scoped to the transaction
// via SET LOCAL, so no explicit cleanup is needed.
type pgReplayTx struct {
	*sql.Tx
}

func (t *pgReplayTx) Commit() error {
	return t.Tx.Commit()
}

func (t *pgReplayTx) Rollback() error {
	return t.Tx.Rollback()
}
