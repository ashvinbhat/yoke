package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <task-id> --by <blocker-id>",
	Short: "Add a dependency (blocker) to a task",
	Long: `Marks a task as blocked by another task.

The blocker task must be completed before the blocked task can proceed.

Examples:
  yoke block 1 --by 2        # Task 1 is blocked by task 2
  yoke block a3f8 --by b2c1  # Using short IDs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		blockerID, _ := cmd.Flags().GetString("by")

		if blockerID == "" {
			return fmt.Errorf("--by flag is required")
		}

		// Get the task to be blocked
		task, err := store.Get(taskID)
		if err != nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		// Get the blocker task
		blocker, err := store.Get(blockerID)
		if err != nil {
			return fmt.Errorf("blocker task not found: %s", blockerID)
		}

		// Can't block self
		if task.ID == blocker.ID {
			return fmt.Errorf("a task cannot block itself")
		}

		// Check if already blocked by this task
		for _, b := range task.Blockers {
			if b == blocker.ID {
				return fmt.Errorf("task #%d is already blocked by #%d", task.Seq, blocker.Seq)
			}
		}

		// Check for cycles
		if wouldCreateCycle(task.ID, blocker.ID) {
			return fmt.Errorf("cannot add blocker: would create a dependency cycle")
		}

		// Add the blocker
		task.AddBlocker(blocker.ID)

		// Update status to blocked if not already complete
		if !task.IsComplete() && task.IsBlocked() {
			task.Block()
		}

		if err := store.Update(task); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Log the event
		store.LogEvent(task.ID, "blocker_added", "", blocker.ID)

		fmt.Printf("Task #%d [%s] is now blocked by #%d [%s]\n",
			task.Seq, task.ID, blocker.Seq, blocker.ID)
		fmt.Printf("  %s → blocked by → %s\n", task.Title, blocker.Title)

		return nil
	},
}

// wouldCreateCycle checks if adding blockerID as a blocker of taskID would create a cycle.
// A cycle exists if taskID is already in the dependency chain of blockerID.
func wouldCreateCycle(taskID, blockerID string) bool {
	visited := make(map[string]bool)
	return hasPathTo(blockerID, taskID, visited)
}

// hasPathTo checks if there's a dependency path from startID to targetID.
func hasPathTo(startID, targetID string, visited map[string]bool) bool {
	if startID == targetID {
		return true
	}

	if visited[startID] {
		return false
	}
	visited[startID] = true

	// Use GetByID since we're working with stored task IDs
	task, err := store.GetByID(startID)
	if err != nil {
		return false
	}

	for _, blockerID := range task.Blockers {
		if hasPathTo(blockerID, targetID, visited) {
			return true
		}
	}

	return false
}

func init() {
	blockCmd.Flags().String("by", "", "ID of the blocker task (required)")
	blockCmd.MarkFlagRequired("by")
	rootCmd.AddCommand(blockCmd)
}
