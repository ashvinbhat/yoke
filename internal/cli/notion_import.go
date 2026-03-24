package cli

import (
	"context"
	"fmt"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var importDryRun bool

var importCmd = &cobra.Command{
	Use:   "import <notion-url>",
	Short: "Import a task from Notion",
	Long: `Creates a new local task from a Notion page.

Imports the task's title, status, priority, and tags from Notion.
The Notion page must be assigned to you (configured assignee) for safety.

Examples:
  yoke import https://notion.so/my-task-abc123...
  yoke import --dry-run https://notion.so/my-task-abc123...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		notionURL := args[0]

		// Check Notion config
		if cfg.Notion.Token == "" {
			return fmt.Errorf("Notion token not configured. Add it to ~/.yoke/config.yaml:\n  notion:\n    token: \"secret_xxx\"")
		}

		// Parse the Notion page ID from URL
		pageID, err := notion.ParsePageID(notionURL)
		if err != nil {
			return fmt.Errorf("invalid Notion URL: %w", err)
		}

		// Create Notion client
		client := notion.NewClient(cfg.Notion.Token, cfg.Notion.DatabaseID, cfg.Notion.AssigneeName)

		// Validate assignee (SAFETY CHECK)
		ctx := context.Background()
		if err := client.ValidateAssignee(ctx, pageID); err != nil {
			return fmt.Errorf("safety check failed: %w", err)
		}

		// Get the page
		page, err := client.GetPage(ctx, pageID)
		if err != nil {
			return fmt.Errorf("failed to access Notion page: %w", err)
		}

		// Extract data from Notion page
		title := notion.GetPageTitle(page)
		notionStatus := notion.GetPageStatus(page)
		yokeStatus := notionStatus.ToYokeStatus()
		priorityStr := notion.GetPagePriority(page)
		priority := notion.NotionPriorityToYoke(priorityStr)
		tags := notion.GetPageTags(page)
		pageURL := notion.GetPageURL(page)

		// Check if already imported
		existing, _ := store.GetByNotionID(pageID)
		if existing != nil {
			return fmt.Errorf("this Notion page is already linked to task #%d [%s]: %s",
				existing.Seq, existing.ID, existing.Title)
		}

		// Dry run - just show what would be created
		if importDryRun {
			fmt.Println("=== DRY RUN - No changes will be made ===\n")
			fmt.Println("Would create task:")
			fmt.Printf("  Title:    %s\n", title)
			fmt.Printf("  Status:   %s (from Notion: %s)\n", yokeStatus, notionStatus)
			fmt.Printf("  Priority: P%d (from Notion: %s)\n", priority, priorityStr)
			if len(tags) > 0 {
				fmt.Printf("  Tags:     %v\n", tags)
			}
			fmt.Printf("  Notion:   %s\n", pageURL)
			return nil
		}

		// Create the local task
		t := task.New(title)
		t.Status = yokeStatus
		t.Priority = priority
		t.NotionID = &pageID
		t.NotionURL = &pageURL
		t.LocalOnly = false
		for _, tag := range tags {
			t.AddTag(tag)
		}

		if err := store.Create(t); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		// Log the event
		store.LogEvent(t.ID, "imported_from_notion", "", pageID)

		fmt.Printf("Imported task #%d [%s] from Notion\n", t.Seq, t.ID)
		fmt.Printf("  Title:    %s\n", t.Title)
		fmt.Printf("  Status:   %s\n", formatStatus(t.Status))
		fmt.Printf("  Priority: P%d\n", t.Priority)
		if len(t.Tags) > 0 {
			fmt.Printf("  Tags:     %v\n", t.Tags)
		}
		fmt.Printf("  Notion:   %s\n", pageURL)

		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without creating")
	rootCmd.AddCommand(importCmd)
}
