package task

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store handles task persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a new task store.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates the database schema.
func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		seq INTEGER UNIQUE,
		notion_id TEXT,
		notion_url TEXT,
		external_ref TEXT,
		title TEXT NOT NULL,
		body TEXT DEFAULT '',
		status TEXT DEFAULT 'pending',
		priority INTEGER DEFAULT 3,
		tags TEXT DEFAULT '[]',
		parent_id TEXT,
		blockers TEXT DEFAULT '[]',
		synced_at DATETIME,
		local_only INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		done_at DATETIME,
		outcome TEXT
	);

	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL REFERENCES tasks(id),
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		event_type TEXT NOT NULL,
		old_value TEXT,
		new_value TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	-- Initialize sequence counter if not exists
	INSERT OR IGNORE INTO meta (key, value) VALUES ('next_seq', '1');

	CREATE TABLE IF NOT EXISTS task_keys (
		key TEXT PRIMARY KEY COLLATE NOCASE,
		task_id TEXT NOT NULL REFERENCES tasks(id)
	);
	CREATE INDEX IF NOT EXISTS task_keys_task ON task_keys(task_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return s.addColumn("tasks", "found_from", "TEXT")
}

// addColumn adds a column that databases made before it don't have.
func (s *Store) addColumn(table, column, decl string) error {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl))
	return err
}

// nextSeq gets and increments the sequence number.
func (s *Store) nextSeq() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var seq int
	err = tx.QueryRow("SELECT value FROM meta WHERE key = 'next_seq'").Scan(&seq)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec("UPDATE meta SET value = ? WHERE key = 'next_seq'", seq+1)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return seq, nil
}

// Create adds a new task to the store.
func (s *Store) Create(t *Task) error {
	seq, err := s.nextSeq()
	if err != nil {
		return fmt.Errorf("failed to get sequence: %w", err)
	}
	t.Seq = seq

	tags, _ := json.Marshal(t.Tags)
	blockers, _ := json.Marshal(t.Blockers)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		INSERT INTO tasks (
			id, seq, notion_id, notion_url, external_ref,
			title, body, status, priority, tags, parent_id, blockers,
			synced_at, local_only, created_at, updated_at, started_at, done_at, outcome, found_from
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.Seq, t.NotionID, t.NotionURL, t.ExternalRef,
		t.Title, t.Body, t.Status, t.Priority, string(tags), t.Parent, string(blockers),
		t.SyncedAt, t.LocalOnly, t.CreatedAt, t.UpdatedAt, t.StartedAt, t.DoneAt, t.Outcome, t.FoundFrom,
	)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}
	if err := insertKeys(tx, t.ID, t.Keys); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// Log creation event
	s.LogEvent(t.ID, "created", "", t.Title)

	return nil
}

