# Timbers — Development Ledger + Narrative Export

**CLI:** `timbers`
**Tagline:** A Git-native development ledger that captures *what/why/how* as structured records attached to history, and exports LLM-ready markdown for narratives.

---

## 0. Positioning

### 0.1 One-sentence description

Timbers is a Go CLI that turns your Git history into a durable, structured "development ledger" by harvesting objective facts from Git (commit ranges, changed files, diffstats, tags) and pairing them with concise human/agent-authored rationale (what/why/how) stored as portable **Git notes** that sync to remotes, then exporting frontmatter-rich markdown packets for changelogs, stakeholder updates, and narrative devlogs.

### 0.2 Paragraph description

Timbers is for solo devs and small teams shipping with AI agents—where code volume is high, commits are frequent, and humans act more like architects/PMs than day-to-day implementers. Git alone is excellent at *what* changed but weak at preserving *why* and *how* in a way that's readable to outsiders. Timbers keeps a clean, Git-native ledger: it automatically collects hard evidence from Git (commit sets, diffstat + path rollups, tags/releases, branch/merge context) and pairs that with concise *what/why/how* summaries authored by the agent or human. From that single canonical source, you can generate internal dev-facing change trails, PM/manager summaries, customer-friendly release notes, and entertaining "dev diaries" in bespoke voices.

### 0.3 Agent DX Philosophy

