package cli

import (
	"fmt"

	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var doneOutcome string

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Complete a task",
	Long: `Marks a task as done.

Examples:
  yoke done a3f8
  yoke done 42 --outcome "Implemented with JWT auth"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		if t.Status == task.StatusDone {
			fmt.Printf("Task #%d [%s] is already done\n", t.Seq, t.ID)
			return nil
		}

		if t.Status == task.StatusDropped {
			return fmt.Errorf("task was dropped, cannot mark as done")
		}

		oldStatus := t.Status
		t.Done(doneOutcome)

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		store.UpdateStatus(t.ID, oldStatus, t.Status)

		fmt.Printf("Completed task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		if doneOutcome != "" {
			fmt.Printf("Outcome: %s\n", doneOutcome)
		}
		return nil
	},
}

func init() {
	doneCmd.Flags().StringVarP(&doneOutcome, "outcome", "o", "", "What happened / what was learned")
}
