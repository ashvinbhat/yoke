# yoke

> The yoke that binds your tasks to your work.

A local-first task management system built in Go. Syncs with Notion, tracks dependencies, and provides a CLI for managing your daily work.

## Philosophy

- **Local-first**: SQLite storage, works offline, you own your data
- **Single source of truth**: All tasks flow through yoke
- **Sync, don't replace**: Links to Notion, doesn't duplicate it
- **Simple core, extensible**: Start minimal, grow as needed

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           NOTION                                │
│                      (Optional sync)                            │
└─────────────────────────────┬───────────────────────────────────┘
                              │ bidirectional sync
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                           YOKE                                  │
│                    (Local task manager)                         │
│  • SQLite storage         • Notion sync                         │
│  • Dependencies           • Hierarchy                           │
│  • Status tracking        • Event log                           │
└─────────────────────────────────────────────────────────────────┘
```

## Installation

```bash
go install github.com/ashvinbhat/yoke/cmd/yoke@latest
```

Or build from source:
```bash
git clone https://github.com/ashvinbhat/yoke.git
cd yoke
make build
```

## Quick Start

```bash
# Initialize yoke
yoke init

# Add a task
yoke add "Implement user authentication"

# List tasks
yoke list

# Start working
yoke start <id>

# Complete
yoke done <id>
```

## Commands

### Core
```bash
yoke init                      # Initialize yoke in ~/.yoke
yoke add "title"               # Create task
yoke list                      # List tasks
yoke show <id>                 # Task details
yoke start <id>                # Mark as in_progress
yoke done <id>                 # Complete task
yoke drop <id>                 # Abandon task
```

### Notes & History
```bash
yoke note <id> "text"          # Add note to task
yoke notes <id>                # Show task notes
yoke log <id>                  # Show task history
yoke log                       # Recent activity
```

### Editing & Tags
```bash
yoke edit <id>                 # Edit task properties
yoke tag <id> <tag>            # Add tag
yoke untag <id> <tag>          # Remove tag
yoke tags                      # List all tags
```

### Hierarchy & Dependencies
```bash
yoke subtask <parent> "title"  # Create subtask
yoke block <id> --by <other>   # Add blocker
yoke unblock <id> <other>      # Remove blocker
yoke ready                     # Show unblocked tasks
yoke tree                      # Show task hierarchy
yoke list --tree               # List as tree view
```

### Search & Filter
```bash
yoke search "keyword"          # Search tasks
yoke list --tag <tag>          # Filter by tag
yoke list --status <status>    # Filter by status
yoke list --blocked            # Show blocked tasks
```

### Notion Sync
```bash
yoke link <id> <notion-url>    # Link local task to Notion page
yoke import <notion-url>       # Import task from Notion
yoke pull [id]                 # Pull updates from Notion
yoke pull --dry-run            # Preview what would change
# push/sync coming soon
```

## Storage

Data stored in `~/.yoke/`:
```
~/.yoke/
├── yoke.db          # SQLite database
├── config.yaml      # Configuration
└── backups/         # Automatic backups
```

## Configuration

```yaml
# ~/.yoke/config.yaml
notion:
  token: "${NOTION_TOKEN}"     # Reference env var from ~/.yoke/.env
  database_id: "xxx"           # Tasks database ID
  assignee_name: "Your Name"   # Safety: only interact with your tasks

defaults:
  priority: 3                  # Default priority (1-5)
  status: "pending"            # Default status
```

Environment variables can be stored in `~/.yoke/.env`:
```bash
NOTION_TOKEN=secret_xxx
```

## Status

Built with love. See [ROADMAP.md](./ROADMAP.md) for development plan.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). `main` is protected — changes land via
pull requests, and CI (build + vet + test) must pass.

## License

[MIT](./LICENSE)
