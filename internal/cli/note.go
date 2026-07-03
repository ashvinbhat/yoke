package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note <id> <text>",
	Short: "Add a note to a task",
	Long: `Adds a note/comment to a task.

Examples:
  yoke note a3f8 "Found the root cause"
  yoke note 42 "Waiting for API access"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		noteText := strings.Join(args[1:], " ")

		// Verify task exists
		t, err := store.Get(taskID)
		if err != nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if err := store.AddNote(t.ID, noteText); err != nil {
			return fmt.Errorf("failed to add note: %w", err)
		}

		fmt.Printf("Added note to task #%d [%s]\n", t.Seq, t.ID)
		return nil
	},
}

var notesJSON bool

var notesCmd = &cobra.Command{
	Use:   "notes <id>",
	Short: "Show notes for a task",
	Long: `Shows all notes attached to a task.

Examples:
  yoke notes a3f8
  yoke notes 42
  yoke notes 42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		notes, err := store.GetNotes(t.ID)
		if err != nil {
			return fmt.Errorf("failed to get notes: %w", err)
		}

		if notesJSON {
			return printJSON(toNotesJSON(notes))
		}

		if len(notes) == 0 {
			fmt.Printf("No notes for task #%d [%s]\n", t.Seq, t.ID)
			return nil
		}

		fmt.Printf("Notes for task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		fmt.Println(strings.Repeat("-", 40))

		for _, n := range notes {
			fmt.Printf("[%s] %s\n", n.CreatedAt.Format("2006-01-02 15:04"), n.Content)
		}

		return nil
	},
}

func init() {
	notesCmd.Flags().BoolVar(&notesJSON, "json", false, "Output as JSON array")
	rootCmd.AddCommand(notesCmd)
}
