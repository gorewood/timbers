# Timbers — Development Ledger + Narrative Export (Draft Spec)

**Working name:** Timbers (CLI: `timbers`)
**Tagline:** A Git-native development ledger that captures *what/why/how* as structured records attached to history, and exports LLM-ready markdown for narratives.

## 0. Positioning

### 0.1 One-sentence description

Timbers is a Go CLI that turns your Git history into a durable, structured “development ledger” by harvesting objective facts from Git (commit ranges, changed files, diffstats, tags) and pairing them with concise human/agent-authored rationale (what/why/how, decisions, verification) stored as portable **Git notes** that sync to remotes, then exporting frontmatter-rich markdown packets (optionally with persona prompts) for changelogs, stakeholder updates, and narrative devlogs.

### 0.2 Paragraph description

Timbers is for solo devs and small teams shipping with AI agents—where code volume is high, commits are frequent, and humans act more like architects/PMs than day-to-day implementers. Git alone is excellent at *what* changed but weak at preserving *why* and *how* in a way that’s readable to outsiders, and leaving piles of markdown in the repo tends to rot. Timbers keeps a clean, Git-native ledger: it automatically collects hard evidence from Git (commit sets, diffstat + path rollups, tags/releases, branch/merge context) and, when available, enriches that evidence with tracker context (Beads/Linear/etc.), while requiring the dev agent (or human) to author the high-signal meaning fields—what/why/how, decisions/tradeoffs, and verification—so the record stays accurate and intentional. From that single canonical source, you can generate internal dev-facing change trails, PM/manager summaries, customer-friendly release notes, and entertaining “dev diaries” in bespoke voices and personas.

### 0.3 30-second elevator pitch

If you’re building with AI agents, you ship a lot of code fast—and the rationale evaporates. Timbers fixes that by creating a Git-native development ledger. **Table stakes:** it reads Git as the ground truth (commit ranges, messages and trailers, changed paths, diffstats, tags/releases, merge commits) and stores structured what/why/how entries as **Git notes** that are configured to **fetch/push to remotes**, so the ledger travels with the repo without adding noisy files. **Optional:** it can integrate deeply with Beads or other trackers like Linear (titles, status, relationships), reference ADRs/specs/plans when they exist, and export LLM-ready markdown packets with frontmatter and an optional prompt preamble to drive anything from strict business release notes to whimsical character-driven dev narratives. Agents author the meaning; Timbers automates evidence collection, syncing, querying, and export.

## 1. Objectives

### 1.1 Primary goals

1. **Capture a durable, queryable record of development** that complements Git commit history.
2. Record **What / Why / How** with tight discipline, optimized for *agent implementation and usage*.
3. Keep the repository **clean of drifting docs**: the canonical ledger must not create long-lived noise in the working tree.
4. Support **archaeology and observability**: reconstruct “why decisions were made” and “how we got here.”
5. Export **markdown packets with frontmatter** and optional prompt preambles to feed downstream narrative generation.

### 1.2 Non-goals (MVP)

* Timbers does **not** directly invoke an LLM to generate narratives (downstream operation).
* Timbers does **not** attempt deep semantic analysis of code diffs beyond basic stats and path summaries.
* Timbers does **not** aim to be a full issue tracker or project-management system.
* Timbers does **not** require any tracker (Beads/Linear/etc.); however, **when a tracker is configured, it is treated as a first-class enrichment source** (not an afterthought), including stable IDs, titles/status, and (where available) relationships.

### 1.3 Design constraints

* **Git is required.**
* **Beads is optional** but tightly integrated when present.
* Must be implementable as a **Go CLI** with deterministic behavior.
* Must be **token-efficient** for agent workflows: the CLI must gather structured facts; the agent adds thoughtful rationale.

---

## 2. Core Approach

### 2.0 Evidence vs Meaning (anti-hallucination boundary)

Timbers explicitly separates:

* **Evidence (machine-collected, required):** Git-derived facts such as commit sets/ranges, diffstat, changed path rollups, branch/merge context, tags/releases, and any structured commit trailers.
* **Meaning (agent/human-authored, required):** concise *What/Why/How* plus decisions/tradeoffs and verification notes.
* **Context enrichment (optional, strongly integrated when enabled):** tracker data (Beads/Linear) and referenced artifacts (ADRs/specs/plans).

