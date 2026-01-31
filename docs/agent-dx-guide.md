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

### 5.2 Dynamic Context over Static Documentation

**Prefer dynamic context injection over static skill generation.**

A `skill` command that outputs static documentation seems useful, but in practice:
- Static docs drift from actual behavior
- Agents need *current state*, not reference material
- The `prime` command already provides workflow context

**Better pattern:** Use `prime` for dynamic context, `onboard` for minimal documentation pointers, and `setup` for automatic injection. If static reference docs are needed, put them in `docs/` where they can be version-controlled and reviewed.

The exception: if your tool is designed to generate plugin/skill files for agent frameworks, a `skill --emit-plugin` command may be warranted. But don't conflate "self-documenting" with "agent-friendly."

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

## 7. Automatic Integration

### 7.1 The Injection Problem

**Good primitives aren't enough.** A tool can have perfect `prime`, `pending`, and `--json` support, but if agents don't *use* them, the tool fails silently.

Real-world failure mode:
```
Session starts → Agent jumps into task → Commits work → Never runs prime
                                                      → Never checks pending
                                                      → Work is undocumented
```

The agent had access to `mytool prime` and `mytool pending`. The CLAUDE.md even documented the workflow. But nothing *triggered* the agent to use them.

**The fix: automatic injection.** Don't rely on agents reading documentation and remembering to run commands. Wire the tool into the environment so context flows automatically.

### 7.2 The Integration Stack

A complete agent-oriented CLI provides four layers of integration:

| Layer | Command | Purpose |
|-------|---------|---------|
| Health | `mytool doctor` | Verify setup, suggest fixes |
| Git Hooks | `mytool hooks install` | Pre-commit warnings, auto-sync |
| Editor Integration | `mytool setup claude` | Session-start context injection |
| Documentation | `mytool onboard` | Minimal AGENTS.md snippet |

Each layer reinforces the others. If hooks fail, doctor catches it. If setup is missing, doctor warns. If an agent reads AGENTS.md, it points to `prime` for dynamic context.

### 7.3 Health Check: `doctor`

**Every tool should have a `doctor` command** that verifies installation health and suggests fixes.

```bash
$ mytool doctor

CORE
  ✓  Storage initialized
  ✓  Remote configured

WORKFLOW
  ⚠  Pending 3 undocumented items
     └─ Run 'mytool pending' to review

INTEGRATION
  ✓  Git hooks installed
  ⚠  Claude integration not configured
     └─ Run 'mytool setup claude' to install

✓ 3 passed  ⚠ 2 warnings  ✖ 0 failed
```

**Design principles:**
- Categorize checks (Core, Workflow, Integration, etc.)
- Show pass/warn/fail with visual indicators
- Provide specific fix commands for each issue
- Support `--fix` to auto-remediate where possible
- Support `--json` for programmatic checking

### 7.4 Git Hooks: `hooks install/uninstall`

**Git hooks provide passive enforcement** without blocking developer workflow.

```bash
$ mytool hooks install
✓ Installed pre-commit hook
  └─ Warns about undocumented work (non-blocking)
```

**Key design decisions:**

1. **Warn, don't block.** Pre-commit hooks should print warnings but allow commits to proceed. Blocking commits creates friction that makes developers disable hooks entirely.

2. **Chain with existing hooks.** Use `--chain` to preserve pre-existing hooks:
   ```bash
   mytool hooks install --chain  # Runs existing hooks, then mytool hook
   ```

3. **Thin shims, not shell scripts.** Hook files should be minimal shims that call back into the CLI:
   ```bash
   #!/bin/sh
   mytool hook run pre-commit "$@"
   ```
   This keeps logic in testable Go/Rust code, not fragile shell.

4. **Graceful degradation.** Hooks should handle missing binaries:
   ```bash
   if command -v mytool >/dev/null 2>&1; then
     mytool hook run pre-commit "$@"
   fi
   ```

5. **Clean uninstall.** `mytool hooks uninstall` should restore any backed-up hooks and leave no traces.

### 7.5 Editor Integration: `setup <editor>`

**Editor-specific hooks inject context at session start** without requiring the agent to remember.

```bash
$ mytool setup claude
✓ Claude Code hook installed
  └─ Will inject 'mytool prime' at session start
```

**Supported patterns:**

```bash
mytool setup claude          # Install Claude Code integration
mytool setup claude --check  # Verify installation
mytool setup claude --remove # Uninstall
mytool setup claude --print  # Show what would be installed (dry-run)
mytool setup --list          # Show available integrations
```

**What setup installs:**

For Claude Code, this typically means adding a `SessionStart` hook that runs `mytool prime` and injects the output into the session context. The agent starts every session with workflow state already loaded.

**Future integrations** might include: Cursor, Windsurf, Aider, Gemini CLI, Copilot.

### 7.6 Documentation Snippets: `onboard`

**Generate minimal documentation that points to dynamic context.**

