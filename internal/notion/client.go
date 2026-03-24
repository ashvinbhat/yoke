package notion

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jomei/notionapi"
)

// Client wraps the Notion API client with safety features.
type Client struct {
	api        *notionapi.Client
	databaseID notionapi.DatabaseID
	assignee   string
}

// NewClient creates a new Notion client.
func NewClient(token, databaseID, assignee string) *Client {
	return &Client{
		api:        notionapi.NewClient(notionapi.Token(token)),
		databaseID: notionapi.DatabaseID(databaseID),
		assignee:   assignee,
	}
}

// ParsePageID extracts the page ID from a Notion URL.
// Supports formats:
//   - https://www.notion.so/workspace/Page-Title-abc123def456...
//   - https://notion.so/abc123def456...
//   - abc123def456... (raw ID)
func ParsePageID(urlOrID string) (string, error) {
	// If it's already a raw ID (32 hex chars, possibly with dashes)
	cleanID := strings.ReplaceAll(urlOrID, "-", "")
	if matched, _ := regexp.MatchString(`^[a-f0-9]{32}$`, cleanID); matched {
		return formatPageID(cleanID), nil
	}

	// Try to extract from URL
	// Pattern: last path segment before any query params, taking the last 32 chars
	re := regexp.MustCompile(`([a-f0-9]{32})(?:\?|$)`)
	matches := re.FindStringSubmatch(urlOrID)
	if len(matches) >= 2 {
		return formatPageID(matches[1]), nil
	}

	// Try another pattern: ID at end of slug (Page-Title-abc123...)
	re2 := regexp.MustCompile(`-([a-f0-9]{32})(?:\?|$)`)
	matches2 := re2.FindStringSubmatch(urlOrID)
	if len(matches2) >= 2 {
		return formatPageID(matches2[1]), nil
	}

	// Try to find any 32-char hex string
	re3 := regexp.MustCompile(`([a-f0-9]{32})`)
	matches3 := re3.FindStringSubmatch(strings.ToLower(urlOrID))
	if len(matches3) >= 2 {
		return formatPageID(matches3[1]), nil
	}

	return "", fmt.Errorf("could not parse Notion page ID from: %s", urlOrID)
}

// formatPageID formats a 32-char ID with dashes (Notion's standard format).
func formatPageID(id string) string {
	id = strings.ReplaceAll(id, "-", "")
	if len(id) != 32 {
		return id
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		id[0:8], id[8:12], id[12:16], id[16:20], id[20:32])
}

// GetPage retrieves a page by ID.
func (c *Client) GetPage(ctx context.Context, pageID string) (*notionapi.Page, error) {
	page, err := c.api.Page.Get(ctx, notionapi.PageID(pageID))
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}
	return page, nil
}

// GetPageTitle extracts the title from a Notion page.
func GetPageTitle(page *notionapi.Page) string {
	for _, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			var parts []string
			for _, rt := range titleProp.Title {
				parts = append(parts, rt.PlainText)
			}
			return strings.Join(parts, "")
		}
	}
	return ""
}

// GetPageStatus extracts the status from a Notion page.
func GetPageStatus(page *notionapi.Page) NotionStatus {
	if prop, ok := page.Properties["Status"]; ok {
		if statusProp, ok := prop.(*notionapi.StatusProperty); ok {
			if statusProp.Status.Name != "" {
				return NotionStatus(statusProp.Status.Name)
			}
		}
		// Also check for Select type (older databases)
		if selectProp, ok := prop.(*notionapi.SelectProperty); ok {
			if selectProp.Select.Name != "" {
				return NotionStatus(selectProp.Select.Name)
			}
		}
	}
	return NotionStatusTodo
}

// GetPagePriority extracts the priority from a Notion page.
func GetPagePriority(page *notionapi.Page) string {
	if prop, ok := page.Properties["Priority"]; ok {
		if selectProp, ok := prop.(*notionapi.SelectProperty); ok {
			if selectProp.Select.Name != "" {
				return selectProp.Select.Name
			}
		}
	}
	return ""
}

// GetPageTags extracts tags from a Notion page.
func GetPageTags(page *notionapi.Page) []string {
	var tags []string
	if prop, ok := page.Properties["Tags"]; ok {
		if multiSelect, ok := prop.(*notionapi.MultiSelectProperty); ok {
			for _, opt := range multiSelect.MultiSelect {
				tags = append(tags, opt.Name)
			}
		}
	}
	return tags
}

// GetPageAssignee extracts the assignee name(s) from a Notion page.
func GetPageAssignee(page *notionapi.Page) []string {
	var assignees []string
	if prop, ok := page.Properties["Assignee"]; ok {
		if peopleProp, ok := prop.(*notionapi.PeopleProperty); ok {
			for _, person := range peopleProp.People {
				assignees = append(assignees, person.Name)
			}
		}
	}
	return assignees
}

// GetPageURL returns the URL for a Notion page.
func GetPageURL(page *notionapi.Page) string {
	return page.URL
}

// QueryDatabase queries the Notion database with optional filters.
func (c *Client) QueryDatabase(ctx context.Context, filter *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	return c.api.Database.Query(ctx, c.databaseID, filter)
}

// DatabaseMetadata contains basic info about a Notion database.
type DatabaseMetadata struct {
	ID         string
	Title      string
	URL        string
	Properties []string
}

// GetDatabaseMetadata retrieves metadata about the configured database.
func (c *Client) GetDatabaseMetadata(ctx context.Context) (*DatabaseMetadata, error) {
	db, err := c.api.Database.Get(ctx, c.databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	// Extract title
	var title string
	for _, rt := range db.Title {
		title += rt.PlainText
	}

	// Extract property names
	var props []string
	for name := range db.Properties {
		props = append(props, name)
	}

	return &DatabaseMetadata{
		ID:         string(db.ID),
		Title:      title,
		URL:        db.URL,
		Properties: props,
	}, nil
}

// UpdatePageStatus updates the status of a Notion page.
func (c *Client) UpdatePageStatus(ctx context.Context, pageID string, status NotionStatus) error {
	_, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Properties: notionapi.Properties{
			"Status": notionapi.StatusProperty{
				Status: notionapi.Status{
					Name: string(status),
				},
			},
		},
	})
	return err
}

// UpdatePageTitle updates the title of a Notion page.
func (c *Client) UpdatePageTitle(ctx context.Context, pageID string, title string) error {
	_, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Properties: notionapi.Properties{
			"Task name": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: title}},
				},
			},
		},
	})
	return err
}

// UpdatePage updates multiple properties of a Notion page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, update *NotionUpdate) error {
	props := notionapi.Properties{}

	// Update status
	for _, change := range update.Changes {
		switch change.Field {
		case "Status":
			props["Status"] = notionapi.StatusProperty{
				Status: notionapi.Status{
					Name: string(update.Status),
				},
			}
		case "Title":
			props["Task name"] = notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: update.Title}},
				},
			}
		case "Priority":
			props["Priority"] = notionapi.SelectProperty{
				Select: notionapi.Option{
					Name: update.Priority,
				},
			}
		}
	}

	if len(props) == 0 {
		return nil // Nothing to update
	}

	_, err := c.api.Page.Update(ctx, notionapi.PageID(pageID), &notionapi.PageUpdateRequest{
		Properties: props,
	})
	return err
}