The CLI must gather evidence deterministically; the agent must supply meaning thoughtfully and accurately.

### 2.1 Canonical storage: Git notes

### 2.1 Canonical storage: Git notes

Timbers stores canonical records in **Git notes** to avoid polluting the repo tree and to maintain strong linkage to Git history.

* Canonical notes ref (default):

  * `refs/notes/timbers`
* Optional thin backlink ref:

  * `refs/notes/timbers-link`

**Rationale:** Git notes attach data to commits without changing commit IDs; they preserve clean trees and support archaeology.

### 2.2 Event model

Timbers is an **event ledger**. Records are stored as structured JSON with a stable schema.

Event kinds:

* `entry` — primary narrative unit (default: **per work batch**, e.g., bead close / merge / issue iteration / time window)
* `commit_link` — optional thin pointer attached to each commit in a workset for fast reverse lookup

**Default unit (alignment with agentic dev):** Timbers optimizes for *batch-first authoring* (work-item close, merge, or iteration) rather than per-commit prose. Per-commit linkage is optional metadata.

### 2.3 Anchor strategy

Each `entry` is attached as a Git note to one **anchor commit**.

**Anchor selection (default):**

* explicit `--anchor <sha>` OR
* `HEAD` if not specified

Common patterns:

* anchor is the **bead close commit** (preferred)
* anchor is a **merge commit** that lands a branch
* anchor is a **release tag commit**

Optional: attach `commit_link` notes to all commits in the workset to point to the anchor entry.

---

## 3. Schema

### 3.1 Schema overview

All documents stored in Git notes MUST be JSON and MUST validate against the schema.

* Devlog schema: `timbers.devlog/v1`
* Export schema: `timbers.export/v1`

**Versioning rule:** breaking changes require new schema version; old versions remain readable.

### 3.2 Common envelope (required for all kinds)

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",
  "created_by": {
    "actor": "human|agent",
    "name": "Bob",
    "tool": "timbers",
    "tool_version": "0.1.0"
  },
  "repo": {
    "remote": "origin",
    "default_branch": "main"
  }
}
```

### 3.3 `entry` kind (canonical narrative record)

#### 3.3.1 Required fields

* `schema`, `kind`, `id`
* `created_at`, `updated_at`, `created_by`
* `work_items[]` (may be empty)
* `workset.anchor_commit`
* `workset.commits[]` (>= 1)
* `summary.what`, `summary.why`, `summary.how`

#### 3.3.2 Full shape

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "entry",
  "id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "created_at": "2026-01-15T15:04:05Z",
  "updated_at": "2026-01-15T15:04:05Z",
  "created_by": {"actor":"agent","name":"TimbersAgent","tool":"timbers","tool_version":"0.1.0"},
  
  "work_items": [
    {"system":"beads","id":"B-1427","title":"Fix heightfield alignment","status":"closed"},
    {"system":"linear","id":"LIN-382","title":"Terrain collider parity","status":"in_progress"}
  ],

  "workset": {
    "anchor_commit": "8f2c1a9d7b0c...",
    "commits": ["8f2c1a9d7b0c...","c11d2a...","a4e9bd..."],
    "range": "c11d2a..8f2c1a",
    "changed_paths_top": ["src/engine/terrain", "docs/physics"],
    "diffstat": {"files": 6, "insertions": 241, "deletions": 88}
  },

  "summary": {
    "what": "...",
    "why": "...",
    "how": "..."
  },

  "decisions": [
    {
      "decision": "Use normalized grid orientation; apply single rotation at collider creation",
      "alternatives": ["Rotate visual mesh instead", "Swap axes at generation only"],
      "tradeoffs": "Keeps physics canonical; requires migration for existing terrains"
    }
  ],

  "quality": {
    "risk": "low|medium|high|unknown",
    "tests": [
      {"kind":"unit|integration|e2e|manual","name":"...","status":"added|updated|existing|none"}
    ],
    "verification": ["..."]
  },

  "references": {
    "adrs": ["ADR-0012"],
    "docs": [
      {"path":"docs/terrain/heightfield-notes.md","kind":"scratch|design|spec|readme","kept":false}
    ],
    "prs": [],
    "issues": []
  },

  "tags": ["terrain","physics"],

  "provenance": {
    "inputs": {
      "git_describe": "v0.9.1-14-g8f2c1a",
      "branch": "feature/terrain-collider"
    }
  },

  "links": {
    "supersedes": [],
    "related_entry_ids": []
  }
}
```

