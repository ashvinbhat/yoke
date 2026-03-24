package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks by keyword",
	Long: `Searches for tasks matching a keyword in title or body.

The search is case-insensitive and matches partial words.

Examples:
  yoke search "api"        # Find tasks containing "api"
  yoke search database     # Find tasks about database
  yoke search "unit test"  # Find tasks about unit tests`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		tasks, err := store.Search(query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Printf("No tasks found matching \"%s\"\n", query)
			return nil
		}

		// Print header
		fmt.Printf("Search results for \"%s\":\n\n", query)
		fmt.Printf("%-4s %-6s %-12s %-3s %s\n", "#", "ID", "STATUS", "PRI", "TITLE")
		fmt.Println(strings.Repeat("-", 60))

		for _, t := range tasks {
			status := formatStatus(t.Status)
			tags := ""
			if len(t.Tags) > 0 {
				tags = " [" + strings.Join(t.Tags, ", ") + "]"
			}
			fmt.Printf("%-4d %-6s %-12s P%-2d %s%s\n",
				t.Seq, t.ID, status, t.Priority, t.Title, tags)
		}

		fmt.Printf("\n%d task(s) found\n", len(tasks))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
