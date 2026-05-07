# PROJECT INSTRUCTIONS

**Timbers** — a Git-native development ledger CLI that captures *what/why/how* as structured records.

## Prime Directive — Gall's Law

> *"A complex system that works is invariably found to have evolved from a simple system that worked. A complex system designed from scratch never works and cannot be patched up to make it work. You have to start over with a working simple system."* — John Gall

**Always grow complexity from a simple system that already works.**

- **Modularity**: Simple parts, clean interfaces
- **Clarity**: Clarity over cleverness
- **Composition**: Design parts to connect with other parts
- **Simplicity**: Add complexity only where you must

In practice:
- Prefer minimal working slices over grand designs
- Avoid speculative architecture and premature abstraction
- Make only small, verifiable changes
- Push back when requests ignore this: *Begin → Learn → Succeed → then add complexity*

---

## Session Orientation

Before starting any work, verify your context:

1. **Branch:** `git branch --show-current` — confirm you're on the expected branch
2. **Worktree:** `git worktree list` — are you in a worktree or the main repo?
3. **Confirm with user:** "I'm on branch X in [worktree/main]. Is this where you want me working?"
4. **Check beads:** `bd ready` — what work is available?

Skipping orientation risks working on the wrong branch, which wastes entire sessions silently.

---

## Session Recovery

Claude Code carries native session continuity (rewind, compact, resume). For cross-session state, beads is the source of truth: `bd ready` and `bd show <id>` reconstruct what's in flight. If the user pastes any prior snapshot as their first message, treat it as starting context and confirm: "Recovered session. [brief summary of where we left off]"

---

## Settled Decisions

Items marked SETTLED should not be revisited unless the user explicitly asks.

<!-- Add decisions as they're made:
| Decision | Date | Rationale | Status |
|----------|------|-----------|--------|
| Example: Auth uses JWT | 2025-01-15 | See docs/plans/auth.md | SETTLED |
-->

---

## Software Engineering Practices

### Commit hygiene and cadence

- **One logical change per commit.** A commit should compile, pass gates, and be revertable in isolation. If the diff spans unrelated concerns, split it.
- **Right-sized commits.** Roughly one bead = one commit for XS/S work; M+ work may produce a few related commits. Don't batch hours of work into one mega-commit; don't fragment a single coherent change across five.
- **Conventional Commits.** Use `type(scope): subject` — `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`, `perf:`, `style:`, `build:`, `ci:`. Imperative subject, ≤72 chars. Body explains *why* when the diff doesn't make it obvious.
- **Reference beads in the body**, not the subject (`Closes bd-abc-123` / `Refs bd-abc-124`).

### Work-item trailer

In addition to bead refs, this repo uses a `Work-item` trailer to link commits to features or issues across systems:

```
feat: add feature description

Detailed explanation of the change.

Work-item: beads:timbers-psc.4
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

**Trailer format**: `Work-item: <system>:<id>`

Examples:
- `Work-item: beads:timbers-q1o.3` — Beads issue tracker
- `Work-item: jira:PROJ-123` — Jira ticket
- `Work-item: gh:owner/repo#42` — GitHub issue

**How catchup uses it**: `timbers log --batch` groups pending commits by Work-item trailer instead of by day, so all commits for a single work item become one ledger entry. Commits without trailers fall into an "untracked" group when other commits have trailers, or are grouped by day if no trailers exist.

### Quality gates per commit

If pre-commit hooks already run lint/typecheck/test, trust them and don't re-run manually. If hooks are missing, partial, or skipped, run `just check` yourself before committing. **Pre-existing failures are still our problem** — "already broken" is not an excuse, and is usually our prior miss.

When file/function/complexity limits trigger, **extract logical sections into well-named companion files** rather than compressing code to fit. Don't combine statements onto one line, strip comments, or shorten names to satisfy a metric.

### just as the primary DX interface