Timbers learns from what makes [beads](https://github.com/steveyegge/beads) successful for agents:

1. **Minimal ceremony** — `timbers log "what" --why "why" --how "how"` captures work in one command
2. **Clear next action** — `timbers pending` shows undocumented commits; no guessing
3. **Context injection** — `timbers prime` outputs workflow state for session start
4. **JSON everywhere** — All commands support `--json` for structured parsing
5. **Sensible defaults** — Anchor to HEAD, auto-detect ranges, minimal required fields
6. **Git-native** — No server, no auth, travels with the repo

**Core insight:** Agents need *velocity* over *completeness*. The happy path must be effortless; enrichment is optional.

---

## 1. Objectives

### 1.1 Primary goals

1. **Capture durable, queryable development records** that complement Git commit history
2. **Minimize authoring friction** — quick-capture in one command, enrich later if desired
3. **Keep the repository clean** — ledger lives in Git notes, not files that drift
4. **Support archaeology** — reconstruct "why decisions were made" and "how we got here"
5. **Export narrative-ready packets** — markdown with frontmatter for downstream LLM generation

### 1.2 Non-goals (MVP)

* Timbers does **not** directly invoke an LLM to generate narratives
* Timbers does **not** attempt deep semantic analysis of code diffs
* Timbers does **not** aim to be an issue tracker (use beads/Linear/GitHub for that)
* Timbers does **not** require any external tracker; integrations are optional enrichment

### 1.3 Design constraints

* **Git is required**
* **Trackers (Beads/Linear) are optional** — enrichment, not dependency
* **Go CLI** with deterministic, reproducible behavior
* **Token-efficient** — CLI gathers facts; agent provides meaning

---

## 2. Core Approach

### 2.1 Evidence vs Meaning (anti-hallucination boundary)

Timbers explicitly separates:

* **Evidence (machine-collected, automatic):** Git-derived facts — commit sets/ranges, diffstat, changed path rollups, branch/merge context, tags/releases, commit trailers
* **Meaning (agent/human-authored, required):** Concise *What/Why/How* — the rationale
* **Enrichment (optional):** Tracker data (Beads/Linear), decisions, verification notes, ADR references

The CLI gathers evidence deterministically. The agent supplies meaning. Enrichment is opt-in.

### 2.2 The Primacy of "Why"

**The "why" is the most valuable field in the ledger.**

Git already captures *what* changed (the diff). Commit messages often describe *how* (implementation details). But *why* — the reasoning, context, requirements, and decisions — lives in ephemeral places: Slack threads, meeting notes, spec documents, the agent's reasoning trace, the human's mental model. This knowledge evaporates after the session ends.

**Timbers exists to capture "why" before it disappears.**

#### What makes a good "why"

| Level | Example | When to use |
|-------|---------|-------------|
| **Trivial** | "Routine maintenance" | Deps updates, formatting, typo fixes |
| **Contextual** | "User reported auth failures on mobile" | Bug fixes with known trigger |
| **Requirements-driven** | "PM requested rate limiting for API launch" | Feature work from stakeholder input |
| **Technical reasoning** | "Existing approach caused N+1 queries under load" | Refactoring with performance motivation |
| **Architectural** | "Chose middleware pattern to enforce auth at single point; considered decorator but rejected due to route sprawl" | Decisions with alternatives considered |

#### Enrichment from session context

When an agent (or human) authors a Timbers entry, they should:

1. **Extract** — Pull what/why/how signals from commit messages, PR descriptions, linked issues
2. **Enrich** — Add context from the current session: requirements discussed, specs referenced, reasoning applied, tradeoffs considered
3. **Synthesize** — Produce a concise summary that someone reading in 6 months will understand

**The goal:** Snapshot the conceptual context that explains the mechanical changes.

#### No hallucination rule

When running `timbers log` on historical commits (not the current session), do NOT fabricate context that wasn't available. Use only:
- Commit messages and trailers
- Linked issue/PR content (if accessible)
- Code comments and documentation

If the "why" is genuinely unknown, say so: "Historical commit; original rationale not captured."

### 2.3 Canonical storage: Git notes

Timbers stores records as **Git notes** attached to commits:

* **Notes ref:** `refs/notes/timbers`
* **One entry per anchor commit** (simple, no merge complexity)

**Why Git notes?**
- Attach data to commits without changing commit IDs
- Sync with repo via fetch/push
- Clean working tree (no file clutter)
- Support archaeology across history

### 2.4 Anchor strategy

Each entry is attached as a Git note to one **anchor commit**.

**Default:** `HEAD` (current commit)

**Override:** `--anchor <sha>` when you want to attach to a specific commit (e.g., merge commit, release tag)

**Simplicity principle:** Don't make agents think about anchors unless they need to.

---

## 3. Schema

### 3.1 Schema philosophy

**Two tiers:**
1. **Minimal entry** — Required fields only. Quick capture.
2. **Full entry** — All optional enrichment fields. For thorough documentation.

Agents can start minimal and enrich later via `timbers enrich`.

### 3.2 Schema version

* Schema: `timbers.devlog/v1`
* Export schema: `timbers.export/v1`

**Versioning:** Breaking changes require new schema version; old versions remain readable.

### 3.3 Minimal entry (quick capture)

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",

  "workset": {
    "anchor_commit": "8f2c1a9d7b0c...",
    "commits": ["8f2c1a9d7b0c..."],
    "diffstat": {"files": 3, "insertions": 45, "deletions": 12}
  },

  "summary": {
    "what": "Fixed authentication bypass vulnerability",
    "why": "User input wasn't being sanitized before JWT validation",
    "how": "Added input validation middleware before auth handler"
  }
}
```

**Required fields:**
- `schema`, `kind`, `id`
- `created_at`, `updated_at`
- `workset.anchor_commit`, `workset.commits[]`
- `summary.what`, `summary.why`, `summary.how`

### 3.4 Full entry (with enrichment)

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",

  "created_by": {
    "actor": "agent",
    "name": "claude",
    "tool": "timbers",
    "tool_version": "0.1.0"
  },

  "work_items": [
    {"system": "beads", "id": "bd-a1b2c3", "title": "Fix auth bypass", "status": "closed"}
  ],

  "workset": {
    "anchor_commit": "8f2c1a9d7b0c...",
    "commits": ["8f2c1a9d7b0c...", "c11d2a...", "a4e9bd..."],
    "range": "c11d2a..8f2c1a",
    "changed_paths_top": ["src/auth", "src/middleware"],
    "diffstat": {"files": 6, "insertions": 241, "deletions": 88}
  },

  "summary": {
    "what": "Fixed authentication bypass vulnerability",
    "why": "User input wasn't being sanitized before JWT validation",
    "how": "Added input validation middleware before auth handler"
  },

  "decisions": [
    {
      "decision": "Validate at middleware layer, not in handler",
      "alternatives": ["Validate in handler", "Use decorator pattern"],
      "rationale": "Middleware catches all routes; single point of enforcement"
    }
  ],

  "verification": {
    "risk": "high",
    "tests_added": ["auth_bypass_test.go"],
    "manual_checks": ["Verified with malformed JWT tokens"]
  },

  "references": {
    "adrs": ["ADR-0007"],
    "docs": ["docs/security/auth.md"]
  },

  "tags": ["security", "auth"],

  "provenance": {
    "branch": "fix/auth-bypass",
    "git_describe": "v1.2.3-5-g8f2c1a"
  }
}
```

