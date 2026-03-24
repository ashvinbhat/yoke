package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag <id> <tag>",
	Short: "Add a tag to a task",
	Long: `Adds a tag to a task.

Examples:
  yoke tag a3f8 backend
  yoke tag 42 urgent`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		tag := strings.ToLower(args[1])

		// Check if already has tag
		for _, existing := range t.Tags {
			if existing == tag {
				fmt.Printf("Task #%d already has tag '%s'\n", t.Seq, tag)
				return nil
			}
		}

		t.AddTag(tag)
		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		store.LogEvent(t.ID, "tag_added", "", tag)
		fmt.Printf("Added tag '%s' to task #%d [%s]\n", tag, t.Seq, t.ID)
		return nil
	},
}

var untagCmd = &cobra.Command{
	Use:   "untag <id> <tag>",
	Short: "Remove a tag from a task",
	Long: `Removes a tag from a task.

Examples:
  yoke untag a3f8 backend
  yoke untag 42 urgent`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		tag := strings.ToLower(args[1])

		// Check if has tag
		found := false
		for _, existing := range t.Tags {
			if existing == tag {
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Task #%d doesn't have tag '%s'\n", t.Seq, tag)
			return nil
		}

		t.RemoveTag(tag)
		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		store.LogEvent(t.ID, "tag_removed", tag, "")
		fmt.Printf("Removed tag '%s' from task #%d [%s]\n", tag, t.Seq, t.ID)
		return nil
	},
}

var tagsCmd = &cobra.Command{
	Use:   "tags [id]",
	Short: "Show tags",
	Long: `Shows tags for a task or all unique tags.

Without an ID, shows all unique tags in use.
With an ID, shows tags for that task.

Examples:
  yoke tags          # All unique tags
  yoke tags a3f8     # Tags for task a3f8`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return showAllTags()
		}

		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		if len(t.Tags) == 0 {
			fmt.Printf("Task #%d has no tags\n", t.Seq)
			return nil
		}

		fmt.Printf("Tags for task #%d [%s]: %s\n", t.Seq, t.ID, strings.Join(t.Tags, ", "))
		return nil
	},
}

func showAllTags() error {
	tasks, err := store.List(task.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	tagCounts := make(map[string]int)
	for _, t := range tasks {
		for _, tag := range t.Tags {
			tagCounts[tag]++
		}
	}

	if len(tagCounts) == 0 {
		fmt.Println("No tags in use")
		return nil
	}

	fmt.Println("Tags in use:")
	for tag, count := range tagCounts {
		fmt.Printf("  %s (%d tasks)\n", tag, count)
	}

	return nil
}

// parseStatus converts a string to a Status.
func parseStatus(s string) task.Status {
	switch strings.ToLower(s) {
	case "pending":
		return task.StatusPending
	case "active":
		return task.StatusActive
	case "in_progress", "in-progress", "inprogress":
		return task.StatusInProgress
	case "blocked":
		return task.StatusBlocked
	case "done":
		return task.StatusDone
	case "dropped":
		return task.StatusDropped
	default:
		return ""
	}
}

func init() {
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(untagCmd)
	rootCmd.AddCommand(tagsCmd)
}
