# Design Decisions

Conscious architectural choices made during code review (2026-01-17). These patterns were flagged by reviewers but intentionally retained.

## Accepted Patterns

### Global `jsonFlag` Variable

**Location**: `cmd/timbers/main.go:18`

**Pattern**: A package-level `var jsonFlag bool` accessed by all command files.

**Why retained**: This is the standard Cobra persistent flag pattern. While it creates implicit coupling between main.go and command files, refactoring (e.g., passing through context or command constructors) would add complexity for minimal benefit. The flag is read-only after initialization and the pattern is well-understood by Go CLI developers.

### Repository Check Duplication

**Location**: `pending.go:69-73`, `show.go:60-64`, `status.go:45-49`, `query.go:64-68`

**Pattern**: Each command checks `!git.IsRepo()` with similar error handling.

**Why retained**: Commands may eventually want different error messages or recovery hints. The duplication is 4-5 lines per command and keeps each command self-contained. Extracting to a helper would save ~15 lines total but reduce clarity.

### No `--ids-only` on `show` Command

**Location**: `cmd/timbers/show.go`

**Pattern**: The `show` command displays full entry details without an `--ids-only` option.

**Why retained**: `show` is for single-entry inspection, not listing. The `query --oneline` flag serves the "list IDs" use case. Adding `--ids-only` to `show` would be semantically odd since you already specify which entry to show.

### No `--count` on `query` Command

**Location**: `cmd/timbers/query.go`

**Pattern**: Query returns full entries rather than having a count-only mode.

**Why retained**: Agents can use `query --json | jq length` for counting. Adding `--count` would duplicate functionality available through composition. May reconsider if profiling shows performance issues with large entry sets.

### No Confirmation Prompt in Auto Mode

**Location**: `cmd/timbers/log_parse.go`

**Pattern**: `--auto` extracts what/why/how from commits non-interactively. The `--yes` flag exists for API compatibility but is currently a no-op.

**Why retained**: The spec description says `--auto — Extract what/why/how from commit messages (non-interactive)`. While a separate spec section mentioned prompting for confirmation, the CLI is designed for agent-driven pipelines where interactive prompts are undesirable. The `--yes` flag ensures forward compatibility if confirmation is added later. For now, `--auto` assumes intent to proceed.

### JSON Output Structure Variation

**Location**: Various commands in `cmd/timbers/`

**Pattern**: Write operations return `{"status": "ok", ...}` wrapper, read operations return raw data.

**Why retained**: This follows REST-like semantics:
- **Write commands** (log, notes init/push/fetch, export to dir) return `{"status": "ok", ...}` to confirm the action completed.
- **Read commands** (show, query, prime) return raw data without wrapper since the presence of data implies success.

This pattern is agent-friendly: status-bearing responses confirm mutations occurred, while read responses can be processed directly without unwrapping. Standardizing all to wrappers would add boilerplate; standardizing all to raw would lose mutation confirmation.

### Removed `--replace` Flag Reference

**Location**: `internal/ledger/storage.go:152`

**Pattern**: Error message for duplicate entries no longer suggests `--replace` flag.

**Why changed**: The spec mentioned a `--replace` flag but it was never implemented. Rather than add another flag, the simpler approach is to rely on `--dry-run` to preview what would happen before committing. Users who need to overwrite can delete the note manually with git.

---

*Document created during M1-complete review. Update when revisiting these decisions.*
