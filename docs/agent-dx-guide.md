# Agent DX Guide for CLI Projects

A pattern language for designing command-line tools that agents can use effectively.

---

## 1. Core Principles

### 1.1 Minimal Ceremony

**One command for the common case.**

Bad:
```bash
mytool init --config ./config.yaml
mytool start
mytool create entry --type log
mytool entry set-field --field title --value "Fixed bug"
mytool entry set-field --field body --value "Details..."
mytool entry commit
```

Good:
```bash
mytool log "Fixed bug" --body "Details..."
```

**Guidelines:**
- Collapse multi-step workflows into single commands
- Use positional arguments for the most common parameter
- Provide sensible defaults for everything else
- Reserve subcommand trees for genuinely distinct operations

### 1.2 Clear Next Action

**The tool should tell agents what to do next.**

Bad:
```bash
$ mytool status
Repository initialized.
3 items tracked.
```

Good:
```bash
$ mytool status
Repository initialized.
3 items tracked.
5 items pending action.

Run: mytool pending    # See what needs attention
Run: mytool process    # Handle pending items
```

**Patterns:**
- `pending` / `ready` commands that surface work
- Human output includes suggested next commands
- JSON output includes `next_action` or `suggested_commands` fields
- Exit messages guide toward logical next steps

### 1.3 Context Injection

**Provide a single command that outputs workflow state for session bootstrapping.**

```bash
$ mytool prime
MyTool workflow context
=======================
Repo: my-project
Branch: main

Last 3 actions:
  2026-01-15  "Fixed auth bypass" (3 items)
  2026-01-14  "Added rate limiting" (7 items)
  2026-01-13  "Initial setup" (12 items)

Pending: 5 items need attention

Quick commands:
  mytool pending     # See pending items
  mytool process     # Handle pending
  mytool show --last # View last action
```

**Why this matters:**
- Agents lose context on session boundaries
- `/clear` and compaction wipe working memory
- `prime` restores workflow state in one call
- Include in shell hooks or CLAUDE.md instructions

### 1.4 JSON Everywhere

**Every command supports `--json` for structured output.**

Human output:
```
Created entry tb_2026-01-15_abc123
  Anchor: abc1234
  Commits: 3
```

JSON output:
```json
{
  "status": "created",
  "id": "tb_2026-01-15_abc123",
  "anchor": "abc1234def5678...",
  "commits": 3
}
```

**Guidelines:**
- `--json` flag on every command, no exceptions
- Errors also return JSON: `{"error": "message", "code": 1}`
- Include machine-readable fields agents need (full SHAs, IDs, counts)
- Human output can be friendlier; JSON output must be complete

### 1.5 Sensible Defaults

**Commands work without flags for the common case.**

Bad:
```bash
mytool log --anchor HEAD --range auto --format default "Message"
```

Good:
```bash
mytool log "Message"
# Anchor defaults to HEAD
# Range defaults to since-last-entry
# Format defaults to human-readable
```

**Guidelines:**
- Defaults should match 80% of use cases
- Explicit flags for the 20% that need customization
- Document defaults clearly in help text
- Never require configuration before first use

---

## 2. Token Efficiency

### 2.1 Batch Operations

**Provide batch modes to reduce round-trips.**

Instead of:
```bash
mytool process item1
mytool process item2
mytool process item3
# 3 tool calls, 3 permission prompts
```

Provide:
```bash
mytool process --batch
# 1 tool call, 1 permission prompt, processes all pending
```

**Patterns:**
- `--batch` flag for processing multiple items
- `--all` flag where appropriate
- Accept multiple positional arguments: `mytool process item1 item2 item3`
- Batch JSON input via stdin: `echo '["item1","item2"]' | mytool process --stdin`

### 2.2 Compact Output Modes

**Provide terse output for high-volume queries.**

```bash
# Full output (default)
mytool query --last 5

# Compact for scanning
mytool query --last 5 --oneline

# IDs only for scripting
mytool query --last 5 --ids-only
```

### 2.3 Allowlist-Friendly Commands

**Design commands that can be pre-approved in permission systems.**

Bad (requires broad "run any mytool command" permission):
```bash
mytool exec --script "arbitrary code here"
```

Good (specific, auditable operations):
```bash
mytool log "message" --why "reason" --how "method"
mytool pending --json
mytool export --last 5
```

**Guidelines:**
- Prefer declarative over imperative commands
- Avoid eval/exec patterns that run arbitrary code
- Make destructive operations explicit: `mytool delete --confirm`
- Support `--dry-run` for write operations

---

## 3. Unix Composability

### 3.1 Pipe-Friendly Output

**Commands should compose with standard Unix tools.**

```bash
# Pipe to other CLIs
mytool export --json | jq '.entries[]'
mytool export --json | claude "Summarize this work"

# Use in scripts
for id in $(mytool query --ids-only); do
  mytool show "$id" --json
done
```

**Guidelines:**
- JSON to stdout (not stderr) when `--json` specified
- Human-readable to stdout, diagnostics to stderr
- Support stdin for batch input where sensible
- Exit codes follow conventions (0=success, 1=user error, 2=system error)