#### 3.3.3 Notes on schema

* `work_items` is the integration point for Beads/Linear/etc.
* `decisions` is ADR-like, but lightweight; formal ADRs may be referenced under `references.adrs`.
* `references.docs[].kept=false` indicates ephemeral docs; the system may optionally archive them externally later.
* `links.supersedes` supports rebase/squash/cherry-pick migrations.

#### 3.3.4 ADR promotion policy (recommended)

Timbers supports two complementary layers of decision recording:

**A) Embedded decisions (default, always):**

* Every `entry` SHOULD include `decisions[]` when any non-trivial tradeoff or approach choice was made.
* These are the *high-frequency* “why” records tied directly to a work batch and its evidence.

**B) Standalone ADRs (promote when architectural):**
Create or update a standalone ADR and reference it via `references.adrs[]` when a decision:

* sets a long-lived constraint or policy (“from now on we do X”),
* affects multiple subsystems or broad direction,
* has meaningful risk/cost or non-obvious tradeoffs,
* is likely to be searched independently of a specific work batch,
* should remain stable across refactors.

**Storage location (recommended default):**

* ADRs live as markdown in-repo (e.g., `docs/adr/ADR-0012-<slug>.md`) for reviewability and discoverability.

**Linkage rule:**

* If a standalone ADR exists, the `entry.decisions[]` SHOULD include a brief summary and the ADR ID MUST appear in `references.adrs[]`.

**MVP stance:**

* Timbers does not manage ADR authoring lifecycle beyond referencing.

**Future helper (non-MVP):**

* `timbers adr draft --from-entry <id>` to scaffold an ADR markdown file from an `entry.decisions[]` item, preserving evidence links and rationale.

### 3.4 `commit_link` kind (optional)

Thin note attached to each commit in the workset.

```json
{
  "schema": "timbers.devlog/v1",
  "kind": "commit_link",
  "id": "tbl_8f2c1a9d",
  "created_at": "2026-01-15T15:04:05Z",
  "beads": ["B-1427"],
  "work_items": [{"system":"linear","id":"LIN-382"}],
  "anchor_entry_id": "tb_2026-01-15T15:04:05Z_8f2c1a",
  "tags": ["terrain","physics"]
}
```

---

## 4. Git Notes Transport and Merging

### 4.1 Making notes move to remotes (required)

Timbers MUST support first-run configuration that ensures notes are fetched and pushed.

#### 4.1.1 Fetch configuration

Timbers should add fetch refspec(s) to `.git/config` for the chosen remote (default `origin`):

* `+refs/notes/timbers:refs/notes/timbers`
* `+refs/notes/timbers-link:refs/notes/timbers-link` (if enabled)

This can be done via:

* `git config --add remote.origin.fetch "+refs/notes/timbers:refs/notes/timbers"`

Timbers should detect if already present.

#### 4.1.2 Push behavior

Timbers should provide:

* `timbers notes push` (push notes refs)
* `timbers write --push-notes` (push after write)

Default push target:

* `git push origin refs/notes/timbers`
* (and link ref if enabled)

Config options:

* `notes.remote` (default `origin`)
* `notes.ref` (default `refs/notes/timbers`)
* `notes.link_ref` (default `refs/notes/timbers-link`)
* `notes.auto_push` (default `false`)
* `notes.auto_fetch` (default `true`)

### 4.2 Merge strategy (required)

Notes can diverge across collaborators and branches.

#### 4.2.1 Canonical merge model

* Each anchor commit has at most **one Timbers note blob** in the canonical ref.
* That blob may contain:

  * a single `entry` (MVP), OR
  * an object `{ "entries": [ ... ] }` (future)

**MVP decision:** store **exactly one `entry` per anchor commit** to reduce complexity. If multiple entries are needed, create multiple anchors or store a bundle in the future.

#### 4.2.2 Conflict detection

When writing a note to a commit that already has a Timbers note:

* `timbers write` MUST fail unless one of:

  * `--merge` is provided
  * `--replace` is provided

#### 4.2.3 Merge semantics (`--merge`)

If merging JSON documents:

