package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ashvinbhat/yoke/task"
)

// These DTOs are yoke's machine-readable output contract, consumed by any
// tool that shells out to `yoke ... --json` (ox, dashboards, scripts).
// Shapes are stable: additive changes only, never rename or repurpose a
// field. Status values are the raw enum strings (e.g. "in_progress"),
// never the human-formatted variants ("IN PROGRESS").

type taskJSON struct {
	ID          string     `json:"id"`
	Seq         int        `json:"seq"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	Tags        []string   `json:"tags"`
	ParentID    *string    `json:"parentId,omitempty"`
	BlockerIDs  []string   `json:"blockerIds,omitempty"`
	NotionURL   *string    `json:"notionUrl,omitempty"`
	ExternalRef *string    `json:"externalRef,omitempty"`
	LocalOnly   bool       `json:"localOnly"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	DoneAt      *time.Time `json:"doneAt,omitempty"`
	Outcome     *string    `json:"outcome,omitempty"`
}

// taskRefJSON is the minimal shape used when a task is embedded as a
// relation (parent / child / blocker) of another task.
type taskRefJSON struct {
	ID     string `json:"id"`
	Seq    int    `json:"seq"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type noteJSON struct {
	TaskID    string    `json:"taskId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type eventJSON struct {
	TaskID    string    `json:"taskId"`
	EventType string    `json:"eventType"`
	OldValue  string    `json:"oldValue,omitempty"`
	NewValue  string    `json:"newValue,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// taskDetailJSON is `yoke show --json`: the task plus resolved relations.
type taskDetailJSON struct {
	taskJSON
	Parent    *taskRefJSON  `json:"parent,omitempty"`
	Children  []taskRefJSON `json:"children"`
	BlockedBy []taskRefJSON `json:"blockedBy"`
}

// taskContextJSON is `yoke context --format json`: everything an agent or
// tool needs to understand a task in one call.
type taskContextJSON struct {
	taskDetailJSON
	Notes  []noteJSON  `json:"notes"`
	Events []eventJSON `json:"events"`
}

func toTaskJSON(t *task.Task) taskJSON {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	return taskJSON{
		ID:          t.ID,
		Seq:         t.Seq,
		Title:       t.Title,
		Body:        t.Body,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Tags:        tags,
		ParentID:    t.Parent,
		BlockerIDs:  t.Blockers,
		NotionURL:   t.NotionURL,
		ExternalRef: t.ExternalRef,
		LocalOnly:   t.LocalOnly,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		StartedAt:   t.StartedAt,
		DoneAt:      t.DoneAt,
		Outcome:     t.Outcome,
	}
}

func toTaskRefJSON(t *task.Task) taskRefJSON {
	return taskRefJSON{ID: t.ID, Seq: t.Seq, Title: t.Title, Status: string(t.Status)}
}

func toTaskListJSON(tasks []*task.Task) []taskJSON {
	out := make([]taskJSON, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskJSON(t))
	}
	return out
}

func toNotesJSON(notes []task.Note) []noteJSON {
	out := make([]noteJSON, 0, len(notes))
	for _, n := range notes {
		out = append(out, noteJSON{TaskID: n.TaskID, Content: n.Content, CreatedAt: n.CreatedAt})
	}
	return out
}

func toEventsJSON(events []task.Event) []eventJSON {
	out := make([]eventJSON, 0, len(events))
	for _, e := range events {
		out = append(out, eventJSON{
			TaskID:    e.TaskID,
			EventType: e.EventType,
			OldValue:  e.OldValue,
			NewValue:  e.NewValue,
			CreatedAt: e.CreatedAt,
		})
	}
	return out
}

// buildTaskDetail resolves parent, children, and blocker relations.
// Resolution failures on relations are non-fatal: a dangling parent or
// blocker ID degrades to omission rather than failing the whole command.
func buildTaskDetail(t *task.Task) taskDetailJSON {
	detail := taskDetailJSON{
		taskJSON:  toTaskJSON(t),
		Children:  []taskRefJSON{},
		BlockedBy: []taskRefJSON{},
	}

	if t.Parent != nil {
		if parent, err := store.GetByID(*t.Parent); err == nil && parent != nil {
			ref := toTaskRefJSON(parent)
			detail.Parent = &ref
		}
	}

	if children, err := store.List(task.ListOptions{ParentID: &t.ID}); err == nil {
		for _, c := range children {
			detail.Children = append(detail.Children, toTaskRefJSON(c))
		}
	}

	for _, bid := range t.Blockers {
		if blocker, err := store.GetByID(bid); err == nil && blocker != nil {
			detail.BlockedBy = append(detail.BlockedBy, toTaskRefJSON(blocker))
		}
	}

	return detail
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
