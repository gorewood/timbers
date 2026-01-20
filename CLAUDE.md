# Timbers - Development Ledger CLI

A Git-native development ledger that captures *what/why/how* as structured records.

## Development Commands

**ALWAYS use `just` commands for development tasks.** Do not run `go` commands directly.

```bash
just setup         # First-time setup (run once)
just check         # REQUIRED: Run before commit (lint + test)
just fix           # Auto-fix lint issues
just run           # Run CLI: just run log "..." --why "..." --how "..."
just build         # Build binary to bin/timbers
just test          # Run tests only
just lint          # Run linter only
just install-local # Install to GOPATH from source (for testing before release)
```

**Quality Gate**: `just check` must pass before any commit. No exceptions.

## Skills

Apply these skills proactively during development:

| Skill | When to Apply |
|-------|---------------|
| `dm-lang:go-pro` | All Go code - patterns, error handling, package design |
| `dm-arch:solid-architecture` | Module boundaries, interfaces, composition |
| `dm-work:tdd` | Before implementing any feature - test first |
| `dm-work:debugging` | When encountering failures - investigate before fixing |

## Tech Stack

- **Language**: Go 1.25+
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **Styling**: [Charmbracelet](https://charm.sh/) ecosystem
  - `lipgloss` - Terminal styling and colors
  - `log` - Structured logging
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
    skill.go          # timbers skill
    notes.go          # timbers notes (subcommands)

internal/
    git/
        git.go        # Git operations via exec
        notes.go      # Notes-specific operations
    ledger/
        entry.go      # Entry struct, ID generation, validation
        storage.go    # Read/write entries, pending commit detection
    export/
        json.go       # JSON export formatting
        markdown.go   # Markdown export formatting
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

## Commit Message Format

Use the `Work-item` trailer to link commits to features or issues:

```
feat: add feature description

Detailed explanation of the change.

Work-item: beads:timbers-psc.4
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

**Trailer format**: `Work-item: <system>:<id>`

Examples:
- `Work-item: beads:timbers-q1o.3` - Beads issue tracker
- `Work-item: jira:PROJ-123` - Jira ticket
- `Work-item: gh:owner/repo#42` - GitHub issue

**How catchup uses it**: `timbers log --batch` groups pending commits by Work-item trailer instead of by day. This enables feature-based documentation where all commits for a single work item become one ledger entry, regardless of when they were made.

Commits without trailers fall into an "untracked" group when other commits have trailers, or are grouped by day if no trailers exist.

## Conventions

- All errors handled explicitly (no `_ = err`)
- Exported identifiers have doc comments
- Table-driven tests for multiple cases
- Test files alongside source: `foo.go` / `foo_test.go`
- Integration tests in `internal/integration/`
- No global state - inject dependencies

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
  "tags": ["security", "auth"],
  "work_items": [{"system": "beads", "id": "bd-a1b2c3"}]
}
```

**Entry ID format**: `tb_<ISO8601-timestamp>_<anchor-short-sha>`

## Key Commands

| Command | Purpose |
|---------|---------|
| `timbers log` | Record work with what/why/how |
| `timbers pending` | Show undocumented commits |
| `timbers prime` | Context injection for session start |
| `timbers status` | Repo/notes state |
| `timbers show` | Display single entry |
| `timbers query` | Search entries |
| `timbers export` | Export for pipelines |
| `timbers skill` | Emit skill content |
| `timbers notes` | Notes management (init/push/fetch) |

## Development Workflow

**Session start:**
```bash
timbers prime        # Get context and workflow instructions
```

**After completing work:**
```bash
timbers pending      # Check for undocumented commits
timbers log "what" --why "why" --how "how"
timbers notes push   # Sync to remote
```

**Before releasing a new version:**
```bash
just install-local   # Install from source with version info
# Test the local build
just release 0.x.x   # Tag and push (triggers GitHub Actions)
```
