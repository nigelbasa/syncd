package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "syncd",
	Short: "Syncd - Two-way database synchronizer",
	Long:  `A lightweight, high-performance Change Data Capture (CDC) and two-way synchronization service for offline-first local applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no command is given, default to printing help
		if err := cmd.Help(); err != nil {
			fmt.Println(err)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
