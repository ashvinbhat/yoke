// Package task provides the core task model and operations for yoke.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Status represents the current state of a task.
type Status string

const (
	StatusPending    Status = "pending"     // Not started
	StatusActive     Status = "active"      // Ready to work
	StatusInProgress Status = "in_progress" // Currently working
	StatusBlocked    Status = "blocked"     // Waiting on something
	StatusDone       Status = "done"        // Completed
	StatusDropped    Status = "dropped"     // Won't do
)

// ValidStatuses returns all valid status values.
func ValidStatuses() []Status {
	return []Status{
		StatusPending,
		StatusActive,
		StatusInProgress,
		StatusBlocked,
		StatusDone,
		StatusDropped,
	}
}

// IsValid checks if a status is valid.
func (s Status) IsValid() bool {
	for _, valid := range ValidStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// Task represents a unit of work.
type Task struct {
	// Identity
	ID  string // Short hash (e.g., "a3f8")
	Seq int    // Human-friendly number (#42)

	// External link
	NotionID    *string // Notion page ID (for sync)
	NotionURL   *string // Direct link to Notion
	ExternalRef *string // Any other external reference

	// Content
	Title string // Task title
	Body  string // Markdown description

	// State
	Status   Status // Current status
	Priority int    // 1 (urgent) to 5 (someday)

	// Organization
	Tags     []string // For filtering and skill matching
	Parent   *string  // Parent task ID for hierarchy
	Blockers []string // Task IDs that block this

	// Sync metadata
	SyncedAt  *time.Time // Last Notion sync
	LocalOnly bool       // Not in Notion

	// Tracking
	CreatedAt time.Time  // When created
	UpdatedAt time.Time  // Last modified
	StartedAt *time.Time // When work started
	DoneAt    *time.Time // When completed
	Outcome   *string    // What happened (for learning)
}

// New creates a new task with the given title.
func New(title string) *Task {
	now := time.Now()
	return &Task{
		ID:        generateID(),
		Title:     title,
		Status:    StatusPending,
		Priority:  3, // Default middle priority
		Tags:      []string{},
		Blockers:  []string{},
		LocalOnly: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateID creates a short random ID.
func generateID() string {
	bytes := make([]byte, 4) // 4 bytes = 8 hex chars, we'll use first 4
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:4]
}

// Start marks the task as in progress.
func (t *Task) Start() {
	now := time.Now()
	t.Status = StatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now
}

// Done marks the task as completed.
func (t *Task) Done(outcome string) {
	now := time.Now()
	t.Status = StatusDone
	t.DoneAt = &now
	t.UpdatedAt = now
	if outcome != "" {
		t.Outcome = &outcome
	}
}

// Drop marks the task as dropped.
func (t *Task) Drop() {
	t.Status = StatusDropped
	t.UpdatedAt = time.Now()
}

// Block marks the task as blocked.
func (t *Task) Block() {
	t.Status = StatusBlocked
	t.UpdatedAt = time.Now()
}

// Activate marks the task as active (ready to work).
func (t *Task) Activate() {
	t.Status = StatusActive
	t.UpdatedAt = time.Now()
}

// AddTag adds a tag if not already present.
func (t *Task) AddTag(tag string) {
	for _, existing := range t.Tags {
		if existing == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// RemoveTag removes a tag if present.
func (t *Task) RemoveTag(tag string) {
	for i, existing := range t.Tags {
		if existing == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			t.UpdatedAt = time.Now()
			return
		}
	}
}

// AddBlocker adds a blocker task ID.
func (t *Task) AddBlocker(blockerID string) {
	for _, existing := range t.Blockers {
		if existing == blockerID {
			return
		}
	}
	t.Blockers = append(t.Blockers, blockerID)
	t.UpdatedAt = time.Now()
}

// RemoveBlocker removes a blocker task ID.
func (t *Task) RemoveBlocker(blockerID string) {
	for i, existing := range t.Blockers {
		if existing == blockerID {
			t.Blockers = append(t.Blockers[:i], t.Blockers[i+1:]...)
			t.UpdatedAt = time.Now()
			return
		}
	}
}

// IsBlocked returns true if the task has blockers.
func (t *Task) IsBlocked() bool {
	return len(t.Blockers) > 0
}

// IsComplete returns true if the task is done or dropped.
func (t *Task) IsComplete() bool {
	return t.Status == StatusDone || t.Status == StatusDropped
}

// IsOpen returns true if the task is not complete.
func (t *Task) IsOpen() bool {
	return !t.IsComplete()
}
