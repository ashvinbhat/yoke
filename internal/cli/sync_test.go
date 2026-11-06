package cli

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/ashvinbhat/yoke/task"
)

func syncStore(t *testing.T) *task.Store {
	t.Helper()
	s, err := task.NewStore(filepath.Join(t.TempDir(), "yoke.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var ticket = notion.Ticket{
	PageID:   "2a1f0c3e-0000-4000-8000-000000000001",
	URL:      "https://www.notion.so/2a1f0c3e000040008000000000000001",
	Key:      "CB-18304",
	Title:    "Video transcription",
	Status:   notion.NotionStatusInProgress,
	Priority: "P1",
	Tags:     []string{"Backend"},
}

func TestSyncCreatesTicketThenLeavesItAlone(t *testing.T) {
	s := syncStore(t)
	now := time.Now()
	r, err := syncTicket(s, ticket, func(string) string { return "the description" }, false, now)
	if err != nil || !r.created {
		t.Fatalf("first sync: %+v, %v; want created", r, err)
	}
	got, err := s.Get("CB-18304")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != task.StatusInProgress || got.Priority != 2 || got.Body != "the description" || got.LocalOnly {
		t.Fatalf("created %+v", got)
	}
	if strings.Join(got.Tags, ",") != "ticket,Backend" || got.NotionID == nil {
		t.Fatalf("tags %v, notion %v", got.Tags, got.NotionID)
	}

	r, err = syncTicket(s, ticket, nil, false, now)
	if err != nil || r.created || len(r.changes) > 0 {
		t.Fatalf("second sync: %+v, %v; want unchanged", r, err)
	}
}

func TestSyncFindsByKeyAndKeepsLocalFields(t *testing.T) {
	s := syncStore(t)
	work := task.New("my own title")
	work.AddKey("CB-18304")
	work.AddTag("durable")
	work.Priority = 1
	if err := s.Create(work); err != nil {
		t.Fatal(err)
	}

	noPriority := ticket
	noPriority.Priority = ""
	noPriority.Status = notion.NotionStatusDone
	r, err := syncTicket(s, noPriority, nil, false, time.Now())
	if err != nil || r.created {
		t.Fatalf("sync: %+v, %v; want the keyed task updated", r, err)
	}
	got, _ := s.GetByID(work.ID)
	if got.NotionID == nil || *got.NotionID != ticket.PageID {
		t.Fatal("the task the key found wasn't linked to the page")
	}
	if got.Title != "Video transcription" || got.Status != task.StatusDone || got.DoneAt == nil {
		t.Fatalf("Notion's fields didn't land: %+v", got)
	}
	if got.Priority != 1 || strings.Join(got.Tags, ",") != "durable,ticket,Backend" {
		t.Fatalf("local fields lost: priority %d tags %v", got.Priority, got.Tags)
	}
}

func TestSyncDryRunWritesNothing(t *testing.T) {
	s := syncStore(t)
	r, err := syncTicket(s, ticket, nil, true, time.Now())
	if err != nil || !r.created {
		t.Fatalf("dry run: %+v, %v", r, err)
	}
	if all, _ := s.List(task.ListOptions{}); len(all) != 0 {
		t.Fatalf("dry run stored %d tasks", len(all))
	}
}