* Must merge by `id` (stable entry id)
* Scalars:

  * default: keep existing unless `--merge-prefer incoming`
* Arrays:

  * append unique by stable key:

    * decisions: hash of `decision`
    * tests: `kind+name`
    * tags: exact string
* Maintain `history[]` if enabled:

  * record prior versions with timestamp and actor

#### 4.2.4 Timeline/stream consistency

Timbers should be able to reconstruct a chronological stream by:

* sorting entries by `created_at` then `id`
* using `workset.anchor_commit` ordering to map onto Git history when needed

### 4.3 Rebase/squash/cherry-pick considerations (future but planned)

Notes attach to commit IDs; rewritten history loses attachment.

Planned commands:

* `timbers relink --old-range A..B --new-range C..D`:

  * match commits by patch-id or subject+diffstat heuristics
  * copy notes to new commits
  * add `links.supersedes` / `related_entry_ids`

---

## 5. CLI UX (Agent-Optimized)

### 5.1 Principles

* CLI commands must produce **structured JSON** for the agent to consume.
* CLI must gather all Git facts: commit lists, diffstat, changed path rollups, existing notes.
* Agent supplies only “meaning fields”: What/Why/How, decisions, verification.

### 5.2 Command set (MVP)

#### 5.2.1 `timbers status --json`

Outputs:

* repo info, branch, head sha
* configured notes refs + remote
* whether notes refs are being fetched
* last N commits and whether they have commit_link notes

#### 5.2.2 `timbers draft --range <A..B> [--work-item <sys:id>] [--anchor <sha>] --json`

Produces a draft `entry` JSON containing:

* computed `workset` (commits, range, diffstat, changed_paths_top)
* detected work items from commit trailers (if present)
* loads existing Timbers note on anchor (if any) for patching

#### 5.2.2a Default work-batch selection policy (MVP)

Timbers SHOULD support multiple batch-selection strategies, chosen deterministically and surfaced in `provenance.inputs`:

**A) Explicit range (highest priority):**

* If `--range` is provided, use it exactly.

**B) Work-item scoped (when `--work-item` provided):**

* Prefer commits that reference the work-item in trailers (e.g., `Bead:` / `Work-Item:`).
* If the configured tracker adapter can provide boundaries (e.g., bead opened/closed timestamps, issue lifecycle events), Timbers MAY further refine the range, but must always output the final commit list explicitly.
* If no commits are discoverable via trailers, fall back to (C) with a warning field in draft output (e.g., `provenance.warnings`).

**C) Since last entry (recommended default when neither range nor work-item is provided):**

* Identify the most recent reachable anchor commit on the current branch that has a Timbers `entry` note in `refs/notes/timbers`.
* Use the range `(last_anchor..HEAD)`.
* If none exists, fall back to (D).

**D) Last N commits (safe fallback):**

* Default `N=10` unless configured.

**Additional rules:**

* Always exclude merge commits from the commits list unless `--include-merges` is set (because merges are often anchors, not content).
* Always include `workset.anchor_commit` in `workset.commits`.
* Always compute and emit `workset.commits[]` explicitly (no implicit ranges in the canonical record).

#### 5.2.3 `timbers write --draft <path> [--anchor <sha>] [--attach-links] [--push-notes] [--merge|--replace]`

* validates JSON
* writes canonical note to anchor commit in `refs/notes/timbers`
* optionally writes commit_link notes to each commit in workset
* optionally pushes notes refs
* prints receipt: entry id, anchor sha, note refs updated

#### 5.2.4 `timbers query [filters...] --json|--md`

Filters:

* `--work-item beads:B-1427`
* `--tag physics`
* `--path-prefix src/engine/terrain`
* `--range A..B`
* `--since 2026-01-01 --until 2026-02-01`

Outputs:

* normalized stream of entries (JSON)
* or a markdown report (MD)

#### 5.2.5 `timbers export --range <A..B> --out <dir> [--bundle release|week|milestone] [--preamble <template>]`

Produces:

* one markdown file per entry (default)
* optional bundle file(s)
* index file

#### 5.2.6 `timbers notes init [--remote origin] [--enable-links]`

* sets fetch refspec(s)
* verifies push permissions
* optionally sets `notes.auto_fetch`

### 5.3 Commit trailer enforcement (recommended)

