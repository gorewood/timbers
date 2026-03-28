+++
title = 'Decision Log'
date = '2026-03-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Decision Log — 2026-03-06 to 2026-03-28

## ADR-1: Suppress Hooks During Interactive Git Operations

**Context:** Timbers hooks (post-commit, pre-tool-use) run automatically during git operations. When an agent performed a rebase, merge, cherry-pick, or revert, the hooks fired mid-operation — the pending check blocked the agent from continuing the rebase, and `timbers log` couldn't commit mid-rebase. This created a deadlock where the agent could neither proceed nor document.

**Decision:** All hooks early-return during interactive git operations. A new `git.IsInteractiveGitOp()` function detects in-progress operations by checking for `.git` state files (e.g., `rebase-merge/`, `MERGE_HEAD`). Hooks silently skip rather than warn or error.

**Consequences:**
- Agents can complete rebases, merges, cherry-picks, and reverts without deadlock
- Work performed during interactive operations is invisible to hooks until the operation completes — a gap in real-time tracking, accepted as necessary
- Detection relies on `.git` state file conventions, which are stable across git versions but technically an implementation detail
- The pattern is conservative: any recognized state file causes all hooks to skip, even if only some hooks would deadlock

## ADR-2: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** After a squash merge, entry `anchor_commit` SHAs reference feature-branch commits that no longer exist in `main`'s history. This caused `pending` to dump every reachable commit as "undocumented" (hundreds of false positives), and `HasPendingCommits` returned an error that blocked hooks. Agents interpreted the false positives literally — attempting to re-document already-covered work and creating duplicate entries.

**Decision:** Stale anchors are treated as a degraded-but-functional state, not an error. `HasPendingCommits` returns `false` (not an error) on `ErrStaleAnchor`. `pending` shows 0 actionable commits with a clear warning explaining the stale anchor, rather than listing unreachable commits. `doctor` gained merge-strategy and stale-anchor health checks to surface the root cause.

**Consequences:**
- Hooks no longer block agents after squash merges — the most common trigger for the stale anchor state
- Agents stop generating duplicate entries from false-positive pending lists
- Genuine pending commits are invisible while the anchor is stale — a real gap in tracking, but preferable to the cascade of duplicates
- The anchor self-heals on the next `timbers log` after a real commit, limiting the window of degraded tracking
- `doctor` can proactively warn about merge strategies (e.g., missing `pull.rebase`) that cause frequent stale anchors

## ADR-3: File-Based Fallback for Entry Discovery After Squash Merge

**Context:** `query --range` found entries by checking whether each entry's `anchor_commit` was an ancestor of commits in the requested range. After squash merge, feature-branch SHAs disappear from `main`'s history, so anchor-based matching found nothing — even though the entry files themselves were clearly present in the diff between the two range endpoints.

**Decision:** When anchor-based matching returns zero results for a `--range` query, fall back to `git diff --name-only A..B -- .timbers/` to discover entries by file presence in the commit range. This was exposed via `EntryPathsInRange` in `storage.go` and `DiffNameOnly` on the `GitOps` interface.

**Consequences:**
- Entries are discoverable after squash merge regardless of whether their anchor SHAs survive in the target branch's history
- The fallback treats "file appeared in this range" as a proxy for "entry belongs to this range" — less precise than anchor matching, but correct for the squash-merge case
- Adds a new method to the `GitOps` interface, expanding the git abstraction surface
- Only triggers on zero results, keeping the faster anchor-based path as the primary strategy

## ADR-4: Union Over Fallback for Anchor and Diff-Based Entry Discovery

**Context:** ADR-3 introduced file-based discovery as a fallback when anchor matching returned zero results. But a subtler case emerged: partial-stale ranges where *some* entries had valid anchors and others had stale feature-branch anchors. Because the fallback only triggered on zero matches, partial-stale cases silently dropped entries — anchor matching returned a non-zero (but incomplete) set, so the fallback never ran.

**Decision:** Run both anchor-based and diff-based discovery unconditionally for every `--range` query, then union and deduplicate results by entry ID. No fallback logic — both paths always execute.

**Consequences:**
- Partial-stale ranges now return complete results — no silent drops
- Slightly more work per query (two discovery passes instead of one), but the `git diff --name-only` call is fast enough to be negligible
- Simpler control flow: no conditional fallback branching, just "run both, merge"
- Any future discovery strategy can be added to the union without restructuring the fallback chain

## ADR-5: Three-Voice Essay Structure Over Stream-of-Consciousness for Devblog Template

**Context:** The devblog template originally used a Carmack `.plan`-style format — stream-of-consciousness technical writing. Generated posts came out as flat recaps of what was done, without narrative arc or insight extraction.

**Decision:** Replaced the stream-of-consciousness format with a structured three-voice essay: Storyteller (narrative and anecdotes), Conceptualist (patterns and abstractions), and Practitioner (concrete techniques and trade-offs). Posts follow a Hook → Work → Insight → Landing scaffolding. A no-headers constraint was added after test generation showed that section headers broke the essay flow.

**Consequences:**
- Generated posts have clearer narrative structure and extractable takeaways
- The voice framework gives the LLM concrete roles to inhabit, producing more varied and engaging prose than an open-ended "write a blog post" prompt
- More opinionated template — less flexibility for posts that don't fit the three-voice model
- The no-headers constraint forces cohesive prose but makes posts harder to skim
- Template complexity increased significantly; future template edits require understanding the voice/scaffolding system
