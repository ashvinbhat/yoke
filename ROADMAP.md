# Yoke Roadmap

> Local-first task management system

## Vision

Replace fragmented task tracking (Notion + mental + notes) with a single local-first system that syncs with Notion and provides the task foundation for AI-assisted work.

## Data Model

```go
type Task struct {
    // Identity
    ID          string     // Local short ID (e.g., "a3f8")
    Seq         int        // Human-friendly number (#42)

    // External link
    NotionID    *string    // Notion page ID (for sync)
    NotionURL   *string    // Direct link to Notion
    ExternalRef *string    // Any other external reference

    // Content
    Title       string
    Body        string     // Markdown

    // State
    Status      Status     // pending, active, in_progress, blocked, done, dropped
    Priority    int        // 1 (urgent) to 5 (someday)

    // Organization
    Tags        []string
    Parent      *string    // Hierarchy
    Blockers    []string   // Dependencies

    // Sync metadata
    SyncedAt    *time.Time // Last Notion sync
    LocalOnly   bool       // Not in Notion

    // Tracking
    CreatedAt   time.Time
    UpdatedAt   time.Time
    StartedAt   *time.Time
    DoneAt      *time.Time
    Outcome     *string    // What happened (for learning)
}

type Status string
const (
    StatusPending    Status = "pending"     // Not started
    StatusActive     Status = "active"      // Ready to work
    StatusInProgress Status = "in_progress" // Currently working
    StatusBlocked    Status = "blocked"     // Waiting on something
    StatusDone       Status = "done"        // Completed
    StatusDropped    Status = "dropped"     // Won't do
)
```

## Task Lifecycle

```
                    ┌─────────────────────────────────────────┐
                    │                                         │
                    ▼                                         │
┌─────────┐    ┌─────────┐    ┌─────────────┐    ┌──────┐    │
│ PENDING │───▶│ ACTIVE  │───▶│ IN_PROGRESS │───▶│ DONE │    │
└─────────┘    └─────────┘    └─────────────┘    └──────┘    │
     │              │               │                        │
     │              │               ▼                        │
     │              │         ┌─────────┐                    │
     │              └────────▶│ BLOCKED │────────────────────┘
     │                        └─────────┘
     │
     └───────────────────────▶┌─────────┐
                              │ DROPPED │
                              └─────────┘
```

---

## Phase Y0: Core CRUD ✅ DONE

**Goal:** Basic task management works

### Deliverables
- [x] `yoke init` - Initialize ~/.yoke directory and database
- [x] `yoke add "title"` - Create a task
- [x] `yoke list` - List all tasks
- [x] `yoke show <id>` - Show task details
- [x] `yoke start <id>` - Mark task as in_progress
- [x] `yoke done <id>` - Mark task as done
- [x] `yoke drop <id>` - Mark task as dropped
- [x] `yoke edit <id>` - Edit task

### Structure
```
yoke/
├── cmd/yoke/
│   └── main.go
├── internal/
│   ├── task/
│   │   ├── task.go        # Task model
│   │   ├── store.go       # SQLite operations
│   │   └── store_test.go
│   ├── config/
│   │   └── config.go
│   └── cli/
│       ├── root.go
│       ├── init.go
│       ├── add.go
│       ├── list.go
│       ├── show.go
│       ├── start.go
│       ├── done.go
│       ├── drop.go
│       └── edit.go
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── ROADMAP.md
```

### Exit Criteria
- Can create, list, view, and complete tasks
- Data persists in SQLite
- Use yoke daily for real work

---

## Phase Y1: Notes & Events ✅ DONE

**Goal:** Track activity and add context

### Deliverables
- [x] `yoke note <id> "text"` - Add note to task
- [x] `yoke notes <id>` - Show all notes for task
- [x] Event logging (all mutations recorded)
- [x] `yoke log <id>` - Show task history
- [x] `yoke tag/untag/tags` - Tag management
- [x] `yoke edit` - Edit task properties with logging