**Optional fields (all):**
- `created_by` — Actor metadata
- `work_items[]` — Linked tracker items
- `workset.range`, `workset.changed_paths_top` — Extended evidence
- `decisions[]` — Architectural choices made
- `verification` — Risk assessment, tests, manual checks
- `references` — ADRs, docs, PRs
- `tags[]` — Searchable labels
- `provenance` — Branch, git describe, etc.

---

## 4. CLI Commands (Agent-Optimized)

### 4.1 Design principles

1. **One command for the common case** — `timbers log` does it all
2. **JSON output everywhere** — `--json` on every command
3. **Sensible defaults** — Minimal flags required
4. **Composable** — Draft → enrich → write for complex workflows

### 4.2 Quick capture: `timbers log`

The primary command for agents. One-liner to record work.

```bash
# Minimal (attaches to HEAD, auto-collects evidence)
timbers log "Fixed auth bypass" --why "Input not sanitized" --how "Added validation"

# With work item
timbers log "Fixed auth bypass" --why "..." --how "..." --work-item beads:bd-a1b2c3

# Custom range
timbers log "Refactored auth" --why "..." --how "..." --range abc123..def456

# From last entry to HEAD (default when no range specified)
timbers log "Sprint work" --why "..." --how "..."

# JSON output of what was written
timbers log "..." --why "..." --how "..." --json
```

**Behavior:**
1. Collects evidence from Git (commits since last entry, or `--range`)
2. Creates minimal entry with what/why/how
3. Attaches as Git note to HEAD (or `--anchor`)
4. Prints confirmation (or JSON receipt with `--json`)

**Flags:**
- `--why` (required unless `--auto` or `--minor`) — Why this work was done
- `--how` (required unless `--auto` or `--minor`) — How it was accomplished
- `--work-item <system:id>` — Link to tracker item
- `--range <A..B>` — Explicit commit range
- `--anchor <sha>` — Override anchor commit (default: HEAD)
- `--tag <tag>` — Add searchable tag (repeatable)
- `--json` — Output JSON receipt
- `--push` — Push notes after write
- `--auto` — Pre-fill from commit messages (see below)
- `--minor` — Use minimal template for trivial changes

#### Auto-fill mode: `--auto`

Pre-fills what/why/how from commit messages, then prompts for enrichment:

```bash
timbers log --auto
# Detected from commits:
#   What: "Fix authentication bypass vulnerability"
#   Why:  "Security audit finding CVE-2026-1234"
#   How:  "Added input validation in auth middleware"
#
# Enrich the 'why' with session context? [Y/n/edit]
# > Added after penetration test revealed JWT parsing flaw.
#   Discussed mitigation options with security team; chose
#   middleware approach for single enforcement point.
```

**Important:** `--auto` is a *starting point*, not the final answer. The extracted why is often shallow (commit messages are terse). Agents and humans should enrich with session context when available.

For agents, `--auto` can be combined with explicit enrichment:
```bash
timbers log --auto --why-append "Implemented per security team recommendation after Q1 audit"
```

#### Minor mode: `--minor`

For trivial changes where full what/why/how is overkill:

```bash
timbers log --minor "Updated dependencies"
# Creates entry with:
#   what: "Updated dependencies"
#   why: "Routine maintenance"
#   how: "Dependency update"

timbers log --minor "Fixed typo in README"
# Creates entry with:
#   what: "Fixed typo in README"
#   why: "Documentation quality"
#   how: "Text correction"
```

Use `--minor` for: dependency updates, formatting fixes, typo corrections, trivial refactors. The entry is still recorded for completeness, but doesn't demand rich context that doesn't exist.

### 4.3 Pending work: `timbers pending`

Shows commits that don't have entries yet. The "what needs documenting?" command.

```bash
timbers pending
# Commits since last entry (5):
#   abc1234 Fix null pointer in auth handler
#   def5678 Add rate limiting middleware
#   ghi9012 Update dependencies
#   jkl3456 Refactor token validation
#   mno7890 Add integration tests
#
# Run: timbers log "..." --why "..." --how "..."

timbers pending --json
# Returns array of commit objects
```

### 4.4 Context injection: `timbers prime`

