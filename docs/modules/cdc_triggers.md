# Module: CDC & Triggers (`internal/cdc`)

## Description
The Change Data Capture (CDC) module is responsible for generating and managing the SQL commands required to create the "Outbox" architecture. Instead of hardcoding SQL files, Syncd creates triggers dynamically using string templates based on the user's defined schema and target tables.

## Key Functions

### `cdc.GenerateOutboxSchema(driver string) string`
Returns the raw SQL string (for either PG or SQLite) to create the `sync_events` and associated log tables.

### `cdc.GenerateTrigger(driver string, tableName string) string`
Constructs the `CREATE TRIGGER` SQL for `INSERT`, `UPDATE`, and `DELETE` actions. 
Crucially, these generated triggers contain the "Ignorer" logic (checking `_syncd_replaying` in SQLite or `syncd.is_replaying` in Postgres) so that the sync engine itself doesn't cause infinite echo loops.

### `cdc.Install(db Database, tables []string)`
Executes the schema and trigger generation strings against the live database connection.

## Structure
```
internal/cdc/
├── schema.go      (Outbox table creation SQL templates)
├── triggers.go    (Trigger SQL templates)
├── installer.go   (Execution engine for the generated SQL)
└── templates/     (Optionally store complex templates as actual .sql or .tmpl files)
```

## Snippets
### Example Generator Logic (SQLite Insert)
```go
func GenerateSqliteInsertTrigger(table string) string {
    return fmt.Sprintf(`
    CREATE TRIGGER IF NOT EXISTS log_%[1]s_insert 
    AFTER INSERT ON %[1]s
    WHEN NOT EXISTS (SELECT 1 FROM sqlite_temp_master WHERE type='table' AND name='_syncd_replaying')
    BEGIN
        INSERT INTO syncd_logs.sync_events (table_name, row_id, action, payload) 
        VALUES ('%[1]s', NEW.id, 'INSERT', json(NEW));
    END;
    `, table)
}
```

### Example Generator Logic (Postgres Update)
```go
func GeneratePostgresUpdateTrigger(table string) string {
    return fmt.Sprintf(`
    CREATE OR REPLACE FUNCTION log_%[1]s_update() RETURNS TRIGGER AS $$
    BEGIN
        -- Ignore engine updates
        IF current_setting('syncd.is_replaying', true) = 'true' THEN
            RETURN NEW;
        END IF;

        INSERT INTO syncd.sync_events (table_name, row_id, action, payload)
        VALUES ('%[1]s', NEW.id, 'UPDATE', row_to_json(NEW));
        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;

    CREATE TRIGGER trigger_%[1]s_update
    AFTER UPDATE ON public.%[1]s
    FOR EACH ROW EXECUTE FUNCTION log_%[1]s_update();
    `, table)
}
```
