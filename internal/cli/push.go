package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var (
	pushDryRun bool
	pushYes    bool
)

var pushCmd = &cobra.Command{
	Use:   "push [task-id]",
	Short: "Push local changes to Notion",
	Long: `Pushes local task changes to Notion.

⚠️  CAUTION: This modifies data in Notion.

Updates Notion pages with changes from linked local tasks.
Only pushes tasks that have been modified locally since last sync.

Safety features:
- Validates assignee before every write
- Requires confirmation (use --yes to skip)
- Use --dry-run to preview changes first

Examples:
  yoke push --dry-run    # Preview what would be pushed
  yoke push 9            # Push specific task (with confirmation)
  yoke push --yes        # Push all without confirmation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check Notion config
		if cfg.Notion.Token == "" {
			return fmt.Errorf("Notion token not configured")
		}

		// Create Notion client
		client := notion.NewClient(cfg.Notion.Token, cfg.Notion.DatabaseID, cfg.Notion.AssigneeName)
		ctx := context.Background()

		if len(args) > 0 {
			// Push specific task
			return pushSingleTask(ctx, client, args[0])
		}

		// Push all linked tasks with local changes
		return pushAllTasks(ctx, client)
	},
}

func pushSingleTask(ctx context.Context, client *notion.Client, taskID string) error {
	t, err := store.Get(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if t.NotionID == nil || *t.NotionID == "" {
		return fmt.Errorf("task #%d is not linked to Notion", t.Seq)
	}

	// SAFETY: Validate assignee before any write
	if err := client.ValidateBeforeWrite(ctx, *t.NotionID); err != nil {
		return fmt.Errorf("safety check failed: %w", err)
	}

	// Get current Notion page to compare
	page, err := client.GetPage(ctx, *t.NotionID)
	if err != nil {
		return fmt.Errorf("failed to fetch Notion page: %w", err)
	}

	update := notion.TaskToNotionUpdate(t, page)

	if !update.HasChanges {
		fmt.Printf("Task #%d [%s] has no changes to push\n", t.Seq, t.ID)
		return nil
	}

	// Dry run - just show what would be pushed
	if pushDryRun {
		plan := &notion.PushPlan{
			Tasks: []notion.PushTaskPlan{{
				Seq:      t.Seq,
				ID:       t.ID,
				Title:    t.Title,
				NotionID: *t.NotionID,
				Changes:  update.Changes,
			}},
		}
		plan.Print()
		return nil
	}

	// Require confirmation unless --yes
	if !pushYes {
		fmt.Printf("\n⚠️  About to update Notion page:\n")
		fmt.Printf("   Task #%d: %s\n\n", t.Seq, t.Title)
		fmt.Printf("Changes:\n")
		for _, c := range update.Changes {
			fmt.Printf("  %s: %q → %q\n", c.Field, c.OldValue, c.NewValue)
		}

		if !notion.ConfirmPush(1) {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// SAFETY: Re-validate assignee immediately before write
	if err := client.ValidateBeforeWrite(ctx, *t.NotionID); err != nil {
		return fmt.Errorf("safety re-check failed: %w", err)
	}

	// Push the changes
	if err := client.UpdatePage(ctx, *t.NotionID, update); err != nil {
		return fmt.Errorf("failed to update Notion page: %w", err)
	}

	// Update local sync time
	now := time.Now()
	t.SyncedAt = &now
	if err := store.Update(t); err != nil {
		fmt.Printf("Warning: failed to update local sync time: %v\n", err)
	}

	store.LogEvent(t.ID, "pushed_to_notion", "", *t.NotionID)

	fmt.Printf("Pushed task #%d [%s] to Notion\n", t.Seq, t.ID)
	for _, c := range update.Changes {
		fmt.Printf("  %s: %q → %q\n", c.Field, c.OldValue, c.NewValue)
	}

	return nil
}

func pushAllTasks(ctx context.Context, client *notion.Client) error {
	linkedTasks, err := store.ListLinked()
	if err != nil {
		return fmt.Errorf("failed to list linked tasks: %w", err)
	}

	if len(linkedTasks) == 0 {
		fmt.Println("No tasks linked to Notion")
		return nil
	}

	// Build push plan
	var plan notion.PushPlan
	var tasksToPush []*pushItem

	for _, t := range linkedTasks {
		if t.NotionID == nil || *t.NotionID == "" {
			continue
		}

		// Check if task has local changes
		if !notion.ShouldPush(t) {
			continue
		}

		// SAFETY: Validate assignee
		if err := client.ValidateBeforeWrite(ctx, *t.NotionID); err != nil {
			continue // Skip tasks not assigned to user
		}

		page, err := client.GetPage(ctx, *t.NotionID)
		if err != nil {
			fmt.Printf("Warning: failed to fetch task #%d: %v\n", t.Seq, err)
			continue
		}

		update := notion.TaskToNotionUpdate(t, page)
		if !update.HasChanges {
			continue
		}

		plan.Tasks = append(plan.Tasks, notion.PushTaskPlan{
			Seq:      t.Seq,
			ID:       t.ID,
			Title:    t.Title,
			NotionID: *t.NotionID,
			Changes:  update.Changes,
		})

		tasksToPush = append(tasksToPush, &pushItem{
			task:   t,
			update: update,
		})
	}

	if len(plan.Tasks) == 0 {
		fmt.Println("No tasks with changes to push")
		return nil
	}

	// Dry run - just show plan
	if pushDryRun {
		plan.Print()
		return nil
	}

	// Show plan and require confirmation
	plan.PrintSummary()

	if !pushYes {
		if !notion.ConfirmPush(len(plan.Tasks)) {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Push each task
	var pushed, failed int
	for _, item := range tasksToPush {
		// SAFETY: Re-validate assignee immediately before each write
		if err := client.ValidateBeforeWrite(ctx, *item.task.NotionID); err != nil {
			fmt.Printf("Skipped #%d: %v\n", item.task.Seq, err)
			failed++
			continue
		}

		if err := client.UpdatePage(ctx, *item.task.NotionID, item.update); err != nil {
			fmt.Printf("Failed #%d: %v\n", item.task.Seq, err)
			failed++
			continue
		}

		// Update local sync time
		now := time.Now()
		item.task.SyncedAt = &now
		store.Update(item.task)
		store.LogEvent(item.task.ID, "pushed_to_notion", "", *item.task.NotionID)

		fmt.Printf("Pushed #%d: %s\n", item.task.Seq, item.task.Title)
		pushed++
	}

	fmt.Printf("\n%d pushed, %d failed\n", pushed, failed)
	return nil
}

type pushItem struct {
	task   *task.Task
	update *notion.NotionUpdate
}

func init() {
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "Preview changes without pushing to Notion")
	pushCmd.Flags().BoolVar(&pushYes, "yes", false, "Skip confirmation prompt")
	rootCmd.AddCommand(pushCmd)
}
