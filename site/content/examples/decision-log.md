+++
title = 'Decision Log'
date = '2026-02-13'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Flat Files Over Git Notes for Entry Storage

**Context:** Timbers originally stored entries as git notes — a built-in Git mechanism for attaching metadata to commits. This worked for single-branch workflows but caused merge conflicts in concurrent worktrees: git notes are a single ref that two branches can't modify independently. The alternatives were: keep notes and document the worktree limitation, add merge tooling for notes, or pivot to a different storage model entirely.

**Decision:** Pivoted to individual JSON files in `.timbers/YYYY/MM/DD/`, where each entry is a standalone file. This eliminates merge conflicts because independent files don't collide. The `GitOps` interface was simplified from 8 methods to 4 (`HEAD`, `Log`, `CommitsReachableFrom`, `GetDiffstat`) since notes-specific operations were no longer needed.

**Consequences:**
- Positive: Entries are inherently merge-safe — concurrent worktrees can't conflict on independent files
- Positive: Atomic writes via temp-file-plus-rename pattern prevent partial entries on crash
- Positive: Standard `git push/pull` syncs entries — no special `notes push/fetch` commands needed
- Positive: Simplified the GitOps interface by half, reducing testing surface
- Negative: Entry commits appear in mainline history (required filtering in `GetPendingCommits` to avoid a chicken-and-egg loop)
- Negative: More files in the repository — potentially thousands of small JSON files over time

---

## ADR-2: YYYY/MM/DD Directory Buckets Over Flat Storage

**Context:** After pivoting to file-based storage, all entries lived in a flat `.timbers/` directory. At high commit volumes (100+ commits/day over months), a single directory would accumulate thousands of files, degrading filesystem performance on some platforms.

**Decision:** Adopted a `YYYY/MM/DD` directory layout parsed from the entry's timestamp. Replaced `ReadDir` with `WalkDir` for recursive discovery. Added `atomicWrite` helper and recursive cleanup in `uninstall`.

**Consequences:**
- Positive: Directory sizes stay bounded — at most a handful of entries per day-bucket
- Positive: Sparse layout works well with git — most directories are small
- Positive: Natural chronological browsing when inspecting `.timbers/` manually
- Negative: More complex path construction and cleanup logic
- Negative: WalkDir is marginally slower than ReadDir for small datasets

---

## ADR-3: MCP Server Over Per-Editor CLI Wrappers

**Context:** Timbers needed to integrate with multiple AI coding environments — Claude Code, Cursor, Windsurf, Gemini CLI, Kilo Code, Continue, Codex, and others. The choice was between writing per-editor integration wrappers (each calling the CLI differently) or implementing a single MCP (Model Context Protocol) server that all MCP-compatible editors can consume.

**Decision:** Built an MCP server (`timbers serve`) with 6 tools over stdio transport. Handlers call `internal/` packages directly, sharing filter functions with the CLI by moving them to `internal/ledger/`.

**Consequences:**
- Positive: One implementation serves all MCP-compatible editors — highest leverage investment
- Positive: Read tools annotated with `idempotentHint` enable client-side caching and retry optimization
- Positive: Shared filter functions between CLI and MCP prevent behavioral drift
- Negative: Depends on go-sdk v1.3.0 which has a known limitation — `google/jsonschema-go` produces `["null","array"]` for Go slices with no override
- Negative: Editors without MCP support still need CLI-based integration

---

## ADR-4: Registry Pattern Over Switch Statement for Agent Environments

**Context:** With the goal of supporting multiple AI agent environments (Claude Code, Gemini CLI, Cursor, Windsurf, Codex), the `init`, `doctor`, `setup`, and `uninstall` commands needed to work across all environments. The choice was between a switch statement that dispatches per environment (centralized, every addition touches multiple files) and a registry pattern where each environment self-registers.

**Decision:** Chose registry pattern with an `AgentEnv` interface (`Detect`/`Install`/`Remove`/`Check` methods). Each environment registers itself via `init()` in its own file. `ClaudeEnv` wraps existing setup functions as the reference implementation.

**Consequences:**
- Positive: Adding a new agent environment is a single-file task — implement the interface, register in `init()`
- Positive: No changes needed to existing code when adding environments
- Positive: Interface methods mirror what `doctor`/`setup`/`init` already need — natural fit
- Positive: Backward-compatible JSON keys preserved (`claude_installed`, `claude_removed`) while step names became generic
- Negative: Registry adds indirection — `AllAgentEnvs()` iterates instead of direct function calls
- Negative: Stable sort ordering needed to keep deterministic output across runs

*Notes from the entry:* "Debated registry vs switch. Registry adds one file per environment but each is self-contained with no changes needed to existing code. Switch would centralize all environments but every addition touches multiple files. Registry won because the goal is to make adding Gemini/Codex later a single-file task."

---

## ADR-5: Notes Field with "Journey vs Verdict" Coaching Over Structured Deliberation Format

**Context:** The `--why` field captures design decisions, but doesn't have room for the full reasoning process — alternatives explored, dead ends encountered, trade-offs weighed. A new field was needed, but the design question was whether to impose structure (e.g., mandatory sections for Alternatives, Decision, Rationale) or keep it free-form with coaching.

**Decision:** Added `notes` as an optional free-form string field, coached by BAD/GOOD examples in `prime` output. The framing: `--why` captures the verdict (one sentence), `--notes` captures the journey (thinking out loud). Coaching by question ("what did you try that didn't work?") rather than structure, following the proven pattern from why-field coaching.

**Consequences:**
- Positive: Agents produce natural "thinking out loud" content instead of mechanical form-filling
- Positive: Optional field means no ceremony overhead for routine work — use selectively
- Positive: Decision-log template can pull from both `why` (verdict) and `notes` (context) for richer ADRs
- Positive: Why coaching was tightened to explicitly differentiate from notes, preventing field confusion
- Negative: Free-form field means inconsistent quality across entries — some notes will be more useful than others
- Negative: Coaching must be in `prime` output (seen every session) to be effective — agents don't read docs

---

## ADR-6: Filtering Ledger-Only Commits in GetPendingCommits Over Caller-Side Filtering

**Context:** After pivoting to file-based storage, every `timbers log` command creates a commit containing the `.timbers/` entry file. This caused a chicken-and-egg problem: `pending` would always report the most recent entry commit as undocumented work, since it was a commit without a corresponding entry documenting *that specific commit*.

**Decision:** Filter ledger-only commits inside `GetPendingCommits` using `git.CommitFiles()` and an `isLedgerOnlyCommit` helper that checks whether all files in a commit are under `.timbers/`. Safe default: if `CommitFiles` fails or returns unknown files, keep the commit visible. Applied to all 3 return paths in `GetPendingCommits`.

**Consequences:**
- Positive: All 7 callers of `GetPendingCommits` benefit from the fix without individual changes
- Positive: Safe default means unknown commits are never hidden — false positives over false negatives
- Negative: Adds a `diff-tree` call per commit when checking pending — minor performance cost
- Negative: `diff-tree` returns empty for merge commits and root commits, which is a known limitation