Outputs workflow context for session start. Like `bd prime` for beads.

```bash
timbers prime
# Timbers workflow context
# ========================
# Repo: timbers (github.com/steveyegge/timbers)
# Branch: main
#
# Last 3 entries:
#   tb_2026-01-15... "Fixed auth bypass" (3 commits)
#   tb_2026-01-14... "Added rate limiting" (7 commits)
#   tb_2026-01-13... "Initial auth system" (12 commits)
#
# Pending commits: 5 (since tb_2026-01-15...)
#
# Quick commands:
#   timbers pending          # See undocumented commits
#   timbers log "..." ...    # Record work
#   timbers query --last 5   # Recent entries
```

### 4.5 Status: `timbers status`

Repository and notes status.

```bash
timbers status
# Repo: timbers
# Branch: main @ abc1234
# Notes ref: refs/notes/timbers (configured, synced)
# Entries: 47 total
# Pending: 5 commits since last entry

timbers status --json
```

### 4.6 Interactive mode: `timbers log -i`

For humans or when agents want guided input. Uses Charm's `huh` library.

```bash
timbers log -i
# Prompts:
#   What did you do? [text input]
#   Why did you do it? [text input]
#   How did you accomplish it? [text input]
#   Link work item? [optional, autocomplete from trailers]
#   Add tags? [optional, multi-select or custom]
```

### 4.7 Draft workflow: `timbers draft` / `timbers write`

For complex entries requiring review/enrichment.

```bash
# Generate draft with auto-collected evidence
timbers draft --json > entry.json

# Or with explicit range
timbers draft --range abc..def --json > entry.json

# Edit entry.json to add meaning, decisions, etc.

# Write to notes
timbers write entry.json

# Or pipe directly
timbers draft --json | jq '.summary.what = "..."' | timbers write -
```

### 4.8 Enrich existing: `timbers enrich`

Add optional fields to an existing entry.

```bash
# Add decision to most recent entry
timbers enrich --last --decision "Used middleware pattern" \
  --alternatives "Handler validation,Decorator" \
  --rationale "Single enforcement point"

# Add verification
timbers enrich tb_2026-01-15... --risk high --tests-added "auth_test.go"

# Add tags
timbers enrich --last --tag security --tag auth
```

### 4.9 Query: `timbers query`

Search and filter entries.

```bash
# Last N entries
timbers query --last 10

# By work item
timbers query --work-item beads:bd-a1b2c3

# By tag
timbers query --tag security

# By path prefix
timbers query --path src/auth

# By date range
timbers query --since 2026-01-01 --until 2026-01-31

# By commit range
timbers query --range v1.0.0..v1.1.0

# Output formats
timbers query --last 5 --json     # JSON array
timbers query --last 5 --md       # Markdown
timbers query --last 5 --oneline  # Compact list
```

### 4.10 Export: `timbers export`

Generate markdown files for narrative generation.

```bash
# Export range to directory
timbers export --range v1.0.0..v1.1.0 --out ./release-notes/

# With narrative preamble
timbers export --range v1.0.0..v1.1.0 --out ./notes/ --preamble changelog

# Bundle by week
timbers export --since 2026-01-01 --bundle week --out ./devlog/
```

### 4.11 Narrative generation: `timbers narrate`

Generate narratives from entries using an LLM. Completes the loop from ledger to publishable content.

```bash
# Generate changelog from recent entries
timbers narrate --last 5 --style changelog

# Release notes from a version range
timbers narrate --range v1.0.0..v1.1.0 --style release-notes

# Developer diary with personality
timbers narrate --last 10 --style devlog --voice "tired engineer at 2am"

# Stakeholder summary (non-technical)
timbers narrate --since 2026-01-01 --style stakeholder

# Output to file
timbers narrate --last 5 --style changelog --out CHANGELOG.md

# JSON output (for further processing)
timbers narrate --last 3 --json
```

**Styles:**
- `changelog` — Conventional changelog format, technical audience
- `release-notes` — User-facing, feature-focused
- `devlog` — Narrative developer diary, conversational
- `stakeholder` — Executive summary, non-technical
- `custom:<file>` — User-provided prompt template

