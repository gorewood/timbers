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

### Separate Commits for Ledger Entries

**Location**: `internal/ledger/filestorage.go` (`DefaultGitCommit`)

**Pattern**: Each `timbers log` creates its own git commit (`timbers: document <id>`) rather than folding the entry into the code commit it documents.

**Why retained**: This design was evaluated against three alternatives — co-committing entries into code commits (via `--amend`), storing entries on a side branch, and deferring commits to a session-end batch. All were rejected after adversarial review. Separate commits are retained because:

1. **Pending detection depends on it.** `filterLedgerOnlyCommits` identifies entry commits by checking whether *every* file in the commit is under `.timbers/`. This is what prevents entry commits from showing as "undocumented work" in `timbers pending`. Co-committed entries (mixed code + `.timbers/` files) would break this filter entirely, requiring a fundamentally different detection mechanism.

2. **Entry IDs contain commit SHAs.** The format `tb_<timestamp>_<short-sha>` bakes the anchor commit's SHA into the entry ID. Amending the code commit changes its SHA, creating a chicken-and-egg: the entry needs the SHA before the commit exists, but the commit includes the entry. Using the parent SHA as anchor was considered but introduces anchor-semantic changes throughout the codebase.

3. **Push timing makes amend unreliable.** The workflow is commit → log → push, but agents and hooks frequently push commits before `timbers log` runs. Once pushed, amending requires force-push. In testing, the amend path would succeed less than 20% of the time, with the remaining 80% falling back to separate commits — creating inconsistent commit patterns that complicate every downstream consumer.

4. **Side branches break clone safety.** Moving entries to an orphan branch means `git clone --single-branch`, `git clone --depth=1`, and many CI environments silently lose all entries. The current model's strongest property is that entries travel with the repo unconditionally.

**The git log noise is real** — roughly 45% of recent commits are entry commits. This is mitigated by:
- Agents never see the noise (`filterLedgerOnlyCommits` handles it)
- `.gitattributes linguist-generated` collapses entries in GitHub diffs
- Human filtering: `git log --invert-grep --grep="^timbers: document"`
- Or add a git alias: `git config alias.lg 'log --oneline --invert-grep --grep="^timbers: document"'`

**The value proposition**: Timbers captures *reasoning* — the why and how behind changes — which git commit messages rarely preserve. The separate-commit noise is the cost of reliable, self-healing documentation that survives rebases, squash merges, and multi-agent workflows without special configuration.

---

*Document created during M1-complete review. Update when revisiting these decisions.*