`just` is the canonical command runner for both humans and agents in this repo. **Always use `just` commands; do not run `go` commands directly.** Treat the justfile as the contract:

```bash
just setup         # First-time setup (run once)
just check         # REQUIRED: lint + test before commit
just fix           # Auto-fix lint issues
just run           # Run CLI: just run log "..." --why "..." --how "..."
just build         # Build binary to bin/timbers
just test          # Run tests only
just lint          # Run linter only
just build-local   # Build bin/timbers with git version info (for dev testing)
```

- All common workflows belong as `just` recipes — agents will read them
- New repeatable commands → add a recipe rather than documenting raw shell
- When a recipe changes, the change is the documentation

If a workflow only exists as a shell snippet in a doc, it's not really a workflow yet — promote it to `just`.

### Output compression: rtk and tokf

When available, use `rtk` (Rust Token Killer) and `tokf` (per-project filter) to compress noisy command output before it reaches the agent's context. Both are transparent: agents call commands normally and the wrappers handle compression.

- **rtk** is the global baseline (npm/git/build output → 60–90% token reduction)
- **tokf** is per-project for repo-specific noise patterns
- Available? Use them. Not installed? Don't block on it.

See `dm-work:output-compression` for setup.

### Pause-for-review cadence

After every **M or larger** feature lands (M+ bead closed, code merged), pause and run a review pass before starting the next M+ chunk. Either:

- `/dm-work:review` for parallel arch/code/security reviewers, or
- A generic subagent review of the recent diff with explicit scope ("read ONLY the diff and the OWN files"), plus an optional Codex second-opinion via the codex plugin for cross-model coverage

The goal is to catch drift, accumulated debt, and integration gaps before they compound. XS/S work doesn't need this; M+ does.

---

## Beads & Timbers

This repo uses **beads** (`bd`) for task tracking and **timbers** (this project, dogfooded) for commit-reasoning logs. Both tools inject their own usage instructions into this file:

- `bd setup claude` adds the `<!-- BEGIN BEADS INTEGRATION -->` block below
- `timbers onboard --target agents` appends timbers usage guidance

**Follow the instructions those tools inject** — they own their respective domains. Don't duplicate that content elsewhere; let `bd setup claude` and `timbers onboard` be the source of truth so they stay current as the tools evolve.

Repo-wide conventions worth stating once (not covered by injections):

