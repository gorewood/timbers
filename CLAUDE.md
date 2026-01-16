# Timbers - Development Ledger CLI

A Git-native development ledger that captures structured what/why/how records as Git notes and exports LLM-ready markdown for narratives.

## Overview

Timbers turns Git history into a durable, queryable development ledger by:
- Harvesting objective facts from Git (commits, diffstats, tags)
- Storing human/agent-authored rationale as portable Git notes
- Exporting frontmatter-rich markdown packets for downstream narrative generation

## Development

```bash
just setup    # First-time setup (mise, deps)
just check    # Run all quality gates (lint, test)
just fix      # Auto-fix lint issues
```

## Tech Stack

- **Language**: Go 1.25+
- **CLI Framework**: Charmbracelet ecosystem
  - `fang` - Command-line flags
  - `bubbletea` - TUI framework
  - `bubbles` - TUI components
  - `huh` - Interactive forms
  - `lipgloss` - Styling
  - `log` - Structured logging

## Architecture

```
cmd/timbers/          # Entry point
internal/
  cli/                # Command definitions (fang-based)
  ledger/             # Core ledger operations
  notes/              # Git notes read/write
  schema/             # JSON schema validation
  export/             # Markdown export
  adapters/           # Tracker adapters (beads, linear)
```

## Conventions

- All errors handled explicitly (no `_ = err`)
- Exported identifiers have doc comments
- Table-driven tests for multiple cases
- JSON output for agent consumption (`--json` flags)
- Human-friendly output for interactive use

## Key Commands (MVP)

- `timbers status` - Repo and notes status
- `timbers draft` - Generate draft entry from Git range
- `timbers write` - Write entry to Git notes
- `timbers query` - Query entries by filters
- `timbers export` - Export to markdown

## Schema

Records use JSON schema `timbers.devlog/v1` stored in `refs/notes/timbers`.
See `docs/spec.md` for full schema details.
