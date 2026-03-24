package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	editTitle    string
	editBody     string
	editPriority int
	editStatus   string
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task",
	Long: `Edit task properties.

Examples:
  yoke edit a3f8 --title "New title"
  yoke edit 42 --priority 1
  yoke edit a3f8 --status active`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		changed := false

		if editTitle != "" && editTitle != t.Title {
			store.LogEvent(t.ID, "title_changed", t.Title, editTitle)
			t.Title = editTitle
			changed = true
		}

		if editBody != "" {
			t.Body = editBody
			changed = true
		}

		if editPriority > 0 && editPriority != t.Priority {
			store.LogEvent(t.ID, "priority_changed", fmt.Sprintf("%d", t.Priority), fmt.Sprintf("%d", editPriority))
			t.Priority = editPriority
			changed = true
		}

		if editStatus != "" {
			newStatus := parseStatus(editStatus)
			if newStatus != "" && newStatus != t.Status {
				store.LogEvent(t.ID, "status_changed", string(t.Status), string(newStatus))
				t.Status = newStatus
				changed = true
			}
		}

		if !changed {
			fmt.Println("No changes specified")
			return nil
		}

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		fmt.Printf("Updated task #%d [%s]\n", t.Seq, t.ID)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "New title")
	editCmd.Flags().StringVar(&editBody, "body", "", "New body/description")
	editCmd.Flags().IntVar(&editPriority, "priority", 0, "New priority (1-5)")
	editCmd.Flags().StringVar(&editStatus, "status", "", "New status")
}
