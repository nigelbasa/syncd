package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/nyanhewe/syncd/internal/config"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin management commands",
}

var adminCreateCmd = &cobra.Command{
	Use:   "create [username]",
	Short: "Create an admin user for the web portal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		password, _ := cmd.Flags().GetString("password")

		if password == "" {
			return fmt.Errorf("--password is required")
		}

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

		// Hash password.
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Create admin users table if it doesn't exist.
		var createTableSQL string
		switch cfg.Database.Driver {
		case "sqlite":
			createTableSQL = `CREATE TABLE IF NOT EXISTS syncd_logs.admin_users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at TEXT DEFAULT (datetime('now'))
			)`
		case "postgres":
			createTableSQL = `CREATE TABLE IF NOT EXISTS syncd.admin_users (
				id BIGSERIAL PRIMARY KEY,
				username TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)`
		}

		ctx := context.Background()
		if _, err := database.ExecContext(ctx, createTableSQL); err != nil {
			return fmt.Errorf("create admin table: %w", err)
		}

		// Insert admin user.
		var insertSQL string
		switch cfg.Database.Driver {
		case "sqlite":
			insertSQL = "INSERT INTO syncd_logs.admin_users (username, password_hash) VALUES (?, ?)"
		case "postgres":
			insertSQL = "INSERT INTO syncd.admin_users (username, password_hash) VALUES ($1, $2)"
		}

		if _, err := database.ExecContext(ctx, insertSQL, username, string(hash)); err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}

		fmt.Printf("✓ Admin user '%s' created successfully\n", username)
		return nil
	},
}

func init() {
	adminCreateCmd.Flags().StringP("config", "c", "syncd.yaml", "Path to config file")
	adminCreateCmd.Flags().StringP("password", "p", "", "Admin password")
	adminCmd.AddCommand(adminCreateCmd)
	rootCmd.AddCommand(adminCmd)
}
