package cli

import (
	"fmt"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var dropCmd = &cobra.Command{
	Use:   "drop <id>",
	Short: "Drop/abandon a task",
	Long: `Marks a task as dropped (won't do).

Examples:
  yoke drop a3f8
  yoke drop 42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		if t.Status == task.StatusDropped {
			fmt.Printf("Task #%d [%s] is already dropped\n", t.Seq, t.ID)
			return nil
		}

		if t.Status == task.StatusDone {
			return fmt.Errorf("task is already done, cannot drop")
		}

		oldStatus := t.Status
		t.Drop()

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		store.UpdateStatus(t.ID, oldStatus, t.Status)

		fmt.Printf("Dropped task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		return nil
	},
}
