package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	contextFormat string
	contextEvents int
)

var contextCmd = &cobra.Command{
	Use:   "context <id>",
	Short: "Assemble the full context for a task",
	Long: `Assembles everything known about a task into a single document:
title, status, description, hierarchy (parent/children), blockers, notes,
and recent activity.

This is the canonical way for tools and agents to load a task's context.
Markdown output is designed to be embedded directly into an agent's
working context; JSON output is for programmatic consumers.

Examples:
  yoke context 42                    # Markdown to stdout
  yoke context a3f8 --format json    # Machine-readable
  yoke context 42 --events 25        # Include more history`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := store.Get(args[0])
		if err != nil {
			return fmt.Errorf("task not found: %s", args[0])
		}

		detail := buildTaskDetail(t)

		notes, err := store.GetNotes(t.ID)
		if err != nil {
			return fmt.Errorf("failed to get notes: %w", err)
		}

		events, err := store.GetEvents(t.ID)
		if err != nil {
			return fmt.Errorf("failed to get events: %w", err)
		}
		if contextEvents > 0 && len(events) > contextEvents {
			events = events[len(events)-contextEvents:]
		}

		ctx := taskContextJSON{
			taskDetailJSON: detail,
			Notes:          toNotesJSON(notes),
			Events:         toEventsJSON(events),
		}

		switch contextFormat {
		case "json":
			return printJSON(ctx)
		case "md", "markdown":
			fmt.Print(renderContextMarkdown(ctx))
			return nil
		default:
			return fmt.Errorf("unknown format %q (expected md or json)", contextFormat)
		}
	},
}

func renderContextMarkdown(ctx taskContextJSON) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Task #%d: %s\n\n", ctx.Seq, ctx.Title)

	meta := []string{
		fmt.Sprintf("**Status:** %s", ctx.Status),
		fmt.Sprintf("**Priority:** P%d", ctx.Priority),
	}
	if len(ctx.Tags) > 0 {
		meta = append(meta, fmt.Sprintf("**Tags:** %s", strings.Join(ctx.Tags, ", ")))
	}
	sb.WriteString(strings.Join(meta, " · "))
	sb.WriteString("\n")
	if ctx.NotionURL != nil {
		fmt.Fprintf(&sb, "**Notion:** %s\n", *ctx.NotionURL)
	}
	if ctx.ExternalRef != nil {
		fmt.Fprintf(&sb, "**External ref:** %s\n", *ctx.ExternalRef)
	}
	sb.WriteString("\n")

	if ctx.Body != "" {
		sb.WriteString("## Description\n\n")
		sb.WriteString(strings.TrimSpace(ctx.Body))
		sb.WriteString("\n\n")
	}

	if ctx.Parent != nil || len(ctx.Children) > 0 {
		sb.WriteString("## Hierarchy\n\n")
		if ctx.Parent != nil {
			fmt.Fprintf(&sb, "Parent: #%d %s (%s)\n", ctx.Parent.Seq, ctx.Parent.Title, ctx.Parent.Status)
		}
		if len(ctx.Children) > 0 {
			sb.WriteString("Subtasks:\n")
			for _, c := range ctx.Children {
				marker := " "
				if c.Status == "done" {
					marker = "x"
				}
				fmt.Fprintf(&sb, "- [%s] #%d %s (%s)\n", marker, c.Seq, c.Title, c.Status)
			}
		}
		sb.WriteString("\n")
	}

	if len(ctx.BlockedBy) > 0 {
		sb.WriteString("## Blocked by\n\n")
		for _, b := range ctx.BlockedBy {
			fmt.Fprintf(&sb, "- #%d %s (%s)\n", b.Seq, b.Title, b.Status)
		}
		sb.WriteString("\n")
	}

	if len(ctx.Notes) > 0 {
		sb.WriteString("## Notes\n\n")
		for _, n := range ctx.Notes {
			fmt.Fprintf(&sb, "- [%s] %s\n", n.CreatedAt.Format("2006-01-02 15:04"), n.Content)
		}
		sb.WriteString("\n")
	}

	if len(ctx.Events) > 0 {
		sb.WriteString("## Recent activity\n\n")
		for _, e := range ctx.Events {
			line := e.EventType
			if e.OldValue != "" || e.NewValue != "" {
				line = fmt.Sprintf("%s: %s → %s", e.EventType, e.OldValue, e.NewValue)
			}
			fmt.Fprintf(&sb, "- [%s] %s\n", e.CreatedAt.Format("2006-01-02 15:04"), line)
		}
		sb.WriteString("\n")
	}

	if ctx.Outcome != nil {
		sb.WriteString("## Outcome\n\n")
		sb.WriteString(strings.TrimSpace(*ctx.Outcome))
		sb.WriteString("\n")
	}

	return sb.String()
}

func init() {
	contextCmd.Flags().StringVarP(&contextFormat, "format", "f", "md", "Output format: md or json")
	contextCmd.Flags().IntVar(&contextEvents, "events", 10, "Max recent events to include (0 = all)")
	rootCmd.AddCommand(contextCmd)
}
