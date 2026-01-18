# Timbers

A Git-native development ledger that captures *what/why/how* as structured records.

## What It Does

Timbers turns Git history into a durable development ledger by:
- Harvesting objective facts from Git (commits, diffstat, changed files)
- Pairing them with agent/human-authored rationale (what/why/how)
- Storing as portable Git notes that sync to remotes
- Exporting structured data for downstream narrative generation

## Quick Start

```bash
# Install
go install github.com/steveyegge/timbers/cmd/timbers@latest

# Record work (the primary operation)
timbers log "Fixed auth bypass" --why "Input not sanitized" --how "Added validation"

# Auto-extract from commit messages
timbers log --auto

# Batch entries by work item
timbers log --batch beads

# See what needs documentation
timbers pending

# Get workflow context at session start
timbers prime

# Export for LLM pipelines
timbers export --last 5 --json | claude "Generate changelog"
```

## Command Reference

**Core Commands**
- `timbers log` - Record development work with what/why/how
- `timbers pending` - Show commits awaiting documentation
- `timbers status` - Display repository and notes state
- `timbers show` - Display a single ledger entry
- `timbers prime` - Emit session context for agent workflow start

**Query & Export**
- `timbers query` - Search ledger entries by tags or content
- `timbers export` - Export entries as JSON or Markdown

**Sync & Management**
- `timbers notes init` - Initialize Git notes remote
- `timbers notes push` - Push notes to remote
- `timbers notes fetch` - Fetch notes from remote
- `timbers notes status` - Check sync status

**Agent Integration**
- `timbers skill` - Emit skill documentation for agents

All commands support `--json` for structured output and `--dry-run` for write operations.

## Core Philosophy

**The "why" is the most valuable field in the ledger.**

Git captures *what* changed (the diff). Commit messages describe *how*. But *why* — the reasoning, context, requirements, decisions — lives in ephemeral places that evaporate after sessions end.

**Timbers exists to capture "why" before it disappears.**

## Agent DX

Timbers is designed for agent consumption:
- `--json` on every command
- `timbers prime` for session context injection
- `timbers pending` for clear next action
- `timbers skill` for self-documentation
- Structured errors with recovery hints
- Pipe-friendly exports

## Documentation

- [Spec](docs/spec.md) - Full specification
- [Agent DX Guide](docs/agent-dx-guide.md) - CLI design patterns for agents

## Development

```bash
just setup    # First-time setup (mise, deps)
just check    # Run all quality gates (lint, test)
just fix      # Auto-fix lint issues
just run      # Run the CLI
```

## License

MIT