### Schema Addition
```sql
CREATE TABLE notes (
    id INTEGER PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id),
    content TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    task_id TEXT,
    event_type TEXT,  -- created, status_changed, note_added, etc.
    old_value TEXT,
    new_value TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Exit Criteria
- Can add notes to tasks
- All changes are logged
- Can see full task history

---

## Phase Y2: Hierarchy ✅ DONE

**Goal:** Break down complex tasks

### Deliverables
- [x] `yoke add "title" --parent <id>` - Create subtask
- [x] `yoke subtask <parent> "title"` - Shorthand for above
- [x] `yoke tree` - Show task hierarchy with icons (○ pending, ◐ active, ● in_progress, ✓ done, ✗ dropped)
- [x] `yoke list --tree` - List with tree view
- [ ] Auto-complete parent when all children done (optional - deferred)

### Exit Criteria
- Can break down tasks into subtasks
- Tree view shows hierarchy
- Parent-child relationship works

---

## Phase Y3: Dependencies ✅ DONE

**Goal:** Track blockers and find ready work

### Deliverables
- [x] `yoke block <id> --by <blocker>` - Add dependency
- [x] `yoke unblock <id> <blocker>` - Remove dependency
- [x] `yoke ready` - Show tasks with no blockers
- [x] `yoke list --blocked` - Show blocked tasks
- [x] Cycle detection in dependency graph

### Exit Criteria
- Can track what blocks what
- `yoke ready` shows actionable tasks
- No circular dependencies allowed

---

## Phase Y4: Tags & Filtering ✅ DONE

**Goal:** Organize and find tasks

### Deliverables
- [x] `yoke add "title" --tag backend --tag urgent`
- [x] `yoke tag <id> <tag>` - Add tag
- [x] `yoke untag <id> <tag>` - Remove tag
- [x] `yoke list --tag backend`
- [x] `yoke list --status active`
- [x] `yoke list --priority 1`
- [x] `yoke search "keyword"` - Full-text search

### Exit Criteria
- Can tag and filter tasks
- Search finds tasks by content

---

## Phase Y5: Notion Sync

**Goal:** Bidirectional sync with Notion

### Deliverables
- [ ] `yoke link <id> <notion-url>` - Link local task to Notion
- [ ] `yoke import <notion-url>` - Import task from Notion
- [ ] `yoke pull` - Pull updates from Notion
- [ ] `yoke push` - Push updates to Notion
- [ ] `yoke sync` - Bidirectional sync
- [ ] Conflict resolution strategy
- [ ] `yoke config notion.token <token>` - Configure Notion

### Sync Strategy
```
Notion Task                    Yoke Task
───────────                    ─────────
Title          ←──────────────▶ Title
Status         ←──────────────▶ Status (mapped)
Properties     ←──────────────▶ Tags
Page content   ←──────────────▶ Body
                               Local notes (not synced)
```

### Exit Criteria
- Tasks sync between Notion and yoke
- Changes in either place reflect in other
- No data loss on sync

---

## Phase Y6: Export & Backup

**Goal:** Data safety and portability

### Deliverables
- [ ] `yoke export` - Export all tasks as JSON/YAML
- [ ] `yoke import --file backup.json` - Import from file
- [ ] Auto-backup before destructive operations
- [ ] `yoke backup` - Manual backup
- [ ] `yoke restore <backup>` - Restore from backup

### Exit Criteria
- Can export/import all data
- Automatic backups work
- Can recover from any state

---

## Phase Y7: Polish & Performance

**Goal:** Production-ready CLI

### Deliverables
- [ ] Shell completions (bash, zsh, fish)
- [ ] Colored output
- [ ] `yoke --json` for all commands (machine-readable)
- [ ] Performance optimization for large task counts
- [ ] `yoke stats` - Task statistics
- [ ] `yoke cleanup` - Archive old completed tasks

### Exit Criteria
- Pleasant CLI experience
- Fast even with 1000+ tasks
- Can script with JSON output

---

## Future Ideas (Not Planned)

- Web UI / TUI dashboard
- Team sync (multiple users)
- Time tracking
- Recurring tasks
- Calendar integration
- Mobile app

---

## Technical Decisions

### Why SQLite?
- Single file, no server
- Full SQL for complex queries
- Transactions for data safety
- Pure Go driver (modernc.org/sqlite) - no CGO

### Why Go?
- Single binary distribution
- Fast compilation
- Excellent CLI tooling (cobra)
- Can be imported as a library

### ID Generation
- Short hash from content + timestamp
- Human-readable sequence number (#1, #2, ...)
- Both work as identifiers

---

## Status Key

- ✅ CURRENT - Active phase
- 🔜 NEXT - Coming up
- 📋 PLANNED - On roadmap
- 💡 IDEA - Not committed
