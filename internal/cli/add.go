package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var (
	addPriority int
	addTags     []string
	addParent   string
	addBody     string
	addKeys     []string
	addFrom     string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Long: `Creates a new task with the given title.

A key (--key) makes the task findable and one of a kind. When a key already
finds a task, that task is used instead: it takes the new keys and tags, a
done or dropped one is reopened, and --body becomes a note on it.

Examples:
  yoke add "Implement user authentication"
  yoke add "Fix login bug" --priority 1 --tag backend
  yoke add "Write tests" --parent a3f8
  yoke add "Segment paging" --key pr:backend-6952 --key thread:local_ab12
  yoke add "Retry helper leaks segments" --tag finding --from 42`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")

		t := task.New(title)
		t.Priority = addPriority
		t.Body = addBody

		for _, tag := range addTags {
			t.AddTag(tag)
		}
		for _, k := range addKeys {
			if !task.ValidKey(k) {
				return fmt.Errorf("%q isn't a key: a key has no spaces and has a ':' or '-' (CB-123, pr:backend-42)", k)
			}
			t.AddKey(k)
		}

		if addParent != "" {
			parent, err := store.Get(addParent)
			if err != nil {
				return fmt.Errorf("parent task not found: %s", addParent)
			}
			t.Parent = &parent.ID
		}
		if addFrom != "" {
			from, err := store.Get(addFrom)
			if err != nil {
				return fmt.Errorf("task to find this from not found: %s", addFrom)
			}
			t.FoundFrom = &from.ID
		}

		got, added, err := store.Add(t)
		if err != nil {
			return fmt.Errorf("failed to add task: %w", err)
		}

		verb := map[task.Added]string{task.AddedNew: "Created", task.AddedFound: "Found", task.AddedReopened: "Reopened"}[added]
		fmt.Printf("%s task #%d [%s]: %s\n", verb, got.Seq, got.ID, got.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().IntVarP(&addPriority, "priority", "p", 3, "Priority (1-5, 1 is highest)")
	addCmd.Flags().StringArrayVarP(&addTags, "tag", "t", []string{}, "Tags (can be repeated)")
	addCmd.Flags().StringVar(&addParent, "parent", "", "Parent task ID")
	addCmd.Flags().StringVarP(&addBody, "body", "b", "", "Task description/body")
	addCmd.Flags().StringArrayVarP(&addKeys, "key", "k", []string{}, "Key that finds the task: CB-123, pr:<repo>-<n>, thread:<id> (can be repeated)")
	addCmd.Flags().StringVar(&addFrom, "from", "", "Task this one was found from")
}