- **Bead-first workflow:** when ad hoc work appears (bug, feature, task) without an existing bead, create one before implementing. Every code change should trace back to a bead.
- **Bead detail discipline:** every bead has an imperative title, a description that lets a cold session start work, explicit dependencies, and a complexity estimate (xs/s/m/l/xl). M+ beads link to a plan doc and call out architectural decisions.
- **Sync model:** beads 1.0+ uses embedded Dolt + git+JSONL transport — `.beads/issues.jsonl` is the source of truth (committed); `.beads/dolt/` is a local cache (gitignored); sync runs through pre-commit / post-merge hooks. `bd dolt push/pull` are safe to run in either configuration: with a Dolt remote they sync against it; without one they exit 0 with an informational message (per fix #3194 in 1.0.3).
- **Worktrees:** always use `bd worktree create`, never `git worktree add`. The `bd` version sets up `.beads/redirect` so worktrees share the main repo's Dolt database.
- **Destructive commands — DO NOT USE**: `bd init --force`, `bd admin reset --force`.

---

## Role

**You are an orchestrator, not an implementer.**

At session start, activate one of these based on your coordination needs:

| Situation | Skill | Mechanism |
|-----------|-------|-----------|
| Standard delegation | `dm-work:orchestrator` | Task() subagents |
| Complex multi-agent work | `dm-team:lead` | [Agent Teams](https://code.claude.com/docs/en/agent-teams) |

Both establish delegation thresholds, quality gates, and file ownership boundaries. See the "Teams vs Subagents vs Direct" table inside `dm-team:lead` for the decision framework. Agent Teams requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` in settings.json.

If you are a **subagent** (delegated by an orchestrator), activate `dm-work:subagent`.
If you are a **teammate** (in an Agent Teams configuration), activate `dm-team:teammate`.

### Worktrees

When creating worktrees for isolated feature work, place them under `.worktrees/` in the repo root. Ensure `.worktrees/` is in `.gitignore` before creating:

```bash
git check-ignore -q .worktrees/ || echo '.worktrees/' >> .gitignore
```

See `dm-work:worktrees` for the full workflow (create → quality gates → user sign-off → merge).

---

## Skills

Apply these skills proactively during development:

| Skill | When to Apply |
|-------|---------------|
| `dm-lang:go-pro` | All Go code — patterns, error handling, package design |
| `dm-arch:solid-architecture` | Module boundaries, interfaces, composition |
| `dm-work:tdd` | Before implementing any feature — test first |
| `dm-work:debugging` | When encountering failures — investigate before fixing |

---

## Memory Layout

| File | Purpose | Committed? |
|------|---------|------------|
| `AGENTS.md` (+ `CLAUDE.md` symlink) | Team-shared project instructions | Yes |
| `CLAUDE.local.md` | Personal project prefs (sandbox URLs, local paths) | No (auto-gitignored) |
| `.claude/rules/*.md` | Modular topic rules, optionally path-scoped | Yes |
| `.claudeignore` | Patterns for CC to skip (build artifacts, large files) | Yes |

For personal prefs that should work across worktrees, use imports: `@~/.claude/my-project-instructions.md` in your `CLAUDE.local.md`.

---

# Timbers Project Reference

The remainder of this file is timbers-specific reference material — tech stack, architecture, schema, and command surface.

## Tech Stack

- **Language**: Go 1.25+
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **Styling**: [Charmbracelet](https://charm.sh/) ecosystem
  - `lipgloss` — Terminal styling and colors
  - `log` — Structured logging
- **Git Operations**: `os/exec` shelling to `git`
- **Schema**: Struct tags + validation (no external schema deps)

## Architecture

```
cmd/timbers/
    main.go           # Entry point, root command
    log.go            # timbers log
    pending.go        # timbers pending
    prime.go          # timbers prime
    status.go         # timbers status
    show.go           # timbers show
    query.go          # timbers query
    export.go         # timbers export
    doctor.go         # timbers doctor
    onboard.go        # timbers onboard

internal/
    git/
        git.go        # Git operations via exec
    ledger/
        entry.go      # Entry struct, ID generation, validation
        storage.go    # Read/write entries, pending commit detection
    export/
        json.go       # JSON export formatting
        markdown.go   # Markdown export formatting
    draft/
        template.go   # Template loading and resolution
        render.go     # Template rendering with entry data
        builtin.go    # Embedded built-in templates
    output/
        human.go      # Human-readable output formatting
        json.go       # JSON output formatting
```

## Agent DX Requirements

Every command MUST support:
- `--json` flag for structured output
- Structured error JSON: `{"error": "message", "code": N}`
- Recovery hints in error messages

Write operations MUST support:
- `--dry-run` flag

See `docs/agent-dx-guide.md` for the full pattern language.

## Output Handling

```go
// Detect if output is piped (disable colors, use JSON-friendly format)
if !term.IsTerminal(os.Stdout.Fd()) {
    // piped - no colors, machine-readable
}

// --json flag takes precedence
if jsonFlag {
    json.NewEncoder(os.Stdout).Encode(result)
} else {
    // human-readable with lipgloss styling
}
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | User error (bad args, missing fields, not found) |
| 2 | System error (git failed, I/O error) |
| 3 | Conflict (entry exists, state mismatch) |

## Generated Files — Do Not Edit Manually

| File | Generated by | Preview |
|------|-------------|---------|
| `CHANGELOG.md` | `just release 0.x.x` | `just changelog` |

These files are produced by `timbers draft` piped through `claude -p --model opus`.
**Never edit them by hand** — the next release overwrites manual changes.

To preview what a release changelog would look like: `just changelog 0.x.x`

## Conventions

- All errors handled explicitly (no `_ = err`)
- Exported identifiers have doc comments
- Table-driven tests for multiple cases
- Test files alongside source: `foo.go` / `foo_test.go`
- Integration tests in `internal/integration/`
- No global state — inject dependencies

## Testing Strategy

**Unit tests**: Pure functions, struct validation, formatting
```go
func TestEntryID(t *testing.T) {
    // Table-driven: same inputs = same outputs
}
```

**Integration tests**: Temp git repos, full workflows
```go
func TestLogPendingCycle(t *testing.T) {
    // Create temp repo, run log, verify pending clears
}
```

## Schema (timbers.devlog/v1)

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",
  "workset": {
    "anchor_commit": "8f2c1a9d...",
    "commits": ["8f2c1a9d...", "c11d2a...", "a4e9bd..."],
    "range": "c11d2a..8f2c1a",
    "diffstat": {"files": 3, "insertions": 45, "deletions": 12}
  },
  "summary": {
    "what": "Fixed authentication bypass vulnerability",
    "why": "User input wasn't being sanitized before JWT validation",
    "how": "Added input validation middleware before auth handler"
  },
  "notes": "Considered rate limiting vs input validation. Validation catches root cause.",
  "tags": ["security", "auth"],
  "work_items": [{"system": "beads", "id": "bd-a1b2c3"}]
}
```

**Entry ID format**: `tb_<ISO8601-timestamp>_<anchor-short-sha>`

## Key Commands

| Command | Purpose |
|---------|---------|
| `timbers init` | Full setup (.timbers/, hooks, Claude integration) |
| `timbers log` | Record work with what/why/how (optional --notes for deliberation) |
| `timbers pending` | Show undocumented commits |
| `timbers prime` | Context injection for session start |
| `timbers status` | Repo/ledger state |
| `timbers show` | Display single entry |
| `timbers query` | Search entries |
| `timbers export` | Export for pipelines |
| `timbers doctor` | Health check and diagnostics |
| `timbers onboard` | Generate CLAUDE.md snippet |

## Development Workflow

**Session start:**
```bash
timbers prime        # Get context and workflow instructions
```

**After completing work:**
```bash
timbers pending      # Check for undocumented commits
timbers log "what" --why "why" --how "how"
git push             # Entries are files, sync via regular push
```

**Before releasing a new version:**
```bash
git pull --rebase    # Sync with remote BEFORE tagging (avoids tag/rebase conflicts)
just build-local     # Build bin/timbers with version info
bin/timbers --version  # Verify the build
just release 0.x.x   # Tag and push (triggers GitHub Actions)
```

---

## Skills & Tools

You have MCPs, skills, and bash tools. Use them. Ensure subagents and teammates know about relevant skills when delegating.

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Dolt-powered version control with native sync
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update <id> --claim --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task atomically**: `bd update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Sync

Beads auto-syncs via `.beads/issues.jsonl` (git-tracked):

- Every `bd` mutation auto-flushes to `.beads/issues.jsonl`
- After `git pull`, the next `bd` command auto-imports changes
- Pre-commit hook auto-stages the file — no manual export/import needed

### Persistent Knowledge

```bash
bd remember "insight"    # Save a factual learning for future sessions
bd memories <keyword>    # Search saved memories by keyword
```

Use `bd remember` to persist factual learnings (library gotchas, API quirks, env-specific behaviors) across sessions. Search with `bd memories <keyword>` when you encounter a problem that might have been solved before — especially at session start or when debugging unfamiliar behavior.

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Session retro** - Harvest factual learnings into `bd remember`. Ask: "What did I learn today that I'd have to rediscover from scratch next time?"
5. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
6. **Clean up** - Clear stashes, prune remote branches
7. **Verify** - All changes committed AND pushed
8. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- END BEADS INTEGRATION -->
