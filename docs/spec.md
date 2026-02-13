# Timbers Spec v1

**CLI:** `timbers`
**Tagline:** A Git-native development ledger that captures *what/why/how* as structured records.

---

## 0. Philosophy

### 0.1 Core Purpose

Timbers turns Git history into a durable development ledger by:
1. Harvesting objective facts from Git (commits, diffstat, changed files)
2. Pairing them with agent/human-authored rationale (what/why/how)
3. Storing as portable JSON files in `.timbers/` that sync via regular Git operations
4. Exporting structured data for downstream narrative generation

### 0.2 Agent DX Principles

1. **Minimal ceremony** — `timbers log "what" --why "why" --how "how"` captures work in one command
2. **Clear next action** — `timbers pending` shows undocumented commits; no guessing
3. **Context injection** — `timbers prime` outputs workflow state for session start
4. **JSON everywhere** — All commands support `--json` for structured parsing
5. **Sensible defaults** — Anchor to HEAD, auto-detect ranges, minimal required fields
6. **Git-native** — No server, no auth, travels with the repo
7. **Composable** — Exports pipe cleanly to LLM CLIs and workflow tools

### 0.3 The Primacy of "Why"

**The "why" is the most valuable field in the ledger.**

Git captures *what* changed (the diff). Commit messages describe *how*. But *why* — the reasoning, context, requirements, decisions — lives in ephemeral places that evaporate after sessions end.

**Timbers exists to capture "why" before it disappears.**

| Level | Example | When to use |
|-------|---------|-------------|
| Trivial | "Routine maintenance" | Deps updates, typo fixes |
| Contextual | "User reported auth failures on mobile" | Bug fixes with known trigger |
| Requirements-driven | "PM requested rate limiting for API launch" | Feature work from stakeholders |
| Technical reasoning | "Existing approach caused N+1 queries" | Performance-motivated refactoring |
| Architectural | "Chose middleware over decorator due to route sprawl" | Decisions with alternatives |

When the "why" captures the verdict, optional **notes** capture the journey — alternatives explored, trade-offs weighed, dead ends encountered. Use `--notes` selectively for non-trivial decisions.

### 0.4 Evidence vs Meaning (Anti-hallucination Boundary)

- **Evidence (automatic):** Git-derived facts — commits, diffstat, changed paths
- **Meaning (authored, required):** What/why/how summaries
- **Enrichment (optional, M2+):** Tracker data, decisions, verification

The CLI gathers evidence deterministically. The agent supplies meaning. Never fabricate.

---

## 1. Scope

### 1.1 Milestone 1 — In Scope

**Commands:**
- `timbers log` — Record work with what/why/how
- `timbers pending` — Show undocumented commits
- `timbers prime` — Context injection for session start
- `timbers status` — Repo/ledger state
- `timbers show` — Display a single entry
- `timbers query` — Retrieve entries (--last N for M1)
- `timbers export` — Structured export for pipelines

**Features:**
- Minimal entry schema
- JSON output on all commands (`--json`)
- `--dry-run` on write operations
- `--batch` mode for multi-entry capture
- Pipe-friendly export for LLM CLI integration

### 1.2 Out of Scope

**Cut entirely (not planned):**
- `timbers narrate` — Use pipeline: `timbers export | claude/gemini/codex`
- `timbers log -i` — Agents don't use TUI
- `timbers draft/write` — `log` suffices
- `timbers enrich` — Log correctly once
- Linear adapter — Support beads or none
- Beads shell hooks — Workflow docs pattern is better

**Deferred to M2:**
- Full query filters (--tag, --work-item, --path, --since, --until)
- Export bundling and preambles
- `--format prompt` wrapper for LLM consumption
- Tracker adapters with hydration
- Config file (`.timbers/config.yaml`)

---

## 2. Technical Decisions

### 2.1 Git Operations

**Decision:** Use `os/exec` to shell out to `git` commands.

**Rationale:** Simpler, matches user's git config, well-understood failure modes.

