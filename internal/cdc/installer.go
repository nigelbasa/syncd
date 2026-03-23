package cdc

import (
	"context"
	"fmt"
	"log"

	"github.com/nyanhewe/syncd/internal/db"
)

// Installer sets up the CDC infrastructure on a live database.
type Installer struct {
	db db.Database
}

// NewInstaller creates a new CDC installer for the given database.
func NewInstaller(database db.Database) *Installer {
	return &Installer{db: database}
}

// Install creates the outbox schema and installs triggers for all
// specified tables. This is idempotent — safe to run multiple times.
func (inst *Installer) Install(ctx context.Context, tables []string) error {
	driver := inst.db.Driver()

	// 1. Create the outbox schema/table.
	schemaSQL := GenerateOutboxSchema(driver)
	if schemaSQL == "" {
		return fmt.Errorf("syncd: unsupported driver %q for CDC", driver)
	}

	log.Printf("[cdc] Creating outbox schema for driver=%s...", driver)
	if _, err := inst.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("syncd: create outbox schema: %w", err)
	}

	// 2. Install triggers for each table.
	for _, table := range tables {
		triggerSQL := GenerateTriggers(driver, table)
		if triggerSQL == "" {
			return fmt.Errorf("syncd: failed to generate triggers for table %q", table)
		}

		log.Printf("[cdc] Installing triggers for table=%s...", table)
		if _, err := inst.db.ExecContext(ctx, triggerSQL); err != nil {
			return fmt.Errorf("syncd: install triggers for %s: %w", table, err)
		}
	}

	log.Printf("[cdc] Successfully installed CDC for %d table(s)", len(tables))
	return nil
}

// Uninstall removes all syncd triggers from the specified tables.
func (inst *Installer) Uninstall(ctx context.Context, tables []string) error {
	driver := inst.db.Driver()

	for _, table := range tables {
		var dropSQL string
		switch driver {
		case "sqlite":
			dropSQL = fmt.Sprintf(`
				DROP TRIGGER IF EXISTS syncd_log_%[1]s_insert;
				DROP TRIGGER IF EXISTS syncd_log_%[1]s_update;
				DROP TRIGGER IF EXISTS syncd_log_%[1]s_delete;
			`, table)
		case "postgres":
			dropSQL = fmt.Sprintf(`
				DROP TRIGGER IF EXISTS syncd_trigger_%[1]s_insert ON public.%[1]s;
				DROP TRIGGER IF EXISTS syncd_trigger_%[1]s_update ON public.%[1]s;
				DROP TRIGGER IF EXISTS syncd_trigger_%[1]s_delete ON public.%[1]s;
				DROP FUNCTION IF EXISTS syncd.log_%[1]s_insert();
				DROP FUNCTION IF EXISTS syncd.log_%[1]s_update();
				DROP FUNCTION IF EXISTS syncd.log_%[1]s_delete();
			`, table)
		}

		log.Printf("[cdc] Removing triggers for table=%s...", table)
		if _, err := inst.db.ExecContext(ctx, dropSQL); err != nil {
			return fmt.Errorf("syncd: uninstall triggers for %s: %w", table, err)
		}
	}

	return nil
}