**Flags:**
- `--style <name>` — Narrative style (required)
- `--voice <description>` — Personality/tone modifier
- `--last <n>` / `--range <A..B>` / `--since <date>` — Entry selection
- `--out <file>` — Write to file (default: stdout)
- `--model <name>` — LLM model (default: from config)
- `--json` — Output structured JSON with narrative + metadata

**Configuration:**

```yaml
# .timbers/config.yaml
narrate:
  provider: anthropic  # or openai, ollama
  model: claude-sonnet-4-20250514
  # API keys via environment: ANTHROPIC_API_KEY, OPENAI_API_KEY
```

**How it works:**
1. Collects entries matching the filter
2. Renders entries to structured prompt using style template
3. Invokes LLM with prompt
4. Returns narrative text

The style templates include the entry evidence (commits, diffstat, paths) and meaning (what/why/how, decisions) so the LLM can synthesize appropriately.

### 4.12 Batch mode: `timbers log --batch`

For catch-up documentation when you have many undocumented commits:

```bash
timbers log --batch
# Found 23 undocumented commits. Grouping by work item...
#
# Group 1: beads:bd-a1b2c3 (8 commits)
#   What: [extracted from commits]
#   Why: [enter or skip]
#   How: [extracted from commits]
#
# Group 2: No work item (5 commits, same day)
#   What: [extracted from commits]
#   ...
```

**Grouping strategies:**
- By work item trailer (if present)
- By day (commits on same day → single entry)
- By author (for multi-contributor repos)

```bash
# Batch with auto-fill (minimal interaction)
timbers log --batch --auto

# Batch all as minor (for historical cleanup)
timbers log --batch --minor
```

### 4.13 Notes management: `timbers notes`

Configure and sync notes refs.

```bash
# First-time setup
timbers notes init

# Push notes to remote
timbers notes push

# Fetch notes from remote
timbers notes fetch

# Check sync status
timbers notes status
```

### 4.14 Skill generation: `timbers skill`

Helps agents build a Timbers skill for their agentic environment.

```bash
timbers skill
# Outputs structured content for building a Timbers skill:
# - Core concepts (evidence vs meaning, primacy of why)
# - Workflow patterns (log, pending, prime cycle)
# - Command quick reference
# - Agent execution contract highlights
# - Integration patterns

timbers skill --format markdown
# Markdown-formatted output (default)

timbers skill --format json
# Structured JSON for programmatic consumption

timbers skill --include-examples
# Include worked examples of good/bad entries
```

**What it outputs:**

The command emits content organized for skill creation, NOT a ready-made skill. The agent transforms this into their environment's skill format.

```markdown
# Timbers Skill Content

## Core Concepts

### Evidence vs Meaning
- Evidence: Machine-collected Git facts (commits, diffstat, paths)
- Meaning: Agent-authored rationale (what/why/how)
- The CLI gathers evidence; you supply meaning

### The Primacy of "Why"
[Key points about enriching why from session context...]

## Workflow Patterns

### After completing work
timbers log "what" --why "why (enriched!)" --how "how"

### At session start
timbers prime  # Understand ledger state

### At session end
timbers pending      # Check for undocumented work
timbers notes push   # Sync to remote

## Command Quick Reference
[Table of commands and key flags...]

## Agent Execution Contract
[Key rules: enrich why, no fabrication, use pending...]

## Integration Patterns
[Beads workflow, work item linking...]
```

**Why this approach:**

1. **Environment-agnostic** — Timbers doesn't know your skill format
2. **Agent-driven** — You shape the skill for your context
3. **Current** — Output reflects latest Timbers capabilities
4. **Composable** — Include/exclude sections as needed

**Usage pattern:**

```bash
# Agent reads skill-creator instructions from their environment
# Agent runs timbers skill to get Timbers-specific content
# Agent synthesizes into a skill file for their environment

timbers skill --format json | agent-skill-creator --name timbers
```

### 4.15 Templates: `timbers log --template`

Pre-filled patterns for common work types.

```bash
timbers log --template bugfix
# Pre-fills: what="Fixed bug in X", prompts for X, why, how

timbers log --template feature
# Pre-fills: what="Added X feature", prompts for X, why, how

timbers log --template refactor
# Pre-fills: what="Refactored X", prompts for X, why, how

timbers log --template chore
# Pre-fills: what="Updated X", prompts for X, why, how
```

---

## 5. Git Notes Transport

### 5.1 Setup: `timbers notes init`

Configures notes to sync with remote:

```bash
git config --add remote.origin.fetch "+refs/notes/timbers:refs/notes/timbers"
```

Verifies push permissions and optionally enables auto-fetch.

### 5.2 Push/Fetch

```bash
timbers notes push   # git push origin refs/notes/timbers
timbers notes fetch  # git fetch origin refs/notes/timbers
```

### 5.3 Configuration

```yaml
# .timbers/config.yaml (optional)
notes:
  remote: origin
  ref: refs/notes/timbers
  auto_push: false    # Push after every write
  auto_fetch: true    # Fetch on status/query
```

### 5.4 Conflict handling

* **MVP:** One entry per anchor commit. No merge complexity.
* **On conflict:** `timbers write` fails with clear error; use `--replace` to overwrite.
* **Future:** `--merge` flag for field-level merge semantics.

---

## 6. Markdown Export Format

### 6.1 File naming

```
YYYY-MM-DD__<entry-id-short>__<anchor-short>.md
```

Example: `2026-01-15__tb_8f2c1a__8f2c1a9.md`

### 6.2 Frontmatter

```yaml
---
schema: timbers.export/v1
id: tb_2026-01-15T15:04:05Z_8f2c1a
date: 2026-01-15
repo: timbers
anchor_commit: 8f2c1a9d7b0c
commit_count: 3
work_items:
  - system: beads
    id: bd-a1b2c3
tags: [security, auth]
risk: high
---
```

### 6.3 Body structure

```markdown
# Fixed authentication bypass vulnerability

**What:** Fixed authentication bypass vulnerability

**Why:** User input wasn't being sanitized before JWT validation

**How:** Added input validation middleware before auth handler

## Decisions

- **Used middleware pattern** instead of handler validation or decorator
  - *Rationale:* Single enforcement point for all routes

## Verification

- Risk: high
- Tests added: auth_bypass_test.go
- Manual checks: Verified with malformed JWT tokens

## Evidence

- Commits: 3 (abc1234, def5678, ghi9012)
- Files changed: 6 (+241/-88)
- Paths: src/auth, src/middleware

## References

- ADR-0007: Authentication architecture
```

### 6.4 Narrative preambles

Optional block for downstream LLM narrative generation:

```markdown
<!-- NARRATIVE_PROMPT
Style: changelog
Audience: developers
Tone: professional
Length: 2-3 paragraphs
-->
```

Templates: `changelog`, `release-notes`, `devlog`, `stakeholder`, `custom:<file>`

---

## 7. Tracker Integrations (Optional)

### 7.1 Philosophy

Trackers are **enrichment, not dependency**. Timbers works fully without any tracker configured.

When enabled, trackers provide:
- ID validation
- Title/status hydration
- Commit candidate suggestions (via trailers)

### 7.2 Beads adapter

Detect from commit trailers (`Bead: bd-xxx`) or `--work-item beads:bd-xxx`.

```bash
# Auto-detect from recent commits
timbers log "..." --why "..." --how "..."
# Detects Bead: bd-a1b2c3 trailer, enriches work_items[]

# Explicit
timbers log "..." --work-item beads:bd-a1b2c3 --why "..." --how "..."
```

### 7.3 Linear adapter

Detect from trailers (`Linear: LIN-xxx`) or `--work-item linear:LIN-xxx`.

### 7.4 Adapter interface

```go
type TrackerAdapter interface {
    ValidateID(id string) (normalized string, ok bool, err error)
    Hydrate(id string) (WorkItem, error)
    SuggestCommits(id string) ([]string, error)  // Optional
}
```

### 7.5 Beads integration hooks

When beads is present, Timbers can integrate at natural documentation moments.

#### Post-close prompt

When you `bd close <id>`, that's the natural moment to document. Beads can trigger a Timbers prompt:

```bash
bd close bd-a1b2c3
# ✓ Closed bd-a1b2c3: Fix authentication bypass
#
# Document this work with Timbers?
# Run: timbers log "Fixed auth bypass" --why "..." --how "..." --work-item beads:bd-a1b2c3
```

Configuration (in `.beads/config.yaml`):
```yaml
hooks:
  post_close:
    - timbers prompt --work-item beads:$ID
```

#### Session-end reminder

At session end, check for undocumented closed beads:

