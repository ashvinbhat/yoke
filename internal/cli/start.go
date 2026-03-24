package cli

import (
	"fmt"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start working on a task",
	Long: `Marks a task as in_progress.

Examples:
  yoke start a3f8
  yoke start 42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		if t.Status == task.StatusDone {
			return fmt.Errorf("task is already done")
		}

		if t.Status == task.StatusDropped {
			return fmt.Errorf("task was dropped")
		}

		if t.Status == task.StatusInProgress {
			fmt.Printf("Task #%d [%s] is already in progress\n", t.Seq, t.ID)
			return nil
		}

		oldStatus := t.Status
		t.Start()

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		store.UpdateStatus(t.ID, oldStatus, t.Status)

		fmt.Printf("Started task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		return nil
	},
}
