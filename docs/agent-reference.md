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
- Entries document completed work; `timbers amend` records supported corrections
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

**Usage**: `timbers log [<what>] --why <why> --how <how> [flags]`

`what` is a capture-time snapshot. When omitted, Timbers derives it from the
selected commit subject(s); provide it explicitly when those subjects are weak.
Later SHA rewrites do not remove the stored text.

**Flags**:
- `--why`: Why — the verdict (required unless --minor/--auto)
- `--how`: How (required unless --minor/--auto)
- `--notes`: Deliberation context — the journey (optional, use selectively)
- `--tag`: Add tag (repeatable)
- `--work-item`: Link work item (system:id)
- `--who`: Replace contributors with `Name <email>` identities (repeatable)
- `--range`: Commit range (A..B)
- `--minor`: Use defaults for trivial changes
- `--auto`: Extract what/why/how from commits
- `--yes`: Skip confirmation in auto mode
- `--batch`: Create entries by work-item/day
- `--dry-run`: Preview without writing
- `--push`: Push to remote after logging

**Examples**:
```bash
timbers log "Switched to cursor pagination" --why "Offset skips rows on concurrent inserts" --how "Cursor tokens with created_at + id"
timbers log "Fix" --why "Race condition in cache" --how "Added mutex" --tag bugfix
timbers log "Added write-through caching" --why "Read-aside served stale data after writes" --how "Cache update in same transaction" \
  --notes "Write-behind was faster but risks data loss on crash. Consistency wins."
timbers log --why "Avoid duplicate updates" --how "Reuse the existing write path"
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

### ack

Record why a commit intentionally does not need a content entry.

**Usage**: `timbers ack <sha> --reason <text> [flags]`

Use `ack` for mechanical or already-documented commits, not as a substitute for
capturing available rationale. A common rewrite case is:

```bash
timbers ack <new-sha> --reason "rebased; content in <original-entry-id>"
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
timbers show --latest
```

### query

Search and retrieve entries

**Usage**: `timbers query [flags]`

**Flags**:
- `--last`: Show last N entries
- `--since`: Entries since duration (24h, 7d) or date
- `--until`: Entries until duration (24h, 7d) or date
- `--range`: Entries whose commits or ledger files appear in a Git range
- `--tag`: Match any supplied tag (repeatable or comma-separated)
- `--oneline`: Compact output

**Examples**:
```bash
timbers query --last 5
timbers query --last 10 --oneline
timbers query --since 7d --tag security
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

**Templates**: `changelog`, `decision-digest`, `devblog`, `pr-description`, `release-notes`, `sprint-report`, `standup`

**Examples**:
```bash
timbers draft changelog --since 7d | claude -p
timbers draft standup --since 1d --model haiku
timbers draft decision-digest --last 20
```

### report

Run a configured report profile. Profiles are ordinary templates with a
`report` frontmatter block that supplies a default scope and compact input.

**Usage**: `timbers report <profile> [flags]`

Without `--model`, report prints the resolved prompt for piping. With a model,
it emits sanitized report content. An explicit `--last`, `--since`, or
`--range` replaces the profile default. An empty selection or configured quiet
result succeeds without artifact content.

```bash
timbers report decision-digest
timbers report decision-digest --model opus
timbers report decision-digest --since 30d --model opus
```

The decision digest is retrospective and non-authoritative. Project-native
ADRs and design documents remain the source of truth.

### Ledger integrity

`doctor` names malformed entry files. Human query output warns once while
preserving its result shape; artifact generation fails rather than silently
producing a report from an incomplete ledger.

### amend

Update an existing ledger entry

**Usage**: `timbers amend <id> [flags]`

**Flags**:
- `--what <text>`: Update the what field
- `--why <text>`: Update the why field
- `--how <text>`: Update the how field
- `--notes <text>`: Update the notes field
- `--tag <name>`: Add tag (repeatable)
- `--who "Name <email>"`: Replace contributors (repeatable; no Git lookup)
- `--dry-run`: Preview without writing
- `--json`: Structured JSON output

**Examples**:
```bash
timbers amend tb_2026-01-15T10:30:00Z_abc123 --why "Updated reasoning"
```

## Contract

**Schema**: `timbers.devlog/v1`

**Contributor attribution**: `entry.contributors` is an optional persisted
capture-time snapshot. Never infer attribution from workset SHAs or prose when
it is absent. See [Contributor attribution](contributor-attribution.md).

**JSON Support**: All commands support --json for structured output

**Error Format**: `{"error": "message", "code": N}`

### Exit Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | Command completed successfully |
| 1 | User error | Bad arguments, missing fields, not found |
| 2 | System error | Git failed, I/O error |
| 3 | Conflict | Entry exists, state mismatch |
