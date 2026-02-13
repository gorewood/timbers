# Timbers Agent Reference

This document provides reference documentation for AI agents working with Timbers.
For dynamic session context, use `timbers prime`. For CLAUDE.md integration, use `timbers onboard`.

## Core Concepts

**Timbers**: A Git-native development ledger that captures what/why/how as structured records.

**Development Ledger**: A persistent record of work that pairs objective facts from Git with human-authored rationale.

**Entry**: A single ledger record documenting a unit of work. It contains a workset (git data), summary (what/why/how), and optional notes.

**Workset**: Captures the Git evidence: anchor commit, commit list, range, and diffstat.

**Summary**: Provides the rationale: what was done, why it was done (the verdict), and how it was accomplished.

**Notes**: Optional free-form deliberation context — the journey to the decision. Use when you explored alternatives or made a real choice. Skip for routine work.

### Key Points

- Entries are stored as JSON files in `.timbers/YYYY/MM/DD/` directories and sync via regular `git push`
- Each entry has a unique ID: tb_<timestamp>_<short-sha>
- The ledger is append-only; entries document completed work
- All commands support --json for structured output

## Workflow Patterns

A typical session follows: prime -> work -> pending -> log -> query

### Prime
**Command**: `timbers prime`

Bootstrap session context.

### Work
**Command**: (git commits)

Do development work.

### Check
**Command**: `timbers pending`

Review undocumented commits.

### Log
**Command**: `timbers log "..." --why "..." --how "..."`

Document work.

### Query
**Command**: `timbers query --last 5`

Review recent entries.

## Command Reference

### log

Record work as a ledger entry

**Usage**: `timbers log <what> --why <why> --how <how> [flags]`

**Flags**:
- `--why`: Why — the verdict (required unless --minor/--auto)
- `--how`: How (required unless --minor/--auto)
- `--notes`: Deliberation context — the journey (optional, use selectively)
- `--tag`: Add tag (repeatable)
- `--work-item`: Link work item (system:id)
- `--range`: Commit range (A..B)
- `--minor`: Use defaults for trivial changes
- `--auto`: Extract what/why/how from commits
- `--yes`: Skip confirmation in auto mode
- `--batch`: Create entries by work-item/day
- `--dry-run`: Preview without writing
- `--push`: Push to remote after logging

**Examples**:
```bash
timbers log "Added auth" --why "JWT chosen over session cookies for stateless scaling" --how "JWT with refresh flow"
timbers log "Fix" --why "Race condition in cache" --how "Added mutex" --tag bugfix
timbers log "Refactored auth" --why "Middleware over decorator for route coverage" --how "Extracted to middleware" \
  --notes "Decorator approach missed 3 routes. Middleware catches all by default."
```

### pending

Show undocumented commits

**Usage**: `timbers pending [flags]`

**Flags**:
- `--count`: Show only count

**Examples**:
```bash
timbers pending
timbers pending --count
```

### prime

Session context injection

**Usage**: `timbers prime [flags]`

**Flags**:
- `--last`: Recent entries (default: 3)

**Examples**:
```bash
timbers prime
timbers prime --last 5
```

### status

Show repository and ledger state

**Usage**: `timbers status [flags]`

**Examples**:
```bash
timbers status
timbers status --json
```

### show

Display a single entry

**Usage**: `timbers show [<id>] [flags]`

**Flags**:
- `--latest`: Show most recent entry

**Examples**:
```bash
timbers show <id>
timbers show --last
```

### query

Search and retrieve entries

**Usage**: `timbers query [flags]`

**Flags**:
- `--last`: Show last N entries
- `--since`: Entries since duration (24h, 7d) or date
- `--until`: Entries until duration (24h, 7d) or date
- `--oneline`: Compact output

**Examples**:
```bash
timbers query --last 5
timbers query --last 10 --oneline
```

### export

Export entries to formats

**Usage**: `timbers export [flags]`

**Flags**:
- `--last`: Export last N
- `--since`: Entries since duration (24h, 7d) or date
- `--until`: Entries until duration (24h, 7d) or date
- `--range`: Commit range (A..B)
- `--format`: json or md
- `--out`: Output directory

**Examples**:
```bash
timbers export --last 5 --json
timbers export --format md --out ./notes/
```

### draft

Render templates with ledger entries for LLM consumption or direct execution

**Usage**: `timbers draft <template> [flags]`

**Flags**:
- `--last N`: Use last N entries
- `--since <duration|date>`: Use entries since duration or date
- `--until <duration|date>`: Use entries until duration or date
- `--range A..B`: Use entries in commit range
- `--append <text>`: Append extra instructions
- `--list`: List available templates
- `--show`: Show template content without rendering
- `-m, --model <name>`: Execute with built-in LLM
- `--json`: Structured JSON output

**Templates**: `changelog`, `devblog`, `exec-summary`, `pr-description`, `release-notes`, `sprint-report`, `decision-log`

**Examples**:
```bash
timbers draft changelog --since 7d | claude -p
timbers draft exec-summary --last 10 --model haiku
timbers draft decision-log --last 20
```

### amend

Update an existing ledger entry

**Usage**: `timbers amend <id> [flags]`

**Flags**:
- `--what <text>`: Update the what field
- `--why <text>`: Update the why field
- `--how <text>`: Update the how field
- `--notes <text>`: Update the notes field
- `--tag <name>`: Add tag (repeatable)
- `--dry-run`: Preview without writing
- `--json`: Structured JSON output

**Examples**:
```bash
timbers amend tb_2026-01-15T10:30:00Z_abc123 --why "Updated reasoning"
```

## Contract

**Schema**: `timbers.devlog/v1`

**JSON Support**: All commands support --json for structured output

**Error Format**: `{"error": "message", "code": N}`

### Exit Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | Command completed successfully |
| 1 | User error | Bad arguments, missing fields, not found |
| 2 | System error | Git failed, I/O error |
| 3 | Conflict | Entry exists, state mismatch |
