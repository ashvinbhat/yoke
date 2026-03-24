package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tasks ready to work on",
	Long: `Lists tasks that have no active blockers and are ready to work on.

A task is ready if:
- It's open (not done or dropped)
- It has no blockers, OR all its blockers are completed

Examples:
  yoke ready           # Show all ready tasks
  yoke ready --tag dev # Show ready tasks with 'dev' tag`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get all open tasks
		opts := task.ListOptions{
			Open: true,
		}

		// Apply tag filter if provided
		tagFilter, _ := cmd.Flags().GetString("tag")
		if tagFilter != "" {
			opts.Tag = &tagFilter
		}

		tasks, err := store.List(opts)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		// Filter to only ready tasks (no active blockers)
		var readyTasks []*task.Task
		for _, t := range tasks {
			if isTaskReady(t) {
				readyTasks = append(readyTasks, t)
			}
		}

		if len(readyTasks) == 0 {
			fmt.Println("No ready tasks")
			return nil
		}

		// Print header
		fmt.Printf("%-4s %-6s %-12s %-3s %s\n", "#", "ID", "STATUS", "PRI", "TITLE")
		fmt.Println(strings.Repeat("-", 60))

		for _, t := range readyTasks {
			status := formatStatus(t.Status)
			tags := ""
			if len(t.Tags) > 0 {
				tags = " [" + strings.Join(t.Tags, ", ") + "]"
			}
			fmt.Printf("%-4d %-6s %-12s P%-2d %s%s\n",
				t.Seq, t.ID, status, t.Priority, t.Title, tags)
		}

		fmt.Printf("\n%d ready task(s)\n", len(readyTasks))
		return nil
	},
}

// isTaskReady returns true if a task has no active blockers.
// A task is ready if it has no blockers, or all its blockers are completed.
func isTaskReady(t *task.Task) bool {
	if len(t.Blockers) == 0 {
		return true
	}

	// Check if all blockers are completed
	for _, blockerID := range t.Blockers {
		// Use GetByID since blockers are stored as exact IDs
		blocker, err := store.GetByID(blockerID)
		if err != nil {
			// If we can't find the blocker, assume it's not blocking
			continue
		}
		if !blocker.IsComplete() {
			// Found an active blocker
			return false
		}
	}

	return true
}

func init() {
	readyCmd.Flags().StringP("tag", "t", "", "Filter by tag")
	rootCmd.AddCommand(readyCmd)
}