Optional hook generator:

* `timbers hooks install`

Commit-msg hook enforces minimal metadata:

* `Work-Item:` or `Bead:` trailer is recommended (configurable)

Rationale: ensures join keys even if notes are written later.

---

## 6. Markdown Export Format

### 6.1 File naming

Default:

* `YYYY-MM-DD__<entry-id>__<short-anchor>.md`
* directory structure optional:

  * by year/month
  * by work item

### 6.2 Frontmatter (required)

Example:

```yaml
---
schema: timbers.export/v1
kind: entry
id: tb_2026-01-15T15:04:05Z_8f2c1a
date: 2026-01-15
repo: <repo-name>
anchor_commit: 8f2c1a9d7b0c...
commit_range: c11d2a..8f2c1a
work_items:
  - system: beads
    id: B-1427
  - system: linear
    id: LIN-382
tags: [terrain, physics]
risk: medium
paths_top: ["src/engine/terrain", "docs/physics"]
---
```

### 6.3 Body structure (stable headings)

```md
# Summary

**What:** ...

**Why:** ...

**How:** ...

## Decisions
- ...

## Verification
- ...

## References
- Anchor commit: ...
- Commits: ...
- Work items: ...
- ADRs: ...
```

### 6.4 Optional prompt preamble

Export may include a preamble block for downstream narrative generation:

```md
<!-- NARRATIVE_PROMPT
Project context:
- <big picture, constraints, terminology>

Preferred style:
- business | changelog | robotic | whimsical | character

Persona (optional):
- name: ...
- motive: ...
- setting: ...

Output constraints:
- length: ...
- include: ...
- exclude: ...
-->
```

Preamble templates:

* `business`
* `changelog`
* `robotic`
* `whimsical`
* `character:<file>` (user-supplied)
* `project:<file>` (user-supplied)

---

## 7. Integrations

### 7.0 Integration philosophy

* **Git is the source of truth** for evidence and timeline.
* **Trackers are optional, but first-class when enabled.** If a tracker adapter is configured, Timbers should:

  * validate IDs,
  * fetch/enrich titles and status where possible,
  * optionally collect relationship/context fields (e.g., parent/child, dependencies),
  * support issue-scoped accumulation over multiple passes.

### 7.1 Tracker adapters (Beads, Linear, others)

Timbers models all trackers as `work_items` with a unified shape:

* `system` (e.g., `beads`, `linear`, `github`)
* `id` (stable tracker ID)
* optional `title`, `status`, `url`, `relations` (where available)

#### 7.1.0 Tracker adapter contract (execution-ready)

When a tracker is enabled, Timbers MUST treat it as first-class enrichment. Implement adapters behind a stable internal interface so additional trackers can be added without modifying ledger semantics.

**Adapter capabilities (MVP):**

* **ValidateID(system, id) -> (normalizedID, ok, err)**
* **HydrateWorkItem(system, id) -> WorkItem**

  * Returns `title`, `status`, optional `url`, optional labels/tags.
* **SuggestCommitCandidates(workItem) -> []CommitRef (optional)**

  * May be implemented as trailer-based heuristics only in MVP.

**Optional capabilities (non-MVP but planned):**

* **GetRelations(workItem) -> Relations** (parent/child, dependencies)
* **GetLifecycleBoundaries(workItem) -> {opened_at, closed_at}**
* **Search(query) -> []WorkItem** (for NL-assisted archaeology)

**Offline + determinism requirements:**

* Adapters MUST support a deterministic mode:

  * If network access is unavailable or disabled, Timbers must still function using:

    * commit trailers,
    * cached work-item snapshots (if present),
    * user-provided flags.
* When online enrichment is used, Timbers MUST record:

  * `provenance.inputs.tracker_enrichment=true`
  * adapter version and fetch timestamp (in `provenance`)

**Caching:**

* Timbers MAY maintain a local cache under `.timbers/cache/` (gitignored) keyed by `system:id` storing last known `title/status/url`.
* Canonical ledger records must not depend on cache to be interpretable; cache only improves draft UX.

#### 7.1.1 Beads (optional, first-class when enabled)

* Detect Bead IDs from commit trailers (e.g., `Bead: B-1427`).
* Support `timbers draft --work-item beads:B-1427`:

  * enrich `work_items[]` (title/status) when accessible,
  * prefer bead-scoped commit selection when bead data provides a boundary.