### 2.2 Schema Validation

**Decision:** Use `encoding/json` with struct tags and validation function.

**Rationale:** No external schema dependencies, compile-time type safety.

### 2.3 "Since Last Entry" Algorithm

```
1. Walk .timbers/YYYY/MM/DD/ directories to find all entry JSON files
2. Parse each entry to extract anchor_commit
3. Find anchor_commit with latest created_at timestamp
4. Return commits from that anchor (exclusive) to HEAD (inclusive)
5. If no entries exist, return all commits reachable from HEAD
```

### 2.4 Entry ID Format

```
tb_<ISO8601-timestamp>_<anchor-short-sha>
```

Example: `tb_2026-01-15T15:04:05Z_8f2c1a`

- Timestamp: UTC, second precision
- Short SHA: First 6 characters of anchor commit
- Determinism: Same anchor + same timestamp = same ID

---

## 3. Schema

### 3.1 Entry Schema (timbers.devlog/v1)

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",

  "workset": {
    "anchor_commit": "8f2c1a9d7b0c3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
    "commits": ["8f2c1a9d7b0c...", "c11d2a...", "a4e9bd..."],
    "range": "c11d2a..8f2c1a",
    "diffstat": {"files": 3, "insertions": 45, "deletions": 12}
  },

  "summary": {
    "what": "Fixed authentication bypass vulnerability",
    "why": "User input wasn't being sanitized before JWT validation",
    "how": "Added input validation middleware before auth handler"
  },

  "notes": "Considered rate limiting vs input validation. Validation catches the root cause; rate limiting only masks symptoms.",

  "tags": ["security", "auth"],

  "work_items": [
    {"system": "beads", "id": "bd-a1b2c3"}
  ]
}
```

**Required fields:**
- `schema`, `kind`, `id`
- `created_at`, `updated_at`
- `workset.anchor_commit`, `workset.commits[]`
- `summary.what`, `summary.why`, `summary.how`

**Optional fields:**
- `notes` — deliberation context (the journey to the decision)
- `workset.range`, `workset.diffstat`
- `tags[]`, `work_items[]`

---

## 4. CLI Commands

### 4.1 Global Flags

All commands support:
- `--json` — Output JSON instead of human-readable text
- `--help` — Show help

### 4.2 `timbers log`

Record work as a ledger entry.

```bash
# Standard usage
timbers log "Fixed auth bypass" --why "Input not sanitized" --how "Added validation"

# With work item and tags
timbers log "Completed feature" --why "User request" --how "New endpoint" \
  --work-item beads:bd-a1b2c3 --tag api --tag feature

# Minor change (defaults why/how)
timbers log --minor "Updated dependencies"

# Auto-extract from commits
timbers log --auto

# Batch mode for multiple entries
timbers log --batch

# Custom range/anchor
timbers log "Sprint work" --why "..." --how "..." --range abc123..def456

