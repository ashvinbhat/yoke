package cli

import (
	"fmt"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var unblockCmd = &cobra.Command{
	Use:   "unblock <task-id> <blocker-id>",
	Short: "Remove a dependency (blocker) from a task",
	Long: `Removes a blocker relationship between tasks.

If the task has no remaining blockers and was blocked, it will be set to active.

Examples:
  yoke unblock 1 2        # Remove task 2 as blocker of task 1
  yoke unblock a3f8 b2c1  # Using short IDs`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		blockerID := args[1]

		// Get the blocked task
		t, err := store.Get(taskID)
		if err != nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		// Get the blocker task (to display info)
		blocker, err := store.Get(blockerID)
		if err != nil {
			return fmt.Errorf("blocker task not found: %s", blockerID)
		}

		// Check if the blocker exists in the task's blockers
		found := false
		for _, b := range t.Blockers {
			if b == blocker.ID {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("task #%d is not blocked by #%d", t.Seq, blocker.Seq)
		}

		// Remove the blocker
		t.RemoveBlocker(blocker.ID)

		// If no more blockers and task was blocked, set to active
		if !t.IsBlocked() && t.Status == task.StatusBlocked {
			t.Activate()
			fmt.Printf("Task #%d [%s] is now unblocked and active\n", t.Seq, t.ID)
		}

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Log the event
		store.LogEvent(t.ID, "blocker_removed", blocker.ID, "")

		fmt.Printf("Removed blocker #%d [%s] from task #%d [%s]\n",
			blocker.Seq, blocker.ID, t.Seq, t.ID)

		if len(t.Blockers) > 0 {
			fmt.Printf("  Remaining blockers: %d\n", len(t.Blockers))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(unblockCmd)
}
