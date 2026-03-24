package cli

import (
	"fmt"

	"github.com/ashvinbhat/yoke/internal/config"
	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize yoke",
	Long:  `Creates the ~/.yoke directory with database and configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already initialized
		if config.Exists() {
			fmt.Printf("yoke already initialized at %s\n", config.YokeDir())
			return nil
		}

		// Create directory
		if err := config.EnsureDir(); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Create database
		s, err := task.NewStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		s.Close()

		// Create default config
		cfg := config.DefaultConfig()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Initialized yoke at %s\n", config.YokeDir())
		fmt.Println("\nNext steps:")
		fmt.Println("  yoke add \"My first task\"  # Create a task")
		fmt.Println("  yoke list                  # List tasks")

		return nil
	},
}
