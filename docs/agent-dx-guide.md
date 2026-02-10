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

### 1.3.1 Verbose Context for Design History

**Context injection should have a verbose mode that gives agents decision history, not just status.**

Standard prime output shows *what* happened. Verbose mode shows *why* decisions were made, enabling agents to make informed choices that are consistent with prior design decisions.

```bash
$ mytool prime --verbose
# ...standard output...

Recent Work
-----------
  tb_2026-02-09_abc123  Added tag-based query filtering
    Why: OR semantics chosen over AND because users filter by any-of, not all-of
    How: Extended query parser with --tag flag accepting comma-separated values
```

**Why this matters:**
- Without design history, agents repeat mistakes or contradict prior decisions
- Why/how context is the highest-signal data for continuity across sessions
- Default mode stays compact (token-efficient); verbose is opt-in when agents need design context

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

### 3.2 Stderr Routing for Pipe Ergonomics

**Separate data from diagnostics when piped.**

Agents frequently pipe CLI output directly to LLMs. Any diagnostic noise — warnings, progress messages, status hints — on stdout breaks downstream parsing. Route all non-data output to stderr.

```go
// Create printer with stderr for diagnostics
printer := output.NewPrinter(cmd.OutOrStdout(), isJSONMode(cmd), output.IsTTY(cmd.OutOrStdout())).
    WithStderr(cmd.ErrOrStderr())

// Errors and warnings automatically route to stderr in human mode
printer.Error(err)   // → stderr (human), stdout (JSON protocol)
printer.Warn("...")  // → stderr (human), stdout (JSON protocol)

// Explicit stderr for status hints when piped
if !printer.IsTTY() {
    printer.Stderr("mytool: rendered %q with %d entries\n", templateName, len(entries))
}
```

**The pattern:**
1. Data (entries, rendered output, query results) always goes to stdout.
2. Errors and warnings go to stderr in human mode, stdout in JSON mode (structured protocol).
3. Status hints (e.g., "rendered template with N entries") go to stderr only when piped.
4. In JSON mode, all structured output goes to stdout (the JSON envelope is the protocol).

**Why this matters:**
- `mytool draft changelog --last 10 | claude "Summarize"` must not include "Warning: 3 entries lack tags" in the LLM prompt
- Agents parsing JSON output get clean data without interleaved diagnostics
- Users still see warnings — they just arrive on the right file descriptor

### 3.3 Output Destination Control

**Separate stdout behavior from file output.**

```bash
# To stdout (for piping)
mytool export --json

# To files (for persistence)
mytool export --json --out ./exports/
```

Don't conflate these — agents need both patterns.

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

### 5.3 Input Coaching

**Don't just tell agents what flags exist — coach them on what good input looks like.**

Agents are surprisingly bad at generating high-quality input without examples. If a flag expects nuanced content (like a "why" field), include concrete BAD/GOOD examples in context injection output.

```
# Writing Good Why Fields
The --why flag captures *design decisions*, not feature descriptions.

BAD (feature description):
  --why "Users needed tag filtering for queries"
  --why "Added amend command for modifying entries"

GOOD (design decision):
  --why "OR semantics chosen over AND because users filter by any-of, not all-of"
  --why "Partial updates via amend avoid re-entering unchanged fields"

Ask yourself: why THIS approach over alternatives? What trade-off did you make?
```

**Where to put coaching:**
- In `prime` output (seen at every session start)
- In `--help` for the specific command
- **Not** in static docs that agents won't read

**Why this matters:**
- Without coaching, agents produce shallow input ("Added feature X" instead of "Chose X over Y because Z")
- The difference between good and bad input determines whether the captured data has long-term value
- Coaching in `prime` output works because it's injected automatically — agents can't skip it

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

**Measured impact:** In dogfood testing, documentation-only integration achieved **0% agent adoption** of the workflow. Adding session-start hook injection achieved **100% adoption** — same agents, same tool, same docs. The only difference was automatic `prime` injection at session start. This isn't a small improvement; it's the difference between a tool that works and one that doesn't.

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
  ok  Storage initialized
  ok  Remote configured
  !!  Version 0.1.5 (latest: 0.2.0)
     -> Update: curl -fsSL .../install.sh | bash

CONFIG
  ok  Config Dir ~/.config/mytool
  ok  Env Files local (.env.local) | keys: anthropic
  ok  Custom Templates none (7 built-in available)

WORKFLOW
  !!  Pending 3 undocumented items
     -> Run 'mytool pending' to review

INTEGRATION
  ok  Git hooks installed
  !!  Claude integration not configured
     -> Run 'mytool setup claude' to install