// Update saves changes to an existing task.
func (s *Store) Update(t *Task) error {
	t.UpdatedAt = time.Now()
	tags, _ := json.Marshal(t.Tags)
	blockers, _ := json.Marshal(t.Blockers)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		UPDATE tasks SET
			notion_id = ?, notion_url = ?, external_ref = ?,
			title = ?, body = ?, status = ?, priority = ?,
			tags = ?, parent_id = ?, blockers = ?,
			synced_at = ?, local_only = ?, updated_at = ?,
			started_at = ?, done_at = ?, outcome = ?, found_from = ?
		WHERE id = ?
	`,
		t.NotionID, t.NotionURL, t.ExternalRef,
		t.Title, t.Body, t.Status, t.Priority,
		string(tags), t.Parent, string(blockers),
		t.SyncedAt, t.LocalOnly, t.UpdatedAt,
		t.StartedAt, t.DoneAt, t.Outcome, t.FoundFrom,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	if err := insertKeys(tx, t.ID, t.Keys); err != nil {
		return err
	}
	return tx.Commit()
}

// insertKeys stores the keys taskID doesn't have yet. A key another task
// holds is an error: a key finds one task.
func insertKeys(tx *sql.Tx, taskID string, keys []string) error {
	for _, k := range keys {
		if !ValidKey(k) {
			return fmt.Errorf("%q isn't a key: a key has no spaces and has a ':' or '-' (CB-123, pr:backend-42)", k)
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO task_keys (key, task_id) VALUES (?, ?)", k, taskID); err != nil {
			return fmt.Errorf("failed to add key %s: %w", k, err)
		}
		var owner string
		if err := tx.QueryRow("SELECT task_id FROM task_keys WHERE key = ?", k).Scan(&owner); err != nil {
			return fmt.Errorf("failed to read key %s: %w", k, err)
		}
		if owner != taskID {
			return fmt.Errorf("key %s already finds task %s", k, owner)
		}
	}
	return nil
}

// loadKeys fills in each task's keys, in the order they were added.
func (s *Store) loadKeys(tasks []*Task) error {
	byID := make(map[string]*Task, len(tasks))
	for _, t := range tasks {
		t.Keys = []string{}
		byID[t.ID] = t
	}
	if len(tasks) == 0 {
		return nil
	}
	query := "SELECT task_id, key FROM task_keys ORDER BY rowid"
	var args []any
	if len(tasks) == 1 {
		query = "SELECT task_id, key FROM task_keys WHERE task_id = ? ORDER BY rowid"
		args = append(args, tasks[0].ID)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return fmt.Errorf("failed to load keys: %w", err)
		}
		if t, ok := byID[id]; ok {
			t.Keys = append(t.Keys, key)
		}
	}
	return rows.Err()
}

// FindByKeys returns the task any of keys finds, or nil when none does.
// Keys that find two different tasks are an error, so the caller says which.
func (s *Store) FindByKeys(keys []string) (*Task, error) {
	found := ""
	for _, k := range keys {
		var id string
		err := s.db.QueryRow("SELECT task_id FROM task_keys WHERE key = ?", k).Scan(&id)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find key %s: %w", k, err)
		}
		if found != "" && found != id {
			return nil, fmt.Errorf("keys %s find two tasks, %s and %s", strings.Join(keys, ", "), found, id)
		}
		found = id
	}
	if found == "" {
		return nil, nil
	}
	return s.GetByID(found)
}

// Added says what Add did.
type Added string

const (
	AddedNew      Added = "created"
	AddedFound    Added = "found"
	AddedReopened Added = "reopened"
)

// Add creates t, unless one of its keys finds a task already. That task
// then takes t's other keys and tags, is reopened if it was done or
// dropped, and gets t's body as a note; its title, parent and found-from
// stay. A task's history stays in one place however often it's added. A
// task linked to Notion isn't reopened: its status is Notion's.
func (s *Store) Add(t *Task) (*Task, Added, error) {
	existing, err := s.FindByKeys(t.Keys)
	if err != nil {
		return nil, "", err
	}
	if existing == nil {
		if err := s.Create(t); err != nil {
			return nil, "", err
		}
		return t, AddedNew, nil
	}

	added := AddedFound
	was := existing.Status
	var earlier *string
	if existing.IsComplete() && !existing.Linked() {
		added = AddedReopened
		earlier = existing.Outcome
		existing.Reopen()
	}
	for _, k := range t.Keys {
		existing.AddKey(k)
	}
	for _, tag := range t.Tags {
		existing.AddTag(tag)
	}
	if err := s.Update(existing); err != nil {
		return nil, "", err
	}
	if added == AddedReopened {
		s.LogEvent(existing.ID, "reopened", string(was), string(existing.Status))
		if earlier != nil {
			s.AddNote(existing.ID, "Outcome before it was reopened: "+*earlier)
		}
	}
	if body := strings.TrimSpace(t.Body); body != "" {
		s.AddNote(existing.ID, body)
	}
	return existing, added, nil
}

// taskColumns is the list of columns for task queries.
const taskColumns = `id, seq, notion_id, notion_url, external_ref,
	title, body, status, priority, tags, parent_id, blockers,
	synced_at, local_only, created_at, updated_at, started_at, done_at, outcome, found_from`

// Get retrieves a task by ID, sequence number or key.
func (s *Store) Get(idOrSeq string) (*Task, error) {
	// Digits resolve as a sequence number first, but fall back to the id
	// lookup on a miss — 4-char hex ids are occasionally all digits.
	isNumeric := len(idOrSeq) > 0
	for _, c := range idOrSeq {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}

	if ValidKey(idOrSeq) {
		t, err := s.FindByKeys([]string{idOrSeq})
		if err != nil {
			return nil, err
		}
		if t == nil {
			return nil, fmt.Errorf("task not found")
		}
		return t, nil
	}

	if isNumeric {
		var seq int
		fmt.Sscanf(idOrSeq, "%d", &seq)
		if t, err := s.scanTask(s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE seq = ?", seq)); err == nil {
			return t, nil
		}
	}

	// ID (exact match or prefix)
	row := s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE id = ? OR id LIKE ?", idOrSeq, idOrSeq+"%")
	return s.scanTask(row)
}

// GetByID retrieves a task by exact ID.
func (s *Store) GetByID(id string) (*Task, error) {
	row := s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE id = ?", id)
	return s.scanTask(row)
}

// scanTask scans a row into a Task struct.
func (s *Store) scanTask(row *sql.Row) (*Task, error) {
	t, err := scanOne(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}
	if err := s.loadKeys([]*Task{t}); err != nil {
		return nil, err
	}
	return t, nil
}

// scanTasks scans every row, then loads the tasks' keys.
func (s *Store) scanTasks(rows *sql.Rows) ([]*Task, error) {
	var tasks []*Task
	for rows.Next() {
		t, err := scanOne(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read tasks: %w", err)
	}
	if err := s.loadKeys(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOne(row rowScanner) (*Task, error) {
	var t Task
	var tags, blockers string
	var notionID, notionURL, extRef, parent, outcome, foundFrom sql.NullString
	var syncedAt, startedAt, doneAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.Seq, &notionID, &notionURL, &extRef,
		&t.Title, &t.Body, &t.Status, &t.Priority, &tags, &parent, &blockers,
		&syncedAt, &t.LocalOnly, &t.CreatedAt, &t.UpdatedAt, &startedAt, &doneAt, &outcome, &foundFrom,
	)
	if err != nil {
		return nil, err
	}

	if foundFrom.Valid {
		t.FoundFrom = &foundFrom.String
	}
	// Handle nullable fields
	if notionID.Valid {
		t.NotionID = &notionID.String
	}
	if notionURL.Valid {
		t.NotionURL = &notionURL.String
	}
	if extRef.Valid {
		t.ExternalRef = &extRef.String
	}
	if parent.Valid {
		t.Parent = &parent.String
	}
	if outcome.Valid {
		t.Outcome = &outcome.String
	}
	if syncedAt.Valid {
		t.SyncedAt = &syncedAt.Time
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if doneAt.Valid {
		t.DoneAt = &doneAt.Time
	}

	// Parse JSON arrays
	json.Unmarshal([]byte(tags), &t.Tags)
	json.Unmarshal([]byte(blockers), &t.Blockers)

	if t.Tags == nil {
		t.Tags = []string{}
	}
	if t.Blockers == nil {
		t.Blockers = []string{}
	}

	return &t, nil
}

// ListOptions specifies filters for listing tasks.
type ListOptions struct {
	Status    *Status
	Priority  *int
	Tag       *string
	ParentID  *string
	FoundFrom *string
	Open      bool // Only open (non-complete) tasks
	Blocked   bool // Only blocked tasks
	Ready     bool // Only unblocked, active tasks
}

// List retrieves tasks matching the given options.
func (s *Store) List(opts ListOptions) ([]*Task, error) {
	query := "SELECT " + taskColumns + " FROM tasks WHERE 1=1"
	args := []interface{}{}

	if opts.Status != nil {
		query += " AND status = ?"
		args = append(args, *opts.Status)
	}

	if opts.Priority != nil {
		query += " AND priority = ?"
		args = append(args, *opts.Priority)
	}

	if opts.Tag != nil {
		query += " AND tags LIKE ?"
		args = append(args, "%\""+*opts.Tag+"\"%")
	}

	if opts.ParentID != nil {
		query += " AND parent_id = ?"
		args = append(args, *opts.ParentID)
	}

	if opts.FoundFrom != nil {
		query += " AND found_from = ?"
		args = append(args, *opts.FoundFrom)
	}

	if opts.Open {
		query += " AND status NOT IN ('done', 'dropped')"
	}

	if opts.Blocked {
		query += " AND status = 'blocked'"
	}

	if opts.Ready {
		query += " AND status IN ('pending', 'active') AND blockers = '[]'"
	}

	query += " ORDER BY priority ASC, created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// Delete removes a task from the store.
func (s *Store) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	if _, err := s.db.Exec("DELETE FROM task_keys WHERE task_id = ?", id); err != nil {
		return fmt.Errorf("failed to delete the task's keys: %w", err)
	}

	s.LogEvent(id, "deleted", "", "")
	return nil
}

// UpdateStatus changes a task's status and logs the event.
func (s *Store) UpdateStatus(id string, oldStatus, newStatus Status) error {
	_, err := s.db.Exec("UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?",
		newStatus, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	s.LogEvent(id, "status_changed", string(oldStatus), string(newStatus))
	return nil
}

// LogEvent records an event in the events table.
func (s *Store) LogEvent(taskID, eventType, oldValue, newValue string) {
	s.db.Exec(`
		INSERT INTO events (task_id, event_type, old_value, new_value)
		VALUES (?, ?, ?, ?)
	`, taskID, eventType, oldValue, newValue)
}

// Search finds tasks matching the query string in title or body.
func (s *Store) Search(query string) ([]*Task, error) {
	searchPattern := "%" + strings.ToLower(query) + "%"

	rows, err := s.db.Query(`
		SELECT `+taskColumns+` FROM tasks
		WHERE LOWER(title) LIKE ? OR LOWER(body) LIKE ?
		ORDER BY
			CASE WHEN status IN ('done', 'dropped') THEN 1 ELSE 0 END,
			priority ASC,
			created_at DESC
	`, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// Note represents a note attached to a task.
type Note struct {
	ID        int
	TaskID    string
	Content   string
	CreatedAt time.Time
}

// AddNote adds a note to a task.
func (s *Store) AddNote(taskID, content string) error {
	_, err := s.db.Exec(`
		INSERT INTO notes (task_id, content) VALUES (?, ?)
	`, taskID, content)
	if err != nil {
		return fmt.Errorf("failed to add note: %w", err)
	}

	s.LogEvent(taskID, "note_added", "", content)
	return nil
}

// GetNotes retrieves all notes for a task.
func (s *Store) GetNotes(taskID string) ([]Note, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, content, created_at FROM notes
		WHERE task_id = ? ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.TaskID, &n.Content, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}

	return notes, nil
}

// Event represents a recorded event/mutation on a task.
type Event struct {
	ID        int
	TaskID    string
	EventType string
	OldValue  string
	NewValue  string
	CreatedAt time.Time
}

// GetEvents retrieves all events for a task.
func (s *Store) GetEvents(taskID string) ([]Event, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, event_type, old_value, new_value, created_at
		FROM events
		WHERE task_id = ? ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var oldVal, newVal sql.NullString
		if err := rows.Scan(&e.ID, &e.TaskID, &e.EventType, &oldVal, &newVal, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		if oldVal.Valid {
			e.OldValue = oldVal.String
		}
		if newVal.Valid {
			e.NewValue = newVal.String
		}
		events = append(events, e)
	}

	return events, nil
}

// GetRecentEvents retrieves recent events across all tasks.
func (s *Store) GetRecentEvents(limit int) ([]Event, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, event_type, old_value, new_value, created_at
		FROM events
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var oldVal, newVal sql.NullString
		if err := rows.Scan(&e.ID, &e.TaskID, &e.EventType, &oldVal, &newVal, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		if oldVal.Valid {
			e.OldValue = oldVal.String
		}
		if newVal.Valid {
			e.NewValue = newVal.String
		}
		events = append(events, e)
	}

	return events, nil
}

// GetByNotionID retrieves a task by its Notion page ID.
func (s *Store) GetByNotionID(notionID string) (*Task, error) {
	row := s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE notion_id = ?", notionID)
	return s.scanTask(row)
}

// ListLinked retrieves all tasks that have a NotionID (linked to Notion).
func (s *Store) ListLinked() ([]*Task, error) {
	query := "SELECT " + taskColumns + " FROM tasks WHERE notion_id IS NOT NULL AND notion_id != '' ORDER BY updated_at DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query linked tasks: %w", err)
	}
	defer rows.Close()

	return s.scanTasks(rows)
}

// UpdateSyncedAt updates the sync timestamp for a task.
func (s *Store) UpdateSyncedAt(id string, syncedAt time.Time) error {
	_, err := s.db.Exec("UPDATE tasks SET synced_at = ?, updated_at = ? WHERE id = ?", syncedAt, time.Now(), id)
	return err
}
