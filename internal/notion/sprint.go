package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

// The EPD Tasks database's names for what sync reads.
const (
	SprintProperty       = "Sprint"        // relation from a task to its sprints
	SprintStatusProperty = "Sprint status" // status on a sprint page
	CurrentSprint        = "Current"
)

// maxPages bounds a paginated query: 100 pages of 100 results is far past
// any sprint.
const maxPages = 100

// Sprint is a page of the sprints database.
type Sprint struct {
	ID   string
	Name string
}

// Ticket is a task page as yoke keeps it.
type Ticket struct {
	PageID   string
	URL      string
	Key      string // the page's Task ID, like CB-18304
	Title    string
	Status   NotionStatus
	Priority string
	Tags     []string
}

// TicketFromPage reads a task page.
func TicketFromPage(page *notionapi.Page) Ticket {
	return Ticket{
		PageID:   string(page.ID),
		URL:      page.URL,
		Key:      TicketKey(page),
		Title:    GetPageTitle(page),
		Status:   GetPageStatus(page),
		Priority: GetPagePriority(page),
		Tags:     GetPageTags(page),
	}
}

// TicketKey is the page's prefixed unique id, like CB-18304, or "" when it
// has none.
func TicketKey(page *notionapi.Page) string {
	for _, prop := range page.Properties {
		if id, ok := prop.(*notionapi.UniqueIDProperty); ok && id.UniqueID.Prefix != nil {
			return id.UniqueID.String()
		}
	}
	return ""
}

// CurrentSprints returns the sprints whose Sprint status is Current, from
// the database the tasks database's Sprint relation points at.
func (c *Client) CurrentSprints(ctx context.Context) ([]Sprint, error) {
	db, err := c.api.Database.Get(ctx, c.databaseID)
	if err != nil {
		return nil, fmt.Errorf("read the tasks database: %w", err)
	}
	rel, ok := db.Properties[SprintProperty].(*notionapi.RelationPropertyConfig)
	if !ok {
		return nil, fmt.Errorf("the tasks database has no %q relation", SprintProperty)
	}
	pages, err := c.queryAll(ctx, rel.Relation.DatabaseID, notionapi.PropertyFilter{
		Property: SprintStatusProperty,
		Status:   &notionapi.StatusFilterCondition{Equals: CurrentSprint},
	})
	if err != nil {
		return nil, fmt.Errorf("find the current sprint: %w", err)
	}
	sprints := make([]Sprint, 0, len(pages))
	for i := range pages {
		sprints = append(sprints, Sprint{ID: string(pages[i].ID), Name: GetPageTitle(&pages[i])})
	}
	return sprints, nil
}

// SprintTickets returns the task pages in sprint assigned to the configured
// assignee.
func (c *Client) SprintTickets(ctx context.Context, sprintID string) ([]notionapi.Page, error) {
	pages, err := c.queryAll(ctx, c.databaseID, notionapi.PropertyFilter{
		Property: SprintProperty,
		Relation: &notionapi.RelationFilterCondition{Contains: sprintID},
	})
	if err != nil {
		return nil, fmt.Errorf("read the sprint's tasks: %w", err)
	}
	return c.FilterByAssignee(pages), nil
}

func (c *Client) queryAll(ctx context.Context, db notionapi.DatabaseID, filter notionapi.Filter) ([]notionapi.Page, error) {
	var pages []notionapi.Page
	req := &notionapi.DatabaseQueryRequest{Filter: filter, PageSize: 100}
	for i := 0; i < maxPages; i++ {
		resp, err := c.api.Database.Query(ctx, db, req)
		if err != nil {
			return nil, err
		}
		pages = append(pages, resp.Results...)
		if !resp.HasMore {
			return pages, nil
		}
		req.StartCursor = resp.NextCursor
	}
	return nil, fmt.Errorf("more than %d pages of results", maxPages)
}