ok 5 passed  !! 2 warnings  XX 0 failed
```

**Design principles:**
- Categorize checks (Core, Config, Workflow, Integration)
- Show pass/warn/fail with visual indicators
- Provide specific fix commands for each issue
- Support `--fix` to auto-remediate where possible
- Support `--json` for programmatic checking

**Check the full operational environment, not just basics.** Beyond storage and hooks, doctor should verify:
- **Config directory** resolution (does it exist? which path was resolved?)
- **Env files** (which are loaded? which API keys are available?)
- **Templates** (custom project-local or global templates found?)
- **Version staleness** (is the installed binary current with the latest release?)

These checks catch real operational issues: missing API keys that cause `--model` failures, stale binaries that lack new features, config directories that were never created.

### 7.4 Git Hooks: `hooks install/uninstall`

**Git hooks provide passive enforcement** without blocking developer workflow.

```bash
$ mytool hooks install
ok Installed pre-commit hook
  -> Warns about undocumented work (non-blocking)
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
ok Claude Code hook installed
  -> Will inject 'mytool prime' at session start
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

  ok Storage created
  ok Remote configured
  ok Git hooks installed (pre-commit)

Optional integrations:
  ? Install Claude Code integration? [Y/n] y
  ok Claude hook installed

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
  - Storage: 16 entries
  - Git hooks: pre-commit
  - Claude integration: installed

? Remove all components? [y/N] y

  ok Claude integration removed
  ok Git hooks removed
  ok Storage refs removed
  ok Config cleaned

Mytool removed. Your git history is unchanged.
```

**Design principles:**
- Show what will be removed before doing it
- Offer `--keep-data` to remove tooling but preserve data
- Reverse everything in the right order (integrations before storage)
- Idempotent (safe to run on already-uninstalled repo)

---

## 8. Configuration

### 8.1 Config Directory Resolution

**Centralize config directory resolution with an explicit override hierarchy.**

```
1. $MYTOOL_CONFIG_HOME       (explicit override — CI, testing, custom setups)
2. $XDG_CONFIG_HOME/mytool   (XDG standard on any platform)
3. %AppData%/mytool           (Windows native)
4. ~/.config/mytool            (macOS/Linux default)
```

```go
func Dir() string {
    if dir := os.Getenv("MYTOOL_CONFIG_HOME"); dir != "" {
        return dir
    }
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "mytool")
    }
    if runtime.GOOS == "windows" {
        if appData := os.Getenv("APPDATA"); appData != "" {
            return filepath.Join(appData, "mytool")
        }
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "mytool")
}
```

**Guidelines:**
- Always provide an env var override (`MYTOOL_CONFIG_HOME`) for CI and testing
- Respect XDG on all platforms (some macOS/Windows users set it)
- Never hardcode `~/.mytool` — use standard config locations
- `doctor` should report which directory was resolved and whether it exists

### 8.2 Env File Loading Chain

**Layer env files for API key management without committing secrets.**

```
Resolution order (first value wins per variable):
  1. Environment variables (always take precedence)
  2. .env.local          (per-repo override, gitignored)
  3. .env                (per-repo defaults)
  4. ~/.config/mytool/env (global fallback — set once, works everywhere)
```

```go
func loadEnvFiles() {
    _ = envfile.Load(".env.local")
    _ = envfile.Load(".env")
    if dir := config.Dir(); dir != "" {
        _ = envfile.Load(filepath.Join(dir, "env"))
    }
}
```

**Guidelines:**
- Environment variables always win (no surprises in CI)
- `.env.local` is gitignored — safe for per-developer API keys
- Global config fallback means developers set keys once, not per-repo
- Silently skip missing files (all env files are optional)
- `doctor` should report which env files are loaded and which API keys are available

### 8.3 Template-Based Document Generation

**Use templates instead of building separate static commands for each output type.**

When your tool generates documents from structured data (changelogs, release notes, reports), use a template system rather than a dedicated command per document type.

```bash
# Template-based (one command, many outputs):
mytool draft changelog --last 10
mytool draft release-notes --since 7d
mytool draft devblog --since 7d --model opus
mytool draft decision-log --last 20
mytool draft --list

