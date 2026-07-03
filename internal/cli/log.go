package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var (
	logLimit int
	logJSON  bool
)

var logCmd = &cobra.Command{
	Use:   "log [id]",
	Short: "Show task history",
	Long: `Shows the event history for a task or recent activity.

Without an ID, shows recent activity across all tasks.
With an ID, shows the full history of that task.

Examples:
  yoke log           # Recent activity (last 20 events)
  yoke log a3f8      # History for task a3f8
  yoke log --limit 50 # Last 50 events`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Show recent events across all tasks
			return showRecentEvents()
		}

		// Show events for specific task
		return showTaskEvents(args[0])
	},
}

func showRecentEvents() error {
	events, err := store.GetRecentEvents(logLimit)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	if logJSON {
		return printJSON(toEventsJSON(events))
	}

	if len(events) == 0 {
		fmt.Println("No activity recorded")
		return nil
	}

	fmt.Println("Recent Activity")
	fmt.Println(strings.Repeat("=", 60))

	for _, e := range events {
		// Try to get task title for context
		task, _ := store.GetByID(e.TaskID)
		taskRef := e.TaskID
		if task != nil {
			taskRef = fmt.Sprintf("#%d [%s]", task.Seq, task.ID)
		}

		timestamp := e.CreatedAt.Format("2006-01-02 15:04")
		desc := formatEventDescription(e)

		fmt.Printf("[%s] %s: %s\n", timestamp, taskRef, desc)
	}

	return nil
}

func showTaskEvents(idOrSeq string) error {
	t, err := store.Get(idOrSeq)
	if err != nil {
		return fmt.Errorf("task not found: %s", idOrSeq)
	}

	events, err := store.GetEvents(t.ID)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	if logJSON {
		return printJSON(toEventsJSON(events))
	}

	if len(events) == 0 {
		fmt.Printf("No history for task #%d [%s]\n", t.Seq, t.ID)
		return nil
	}

	fmt.Printf("History for Task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
	fmt.Println(strings.Repeat("=", 60))

	for _, e := range events {
		timestamp := e.CreatedAt.Format("2006-01-02 15:04")
		desc := formatEventDescription(e)
		fmt.Printf("[%s] %s\n", timestamp, desc)
	}

	return nil
}

func formatEventDescription(e task.Event) string {
	switch e.EventType {
	case "created":
		return fmt.Sprintf("Created: %s", e.NewValue)
	case "status_changed":
		return fmt.Sprintf("Status: %s → %s", e.OldValue, e.NewValue)
	case "note_added":
		note := e.NewValue
		if len(note) > 50 {
			note = note[:47] + "..."
		}
		return fmt.Sprintf("Note added: %s", note)
	case "deleted":
		return "Deleted"
	case "priority_changed":
		return fmt.Sprintf("Priority: P%s → P%s", e.OldValue, e.NewValue)
	case "title_changed":
		return fmt.Sprintf("Title changed: %s → %s", e.OldValue, e.NewValue)
	case "tag_added":
		return fmt.Sprintf("Tag added: %s", e.NewValue)
	case "tag_removed":
		return fmt.Sprintf("Tag removed: %s", e.OldValue)
	case "blocker_added":
		return fmt.Sprintf("Blocked by: %s", e.NewValue)
	case "blocker_removed":
		return fmt.Sprintf("Unblocked: %s", e.OldValue)
	case "parent_set":
		return fmt.Sprintf("Parent set: %s", e.NewValue)
	default:
		if e.OldValue != "" && e.NewValue != "" {
			return fmt.Sprintf("%s: %s → %s", e.EventType, e.OldValue, e.NewValue)
		} else if e.NewValue != "" {
			return fmt.Sprintf("%s: %s", e.EventType, e.NewValue)
		}
		return e.EventType
	}
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Number of events to show")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "Output as JSON array")
}
