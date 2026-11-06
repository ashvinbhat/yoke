package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key <id> <key>...",
	Short: "Add keys that find a task",
	Long: `Adds keys to a task. A key finds one task: CB-123 for a ticket,
pr:<repo>-<n> for a PR, thread:<id> for a thread. Any yoke command that
takes an id also takes a key, and adding a task under a key it already has
finds that task instead of creating another.

Examples:
  yoke key 42 pr:backend-6952
  yoke key CB-18304 thread:local_ab12`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}
		for _, k := range args[1:] {
			if !task.ValidKey(k) {
				return fmt.Errorf("%q isn't a key: a key has no spaces and has a ':' or '-' (CB-123, pr:backend-42)", k)
			}
			t.AddKey(k)
		}
		if err := store.Update(t); err != nil {
			return err
		}
		store.LogEvent(t.ID, "keys_added", "", strings.Join(args[1:], ", "))
		fmt.Printf("Task #%d [%s] keys: %s\n", t.Seq, t.ID, strings.Join(t.Keys, ", "))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(keyCmd)
}
