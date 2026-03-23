# Module: Database Adapter (`internal/db`)

## Description
The Database Adapter module provides a unified interface over standard `database/sql` driver implementations for both PostgreSQL and SQLite. Its main purpose is to smoothly handle the "Ignorer" pattern (setting session variables or temp tables) securely and generically whenever the sync engine needs to write data.

## Key Interfaces & Functions

### `type Database interface`
```go
type Database interface {
    // Standard execution
    Query(query string, args ...any) (*sql.Rows, error)
    Exec(query string, args ...any) (sql.Result, error)
    
    // Core Syncd Requirement:
    BeginReplayTx(ctx context.Context) (ReplayTransaction, error)
    
    // Information Schema
    GetTables() ([]string, error)
    GetPrimaryKey(table string) (string, error)
}
```

### `db.NewPostgresAdapter(dsn string)` and `db.NewSqliteAdapter(filePath string)`
Initializers that return structs satisfying the `Database` interface.

## Structure
```
internal/db/
├── db.go        (Interfaces and generic types)
├── postgres.go  (PG specific implementation & session var logic)
├── sqlite.go    (SQLite specific implementation & ATTACH logic)
└── query_builder.go (Helpers for dynamic SQL generation)
```

## Snippets
### Postgres `BeginReplayTx` Implementation
```go
func (p *PostgresDB) BeginReplayTx(ctx context.Context) (ReplayTransaction, error) {
    tx, err := p.conn.BeginTx(ctx, nil)
    if err != nil { return nil, err }
    
    // Set the session variable so triggers ignore these changes
    _, err = tx.ExecContext(ctx, "SET LOCAL syncd.is_replaying = 'true'")
    if err != nil {
        tx.Rollback()
        return nil, err
    }
    
    return &pgReplayTx{Tx: tx}, nil
}
```

### SQLite `BeginReplayTx` Implementation
```go
func (s *SqliteDB) BeginReplayTx(ctx context.Context) (ReplayTransaction, error) {
    tx, err := s.conn.BeginTx(ctx, nil)
    if err != nil { return nil, err }
    
    // Create temp table flag so triggers ignore these changes
    _, err = tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS _syncd_replaying (flag INT)")
    if err != nil { /* rollback and return */ }
    
    return &sqliteReplayTx{Tx: tx}, nil
}
```
