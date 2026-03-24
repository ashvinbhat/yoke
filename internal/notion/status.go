package notion

import "github.com/ashvinbhat/yoke/task"

// NotionStatus represents a Notion status value.
type NotionStatus string

// Notion status constants matching user's database.
const (
	NotionStatusTodo       NotionStatus = "Todo"
	NotionStatusInProgress NotionStatus = "In Progress"
	NotionStatusInReview   NotionStatus = "In Review"
	NotionStatusOnQA       NotionStatus = "On QA"
	NotionStatusVerify     NotionStatus = "Verify"
	NotionStatusDone       NotionStatus = "Done"
	NotionStatusBlocked    NotionStatus = "Blocked"
)

// ToYokeStatus converts a Notion status to a yoke status.
func (n NotionStatus) ToYokeStatus() task.Status {
	switch n {
	case NotionStatusTodo:
		return task.StatusPending
	case NotionStatusInProgress, NotionStatusInReview, NotionStatusOnQA, NotionStatusVerify:
		return task.StatusInProgress
	case NotionStatusDone:
		return task.StatusDone
	case NotionStatusBlocked:
		return task.StatusBlocked
	default:
		return task.StatusPending
	}
}

// YokeToNotionStatus converts a yoke status to a Notion status.
func YokeToNotionStatus(s task.Status) NotionStatus {
	switch s {
	case task.StatusPending, task.StatusActive:
		return NotionStatusTodo
	case task.StatusInProgress:
		return NotionStatusInProgress
	case task.StatusBlocked:
		return NotionStatusBlocked
	case task.StatusDone, task.StatusDropped:
		return NotionStatusDone
	default:
		return NotionStatusTodo
	}
}

// NotionPriorityToYoke converts Notion priority to yoke priority (1-5).
func NotionPriorityToYoke(priority string) int {
	switch priority {
	case "High":
		return 1
	case "Medium":
		return 3
	case "Low":
		return 4
	default:
		return 3 // Default medium priority
	}
}

// YokePriorityToNotion converts yoke priority to Notion priority string.
func YokePriorityToNotion(priority int) string {
	switch priority {
	case 1, 2:
		return "High"
	case 3:
		return "Medium"
	case 4, 5:
		return "Low"
	default:
		return "Medium"
	}
}
