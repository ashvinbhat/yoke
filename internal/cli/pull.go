package cli

import (
	"context"
	"fmt"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/ashvinbhat/yoke/task"
	"github.com/jomei/notionapi"
	"github.com/spf13/cobra"
)

var pullDryRun bool

var pullCmd = &cobra.Command{
	Use:   "pull [task-id]",
	Short: "Pull updates from Notion",
	Long: `Pulls updates from Notion to local tasks.

Updates local tasks with changes from their linked Notion pages.
Only pulls tasks that have been modified in Notion since last sync.

Examples:
  yoke pull              # Pull all linked tasks
  yoke pull 9            # Pull specific task
  yoke pull --dry-run    # Show what would change without applying`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check Notion config
		if cfg.Notion.Token == "" {
			return fmt.Errorf("Notion token not configured")
		}

		// Create Notion client
		client := notion.NewClient(cfg.Notion.Token, cfg.Notion.DatabaseID, cfg.Notion.AssigneeName)
		ctx := context.Background()

		if len(args) > 0 {
			// Pull specific task
			t, err := store.Get(args[0])
			if err != nil {
				return fmt.Errorf("task not found: %s", args[0])
			}
			if t.NotionID == nil || *t.NotionID == "" {
				return fmt.Errorf("task #%d is not linked to Notion", t.Seq)
			}

			// Validate assignee
			if err := client.ValidateAssignee(ctx, *t.NotionID); err != nil {
				return fmt.Errorf("safety check failed: %w", err)
			}

			page, err := client.GetPage(ctx, *t.NotionID)
			if err != nil {
				return fmt.Errorf("failed to fetch Notion page: %w", err)
			}

			// Fetch page content (body)
			body, err := client.GetPageContent(ctx, *t.NotionID)
			if err != nil {
				fmt.Printf("Warning: could not fetch page content: %v\n", err)
				body = t.Body // Keep existing body
			}

			update := notion.PageToTaskUpdate(page, t)

			// Check if body changed
			if body != t.Body {
				update.HasChanges = true
				oldLen := len(t.Body)
				newLen := len(body)
				update.Changes = append(update.Changes, notion.FieldChange{
					Field:    "Body",
					OldValue: fmt.Sprintf("%d chars", oldLen),
					NewValue: fmt.Sprintf("%d chars", newLen),
				})
			}

			if pullDryRun {
				printPullDryRun([]*pullTask{{task: t, update: update, body: body}})
				return nil
			}

			if !update.HasChanges && !notion.ShouldUpdate(page, t) {
				fmt.Printf("Task #%d [%s] is up to date\n", t.Seq, t.ID)
				return nil
			}

			update.ApplyToTask(t)
			t.Body = body // Apply body separately
			if err := store.Update(t); err != nil {
				return fmt.Errorf("failed to update task: %w", err)
			}

			store.LogEvent(t.ID, "pulled_from_notion", "", *t.NotionID)
			fmt.Printf("Pulled task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
			printChanges(update.Changes)
			return nil
		}

		// Pull all linked tasks
		linkedTasks, err := store.ListLinked()
		if err != nil {
			return fmt.Errorf("failed to list linked tasks: %w", err)
		}

		if len(linkedTasks) == 0 {
			fmt.Println("No tasks linked to Notion")
			return nil
		}

		var pullTasks []*pullTask
		var skipped int

		for _, t := range linkedTasks {
			if t.NotionID == nil || *t.NotionID == "" {
				continue
			}

			// Validate assignee
			if err := client.ValidateAssignee(ctx, *t.NotionID); err != nil {
				skipped++
				continue
			}

			page, err := client.GetPage(ctx, *t.NotionID)
			if err != nil {
				fmt.Printf("Warning: failed to fetch task #%d: %v\n", t.Seq, err)
				continue
			}

			// Fetch page content (body)
			body, err := client.GetPageContent(ctx, *t.NotionID)
			if err != nil {
				body = t.Body // Keep existing
			}

			update := notion.PageToTaskUpdate(page, t)

			// Check if body changed
			if body != t.Body {
				update.HasChanges = true
				update.Changes = append(update.Changes, notion.FieldChange{
					Field:    "Body",
					OldValue: fmt.Sprintf("%d chars", len(t.Body)),
					NewValue: fmt.Sprintf("%d chars", len(body)),
				})
			}

			pullTasks = append(pullTasks, &pullTask{task: t, update: update, page: page, body: body})
		}

		if pullDryRun {
			printPullDryRun(pullTasks)
			if skipped > 0 {
				fmt.Printf("\n%d task(s) skipped (not assigned to you)\n", skipped)
			}
			return nil
		}

		// Apply updates
		var updated, unchanged int
		for _, pt := range pullTasks {
			if !pt.update.HasChanges && !notion.ShouldUpdate(pt.page, pt.task) {
				unchanged++
				continue
			}

			pt.update.ApplyToTask(pt.task)
			pt.task.Body = pt.body // Apply body separately
			if err := store.Update(pt.task); err != nil {
				fmt.Printf("Warning: failed to update task #%d: %v\n", pt.task.Seq, err)
				continue
			}

			store.LogEvent(pt.task.ID, "pulled_from_notion", "", *pt.task.NotionID)
			updated++

			fmt.Printf("Pulled task #%d [%s]: %s\n", pt.task.Seq, pt.task.ID, pt.task.Title)
			printChanges(pt.update.Changes)
		}

		fmt.Printf("\n%d updated, %d unchanged", updated, unchanged)
		if skipped > 0 {
			fmt.Printf(", %d skipped", skipped)
		}
		fmt.Println()

		return nil
	},
}

type pullTask struct {
	task   *task.Task
	update *notion.TaskUpdate
	page   *notionapi.Page
	body   string // Page content as markdown
}

func printPullDryRun(tasks []*pullTask) {
	fmt.Println("=== DRY RUN - No changes will be made ===")

	var withChanges, upToDate int
	for _, pt := range tasks {
		if pt.update.HasChanges {
			withChanges++
			fmt.Printf("Task #%d [%s]: %s\n", pt.task.Seq, pt.task.ID, pt.task.Title)
			for _, c := range pt.update.Changes {
				fmt.Printf("  %s: %q → %q\n", c.Field, c.OldValue, c.NewValue)
			}
			fmt.Println()
		} else {
			upToDate++
		}
	}

	if withChanges == 0 {
		fmt.Println("All tasks are up to date")
	} else {
		fmt.Printf("%d task(s) would be updated, %d up to date\n", withChanges, upToDate)
	}
}

func printChanges(changes []notion.FieldChange) {
	for _, c := range changes {
		fmt.Printf("  %s: %q → %q\n", c.Field, c.OldValue, c.NewValue)
	}
}

func init() {
	pullCmd.Flags().BoolVar(&pullDryRun, "dry-run", false, "Show what would change without applying")
	rootCmd.AddCommand(pullCmd)
}
