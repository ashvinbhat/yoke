package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var subtaskPriority int
var subtaskTags []string

var subtaskCmd = &cobra.Command{
	Use:   "subtask <parent-id> <title>",
	Short: "Create a subtask",
	Long: `Creates a subtask under a parent task.

This is a shorthand for: yoke add "title" --parent <id>

Examples:
  yoke subtask a3f8 "Write unit tests"
  yoke subtask 1 "Design database schema" --priority 2`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		parentID := args[0]
		title := strings.Join(args[1:], " ")

		// Verify parent exists
		parent, err := store.Get(parentID)
		if err != nil {
			return fmt.Errorf("parent task not found: %s", parentID)
		}

		// Create subtask
		t := task.New(title)
		t.Priority = subtaskPriority
		t.Parent = &parent.ID

		for _, tag := range subtaskTags {
			t.AddTag(tag)
		}

		if err := store.Create(t); err != nil {
			return fmt.Errorf("failed to create subtask: %w", err)
		}

		fmt.Printf("Created subtask #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		fmt.Printf("  └─ under #%d [%s]: %s\n", parent.Seq, parent.ID, parent.Title)
		return nil
	},
}

func init() {
	subtaskCmd.Flags().IntVarP(&subtaskPriority, "priority", "p", 3, "Priority (1-5)")
	subtaskCmd.Flags().StringArrayVarP(&subtaskTags, "tag", "t", []string{}, "Tags")
	rootCmd.AddCommand(subtaskCmd)
}