* Future: adapter may read local Beads files or call a Beads CLI/API if present (implementation TBD).

#### 7.1.2 Linear (optional, first-class when enabled)

* Detect Linear IDs from commit trailers (e.g., `Work-Item: LIN-382`).
* Support issue-scoped accumulation:

  * `timbers draft --work-item linear:LIN-382 --since last`
  * updates a single logical narrative entry over time (requires `links`/history strategy)
* Future: Linear adapter can enrich title/status/labels and (optionally) relationships.

#### 7.1.3 Other trackers (future)

* GitHub Issues, Jira, etc. can be supported via the same adapter interface.

### 7.2 Spec-driven dev tools and agent planning formats (optional)

Goal: make Timbers compatible with workflows like Claude/Windsurf/Kilo planning files by **referencing** them, not owning them.

Planned features:

* Import references to spec artifacts:

  * `references.docs[].kind = spec|plan`
* Export a “planning packet”:

  * `timbers export --format md --include planning`

Out-of-scope for MVP: direct parsing of each tool’s proprietary formats.

## 8. Quality, Reliability, and Testing

### 8.1 Determinism

* All commands that output JSON must be stable across runs given identical repo state.
* IDs must be stable within an operation.

### 8.2 Validation

* Strict JSON schema validation on write.
* Fail fast with actionable errors.

### 8.3 Test plan (minimum)

* Unit tests:

  * schema validation
  * merge semantics
  * export formatting
* Integration tests (temp git repos):

  * init notes refs, write entry, push/fetch notes
  * attach commit_links
  * query by range/tag/path
  * merge conflict simulation (two clones)
* Future integration tests:

  * rebase/squash relink workflows

---

## 9. Open Questions (must resolve for MVP finalization)

1. **One entry per anchor commit vs bundle-per-anchor**: MVP uses one entry per anchor. Is that acceptable, or do we need multiple entries on a single anchor immediately?
2. **ID generation**: should `id` be time+shortsha (as shown) or UUID? (Shortsha is human-friendly but time/ordering assumptions must be clear.)
3. **Commit range selection**: for `draft --range`, do we default to last N commits, since last entry, or since last tag?
4. **Commit link notes**: do we enable by default? (Great for query speed, but doubles note writes.)
5. **Auto-push notes**: default off to avoid surprises, but agents may want it on. Final default?
6. **Work-item detection standard**: commit trailers keys (`Bead`, `Work-Item`, `Refs`)—finalize canonical names.
7. **Handling private repos / multiple remotes**: do we support selecting remote per command?
8. **Security/privacy for prompt preambles**: should export support redaction rules for secrets or internal terms?

---

## 10. Future Considerations

* **Relink tool** for rewritten history (patch-id heuristics).
* **Bundle and release artifacts**:

  * release notes generation directly from Timbers entries + conventional commits.
* **HTML site generation**:

  * publishable dev diary.
* **Cross-repo aggregation**:

  * multi-repo story for an org.
* **External archive of ephemeral docs**:

  * optional store (S3, GDrive, etc.) keyed by entry id.
* **Pluggable integrations**:

  * Beads CLI adapter, Linear API adapter, GitHub issues adapter.

---

## 11. Proposed MVP Milestones

### Milestone 1 — Ledger core

* `notes init`
* `draft` (range + anchor)
* `write` (entry)
* `notes push/fetch`

### Milestone 2 — Query + export

* `query` (range/tag/path/work-item)
* `export` (md + frontmatter + optional preamble)
* example preamble templates

### Milestone 3 — Optional enhancements

* `commit_link` notes
* hooks installer
* merge semantics + history retention

---

## 12. Agent Execution Contract (for coding agents)

When implementing or using Timbers:

1. **Do not infer** Git state via ad-hoc parsing; call `timbers` subcommands and consume JSON.
2. The agent is responsible for authoring accurate **What/Why/How** and decisions based on evidence:

   * commit subjects
   * diffstat + changed paths
   * relevant specs/ADRs
   * work item context (Beads/Linear) when available
3. The agent must never fabricate tests, decisions, or verification steps; if unknown, set to `unknown` or omit optional fields.
4. All writes must be validated; on failure, agent must correct data and retry deterministically.