# Dry run
timbers log "Test" --why "Test" --how "Test" --dry-run --json
```

**Arguments:**
- `<what>` (positional) — What was done (required unless --auto or --batch)

**Flags:**
- `--why <text>` — Why it was done — the verdict (required unless --minor, --auto, or --batch)
- `--how <text>` — How it was done (required unless --minor, --auto, or --batch)
- `--notes <text>` — Deliberation context — the journey (optional, use selectively)
- `--range <A..B>` — Explicit commit range (default: since last entry)
- `--anchor <sha>` — Override anchor commit (default: HEAD)
- `--tag <tag>` — Add tag (repeatable)
- `--work-item <system:id>` — Link work item (repeatable)
- `--minor` — Use defaults for trivial changes
- `--auto` — Extract what/why/how from commit messages (non-interactive)
- `--batch` — Process multiple commit groups interactively
- `--push` — Push to remote after write
- `--dry-run` — Show what would be written without writing
- `--json` — Output JSON receipt

**Exit codes:**
- 0: Success
- 1: Invalid arguments or missing required fields
- 2: Git operation failed
- 3: Entry already exists for anchor (use --replace)

**JSON output (success):**
```json
{
  "status": "created",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "anchor": "8f2c1a9d7b0c...",
  "commits": 3
}
```

**JSON output (error):**
```json
{
  "error": "missing required flag: --why",
  "code": 1
}
```

#### Auto Mode

`--auto` extracts what/why/how from commit messages:

1. Concatenates commit subjects → "what" candidate
2. Extracts commit body content → "why/how" candidates
3. Prompts for confirmation/refinement (or uses as-is with `--auto --yes`)

**Important:** `--auto` is a starting point. Agents should enrich with session context.

#### Batch Mode

`--batch` processes multiple commit groups:

1. Groups pending commits by work item trailer, or by day
2. For each group, prompts for what/why/how (or uses `--auto`)
3. Creates one entry per group

Token-efficient alternative to looping `timbers log` calls.

### 4.3 `timbers pending`

Show commits without entries.

```bash
timbers pending
timbers pending --json
timbers pending --count
```

**Flags:**
- `--count` — Only show count
- `--json` — Output JSON

**Output (human):**
```
Commits since last entry (5):
  abc1234 Fix null pointer in auth handler
  def5678 Add rate limiting middleware
  ghi9012 Update dependencies
  ...

Run: timbers log "..." --why "..." --how "..."
```

**Output (JSON):**
```json
{
  "count": 5,
  "last_entry": "tb_2026-01-14T10:00:00Z_xyz789",
  "commits": [
    {"sha": "abc1234...", "short": "abc1234", "subject": "Fix null pointer..."}
  ]
}
```

### 4.4 `timbers prime`

Output workflow context for session start.

```bash
timbers prime
timbers prime --json
```

**Output (human):**
```
Timbers workflow context
========================
Repo: timbers
Branch: main

Last 3 entries:
  tb_2026-01-15... "Fixed auth bypass" (3 commits)
  tb_2026-01-14... "Added rate limiting" (7 commits)
  tb_2026-01-13... "Initial auth system" (12 commits)

Pending commits: 5 (since tb_2026-01-15...)

Quick commands:
  timbers pending          # See undocumented commits
  timbers log "..." ...    # Record work
  timbers show --last      # View last entry
```

### 4.5 `timbers status`

Show repository and ledger status.

```bash
timbers status
timbers status --json
```

**Output (JSON):**
```json
{
  "repo": "timbers",
  "branch": "main",
  "head": "abc1234...",
  "storage_dir": ".timbers/",
  "entry_count": 47,
  "pending_count": 5
}
```

### 4.6 `timbers show`

Display a single entry.

```bash
timbers show tb_2026-01-15T15:04:05Z_8f2c1a
timbers show --last
timbers show --last --json
```

**Flags:**
- `--last` — Show most recent entry
- `--json` — Output JSON

### 4.7 `timbers query`

Search and retrieve entries.

```bash
timbers query --last 5
timbers query --last 10 --json
timbers query --last 3 --oneline
```

**M1 Flags:**
- `--last <n>` — Last N entries (required for M1)
- `--json` — Output JSON array
- `--oneline` — Compact format

**M2 Flags (deferred):**
- `--tag <tag>`, `--work-item <system:id>`, `--path <prefix>`
- `--since <date>`, `--until <date>`
- `--range <A..B>`

### 4.8 `timbers export`

Export entries to structured formats for pipelines.

```bash
# Export to stdout (for piping)
timbers export --last 5 --json

# Export to directory
timbers export --last 5 --out ./exports/

# Export for LLM consumption
timbers export --range v1.0.0..v1.1.0 --json | claude "Generate changelog"

