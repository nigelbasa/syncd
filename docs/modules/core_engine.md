# Module: Core Engine (`internal/engine`)

## Description
The `core_engine` module is the heart of Syncd. It is responsible for orchestrating the flow of sync events between the local client and the remote server. It contains the Watcher (polling for local changes), Transporter (API communication), Applicator (executing changes safely), and Conflict Resolver.

## Key Functions

### `engine.Start(ctx context.Context)`
Starts the background sync loop for the daemon.

### `engine.PullAndApplyRemoteChanges()`
Fetches newly queued events from the remote server API, resolves conflicts, and applies them to the local database via the `applicator`.

### `engine.PushLocalChanges()`
Reads new events from the local "Outbox" (`sync_events` table) and POSTs them to the remote server.

### `applicator.ApplyEvent(tx db.Transaction, event SyncEvent)`
Takes a deserialized sync event and constructs the actual `INSERT/UPDATE/DELETE` query, safely executing it within a "replaying" transaction to bypass trigger echoes.

## Structure
```
internal/engine/
├── daemon.go       (Main loop & orchestration)
├── applicator.go   (Query reconstruction & safe execution)
├── conflict.go     (LWW resolution logic)
├── watcher.go      (Local outbox polling)
└── transporter.go  (HTTP/WS client wrapper)
```

## Expected Inputs and Outputs
**Input**: A `SyncEvent` containing `table_name`, `action` (INSERT/UPDATE/DELETE), `row_id`, `payload` (JSON of the row), and `updated_at`.
**Output**: A success boolean and an updated "Last Synced Cursor" ID to track progress.

## Code Snippet (Applicator Example)
```go
func (a *Applicator) ApplyEvent(ctx context.Context, e types.SyncEvent) error {
    // 1. Begin a special "Replay" transaction 
    //    (sets syncd.is_replaying=true or creates temp table)
    tx, err := a.db.BeginReplayTx(ctx)
    if err != nil { return err }
    defer tx.Rollback()

    // 2. Resolve Conflict
    shouldApply, err := a.conflictResolver.Check(tx, e)
    if err != nil || !shouldApply {
        return nil // Skip safely
    }

    // 3. Execute
    query, args := buildQueryFromEvent(e)
    if _, err := tx.ExecContext(ctx, query, args...); err != nil {
        return err
    }

    return tx.Commit()
}
```
