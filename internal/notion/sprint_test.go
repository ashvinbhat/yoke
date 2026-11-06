package notion

import (
	"testing"

	"github.com/ashvinbhat/yoke/task"
	"github.com/jomei/notionapi"
)

func TestTicketFromPage(t *testing.T) {
	cb := "CB"
	page := &notionapi.Page{
		ID:  "2a1f0c3e-0000-4000-8000-000000000001",
		URL: "https://www.notion.so/x",
		Properties: notionapi.Properties{
			"Task ID":   &notionapi.UniqueIDProperty{UniqueID: notionapi.UniqueID{Prefix: &cb, Number: 18304}},
			"Task name": &notionapi.TitleProperty{Title: []notionapi.RichText{{PlainText: "Video transcription"}}},
			"Status":    &notionapi.StatusProperty{Status: notionapi.Status{Name: "Merged on QA - Pending Verification"}},
			"Priority":  &notionapi.SelectProperty{Select: notionapi.Option{Name: "P0"}},
		},
	}
	tk := TicketFromPage(page)
	if tk.Key != "CB-18304" || tk.Title != "Video transcription" || tk.Priority != "P0" {
		t.Fatalf("ticket = %+v", tk)
	}
	if tk.Status.ToYokeStatus() != task.StatusInProgress {
		t.Fatalf("%q maps to %s", tk.Status, tk.Status.ToYokeStatus())
	}
}

func TestEPDStatusesAndPriorities(t *testing.T) {
	for status, want := range map[NotionStatus]task.Status{
		"Todo":                                task.StatusPending,
		"In Review":                           task.StatusInProgress,
		"Merged on QA - Pending Verification": task.StatusInProgress,
		"Verified on QA":                      task.StatusInProgress,
		"RCA Pending":                         task.StatusInProgress,
		"Blocked":                             task.StatusBlocked,
		"Done":                                task.StatusDone,
		"Cancelled":                           task.StatusDropped,
	} {
		if got := status.ToYokeStatus(); got != want {
			t.Errorf("%q → %s, want %s", status, got, want)
		}
	}
	for priority, want := range map[string]int{"P0": 1, "High": 1, "P1": 2, "P2": 3, "P3": 4, "Low": 4} {
		if got := NotionPriorityToYoke(priority); got != want {
			t.Errorf("%q → %d, want %d", priority, got, want)
		}
	}
}
