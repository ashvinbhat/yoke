package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var (
	listStatus   string
	listPriority int
	listTag      string
	listOpen     bool
	listBlocked  bool
	listReady    bool
	listAll      bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: `Lists tasks with optional filters.

Examples:
  yoke list                  # Open tasks
  yoke list --all            # All tasks including done
  yoke list --status active  # Only active tasks
  yoke list --tag backend    # Tasks with tag
  yoke list --ready          # Unblocked, ready to work`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := task.ListOptions{
			Open: !listAll,
		}

		if listStatus != "" {
			s := task.Status(listStatus)
			if !s.IsValid() {
				return fmt.Errorf("invalid status: %s", listStatus)
			}
			opts.Status = &s
			opts.Open = false // Override open filter when status is specified
		}

		if listPriority > 0 {
			opts.Priority = &listPriority
		}

		if listTag != "" {
			opts.Tag = &listTag
		}

		if listBlocked {
			opts.Blocked = true
			opts.Open = false
		}

		if listReady {
			opts.Ready = true
			opts.Open = false
		}

		tasks, err := store.List(opts)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		// Print header
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

		fmt.Printf("\n%d task(s)\n", len(tasks))
		return nil
	},
}

func formatStatus(s task.Status) string {
	switch s {
	case task.StatusPending:
		return "pending"
	case task.StatusActive:
		return "active"
	case task.StatusInProgress:
		return "IN PROGRESS"
	case task.StatusBlocked:
		return "BLOCKED"
	case task.StatusDone:
		return "done"
	case task.StatusDropped:
		return "dropped"
	default:
		return string(s)
	}
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status")
	listCmd.Flags().IntVarP(&listPriority, "priority", "p", 0, "Filter by priority")
	listCmd.Flags().StringVarP(&listTag, "tag", "t", "", "Filter by tag")
	listCmd.Flags().BoolVar(&listOpen, "open", false, "Only open tasks (default)")
	listCmd.Flags().BoolVar(&listBlocked, "blocked", false, "Only blocked tasks")
	listCmd.Flags().BoolVar(&listReady, "ready", false, "Only ready tasks (unblocked)")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Include completed tasks")
}
