package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nyanhewe/syncd/internal/cdc"
	"github.com/nyanhewe/syncd/internal/config"
)

var triggersCmd = &cobra.Command{
	Use:   "install-triggers",
	Short: "Install CDC triggers on database tables",
	Long:  `Reads the config and installs the necessary INSERT/UPDATE/DELETE triggers for the specified tables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		database, err := openDatabase(cfg)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		installer := cdc.NewInstaller(database)
		tables := cfg.Sync.Tables

		// Allow overriding tables from CLI.
		tableFlag, _ := cmd.Flags().GetStringSlice("table")
		if len(tableFlag) > 0 {
			tables = tableFlag
		}

		if err := installer.Install(context.Background(), tables); err != nil {
			return fmt.Errorf("trigger installation failed: %w", err)
		}

		fmt.Printf("✓ CDC triggers installed for %d table(s)\n", len(tables))
		return nil
	},
}

var uninstallTriggersCmd = &cobra.Command{
	Use:   "uninstall-triggers",
	Short: "Remove CDC triggers from database tables",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		database, err := openDatabase(cfg)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		installer := cdc.NewInstaller(database)
		tables := cfg.Sync.Tables

		tableFlag, _ := cmd.Flags().GetStringSlice("table")
		if len(tableFlag) > 0 {
			tables = tableFlag
		}

		if err := installer.Uninstall(context.Background(), tables); err != nil {
			return fmt.Errorf("trigger uninstallation failed: %w", err)
		}

		fmt.Printf("✓ CDC triggers removed for %d table(s)\n", len(tables))
		return nil
	},
}

func init() {
	triggersCmd.Flags().StringP("config", "c", "syncd.yaml", "Path to config file")
	triggersCmd.Flags().StringSliceP("table", "t", nil, "Specific tables (overrides config)")

	uninstallTriggersCmd.Flags().StringP("config", "c", "syncd.yaml", "Path to config file")
	uninstallTriggersCmd.Flags().StringSliceP("table", "t", nil, "Specific tables (overrides config)")

	rootCmd.AddCommand(triggersCmd)
	rootCmd.AddCommand(uninstallTriggersCmd)
}