```bash
# In Claude Code hook or shell prompt
bd list --status closed --since "1 hour ago" --json | \
  timbers check-undocumented --work-items -
# Warning: 2 closed beads have no Timbers entries
#   bd-a1b2c3: Fix auth bypass
#   bd-d4e5f6: Add rate limiting
```

#### Automatic work item detection

When running `timbers log` without `--work-item`, Timbers checks recent commits for `Bead:` trailers and auto-links:

```bash
timbers log "Completed auth work" --why "..." --how "..."
# Detected work item from commits: beads:bd-a1b2c3
# Linked automatically. Use --no-auto-link to disable.
```

---

## 8. Agent Execution Contract

When implementing or using Timbers as an agent:

1. **Use CLI, not Git parsing** — Call `timbers` commands, consume JSON output
2. **Author accurate meaning** — What/why/how must reflect actual work, not fabrication
3. **Enrich "why" from session context** — This is your primary value-add (see below)
4. **Use `timbers pending`** — Know what needs documentation before writing
5. **Start minimal, enrich if needed** — Quick capture first, add decisions/verification later
6. **Never fabricate** — If unknown, omit optional fields rather than guess
7. **Validate writes** — Check JSON receipt for success

### Enriching "why" — The agent's key contribution

As an agent, you have access to context that will be lost after the session:

- **Requirements discussed** — What the user asked for and why
- **Specs and docs referenced** — Design documents, ADRs, external resources consulted
- **Reasoning applied** — Why you chose approach A over approach B
- **Tradeoffs considered** — Performance vs. readability, simplicity vs. flexibility
- **Constraints discovered** — Limitations encountered, workarounds applied
- **User feedback incorporated** — Iterations based on review comments

**Your job:** Synthesize this ephemeral context into a concise "why" that someone reading in 6 months will understand.

**Example transformation:**

| Source | Raw | Enriched "why" |
|--------|-----|----------------|
| Commit message | "fix auth bug" | "User reported JWT validation failures on mobile; root cause was missing audience claim check; chose strict validation per OWASP guidelines" |
| PR description | "adds rate limiting" | "PM requested rate limiting before API launch to prevent abuse; implemented token bucket algorithm; chose 100 req/min based on expected load analysis" |
| Session context | [agent reasoning] | "Refactored to middleware pattern after discovering handler-level auth was bypassed on 3 routes; consolidated to single enforcement point per security team recommendation" |

### Recommended workflow

```bash
# At work completion — capture while context is fresh
timbers log "What I did" \
  --why "Why I did it (with session context!)" \
  --how "How I did it"

# For trivial changes
timbers log --minor "Updated dependencies"

# At session end
timbers pending      # Any undocumented work?
timbers notes push   # Sync to remote
```

### Historical documentation

When documenting historical commits (not current session):

```bash
timbers log --range old..older --auto
# Use --auto to extract from commits
# Do NOT fabricate context you don't have
# Acceptable: "Historical commit; original rationale not captured"
```

---

## 9. Quality & Testing

### 9.1 Determinism

* All JSON output stable across runs given identical repo state
* IDs are deterministic (timestamp + anchor short-sha)

### 9.2 Validation

* Strict JSON schema validation on write
* Fail fast with actionable error messages

### 9.3 Test plan

**Unit tests:**
- Schema validation
- Evidence collection (diffstat, paths)
- Export formatting

**Integration tests (temp Git repos):**
- `notes init` → `log` → `notes push/fetch`
- `pending` detection
- `query` filters
- Conflict handling

---

## 10. Resolved Design Decisions

Based on learnings from beads agent DX:

| Question | Decision | Rationale |
|----------|----------|-----------|
| One entry per anchor vs bundle? | One entry | Simpler merge, clearer ownership |
| ID format? | `tb_<timestamp>_<short-sha>` | Human-readable, sortable, unique |
| Default range? | Since last entry | Most common case; explicit `--range` overrides |
| Commit links? | Deferred to v2 | Doubles writes; query by range is sufficient |
| Auto-push? | Off by default | Avoid surprises; explicit `--push` or `notes push` |
| Trailer keys? | `Bead:`, `Linear:`, `Work-Item:` | Match existing conventions |

---

## 11. MVP Milestones

### Milestone 1 — Agent DX Core

**Goal:** An agent can document work in one command.

