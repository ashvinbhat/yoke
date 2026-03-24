package cli

import (
	"context"
	"fmt"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/jomei/notionapi"
	"github.com/spf13/cobra"
)

// getPageSelect extracts a select property value from a page.
func getPageSelect(page *notionapi.Page, propName string) string {
	if prop, ok := page.Properties[propName]; ok {
		if selectProp, ok := prop.(*notionapi.SelectProperty); ok {
			return selectProp.Select.Name
		}
	}
	return ""
}

var notionTestCmd = &cobra.Command{
	Use:   "notion-test",
	Short: "Test Notion connection and list your tasks",
	Long: `Tests the Notion API connection and lists tasks assigned to you.

This command verifies:
- Notion token is valid
- Database is accessible
- Assignee filter works

Examples:
  yoke notion-test`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check Notion config
		if cfg.Notion.Token == "" {
			return fmt.Errorf("Notion token not configured")
		}
		if cfg.Notion.DatabaseID == "" {
			return fmt.Errorf("Notion database_id not configured")
		}

		fmt.Printf("Testing Notion connection...\n")
		fmt.Printf("  Database ID: %s\n", cfg.Notion.DatabaseID)
		fmt.Printf("  Assignee:    %s\n", cfg.Notion.AssigneeName)

		// Create Notion client
		client := notion.NewClient(cfg.Notion.Token, cfg.Notion.DatabaseID, cfg.Notion.AssigneeName)
		ctx := context.Background()

		// Get database metadata
		dbMeta, err := client.GetDatabaseMetadata(ctx)
		if err != nil {
			return fmt.Errorf("failed to get database metadata: %w", err)
		}
		fmt.Printf("\nDatabase Metadata:\n")
		fmt.Printf("  Title: %s\n", dbMeta.Title)
		fmt.Printf("  URL:   %s\n", dbMeta.URL)
		fmt.Printf("  Properties: %v\n", dbMeta.Properties)

		// Query database (just first 100 for speed)
		req := &notionapi.DatabaseQueryRequest{
			PageSize: 100,
		}
		resp, err := client.QueryDatabase(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to query database: %w", err)
		}
		allPages := resp.Results

		fmt.Printf("\nFound %d pages (first page)\n", len(allPages))

		// Debug: Show Sprint property type and values
		fmt.Printf("\nSprint property details (first 5 tasks):\n")
		for i, page := range allPages {
			if i >= 5 {
				break
			}
			title := notion.GetPageTitle(&page)
			if prop, ok := page.Properties["Sprint"]; ok {
				fmt.Printf("  %d. %s\n", i+1, title)
				fmt.Printf("     Sprint type: %T\n", prop)
				fmt.Printf("     Sprint value: %+v\n", prop)
			}
		}

		// Debug: show unique assignees
		assigneeSet := make(map[string]int)
		for _, page := range allPages {
			assignees := notion.GetPageAssignee(&page)
			for _, a := range assignees {
				assigneeSet[a]++
			}
			if len(assignees) == 0 {
				assigneeSet["(no assignee)"]++
			}
		}
		fmt.Printf("\nAssignees in database:\n")
		for name, count := range assigneeSet {
			fmt.Printf("  - %q: %d tasks\n", name, count)
		}

		// Filter by assignee
		assigned := client.FilterByAssignee(allPages)
		fmt.Printf("\nAssigned to %q: %d\n", cfg.Notion.AssigneeName, len(assigned))

		if len(assigned) > 0 {
			fmt.Printf("\nYour tasks:\n")
			for i, page := range assigned {
				if i >= 10 {
					fmt.Printf("  ... and %d more\n", len(assigned)-10)
					break
				}
				title := notion.GetPageTitle(&page)
				status := notion.GetPageStatus(&page)
				fmt.Printf("  - [%s] %s\n", status, title)
				fmt.Printf("    ID: %s\n", page.ID)
			}
		}

		fmt.Println("\nNotion connection successful!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(notionTestCmd)
}