# Export to markdown files
timbers export --last 10 --format md --out ./release-notes/
```

**Flags:**
- `--last <n>` — Export last N entries
- `--range <A..B>` — Export entries in commit range
- `--format <json|md>` — Output format (default: json for stdout, md for --out)
- `--out <dir>` — Output directory (if omitted, writes to stdout)
- `--json` — Shorthand for --format json

**Pipe-friendly design:**
- Without `--out`: writes to stdout for piping
- With `--out`: writes files to directory

**Markdown file format:**
```markdown
---
schema: timbers.export/v1
id: tb_2026-01-15T15:04:05Z_8f2c1a
date: 2026-01-15
anchor_commit: 8f2c1a9d7b0c
commit_count: 3
tags: [security, auth]
---

# Fixed authentication bypass vulnerability

**What:** Fixed authentication bypass vulnerability

**Why:** User input wasn't being sanitized before JWT validation

**How:** Added input validation middleware before auth handler

## Evidence

- Commits: 3 (c11d2a..8f2c1a)
- Files changed: 6 (+241/-88)
```

---

## 5. Error Handling

### 5.1 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Invalid arguments, missing fields, or entry not found |
| 2 | Git operation failed |
| 3 | Entry conflict or I/O error |

### 5.2 Error JSON Format

```json
{
  "error": "human-readable error message",
  "code": 1
}
```

---

## 6. Agent Execution Contract

When using Timbers as an agent:

1. **Use CLI, not Git parsing** — Call `timbers` commands, consume JSON output
2. **Author accurate meaning** — What/why/how must reflect actual work
3. **Enrich "why" from session context** — This is your primary value-add
4. **Use `timbers pending`** — Know what needs documentation
5. **Never fabricate** — If unknown, omit optional fields
6. **Validate writes** — Use `--dry-run` before committing

### 6.1 Enriching "Why" — The Agent's Key Contribution

As an agent, you have access to context that will be lost after the session:

- **Requirements discussed** — What the user asked for and why
- **Specs and docs referenced** — Design documents, ADRs, external resources
- **Reasoning applied** — Why you chose approach A over B
- **Tradeoffs considered** — Performance vs. readability, simplicity vs. flexibility
- **Constraints discovered** — Limitations encountered, workarounds applied
- **User feedback incorporated** — Iterations based on review

**Your job:** Synthesize this ephemeral context into a concise "why" (the verdict) that someone reading in 6 months will understand. When you explored alternatives or made a real choice, capture the journey in `--notes`.

**Example transformation:**

| Source | Raw | Enriched "why" |
|--------|-----|----------------|
| Commit | "fix auth bug" | "User reported JWT validation failures on mobile; root cause was missing audience claim check" |
| PR | "adds rate limiting" | "PM requested rate limiting before API launch; chose token bucket algorithm based on expected load" |
| Session | [reasoning] | "Refactored to middleware after discovering handler-level auth was bypassed on 3 routes" |

### 6.2 Recommended Workflow

```bash
# At work completion — capture while context is fresh
timbers log "What I did" \
  --why "Why I did it (the verdict)" \
  --how "How I did it" \
  --notes "Alternatives explored, trade-offs weighed (optional, use selectively)"

# For trivial changes
timbers log --minor "Updated dependencies"

# At session end
timbers pending      # Any undocumented work?
git push             # Sync to remote
```

### 6.3 Historical Documentation

When documenting old commits:

```bash
timbers log --range old..older --auto
# Use --auto to extract from commits
# Do NOT fabricate context you don't have
# Acceptable: "Historical commit; original rationale not captured"
```

### 6.4 Pipeline Integration

For narrative generation, pipe to your preferred LLM CLI:

```bash
# Generate changelog
timbers export --last 10 --json | claude "Generate a changelog from these dev log entries"

# Generate release notes
timbers export --range v1.0.0..v1.1.0 --json | gemini "Write user-facing release notes"

# Use mdflow for complex pipelines
timbers export --last 20 --json | mdflow run devlog-pipeline
```

---

## 7. Integration Patterns

### 7.1 With Beads

Beads tracks *what to do*; Timbers records *what was done*.

```bash
# Beads: Track and complete work
bd create "Fix auth bypass" --type bug
bd update bd-a1b2c3 --claim
# ... do the work ...
bd close bd-a1b2c3

