package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task details",
	Long: `Shows detailed information about a task.

The id can be the task ID (e.g., a3f8) or sequence number (e.g., 42).

Examples:
  yoke show a3f8
  yoke show 42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		// Header
		fmt.Printf("Task #%d [%s]\n", t.Seq, t.ID)
		fmt.Println(strings.Repeat("=", 40))

		// Title
		fmt.Printf("\nTitle:    %s\n", t.Title)
		fmt.Printf("Status:   %s\n", formatStatus(t.Status))
		fmt.Printf("Priority: P%d\n", t.Priority)

		// Tags
		if len(t.Tags) > 0 {
			fmt.Printf("Tags:     %s\n", strings.Join(t.Tags, ", "))
		}

		// Body
		if t.Body != "" {
			fmt.Printf("\nDescription:\n%s\n", t.Body)
		}

		// Hierarchy
		if t.Parent != nil {
			parent, _ := store.GetByID(*t.Parent)
			if parent != nil {
				fmt.Printf("\nParent:   #%d [%s] %s\n", parent.Seq, parent.ID, parent.Title)
			}
		}

		// Blockers
		if len(t.Blockers) > 0 {
			fmt.Println("\nBlocked by:")
			for _, bid := range t.Blockers {
				blocker, _ := store.GetByID(bid)
				if blocker != nil {
					fmt.Printf("  - #%d [%s] %s\n", blocker.Seq, blocker.ID, blocker.Title)
				}
			}
		}

		// External links
		if t.NotionURL != nil {
			fmt.Printf("\nNotion:   %s\n", *t.NotionURL)
		}

		// Timestamps
		fmt.Println("\nTimeline:")
		fmt.Printf("  Created:  %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("  Updated:  %s\n", t.UpdatedAt.Format("2006-01-02 15:04"))
		if t.StartedAt != nil {
			fmt.Printf("  Started:  %s\n", t.StartedAt.Format("2006-01-02 15:04"))
		}
		if t.DoneAt != nil {
			fmt.Printf("  Done:     %s\n", t.DoneAt.Format("2006-01-02 15:04"))
		}

		// Outcome
		if t.Outcome != nil {
			fmt.Printf("\nOutcome:\n%s\n", *t.Outcome)
		}

		// Notes
		notes, _ := store.GetNotes(t.ID)
		if len(notes) > 0 {
			fmt.Println("\nNotes:")
			for _, n := range notes {
				fmt.Printf("  [%s] %s\n", n.CreatedAt.Format("01-02 15:04"), n.Content)
			}
		}

		return nil
	},
}
