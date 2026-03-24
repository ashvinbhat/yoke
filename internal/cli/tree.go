package cli

import (
	"fmt"
	"strings"

	"github.com/ashvinbhat/yoke/internal/task"
	"github.com/spf13/cobra"
)

var treeAll bool

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show task hierarchy",
	Long: `Displays tasks in a tree structure showing parent-child relationships.

Examples:
  yoke tree           # Show open tasks as tree
  yoke tree --all     # Include completed tasks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := task.ListOptions{
			Open: !treeAll,
		}

		tasks, err := store.List(opts)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		printTree(tasks)
		return nil
	},
}

// printTree displays tasks in a hierarchical tree format.
func printTree(tasks []*task.Task) {
	// Build a map of tasks by ID
	taskMap := make(map[string]*task.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Find root tasks (no parent or parent not in list)
	var roots []*task.Task
	children := make(map[string][]*task.Task)

	for _, t := range tasks {
		if t.Parent == nil {
			roots = append(roots, t)
		} else {
			// Check if parent is in our task list
			if _, exists := taskMap[*t.Parent]; exists {
				children[*t.Parent] = append(children[*t.Parent], t)
			} else {
				// Parent not in list, treat as root
				roots = append(roots, t)
			}
		}
	}

	// Print the tree
	for i, root := range roots {
		isLast := i == len(roots)-1
		printNode(root, "", isLast, children)
	}

	fmt.Printf("\n%d task(s)\n", len(tasks))
}

// printNode prints a single node and its children.
func printNode(t *task.Task, prefix string, isLast bool, children map[string][]*task.Task) {
	// Choose connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Format status indicator
	statusIcon := getStatusIcon(t.Status)

	// Format tags
	tags := ""
	if len(t.Tags) > 0 {
		tags = " [" + strings.Join(t.Tags, ", ") + "]"
	}

	// Print this node
	fmt.Printf("%s%s%s #%d %s%s\n", prefix, connector, statusIcon, t.Seq, t.Title, tags)

	// Determine prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Print children
	childList := children[t.ID]
	for i, child := range childList {
		isChildLast := i == len(childList)-1
		printNode(child, childPrefix, isChildLast, children)
	}
}

// getStatusIcon returns an icon for the status.
func getStatusIcon(s task.Status) string {
	switch s {
	case task.StatusPending:
		return "○"
	case task.StatusActive:
		return "◐"
	case task.StatusInProgress:
		return "●"
	case task.StatusBlocked:
		return "✖"
	case task.StatusDone:
		return "✓"
	case task.StatusDropped:
		return "✗"
	default:
		return "?"
	}
}

// TreeView generates a tree view string for tasks (used by list --tree).
func TreeView(tasks []*task.Task) string {
	if len(tasks) == 0 {
		return "No tasks found"
	}

	var sb strings.Builder

	// Build a map of tasks by ID
	taskMap := make(map[string]*task.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Find root tasks
	var roots []*task.Task
	children := make(map[string][]*task.Task)

	for _, t := range tasks {
		if t.Parent == nil {
			roots = append(roots, t)
		} else {
			if _, exists := taskMap[*t.Parent]; exists {
				children[*t.Parent] = append(children[*t.Parent], t)
			} else {
				roots = append(roots, t)
			}
		}
	}

	// Build tree string
	for i, root := range roots {
		isLast := i == len(roots)-1
		buildTreeString(&sb, root, "", isLast, children)
	}

	return sb.String()
}

func buildTreeString(sb *strings.Builder, t *task.Task, prefix string, isLast bool, children map[string][]*task.Task) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	statusIcon := getStatusIcon(t.Status)
	tags := ""
	if len(t.Tags) > 0 {
		tags = " [" + strings.Join(t.Tags, ", ") + "]"
	}

	sb.WriteString(fmt.Sprintf("%s%s%s #%d %s%s\n", prefix, connector, statusIcon, t.Seq, t.Title, tags))

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	childList := children[t.ID]
	for i, child := range childList {
		isChildLast := i == len(childList)-1
		buildTreeString(sb, child, childPrefix, isChildLast, children)
	}
}

func init() {
	treeCmd.Flags().BoolVarP(&treeAll, "all", "a", false, "Include completed tasks")
	rootCmd.AddCommand(treeCmd)
}
