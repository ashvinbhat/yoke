package notion

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmPush prompts the user to confirm a push operation.
func ConfirmPush(taskCount int) bool {
	if taskCount == 0 {
		return false
	}

	fmt.Printf("\n⚠️  About to update %d Notion page(s).\n", taskCount)
	fmt.Print("Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ConfirmSinglePush prompts for a single task push.
func ConfirmSinglePush(seq int, title string) bool {
	fmt.Printf("\n⚠️  About to update Notion page:\n")
	fmt.Printf("   Task #%d: %s\n", seq, title)
	fmt.Print("Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// PushPlan represents what will be pushed to Notion.
type PushPlan struct {
	Tasks []PushTaskPlan
}

// PushTaskPlan represents a single task to push.
type PushTaskPlan struct {
	Seq       int
	ID        string
	Title     string
	NotionID  string
	Changes   []FieldChange
}

// Print displays the push plan.
func (p *PushPlan) Print() {
	fmt.Println("=== DRY RUN - No changes will be made to Notion ===")

	if len(p.Tasks) == 0 {
		fmt.Println("No tasks to push")
		return
	}

	fmt.Printf("Would push %d task(s) to Notion:\n\n", len(p.Tasks))

	for _, t := range p.Tasks {
		fmt.Printf("Task #%d [%s]: %s\n", t.Seq, t.ID, t.Title)
		for _, c := range t.Changes {
			fmt.Printf("  %s: %q → %q\n", c.Field, c.OldValue, c.NewValue)
		}
		fmt.Println()
	}
}

// PrintSummary shows a brief summary.
func (p *PushPlan) PrintSummary() {
	if len(p.Tasks) == 0 {
		fmt.Println("Nothing to push")
		return
	}

	fmt.Printf("\nSummary: %d task(s) to push to Notion\n", len(p.Tasks))
	for _, t := range p.Tasks {
		fmt.Printf("  • #%d %s (%d changes)\n", t.Seq, t.Title, len(t.Changes))
	}
}