### 3.2 Output Destination Control

**Separate stdout behavior from file output.**

```bash
# To stdout (for piping)
mytool export --json

# To files (for persistence)
mytool export --json --out ./exports/
```

Don't conflate these—agents need both patterns.

---

## 4. Error Handling

### 4.1 Structured Errors

**Errors are data, not just messages.**

```json
{
  "error": "Missing required flag: --why",
  "code": 1,
  "hint": "Use --minor for trivial changes that don't need explanation"
}
```

### 4.2 Consistent Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | User error (bad args, missing fields, not found) |
| 2 | System error (git failed, network error) |
| 3 | Conflict (already exists, state mismatch) |

### 4.3 Recoverable Failures

**Tell agents how to recover.**

Bad:
```
Error: Entry already exists.
```

Good:
```
Error: Entry already exists for anchor abc1234.
Use --replace to overwrite, or --anchor <different-sha> to target another commit.
```

---

## 5. Documentation & Discoverability

### 5.1 Self-Documenting Help

```bash
$ mytool --help
A development ledger for capturing what/why/how.

Usage:
  mytool <command> [flags]

Core Commands:
  log       Record work with what/why/how
  pending   Show items needing attention
  prime     Output workflow context for session start

Query Commands:
  show      Display a single entry
  query     Search and retrieve entries
  export    Export for pipelines

All commands support --json for structured output.

Run 'mytool <command> --help' for details.
```

### 5.2 Skill Generation

**Provide a command that outputs content for building agent skills.**

```bash
mytool skill
mytool skill --format json
mytool skill --include-examples
```

This outputs:
- Core concepts and mental model
- Workflow patterns
- Command quick reference
- Common recipes
- Error recovery patterns

The tool knows itself best—let it generate its own documentation.

---

## 6. Agent Execution Contract

When documenting your CLI for agents, include an explicit contract:

```markdown
## Agent Contract

1. **Use CLI, not internals** — Call commands, consume JSON output
2. **Trust the tool** — Don't parse config files or data stores directly
3. **Use --dry-run** — Validate before writing
4. **Enrich, don't fabricate** — Add context you have, omit what you don't
5. **Check pending** — Know what needs attention before starting
6. **Prime on session start** — Restore context after /clear or compaction
```

---

## 7. Integration Patterns

### 7.1 Hook Points

Design for integration with agent hooks:

```bash
# Session start hook
mytool prime >> $SESSION_CONTEXT

# Pre-commit hook
mytool pending --count | grep -q "^0$" || echo "Warning: undocumented work"

# Post-work hook
mytool log "$WORK_SUMMARY" --why "$WORK_RATIONALE" --how "$WORK_METHOD"
```

### 7.2 Workflow Documentation

Provide copy-paste blocks for CLAUDE.md / AGENTS.md:

```markdown
## Development Workflow

This project uses MyTool for tracking work.

At session start:
  mytool prime

After completing work:
  mytool log "what" --why "why" --how "how"

At session end:
  mytool pending    # Check for undocumented work
```

---

## 8. Checklist

Use this checklist when designing agent-oriented CLIs:

### Essential
- [ ] `--json` flag on every command
- [ ] Structured error output with codes
- [ ] `prime` or equivalent context injection command
- [ ] `pending` or equivalent "what needs attention" command
- [ ] Sensible defaults—works without config
- [ ] `--dry-run` on write operations

### Recommended
- [ ] `--batch` mode for multi-item operations
- [ ] `--oneline` / `--ids-only` for compact output
- [ ] `skill` command for self-documentation
- [ ] Suggested next commands in output
- [ ] Recoverable error messages with hints

### Bonus
- [ ] Stdin support for batch input
- [ ] Exit code conventions documented
- [ ] Integration hooks documented
- [ ] CLAUDE.md workflow snippet provided

---

## 9. Examples in the Wild

### Beads (Issue Tracking)

- `bd prime` — Context injection
- `bd ready` — Clear next action
- `bd close <id>` — Minimal ceremony
- `--json` on all commands

### Timbers (Development Ledger)

- `timbers prime` — Context injection
- `timbers pending` — Clear next action
- `timbers log "what" --why "why" --how "how"` — Single command capture
- `timbers export --json | claude "..."` — Unix composability
- `timbers skill` — Self-documentation
- `--batch` mode for efficiency

---

## 10. Anti-Patterns

### Configuration-First
Requiring config files before any command works. Agents struggle with multi-step setup.

### Interactive-Only
TUI-only workflows with no scriptable alternative. Agents can't use curses interfaces.

### Implicit State
Commands that depend on hidden state without inspection commands. Agents need to query state.

### Verbose-Only Output
No compact modes for high-volume queries. Token budgets matter.

### Undocumented Errors
Generic error messages without codes or recovery hints. Agents need structured feedback.

### Broad Permissions
Commands that require "run anything" permissions. Allowlist-friendly commands can be pre-approved.
