package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
)

// ValidateAssignee checks if a Notion page is assigned to the required user.
// Returns an error if the assignee doesn't match.
func (c *Client) ValidateAssignee(ctx context.Context, pageID string) error {
	page, err := c.GetPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("failed to fetch page for assignee validation: %w", err)
	}

	return c.ValidatePageAssignee(page)
}

// ValidatePageAssignee checks if a page is assigned to the required user.
func (c *Client) ValidatePageAssignee(page *notionapi.Page) error {
	assignees := GetPageAssignee(page)

	if len(assignees) == 0 {
		return fmt.Errorf("page has no assignee - refusing to interact (required: %s)", c.assignee)
	}

	for _, assignee := range assignees {
		if strings.EqualFold(assignee, c.assignee) {
			return nil // Found matching assignee
		}
	}

	return fmt.Errorf("page assigned to %v, not %s - refusing to interact",
		assignees, c.assignee)
}

// ValidateBeforeWrite performs all safety checks before any Notion write operation.
func (c *Client) ValidateBeforeWrite(ctx context.Context, pageID string) error {
	page, err := c.GetPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("page not found or inaccessible: %w", err)
	}

	// Check if page is archived
	if page.Archived {
		return fmt.Errorf("page is archived - refusing to modify")
	}

	// Validate assignee
	if err := c.ValidatePageAssignee(page); err != nil {
		return err
	}

	return nil
}

// AssigneeFilter returns a database filter for the required assignee.
// Note: Notion's People filter requires the user ID, not name.
// This is a helper that would need the user ID to be configured.
func (c *Client) AssigneeFilter() *notionapi.PropertyFilter {
	// For People properties, we need to filter by user ID
	// The assignee name alone won't work for API filtering
	// We'll filter client-side after fetching results
	return nil
}

// FilterByAssignee filters a list of pages to only those assigned to the required user.
func (c *Client) FilterByAssignee(pages []notionapi.Page) []notionapi.Page {
	var filtered []notionapi.Page
	for _, page := range pages {
		if err := c.ValidatePageAssignee(&page); err == nil {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

// SafetyReport generates a report of what would happen during a sync operation.
type SafetyReport struct {
	TotalPages     int
	AssigneeMatch  int
	AssigneeMiss   int
	Archived       int
	WouldProcess   []string // Page IDs that would be processed
	WouldSkip      []string // Page IDs that would be skipped
	SkipReasons    map[string]string
}

// GenerateSafetyReport checks all pages in the database and reports safety status.
func (c *Client) GenerateSafetyReport(ctx context.Context) (*SafetyReport, error) {
	report := &SafetyReport{
		SkipReasons: make(map[string]string),
	}

	// Query all pages in database
	resp, err := c.QueryDatabase(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	for _, page := range resp.Results {
		report.TotalPages++
		pageID := string(page.ID)

		if page.Archived {
			report.Archived++
			report.WouldSkip = append(report.WouldSkip, pageID)
			report.SkipReasons[pageID] = "archived"
			continue
		}

		if err := c.ValidatePageAssignee(&page); err != nil {
			report.AssigneeMiss++
			report.WouldSkip = append(report.WouldSkip, pageID)
			report.SkipReasons[pageID] = err.Error()
			continue
		}

		report.AssigneeMatch++
		report.WouldProcess = append(report.WouldProcess, pageID)
	}

	return report, nil
}

// PrintSafetyReport prints a safety report to stdout.
func (r *SafetyReport) Print() {
	fmt.Printf("Safety Report:\n")
	fmt.Printf("  Total pages in database: %d\n", r.TotalPages)
	fmt.Printf("  Assigned to you:         %d\n", r.AssigneeMatch)
	fmt.Printf("  Assigned to others:      %d\n", r.AssigneeMiss)
	fmt.Printf("  Archived:                %d\n", r.Archived)
	fmt.Printf("  Would process:           %d\n", len(r.WouldProcess))
	fmt.Printf("  Would skip:              %d\n", len(r.WouldSkip))
}
