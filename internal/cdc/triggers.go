package cdc

import "fmt"

// GenerateTriggers returns the full set of CREATE TRIGGER SQL statements
// (INSERT, UPDATE, DELETE) for the given table, with the "Ignorer" guard
// to prevent infinite sync echo loops.
func GenerateTriggers(driver, table string) string {
	switch driver {
	case "sqlite":
		return generateSqliteTriggers(table)
	case "postgres":
		return generatePostgresTriggers(table)
	default:
		return ""
	}
}

// --- SQLite Triggers ---

func generateSqliteTriggers(table string) string {
	return generateSqliteInsertTrigger(table) +
		generateSqliteUpdateTrigger(table) +
		generateSqliteDeleteTrigger(table)
}

func generateSqliteInsertTrigger(table string) string {
	return fmt.Sprintf(`
CREATE TRIGGER IF NOT EXISTS syncd_log_%[1]s_insert
AFTER INSERT ON %[1]s
WHEN NOT EXISTS (SELECT 1 FROM sqlite_temp_master WHERE type='table' AND name='_syncd_replaying')
BEGIN
    INSERT INTO syncd_logs.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', CAST(NEW.id AS TEXT), 'INSERT', json_object(
        'data', json(CASE TYPEOF(NEW.id) WHEN 'null' THEN '{}' ELSE json_group_array(NEW.*) END)
    ));
END;
`, table)
}

func generateSqliteUpdateTrigger(table string) string {
	return fmt.Sprintf(`
CREATE TRIGGER IF NOT EXISTS syncd_log_%[1]s_update
AFTER UPDATE ON %[1]s
WHEN NOT EXISTS (SELECT 1 FROM sqlite_temp_master WHERE type='table' AND name='_syncd_replaying')
BEGIN
    INSERT INTO syncd_logs.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', CAST(NEW.id AS TEXT), 'UPDATE', json_object(
        'data', json(CASE TYPEOF(NEW.id) WHEN 'null' THEN '{}' ELSE json_group_array(NEW.*) END)
    ));
END;
`, table)
}

func generateSqliteDeleteTrigger(table string) string {
	return fmt.Sprintf(`
CREATE TRIGGER IF NOT EXISTS syncd_log_%[1]s_delete
AFTER DELETE ON %[1]s
WHEN NOT EXISTS (SELECT 1 FROM sqlite_temp_master WHERE type='table' AND name='_syncd_replaying')
BEGIN
    INSERT INTO syncd_logs.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', CAST(OLD.id AS TEXT), 'DELETE', '{}');
END;
`, table)
}

// --- PostgreSQL Triggers ---

func generatePostgresTriggers(table string) string {
	return generatePostgresInsertTrigger(table) +
		generatePostgresUpdateTrigger(table) +
		generatePostgresDeleteTrigger(table)
}

func generatePostgresInsertTrigger(table string) string {
	return fmt.Sprintf(`
CREATE OR REPLACE FUNCTION syncd.log_%[1]s_insert() RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('syncd.is_replaying', true) = 'true' THEN
        RETURN NEW;
    END IF;

    INSERT INTO syncd.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', NEW.id::TEXT, 'INSERT', row_to_json(NEW));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER syncd_trigger_%[1]s_insert
AFTER INSERT ON public.%[1]s
FOR EACH ROW EXECUTE FUNCTION syncd.log_%[1]s_insert();
`, table)
}

func generatePostgresUpdateTrigger(table string) string {
	return fmt.Sprintf(`
CREATE OR REPLACE FUNCTION syncd.log_%[1]s_update() RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('syncd.is_replaying', true) = 'true' THEN
        RETURN NEW;
    END IF;

    INSERT INTO syncd.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', NEW.id::TEXT, 'UPDATE', row_to_json(NEW));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER syncd_trigger_%[1]s_update
AFTER UPDATE ON public.%[1]s
FOR EACH ROW EXECUTE FUNCTION syncd.log_%[1]s_update();
`, table)
}

func generatePostgresDeleteTrigger(table string) string {
	return fmt.Sprintf(`
CREATE OR REPLACE FUNCTION syncd.log_%[1]s_delete() RETURNS TRIGGER AS $$
BEGIN
    IF current_setting('syncd.is_replaying', true) = 'true' THEN
        RETURN OLD;
    END IF;

    INSERT INTO syncd.sync_events (table_name, row_id, action, payload)
    VALUES ('%[1]s', OLD.id::TEXT, 'DELETE', '{}');
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER syncd_trigger_%[1]s_delete
AFTER DELETE ON public.%[1]s
FOR EACH ROW EXECUTE FUNCTION syncd.log_%[1]s_delete();
`, table)
}
