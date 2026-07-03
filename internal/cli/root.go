// Package cli implements the yoke command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/ashvinbhat/yoke/internal/config"
	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var (
	store *task.Store
	cfg   *config.Config
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "yoke",
	Short: "The yoke that binds your tasks to your work",
	Long: `yoke is a local-first task management system.

It keeps your tasks organized with SQLite storage, Notion sync,
and a powerful CLI for managing your daily work.

Get started:
  yoke init                 Initialize yoke
  yoke add "My first task"  Create a task
  yoke list                 List all tasks`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip setup for commands that must work before/without a database
		if cmd.Name() == "init" || cmd.Name() == "docs" {
			return nil
		}

		// Check if initialized
		if !config.Exists() {
			return fmt.Errorf("yoke not initialized. Run 'yoke init' first")
		}

		// Load config
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Open store
		store, err = task.NewStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if store != nil {
			store.Close()
		}
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(dropCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(editCmd)
}
