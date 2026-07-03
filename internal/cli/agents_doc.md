# yoke — task management CLI

yoke is the local task tracker. It is the single source of truth for what
work exists, what state it's in, and what's known about it (notes, history,
hierarchy, blockers). Use it directly — there is no wrapper.

Everything is stored locally in SQLite (`~/.yoke/yoke.db`). Some tasks are
linked to Notion pages and can sync bidirectionally.

## Conventions

**Identifiers.** Every task has two identifiers, interchangeable in every
command that takes an `<id>`:
- `seq` — human-friendly number (`42`). Stable, never reused.
- `id` — 4-char hash (`a3f8`). Use when scripting; prefix-match supported.

**Statuses** (exact strings, used in `--status` filters and JSON output):
| status | meaning |
|---|---|
| `pending` | not started |
| `active` | ready to work |
| `in_progress` | currently being worked |
| `blocked` | waiting on a blocker |
| `done` | completed |
| `dropped` | won't do |

**Priority.** `1` (urgent) … `5` (someday). Default `3`.

**Tags.** Free-form labels (`backend`, `bug`, …). Used for filtering and
routing. Set at creation (`-t`) or later (`yoke tag`).

## Machine-readable output

Read commands accept `--json`. JSON field names are a stable contract
(additive changes only). Status values in JSON are always the raw enum
strings above — never display formatting.

```
yoke show 42 --json      # task + resolved parent/children/blockers
yoke list --json         # array of tasks
yoke notes 42 --json     # array of notes
yoke log 42 --json       # array of events
yoke ready --json        # array of unblocked tasks
yoke context 42 -f json  # everything in one object
```

## Loading task context

`yoke context <id>` assembles the full picture of a task — description,
hierarchy, blockers, notes, recent activity — as markdown (default) or JSON.
This is the recommended first command when starting work on a task:

```
yoke context 42            # markdown, ready to read
yoke context 42 -f json    # same data, machine-readable
yoke context 42 --events 0 # include full history
```

## Command reference

### Creating & editing
```
yoke add "Title" [-t tag]... [-p N] [-b "body"]   # create (note: -t repeats, NOT --tags)
yoke edit <id> --title "..." | --body "..." | --priority N | --status <status>
yoke subtask <parent-id> "Title"                  # create child task
```

### Working a task
```
yoke start <id>                    # mark in_progress
yoke done <id> [--outcome "..."]   # mark done, record what happened
yoke drop <id>                     # mark dropped (won't do)
yoke note <id> "text"              # append a note (progress, findings, links)
```

Record meaningful progress as notes while working — notes are the task's
memory and surface in `yoke context` for whoever (or whatever) picks the
task up next.

### Dependencies & hierarchy
```
yoke block <id> --by <blocker-id>  # this task is blocked by another
yoke unblock <id> <blocker-id>     # remove the dependency
yoke tree                          # parent/child hierarchy view
```

### Finding tasks
```
yoke list                          # open tasks
yoke list --all                    # include done/dropped
yoke list --status in_progress     # filter by status
yoke list --tag backend            # filter by tag
yoke ready                         # unblocked tasks, ready to pick up
yoke search "query"                # full-text search in title/body
yoke show <id>                     # full detail for one task
yoke log [id]                      # event history (task or global)
```

### Tags
```
yoke tag <id> <tag>       # add
yoke untag <id> <tag>     # remove
yoke tags [id]            # list all tags, or one task's
```

### Notion sync
Tasks may be linked to Notion pages. Sync is explicit, never automatic,
and always validates the page's assignee before writing.
```
yoke import <notion-url>     # create local task from a Notion page
yoke link <id> <notion-url>  # link an existing local task
yoke pull <id> [--dry-run]   # apply Notion changes locally
yoke push <id> [--dry-run]   # push local changes to Notion
```

### Meta
```
yoke init      # one-time setup (~/.yoke)
yoke docs      # refresh + print the path of this document
```

## Recommended workflow when picking up a task

```
yoke context <id>                  # 1. load everything known
yoke start <id>                    # 2. mark in_progress
yoke note <id> "approach: ..."     # 3. record decisions as you go
yoke note <id> "PR: <url>"         # 4. link artifacts you produce
yoke done <id> --outcome "..."     # 5. close with what actually happened
```

If you discover follow-up work, create it (`yoke add` / `yoke subtask`)
rather than leaving it in prose — untracked work is lost work.