# vs. Static commands (one command per type — doesn't scale):
mytool changelog --last 10
mytool release-notes --since 7d
```

**Why templates win:**
- Adding a new document type is adding a template file, not writing a new command
- Users can override built-in templates (project-local > global > built-in)
- Built-in templates can be tightly prompt-engineered for quality
- The LLM integration (`--model`) works identically across all templates
- `--show` lets users inspect templates before rendering

**Template resolution order:**
```
1. .mytool/templates/    (project-local overrides)
2. ~/.config/mytool/templates/  (global custom templates)
3. Built-in templates    (shipped with binary)
```

**Anti-pattern learned the hard way:** We built a standalone `changelog` command, then realized `draft changelog` was strictly better — same output quality, but composable with `--model`, `--append`, pipe-to-LLM workflows, and custom template overrides. The standalone command was dropped.

---

## 9. Checklist

Use this checklist when designing agent-oriented CLIs:

### Essential (Core DX)
- [ ] `--json` flag on every command
- [ ] Structured error output with codes
- [ ] `prime` or equivalent context injection command
- [ ] `pending` or equivalent "what needs attention" command
- [ ] Sensible defaults — works without config
- [ ] `--dry-run` on write operations

### Essential (Integration)
- [ ] `doctor` command for health checking
- [ ] `hooks install/uninstall` for git integration
- [ ] `setup <editor>` for editor-specific hooks
- [ ] `init` that orchestrates full setup
- [ ] `uninstall` that reverses all setup

### Essential (Pipe Ergonomics)
- [ ] Errors and warnings route to stderr in human mode
- [ ] Status hints to stderr when output is piped
- [ ] JSON mode keeps all structured output on stdout
- [ ] No diagnostic noise on stdout when piped

### Recommended
- [ ] `--batch` mode for multi-item operations
- [ ] `--oneline` / `--ids-only` for compact output
- [ ] `onboard` command for documentation snippets
- [ ] `prime --verbose` for design decision history
- [ ] Input coaching (BAD/GOOD examples) in context injection
- [ ] Suggested next commands in output
- [ ] Recoverable error messages with hints
- [ ] `doctor --fix` for auto-remediation
- [ ] Config directory with env var override
- [ ] Env file loading chain for API key management
- [ ] Template-based document generation

### Bonus
- [ ] Stdin support for batch input
- [ ] Exit code conventions documented
- [ ] `hooks install --chain` to preserve existing hooks
- [ ] `setup --check` and `setup --remove` flags
- [ ] Multiple editor integrations (claude, cursor, gemini, etc.)
- [ ] `doctor` checks config dir, env files, API keys, templates, version staleness
- [ ] Template resolution: project-local > global > built-in

---

## 10. Examples in the Wild

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
- `timbers prime` — Context injection with pending count and workflow coaching
- `timbers prime --verbose` — Design decision history (why/how in recent entries)
- `timbers pending` — Clear next action
- `timbers log "what" --why "why" --how "how"` — Single command capture
- `timbers draft <template>` — Template-based document generation (changelog, release-notes, devblog, decision-log, adr, exec-summary, standup)
- `timbers draft --list` — Discover available templates
- `timbers draft release-notes --last 10 --model opus` — Generate with built-in LLM
- `timbers export --json | claude "..."` — Unix composability
- `--batch` mode for efficiency

**Integration stack:**
- `timbers doctor` — Health check across CORE, CONFIG, WORKFLOW, INTEGRATION
  - CONFIG checks: config dir, env files, API keys, custom templates, version staleness
- `timbers hooks install` — Pre-commit warning for undocumented work
- `timbers setup claude` — Session-start prime injection
- `timbers onboard` — Minimal CLAUDE.md/AGENTS.md snippet
- `timbers init` — Full setup with optional Claude integration
- `timbers uninstall` — Clean removal of all components

**Pipe ergonomics:**
- Errors/warnings route to stderr in human mode
- Status hints to stderr when piped (`timbers: rendered "changelog" with 10 entries`)
- JSON mode: all structured output on stdout

**Configuration:**
- Config dir: `$TIMBERS_CONFIG_HOME` > `$XDG_CONFIG_HOME/timbers` > `%AppData%/timbers` > `~/.config/timbers`
- Env chain: `.env.local` > `.env` > `~/.config/timbers/env` (env vars always win)
- Templates: `.timbers/templates/` > `~/.config/timbers/templates/` > built-in

---

## 11. Anti-Patterns

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
Relying on CLAUDE.md or AGENTS.md instructions without automatic injection. Agents don't reliably read and follow documentation — they jump into tasks. If your only integration is "add this to your CLAUDE.md", agents will skip the workflow steps. Wire integration into hooks so context flows automatically. **Measured: 0% adoption without injection, 100% with injection.** This isn't a minor optimization — it's the make-or-break pattern.

### Blocking Git Hooks
Pre-commit hooks that fail and block commits. Developers disable blocking hooks. Use warnings instead — inform without obstructing. The goal is awareness, not enforcement.

### One Command Per Output Type
Building a separate command for each document type (changelog, release-notes, etc.) instead of using a template system. This creates maintenance burden, inconsistent flags across commands, and limits user extensibility. Use `draft <template>` with a resolution chain (project > global > built-in) instead.

### Stdout Pollution When Piped
Mixing diagnostic output (warnings, progress, hints) with data on stdout. When agents pipe output to LLMs or other tools, diagnostic noise breaks parsing. Route all non-data output to stderr.
