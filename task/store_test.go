package task

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "yoke.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func create(t *testing.T, s *Store, title string, keys ...string) *Task {
	t.Helper()
	tk := New(title)
	for _, k := range keys {
		tk.AddKey(k)
	}
	if err := s.Create(tk); err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestValidKey(t *testing.T) {
	for k, want := range map[string]bool{
		"CB-18304":          true,
		"pr:backend-6952":   true,
		"thread:local_ab12": true,
		"a3f8":              false,
		"42":                false,
		"":                  false,
		"CB 18304":          false,
	} {
		if got := ValidKey(k); got != want {
			t.Errorf("ValidKey(%q) = %v, want %v", k, got, want)
		}
	}
}

func TestGetFindsByKeyIgnoringCase(t *testing.T) {
	s := newStore(t)
	made := create(t, s, "Video transcription", "CB-18304", "pr:backend-6952")

	got, err := s.Get("cb-18304")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != made.ID {
		t.Fatalf("Get by key found %s, want %s", got.ID, made.ID)
	}
	if strings.Join(got.Keys, ",") != "CB-18304,pr:backend-6952" {
		t.Fatalf("keys = %v, want them in the order added", got.Keys)
	}
	if _, err := s.Get("CB-1"); err == nil {
		t.Fatal("Get of an unknown key found a task")
	}
}

func TestKeyFindsOneTask(t *testing.T) {
	s := newStore(t)
	create(t, s, "first", "CB-1")
	second := create(t, s, "second", "CB-2")

	second.AddKey("cb-1")
	if err := s.Update(second); err == nil {
		t.Fatal("Update gave a second task a key another task holds")
	}
	if _, err := s.FindByKeys([]string{"CB-1", "CB-2"}); err == nil {
		t.Fatal("keys that find two tasks didn't fail")
	}
	got, err := s.Get("CB-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Keys) != 1 {
		t.Fatalf("the failed update stored keys %v", got.Keys)
	}
}

func TestAddCreatesThenFinds(t *testing.T) {
	s := newStore(t)
	first := New("Segment paging")
	first.AddKey("pr:backend-6952")
	got, added, err := s.Add(first)
	if err != nil || added != AddedNew {
		t.Fatalf("first Add = %v, %v; want created", added, err)
	}

	again := New("another title")
	again.AddKey("pr:backend-6952")
	again.AddKey("thread:local_b")
	again.AddTag("work")
	again.Body = "Ashvin: fix the paging"
	found, added, err := s.Add(again)
	if err != nil || added != AddedFound {
		t.Fatalf("second Add = %v, %v; want found", added, err)
	}
	if found.ID != got.ID || found.Title != "Segment paging" {
		t.Fatalf("found %s %q, want %s with its own title", found.ID, found.Title, got.ID)
	}
	stored, _ := s.GetByID(got.ID)
	if strings.Join(stored.Keys, ",") != "pr:backend-6952,thread:local_b" || strings.Join(stored.Tags, ",") != "work" {
		t.Fatalf("keys %v tags %v, want the new key and tag added", stored.Keys, stored.Tags)
	}
	notes, _ := s.GetNotes(got.ID)
	if len(notes) != 1 || notes[0].Content != "Ashvin: fix the paging" {
		t.Fatalf("notes = %v, want the body as a note", notes)
	}
}

func TestAddReopensDoneTask(t *testing.T) {
	s := newStore(t)
	done := create(t, s, "Video transcription", "CB-18304")
	done.Done("shipped in #6410")
	if err := s.Update(done); err != nil {
		t.Fatal(err)
	}

	again := New("Video transcription")
	again.AddKey("CB-18304")
	got, added, err := s.Add(again)
	if err != nil || added != AddedReopened {
		t.Fatalf("Add = %v, %v; want reopened", added, err)
	}
	if got.Status != StatusPending || got.DoneAt != nil || got.Outcome != nil {
		t.Fatalf("reopened task is %s, doneAt %v, outcome %v", got.Status, got.DoneAt, got.Outcome)
	}
	notes, _ := s.GetNotes(got.ID)
	if len(notes) != 1 || !strings.Contains(notes[0].Content, "shipped in #6410") {
		t.Fatalf("notes = %v, want the earlier outcome kept", notes)
	}
}

func TestFoundFrom(t *testing.T) {
	s := newStore(t)
	work := create(t, s, "work")
	finding := New("finding")
	finding.FoundFrom = &work.ID
	if err := s.Create(finding); err != nil {
		t.Fatal(err)
	}
	create(t, s, "unrelated")

	got, err := s.List(ListOptions{FoundFrom: &work.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != finding.ID || *got[0].FoundFrom != work.ID {
		t.Fatalf("found from work: %v", got)
	}
}

func TestOldDatabaseGainsTheNewColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "yoke.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE tasks (
		id TEXT PRIMARY KEY, seq INTEGER UNIQUE, notion_id TEXT, notion_url TEXT, external_ref TEXT,
		title TEXT NOT NULL, body TEXT DEFAULT '', status TEXT DEFAULT 'pending', priority INTEGER DEFAULT 3,
		tags TEXT DEFAULT '[]', parent_id TEXT, blockers TEXT DEFAULT '[]', synced_at DATETIME,
		local_only INTEGER DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, started_at DATETIME, done_at DATETIME, outcome TEXT);
		INSERT INTO tasks (id, seq, title) VALUES ('ab12', 1, 'from before');`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Get("1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "from before" || got.FoundFrom != nil || len(got.Keys) != 0 {
		t.Fatalf("old task read back as %+v", got)
	}
}

func TestAddFindsLinkedTaskWithoutReopening(t *testing.T) {
	s := newStore(t)
	ticket := New("Video transcription")
	ticket.AddKey("CB-18304")
	page := "2a1f0c3e-0000-4000-8000-000000000001"
	ticket.NotionID = &page
	ticket.Done("")
	if err := s.Create(ticket); err != nil {
		t.Fatal(err)
	}

	again := New("Video transcription")
	again.AddKey("CB-18304")
	got, added, err := s.Add(again)
	if err != nil || added != AddedFound || got.Status != StatusDone {
		t.Fatalf("Add = %v, %s, %v; want found and still done", added, got.Status, err)
	}
}
