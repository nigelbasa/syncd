package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize syncd configuration",
	Long:  `Generates a standard syncd.yaml configuration file in the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		const defaultConfig = `app:
  mode: "client" # or "server"
  port: 8080
  remote_url: "http://localhost:8080" # Only required if mode=client

database:
  driver: "sqlite" # or "postgres"
  dsn: "./local_data/app.db"
  sync_db: "./local_data/syncd_logs.db" # Required for SQLite Outbox

sync:
  api_key: "default-secret-key-change-me"
  poll_interval: "5s"
  tables: 
    - "users"
    - "items"
`
		err := os.WriteFile("syncd.yaml", []byte(defaultConfig), 0644)
		if err != nil {
			fmt.Printf("Failed to generate syncd.yaml: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully created syncd.yaml")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
