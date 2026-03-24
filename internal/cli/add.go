package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var (
	addPriority int
	addTags     []string
	addParent   string
	addBody     string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Long: `Creates a new task with the given title.

Examples:
  yoke add "Implement user authentication"
  yoke add "Fix login bug" --priority 1 --tag backend
  yoke add "Write tests" --parent a3f8`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")

		t := task.New(title)
		t.Priority = addPriority
		t.Body = addBody

		for _, tag := range addTags {
			t.AddTag(tag)
		}

		if addParent != "" {
			// Verify parent exists
			_, err := store.Get(addParent)
			if err != nil {
				return fmt.Errorf("parent task not found: %s", addParent)
			}
			t.Parent = &addParent
		}

		if err := store.Create(t); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Printf("Created task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().IntVarP(&addPriority, "priority", "p", 3, "Priority (1-5, 1 is highest)")
	addCmd.Flags().StringArrayVarP(&addTags, "tag", "t", []string{}, "Tags (can be repeated)")
	addCmd.Flags().StringVar(&addParent, "parent", "", "Parent task ID")
	addCmd.Flags().StringVarP(&addBody, "body", "b", "", "Task description/body")
}
