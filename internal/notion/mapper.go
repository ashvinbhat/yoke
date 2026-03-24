package notion

import (
	"time"

	"github.com/ashvinbhat/yoke/task"
	"github.com/jomei/notionapi"
)

// TaskUpdate represents changes to apply to a local task from Notion.
type TaskUpdate struct {
	NotionID    string
	Title       string
	Status      task.Status
	Priority    int
	Tags        []string
	LastEdited  time.Time
	HasChanges  bool
	Changes     []FieldChange
}

// FieldChange describes a single field change.
type FieldChange struct {
	Field    string
	OldValue string
	NewValue string
}

// PageToTaskUpdate converts a Notion page to a TaskUpdate.
func PageToTaskUpdate(page *notionapi.Page, existingTask *task.Task) *TaskUpdate {
	update := &TaskUpdate{
		NotionID:   string(page.ID),
		Title:      GetPageTitle(page),
		Status:     GetPageStatus(page).ToYokeStatus(),
		Priority:   NotionPriorityToYoke(GetPagePriority(page)),
		Tags:       GetPageTags(page),
		LastEdited: page.LastEditedTime,
	}

	// Compare with existing task to detect changes
	if existingTask != nil {
		if existingTask.Title != update.Title {
			update.Changes = append(update.Changes, FieldChange{
				Field:    "Title",
				OldValue: existingTask.Title,
				NewValue: update.Title,
			})
		}
		if existingTask.Status != update.Status {
			update.Changes = append(update.Changes, FieldChange{
				Field:    "Status",
				OldValue: string(existingTask.Status),
				NewValue: string(update.Status),
			})
		}
		if existingTask.Priority != update.Priority {
			update.Changes = append(update.Changes, FieldChange{
				Field:    "Priority",
				OldValue: priorityString(existingTask.Priority),
				NewValue: priorityString(update.Priority),
			})
		}
		// Check tags changes
		if !tagsEqual(existingTask.Tags, update.Tags) {
			update.Changes = append(update.Changes, FieldChange{
				Field:    "Tags",
				OldValue: tagsString(existingTask.Tags),
				NewValue: tagsString(update.Tags),
			})
		}

		update.HasChanges = len(update.Changes) > 0
	}

	return update
}

// ApplyToTask applies the update to a task.
func (u *TaskUpdate) ApplyToTask(t *task.Task) {
	t.Title = u.Title
	t.Status = u.Status
	t.Priority = u.Priority
	t.Tags = u.Tags
	now := time.Now()
	t.SyncedAt = &now
	t.UpdatedAt = now
}

// ShouldUpdate returns true if the Notion page is newer than local sync.
func ShouldUpdate(page *notionapi.Page, t *task.Task) bool {
	if t.SyncedAt == nil {
		return true // Never synced, should update
	}
	return page.LastEditedTime.After(*t.SyncedAt)
}

func priorityString(p int) string {
	return "P" + string(rune('0'+p))
}

func tagsString(tags []string) string {
	if len(tags) == 0 {
		return "(none)"
	}
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += ", "
		}
		result += t
	}
	return result
}

func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool)
	for _, t := range a {
		aMap[t] = true
	}
	for _, t := range b {
		if !aMap[t] {
			return false
		}
	}
	return true
}

// NotionUpdate represents changes to push to Notion.
type NotionUpdate struct {
	TaskID     string
	NotionID   string
	Title      string
	Status     NotionStatus
	Priority   string
	HasChanges bool
	Changes    []FieldChange
}

// TaskToNotionUpdate compares local task with Notion page and returns changes to push.
func TaskToNotionUpdate(t *task.Task, page *notionapi.Page) *NotionUpdate {
	update := &NotionUpdate{
		TaskID:   t.ID,
		NotionID: string(page.ID),
		Title:    t.Title,
		Status:   YokeToNotionStatus(t.Status),
		Priority: YokePriorityToNotion(t.Priority),
	}

	// Compare with Notion page to detect changes
	notionTitle := GetPageTitle(page)
	notionStatus := GetPageStatus(page)
	notionPriority := GetPagePriority(page)

	if t.Title != notionTitle {
		update.Changes = append(update.Changes, FieldChange{
			Field:    "Title",
			OldValue: notionTitle,
			NewValue: t.Title,
		})
	}

	if update.Status != notionStatus {
		update.Changes = append(update.Changes, FieldChange{
			Field:    "Status",
			OldValue: string(notionStatus),
			NewValue: string(update.Status),
		})
	}

	if update.Priority != notionPriority && notionPriority != "" {
		update.Changes = append(update.Changes, FieldChange{
			Field:    "Priority",
			OldValue: notionPriority,
			NewValue: update.Priority,
		})
	}

	update.HasChanges = len(update.Changes) > 0
	return update
}

// ShouldPush returns true if local task is newer than last sync.
func ShouldPush(t *task.Task) bool {
	if t.SyncedAt == nil {
		return true // Never synced
	}
	return t.UpdatedAt.After(*t.SyncedAt)
}