# Timbers: Document with session context
timbers log "Fixed auth bypass" \
  --why "Security audit revealed JWT parsing flaw; chose middleware per team discussion" \
  --how "Added validation in auth middleware" \
  --work-item beads:bd-a1b2c3

# Timbers: Generate narrative (via pipeline)
timbers export --last 5 --json | claude "Write a developer diary entry"
```

### 7.2 Workflow Documentation

Add to CLAUDE.md or AGENTS.md:

```markdown
## Development Ledger

This project uses Timbers for development documentation.

After completing work:
timbers log "what" --why "why (enrich!)" --how "how"

At session start:
timbers prime

At session end:
timbers pending      # Check for undocumented work
git push             # Sync to remote
```

---

## 8. Acceptance Criteria

### 8.1 Core Workflow

```bash
# AC1: Basic log creates entry
timbers log "Test entry" --why "Testing" --how "Via CLI"
# Exit 0, entry visible in `timbers show --last`

# AC2: Pending shows undocumented commits
git commit --allow-empty -m "Test commit"
timbers pending
# Shows the new commit

# AC3: Log clears pending
timbers log "Documented" --why "Test" --how "Test"
timbers pending
# Shows 0 pending commits

# AC4: Prime outputs context
timbers prime
# Shows last entries and pending count

# AC5: JSON output works
timbers log "JSON test" --why "Test" --how "Test" --json
# Returns valid JSON with status, id, anchor, commits

# AC6: Dry run doesn't write
timbers log "Dry run" --why "Test" --how "Test" --dry-run
# Entry not created

# AC7: Batch mode works
timbers log --batch --auto
# Processes pending commits in groups

# AC8: Export pipes cleanly
timbers export --last 3 --json | jq '.entries | length'
# Returns 3
```

### 8.2 Error Cases

```bash
# AC9: Missing --why fails
timbers log "No why"
# Exit 1, error mentions --why

# AC10: Not in git repo fails
cd /tmp && timbers status
# Exit 2, error mentions "not a git repository"

# AC11: Duplicate anchor fails
timbers log "First" --why "Test" --how "Test"
timbers log "Second" --why "Test" --how "Test" --anchor HEAD
# Exit 3, error mentions --replace
```

---

## 9. Implementation Notes

### 9.1 Package Structure

```
cmd/
  timbers/
    main.go           # Entry point
    log.go            # log command
    pending.go        # pending command
    prime.go          # prime command
    status.go         # status command
    show.go           # show command
    query.go          # query command
    export.go         # export command
internal/
  git/
    git.go            # Git operations via exec
  ledger/
    entry.go          # Entry struct and validation
    filestorage.go    # Read/write entries to .timbers/ files
    pending.go        # Since-last-entry algorithm
  export/
    json.go           # JSON export
    markdown.go       # Markdown export
```

### 9.2 Dependencies

- `github.com/spf13/cobra` — CLI framework (already in go.mod)
- Standard library for git operations (`os/exec`)
- Charm libraries for TUI elements (status, progress) if desired

### 9.3 Testing Strategy

**Unit tests:**
- Entry struct validation
- ID generation determinism
- Markdown/JSON formatting
- Since-last-entry algorithm (mock git output)

**Integration tests:**
- Create temp git repo
- Run full workflows (log → pending → query → export)
- Verify entry files in .timbers/
- Test error cases
- Test pipeline output

---

## 10. Future Milestones

### M2: Query, Export, Pipelines

- Full query filters (--tag, --work-item, --path, --since, --until)
- `--format prompt` wrapper for LLM-ready output
- Export bundling
- Tracker adapters with hydration

### M3: Enrichment & Polish

- `timbers log -i` interactive mode (if human use grows)
- Config file support
- Template system

### M4: Advanced

- Relink for rebased history
- Merge semantics for concurrent edits
- HTML generation
- Cross-repo aggregation
