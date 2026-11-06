package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ashvinbhat/yoke/internal/notion"
	"github.com/ashvinbhat/yoke/task"
	"github.com/spf13/cobra"
)

// TicketTag marks a task that is a Notion ticket.
const TicketTag = "ticket"

var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Bring in your current-sprint tickets from Notion",
	Long: `Brings the Notion tickets assigned to you in the current sprint (the
sprint whose Sprint status is Current) into yoke, then refreshes every other
open task linked to a Notion page, so a ticket keeps its status after its
sprint ends.

A new ticket becomes a task tagged "ticket", keyed by its Task ID (CB-18304)
and linked to its page. A ticket yoke already has, found by its page or its
key, takes Notion's title, status and priority; its tags, keys, notes and
findings stay. sync only reads Notion.

Examples:
  yoke sync
  yoke sync --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Notion.Token == "" {
			return fmt.Errorf("Notion token not configured")
		}
		client := notion.NewClient(cfg.Notion.Token, cfg.Notion.DatabaseID, cfg.Notion.AssigneeName)
		ctx := context.Background()

		sprints, err := client.CurrentSprints(ctx)
		if err != nil {
			return err
		}
		var names []string
		var tickets []notion.Ticket
		inSprint := map[string]bool{}
		for _, sp := range sprints {
			names = append(names, sp.Name)
			pages, err := client.SprintTickets(ctx, sp.ID)
			if err != nil {
				return err
			}
			for i := range pages {
				tk := notion.TicketFromPage(&pages[i])
				inSprint[pageKey(tk.PageID)] = true
				tickets = append(tickets, tk)
			}
		}
		if len(names) == 0 {
			fmt.Println("No current sprint in Notion")
		} else {
			fmt.Printf("Sprint: %s\n", strings.Join(names, ", "))
		}

		linked, err := store.ListLinked()
		if err != nil {
			return err
		}
		skipped := 0
		for _, t := range linked {
			if t.IsComplete() || inSprint[pageKey(*t.NotionID)] {
				continue
			}
			page, err := client.GetPage(ctx, *t.NotionID)
			if err != nil {
				fmt.Printf("Warning: task #%d: %v\n", t.Seq, err)
				continue
			}
			if client.ValidatePageAssignee(page) != nil {
				skipped++
				continue
			}
			tickets = append(tickets, notion.TicketFromPage(page))
		}

		body := func(pageID string) string {
			text, err := client.GetPageContent(ctx, pageID)
			if err != nil {
				fmt.Printf("Warning: could not fetch page content: %v\n", err)
			}
			return text
		}
		created, updated, unchanged := 0, 0, 0
		for _, tk := range tickets {
			r, err := syncTicket(store, tk, body, syncDryRun, time.Now())
			if err != nil {
				fmt.Printf("Warning: %s %s: %v\n", tk.Key, tk.Title, err)
				continue
			}
			switch {
			case r.created && syncDryRun:
				created++
				fmt.Printf("Would create %s %s\n", tk.Key, r.task.Title)
			case r.created:
				created++
				fmt.Printf("Created #%d [%s] %s %s\n", r.task.Seq, r.task.ID, tk.Key, r.task.Title)
			case len(r.changes) > 0:
				updated++
				fmt.Printf("%s #%d [%s] %s %s: %s\n", pick(syncDryRun, "Would update", "Updated"), r.task.Seq, r.task.ID, tk.Key, r.task.Title, strings.Join(r.changes, "; "))
			default:
				unchanged++
			}
		}

		fmt.Printf("%d created, %d updated, %d unchanged", created, updated, unchanged)
		if skipped > 0 {
			fmt.Printf(", %d skipped (no longer assigned to you)", skipped)
		}
		fmt.Println()
		return nil
	},
}

func pick(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

func pageKey(id string) string {
	return strings.ReplaceAll(strings.ToLower(id), "-", "")
}

type syncResult struct {
	task    *task.Task
	created bool
	changes []string
}

// syncTicket brings one ticket into s: the task linked to its page, else the
// one its key finds, else a new one. body fetches a new ticket's
// description; Notion owns title, status and priority, yoke owns the rest.
func syncTicket(s *task.Store, tk notion.Ticket, body func(pageID string) string, dryRun bool, now time.Time) (syncResult, error) {
	t, _ := s.GetByNotionID(tk.PageID)
	if t == nil && tk.Key != "" {
		var err error
		if t, err = s.FindByKeys([]string{tk.Key}); err != nil {
			return syncResult{}, err
		}
	}
	status := tk.Status.ToYokeStatus()

	if t == nil {
		t = task.New(tk.Title)
		t.Priority = notion.NotionPriorityToYoke(tk.Priority)
		t.NotionID = &tk.PageID
		t.NotionURL = &tk.URL
		t.LocalOnly = false
		t.SyncedAt = &now
		t.AddTag(TicketTag)
		for _, tag := range tk.Tags {
			t.AddTag(tag)
		}
		if tk.Key != "" {
			t.AddKey(tk.Key)
		}
		setStatus(t, status, now)
		if dryRun {
			return syncResult{task: t, created: true}, nil
		}
		if body != nil {
			t.Body = body(tk.PageID)
		}
		if err := s.Create(t); err != nil {
			return syncResult{}, err
		}
		s.LogEvent(t.ID, "synced_from_notion", "", tk.PageID)
		return syncResult{task: t, created: true}, nil
	}

	var changes []string
	was := t.Status
	if t.Title != tk.Title {
		changes = append(changes, fmt.Sprintf("title %q → %q", t.Title, tk.Title))
		t.Title = tk.Title
	}
	if t.Status != status {
		changes = append(changes, fmt.Sprintf("status %s → %s", t.Status, status))
		setStatus(t, status, now)
	}
	if tk.Priority != "" {
		if p := notion.NotionPriorityToYoke(tk.Priority); p != t.Priority {
			changes = append(changes, fmt.Sprintf("priority P%d → P%d", t.Priority, p))
			t.Priority = p
		}
	}
	if t.NotionID == nil || *t.NotionID == "" {
		changes = append(changes, "linked to Notion")
		t.NotionID = &tk.PageID
		t.NotionURL = &tk.URL
		t.LocalOnly = false
	}
	for _, tag := range append([]string{TicketTag}, tk.Tags...) {
		if !hasFold(t.Tags, tag) {
			changes = append(changes, "tag "+tag)
			t.AddTag(tag)
		}
	}
	if tk.Key != "" && !hasFold(t.Keys, tk.Key) {
		changes = append(changes, "key "+tk.Key)
		t.AddKey(tk.Key)
	}
	if len(changes) == 0 || dryRun {
		return syncResult{task: t, changes: changes}, nil
	}

	t.SyncedAt = &now
	if err := s.Update(t); err != nil {
		return syncResult{}, err
	}
	if was != t.Status {
		s.LogEvent(t.ID, "status_changed", string(was), string(t.Status))
	}
	s.LogEvent(t.ID, "synced_from_notion", "", strings.Join(changes, "; "))
	return syncResult{task: t, changes: changes}, nil
}

// setStatus moves t to status the way the matching command would.
func setStatus(t *task.Task, status task.Status, now time.Time) {
	t.Status = status
	switch {
	case status == task.StatusDone && t.DoneAt == nil:
		t.DoneAt = &now
	case status != task.StatusDone:
		t.DoneAt = nil
	}
	if status == task.StatusInProgress && t.StartedAt == nil {
		t.StartedAt = &now
	}
}

func hasFold(list []string, s string) bool {
	for _, x := range list {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would change without writing")
	rootCmd.AddCommand(syncCmd)
}
