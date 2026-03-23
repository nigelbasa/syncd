package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/nyanhewe/syncd/internal/config"
	"github.com/nyanhewe/syncd/internal/db"
	"github.com/nyanhewe/syncd/internal/engine"
	"github.com/nyanhewe/syncd/internal/web"
)

var cfgFile string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the Syncd daemon",
	Long:  `Starts the background sync loop. In server mode, it also boots up the web API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config.
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize database adapter.
		database, err := openDatabase(cfg)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		// Create engine.
		var eng *engine.Engine
		if cfg.App.Mode == "server" {
			eng = engine.NewServerEngine(cfg, database)
		} else {
			eng = engine.New(cfg, database)
		}

		// Context with graceful shutdown.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Start the sync engine in the background.
		errCh := make(chan error, 1)
		go func() {
			errCh <- eng.Start(ctx)
		}()

		// In server mode, also start the web server.
		if cfg.App.Mode == "server" {
			srv := web.NewServer(cfg, eng)
			go func() {
				if err := srv.Start(); err != nil {
					log.Printf("[main] Web server error: %v", err)
				}
			}()
			defer srv.Shutdown(ctx)
		}

		log.Printf("[main] Syncd is running in %s mode. Press Ctrl+C to stop.", cfg.App.Mode)

		// Wait for shutdown signal or error.
		select {
		case sig := <-sigCh:
			log.Printf("[main] Received signal: %v, shutting down...", sig)
			cancel()
		case err := <-errCh:
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "syncd.yaml", "Path to config file")
	rootCmd.AddCommand(startCmd)
}

// openDatabase creates the appropriate database adapter based on config.
func openDatabase(cfg *config.Config) (db.Database, error) {
	switch cfg.Database.Driver {
	case "sqlite":
		return db.NewSqliteAdapter(cfg.Database.DSN, cfg.Database.SyncDB)
	case "postgres":
		return db.NewPostgresAdapter(cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Database.Driver)
	}
}
