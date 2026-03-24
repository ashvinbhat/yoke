package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <task-id> <notion-url>",
	Short: "Link a local task to a Notion page",
	Long: `Links an existing local task to a Notion page.

This stores the Notion page ID and URL in the local task for future sync operations.
No data is synced during linking - this just establishes the connection.

The Notion page must be assigned to you (configured assignee) for safety.

Examples:
  yoke link 1 https://notion.so/my-task-abc123...
  yoke link a3f8 https://www.notion.so/workspace/Task-Name-def456...`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		notionURL := args[1]

		// Get the local task
		t, err := store.Get(taskID)
		if err != nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		// Check if already linked
		if t.NotionID != nil && *t.NotionID != "" {
			return fmt.Errorf("task #%d is already linked to Notion (ID: %s)\nUse 'yoke show %d' to see details",
				t.Seq, *t.NotionID, t.Seq)
		}

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

		// Get the page to verify it exists and get the URL
		page, err := client.GetPage(ctx, pageID)
		if err != nil {
			return fmt.Errorf("failed to access Notion page: %w", err)
		}

		// Update local task with Notion info
		t.NotionID = &pageID
		notionURLClean := notion.GetPageURL(page)
		t.NotionURL = &notionURLClean
		t.LocalOnly = false
		now := time.Now()
		t.SyncedAt = &now

		if err := store.Update(t); err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		// Log the event
		store.LogEvent(t.ID, "linked_to_notion", "", pageID)

		// Get Notion page title for display
		notionTitle := notion.GetPageTitle(page)

		fmt.Printf("Linked task #%d [%s] to Notion\n", t.Seq, t.ID)
		fmt.Printf("  Local:  %s\n", t.Title)
		fmt.Printf("  Notion: %s\n", notionTitle)
		fmt.Printf("  URL:    %s\n", notionURLClean)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
