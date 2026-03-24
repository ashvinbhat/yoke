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
	`

	_, err := s.db.Exec(schema)
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

	_, err = s.db.Exec(`
		INSERT INTO tasks (
			id, seq, notion_id, notion_url, external_ref,
			title, body, status, priority, tags, parent_id, blockers,
			synced_at, local_only, created_at, updated_at, started_at, done_at, outcome
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.Seq, t.NotionID, t.NotionURL, t.ExternalRef,
		t.Title, t.Body, t.Status, t.Priority, string(tags), t.Parent, string(blockers),
		t.SyncedAt, t.LocalOnly, t.CreatedAt, t.UpdatedAt, t.StartedAt, t.DoneAt, t.Outcome,
	)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	// Log creation event
	s.logEvent(t.ID, "created", "", t.Title)

	return nil
}

// Update saves changes to an existing task.
func (s *Store) Update(t *Task) error {
	t.UpdatedAt = time.Now()
	tags, _ := json.Marshal(t.Tags)
	blockers, _ := json.Marshal(t.Blockers)

	_, err := s.db.Exec(`
		UPDATE tasks SET
			notion_id = ?, notion_url = ?, external_ref = ?,
			title = ?, body = ?, status = ?, priority = ?,
			tags = ?, parent_id = ?, blockers = ?,
			synced_at = ?, local_only = ?, updated_at = ?,
			started_at = ?, done_at = ?, outcome = ?
		WHERE id = ?
	`,
		t.NotionID, t.NotionURL, t.ExternalRef,
		t.Title, t.Body, t.Status, t.Priority,
		string(tags), t.Parent, string(blockers),
		t.SyncedAt, t.LocalOnly, t.UpdatedAt,
		t.StartedAt, t.DoneAt, t.Outcome,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// taskColumns is the list of columns for task queries.
const taskColumns = `id, seq, notion_id, notion_url, external_ref,
	title, body, status, priority, tags, parent_id, blockers,
	synced_at, local_only, created_at, updated_at, started_at, done_at, outcome`

// Get retrieves a task by ID or sequence number.
func (s *Store) Get(idOrSeq string) (*Task, error) {
	var row *sql.Row

	// Try as sequence number first (if it's purely numeric)
	isNumeric := true
	for _, c := range idOrSeq {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}

	if isNumeric && len(idOrSeq) > 0 {
		var seq int
		fmt.Sscanf(idOrSeq, "%d", &seq)
		row = s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE seq = ?", seq)
	} else {
		// Try as ID (exact match or prefix)
		row = s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE id = ? OR id LIKE ?", idOrSeq, idOrSeq+"%")
	}

	return s.scanTask(row)
}

// GetByID retrieves a task by exact ID.
func (s *Store) GetByID(id string) (*Task, error) {
	row := s.db.QueryRow("SELECT "+taskColumns+" FROM tasks WHERE id = ?", id)
	return s.scanTask(row)
}

// scanTask scans a row into a Task struct.
func (s *Store) scanTask(row *sql.Row) (*Task, error) {
	var t Task
	var tags, blockers string
	var notionID, notionURL, extRef, parent, outcome sql.NullString
	var syncedAt, startedAt, doneAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.Seq, &notionID, &notionURL, &extRef,
		&t.Title, &t.Body, &t.Status, &t.Priority, &tags, &parent, &blockers,
		&syncedAt, &t.LocalOnly, &t.CreatedAt, &t.UpdatedAt, &startedAt, &doneAt, &outcome,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan task: %w", err)
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
	Status   *Status
	Priority *int
	Tag      *string
	ParentID *string
	Open     bool // Only open (non-complete) tasks
	Blocked  bool // Only blocked tasks
	Ready    bool // Only unblocked, active tasks
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

	var tasks []*Task
	for rows.Next() {
		var t Task
		var tags, blockers string
		var notionID, notionURL, extRef, parent, outcome sql.NullString
		var syncedAt, startedAt, doneAt sql.NullTime

		err := rows.Scan(
			&t.ID, &t.Seq, &notionID, &notionURL, &extRef,
			&t.Title, &t.Body, &t.Status, &t.Priority, &tags, &parent, &blockers,
			&syncedAt, &t.LocalOnly, &t.CreatedAt, &t.UpdatedAt, &startedAt, &doneAt, &outcome,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
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

		json.Unmarshal([]byte(tags), &t.Tags)
		json.Unmarshal([]byte(blockers), &t.Blockers)

		if t.Tags == nil {
			t.Tags = []string{}
		}
		if t.Blockers == nil {
			t.Blockers = []string{}
		}

		tasks = append(tasks, &t)
	}

	return tasks, nil
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

	s.logEvent(id, "deleted", "", "")
	return nil
}

// UpdateStatus changes a task's status and logs the event.
func (s *Store) UpdateStatus(id string, oldStatus, newStatus Status) error {
	_, err := s.db.Exec("UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?",
		newStatus, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	s.logEvent(id, "status_changed", string(oldStatus), string(newStatus))
	return nil
}

// logEvent records an event in the events table.
func (s *Store) logEvent(taskID, eventType, oldValue, newValue string) {
	s.db.Exec(`
		INSERT INTO events (task_id, event_type, old_value, new_value)
		VALUES (?, ?, ?, ?)
	`, taskID, eventType, oldValue, newValue)
}

// Search finds tasks matching the query string.
func (s *Store) Search(query string) ([]*Task, error) {
	// TODO: Implement proper FTS search
	// For now, just return all open tasks
	_ = "%" + strings.ToLower(query) + "%" // placeholder for future search
	return s.List(ListOptions{Open: true})
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

	s.logEvent(taskID, "note_added", "", content)
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
