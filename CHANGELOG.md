# Changelog

All notable changes to this project are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Keys: `CB-123`, `pr:<repo>-<n>`, `thread:<id>`. A key finds one task, and
  every command that takes an id takes a key. `yoke add --key` finds or
  reopens the task a key names instead of creating a second one; `yoke key`
  adds keys to a task.
- Found from: `yoke add --from <id>` records the task a new one was noticed
  in; `show`, `context` and `list --from` show it. JSON gains `keys`,
  `foundFromId`, and on `show`/`context`, `foundFrom` and `found`.
- `yoke sync`: brings in the Notion tickets assigned to you in the current
  sprint, tagged `ticket` and keyed by Task ID, and refreshes every other
  open linked task. Read-only.

### Fixed
- Notion's Merged on QA, Verified on QA and RCA Pending map to `in_progress`,
  Cancelled to `dropped`, and P0–P3 priorities to 1–4.
- `yoke add --parent 42` stores the parent's id, not the `42` typed.

## [0.1.0] - 2026-08-19

Initial public release.

### Added
- Local-first task management on SQLite: add, list, show, start, done, drop.
- Notes & history, tags, hierarchy (parent/child) and blocker dependencies.
- Search and filtering; `--json` output for scripting and tool integration.
- Optional Notion sync.
- `yoke docs` — agent-facing usage guide (AGENTS.md).

[Unreleased]: https://github.com/ashvinbhat/yoke/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ashvinbhat/yoke/releases/tag/v0.1.0