- [ ] `timbers log` with --why, --how, --work-item, --range
- [ ] `timbers log --auto` — pre-fill from commits
- [ ] `timbers log --minor` — trivial change shorthand
- [ ] `timbers pending` — show undocumented commits
- [ ] `timbers prime` — context injection
- [ ] `timbers status` — repo/notes state
- [ ] `timbers notes init/push/fetch`
- [ ] `timbers skill` — emit content for agent skill creation
- [ ] Minimal entry schema
- [ ] JSON output on all commands

### Milestone 2 — Query, Export & Narrate

**Goal:** Archaeology and narrative generation — complete the loop.

- [ ] `timbers query` with filters
- [ ] `timbers export` to markdown with frontmatter
- [ ] `timbers narrate` — LLM-integrated narrative generation
- [ ] Built-in styles: changelog, release-notes, devlog, stakeholder
- [ ] Preamble templates for export

### Milestone 3 — Enrichment & Polish

**Goal:** Full schema support, better UX, batch workflows.

- [ ] `timbers enrich` for adding decisions, verification
- [ ] `timbers log -i` interactive mode (huh forms)
- [ ] `timbers log --batch` for catch-up documentation
- [ ] `timbers log --template` for common patterns
- [ ] Full entry schema with all optional fields
- [ ] Beads/Linear adapter enrichment
- [ ] Beads post-close integration hooks

### Milestone 4 — Advanced (Post-MVP)

- [ ] `timbers relink` for rebased history
- [ ] Merge semantics for concurrent edits
- [ ] HTML site generation
- [ ] Cross-repo aggregation
- [ ] Custom narrate styles and voice training

---

## 12. Future Considerations

* **Relink tool** — Migrate notes after rebase/squash via patch-id heuristics
* **Release bundling** — Aggregate entries between tags for release notes
* **HTML dev diary** — Publishable narrative site from entries
* **Cross-repo** — Aggregate across org's repositories
* **Voice training** — Fine-tune narrative style from examples
* **MCP server** — Expose Timbers as MCP tools for broader agent integration

---

## Appendix A: Command Reference

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `timbers log` | Quick capture entry | `--why`, `--how`, `--auto`, `--minor`, `--batch`, `--work-item`, `--range` |
| `timbers pending` | Show undocumented commits | `--json`, `--count` |
| `timbers prime` | Context injection | — |
| `timbers status` | Repo/notes state | `--json` |
| `timbers narrate` | Generate narrative via LLM | `--style`, `--voice`, `--last`, `--range`, `--out`, `--model` |
| `timbers draft` | Generate draft JSON | `--range`, `--json` |
| `timbers write` | Write entry from JSON | `--replace` |
| `timbers enrich` | Add optional fields | `--decision`, `--risk`, `--tag`, `--last` |
| `timbers query` | Search entries | `--last`, `--tag`, `--work-item`, `--range`, `--json`, `--md` |
| `timbers export` | Generate markdown files | `--out`, `--preamble`, `--bundle` |
| `timbers notes init` | Configure sync | `--remote` |
| `timbers notes push` | Push to remote | — |
| `timbers notes fetch` | Fetch from remote | — |
| `timbers skill` | Emit content for building agent skill | `--format`, `--include-examples` |

## Appendix B: Comparison with Beads

| Aspect | Beads | Timbers |
|--------|-------|---------|
| Purpose | Issue tracking (what to do) | Development ledger (what was done) |
| Storage | JSONL in `.beads/` | Git notes on commits |
| Key command | `bd ready` | `timbers pending` |
| Quick action | `bd create "..."` | `timbers log "..." --why --how` |
| Context | `bd prime` | `timbers prime` |
| Sync | `bd sync` | `timbers notes push` |
| Schema | Issues with dependencies | Entries with evidence + meaning |
| Output | Issue lists, graphs | Narratives via `timbers narrate` |

**Complementary usage:** Beads tracks *what to do*; Timbers records *what was done and why*.

**Integrated workflow:**
```bash
# Beads: Track work
bd create "Fix auth bypass" --type bug
bd update bd-xxx --claim

# ... do the work ...

# Beads: Close issue
bd close bd-xxx

# Timbers: Document why (with session context!)
timbers log "Fixed auth bypass" \
  --why "Security audit revealed JWT parsing flaw; chose middleware approach per team discussion" \
  --how "Added validation in auth middleware" \
  --work-item beads:bd-xxx

# Timbers: Generate narrative
timbers narrate --last 5 --style devlog
```