```bash
$ mytool onboard

## Development Workflow

This project uses **mytool** for tracking work.
Run `mytool prime` for workflow context, or install hooks (`mytool hooks install`) for auto-injection.

**Quick reference:**
- `mytool pending` - Check for undocumented work
- `mytool log "what" --why "why" --how "how"` - Record work

For full workflow details: `mytool prime`
```

**Why minimal?** Documentation rots. If AGENTS.md contains detailed workflow instructions, they'll drift from actual behavior. Instead:
- Keep the snippet short (< 15 lines)
- Point to `mytool prime` for dynamic context
- Let `prime` provide the authoritative workflow documentation

### 7.7 Orchestrated Setup: `init`

**A single command should set up everything.**

```bash
$ mytool init

Initializing mytool in my-project...

  ✓ Storage created
  ✓ Remote configured
  ✓ Git hooks installed (pre-commit)

Optional integrations:
  ? Install Claude Code integration? [Y/n] y
  ✓ Claude hook installed

Next steps:
  1. Add workflow snippet to CLAUDE.md:
     mytool onboard >> CLAUDE.md

  2. Verify setup:
     mytool doctor
```

**Design principles:**
- Single entry point for new users
- Interactive prompts for optional integrations
- `--yes` flag for scripted/automated setup
- Idempotent (safe to run again)
- Clear next-steps guidance

### 7.8 Clean Removal: `uninstall`

**Uninstall should reverse everything init/hooks/setup created.**

```bash
$ mytool uninstall

Components found:
  • Storage: 16 entries
  • Git hooks: pre-commit
  • Claude integration: installed

? Remove all components? [y/N] y

  ✓ Claude integration removed
  ✓ Git hooks removed
  ✓ Storage refs removed
  ✓ Config cleaned

Mytool removed. Your git history is unchanged.
```

**Design principles:**
- Show what will be removed before doing it
- Offer `--keep-data` to remove tooling but preserve data
- Reverse everything in the right order (integrations before storage)
- Idempotent (safe to run on already-uninstalled repo)

---

## 8. Checklist

Use this checklist when designing agent-oriented CLIs:

### Essential (Core DX)
- [ ] `--json` flag on every command
- [ ] Structured error output with codes
- [ ] `prime` or equivalent context injection command
- [ ] `pending` or equivalent "what needs attention" command
- [ ] Sensible defaults—works without config
- [ ] `--dry-run` on write operations

### Essential (Integration)
- [ ] `doctor` command for health checking
- [ ] `hooks install/uninstall` for git integration
- [ ] `setup <editor>` for editor-specific hooks
- [ ] `init` that orchestrates full setup
- [ ] `uninstall` that reverses all setup

### Recommended
- [ ] `--batch` mode for multi-item operations
- [ ] `--oneline` / `--ids-only` for compact output
- [ ] `onboard` command for documentation snippets
- [ ] Suggested next commands in output
- [ ] Recoverable error messages with hints
- [ ] `doctor --fix` for auto-remediation

### Bonus
- [ ] Stdin support for batch input
- [ ] Exit code conventions documented
- [ ] `hooks install --chain` to preserve existing hooks
- [ ] `setup --check` and `setup --remove` flags
- [ ] Multiple editor integrations (claude, cursor, gemini, etc.)

---

## 9. Examples in the Wild

### Beads (Issue Tracking)

**Core DX:**
- `bd prime` — Context injection
- `bd ready` — Clear next action
- `bd close <id>` — Minimal ceremony
- `--json` on all commands

**Integration stack:**
- `bd doctor` — Comprehensive health check with `--fix`
- `bd hooks install` — Git hooks with `--chain` support
- `bd setup claude` — Claude Code integration with `--check`/`--remove`
- `bd onboard` — Minimal AGENTS.md snippet generator
- `bd init` — Orchestrated setup with interactive prompts

### Timbers (Development Ledger)

**Core DX:**
- `timbers prime` — Context injection with pending count
- `timbers pending` — Clear next action
- `timbers log "what" --why "why" --how "how"` — Single command capture
- `timbers export --json | claude "..."` — Unix composability
- `--batch` mode for efficiency

**Integration stack:**
- `timbers doctor` — Health check for notes, hooks, integrations
- `timbers hooks install` — Pre-commit warning for undocumented work
- `timbers setup claude` — Session-start prime injection
- `timbers onboard` — Minimal CLAUDE.md snippet
- `timbers init` — Full setup with optional Claude integration
- `timbers uninstall` — Clean removal of all components

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

### Documentation-Only Integration
Relying on CLAUDE.md or AGENTS.md instructions without automatic injection. Agents don't reliably read and follow documentation—they jump into tasks. If your only integration is "add this to your CLAUDE.md", agents will skip the workflow steps. Wire integration into hooks so context flows automatically.

### Blocking Git Hooks
Pre-commit hooks that fail and block commits. Developers disable blocking hooks. Use warnings instead—inform without obstructing. The goal is awareness, not enforcement.
