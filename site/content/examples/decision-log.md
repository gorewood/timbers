+++
title = 'Decision Log'
date = '2026-03-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Decision Log — March 2026

## ADR-1: Suppress Hooks During Interactive Git Operations

**Context:** Timbers hooks (post-commit, pre-push) run pending-commit checks and other validation. During interactive git operations — rebase, merge, cherry-pick, revert — these hooks created a deadlock: the agent was blocked by the pending check mid-rebase but couldn't log work or commit to clear it, because the rebase was still in progress. The agent was stuck with no path forward.

**Decision:** All hooks early-return during interactive git operations. `git.IsInteractiveGitOp()` checks for `.git` state files (e.g., `rebase-merge/`, `MERGE_HEAD`) and suppresses hook execution entirely when detected. The approach was validated by a code-review agent that caught two issues before commit: relative `gitDir` paths not resolved against the repo root (breaks when CWD isn't repo root), and redundant `GetPendingCommits` calls in `pending.go` after the operation check.

**Consequences:**
- Agents can rebase, merge, cherry-pick, and revert without getting deadlocked by timbers hooks
- Work done during interactive operations is invisible to timbers until the operation completes — a gap in tracking, but acceptable since the alternative was total blockage
- The state-file detection approach is git-implementation-dependent; if git changes its state file conventions, detection could silently break

## ADR-2: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** Entry anchor commits reference the SHA that was HEAD when the entry was created. After a squash-merge or rebase, those feature-branch SHAs disappear from `main`'s history. `HasPendingCommits` errored on stale anchors, which blocked hooks. `pending` dumped every reachable commit as "undocumented," and agents dutifully tried to re-document hundreds of already-documented commits, creating duplicates.

**Decision:** Graceful degradation: `HasPendingCommits` returns `false` (not an error) when it encounters a stale anchor. `pending` reports 0 actionable commits with a clear warning message instead of listing commits. `doctor` gained merge-strategy checks (`pull.rebase`, `merge.ff`) and stale-anchor detection to surface the root cause proactively.

**Consequences:**
- Hooks no longer block on stale anchors — agents can continue working after squash-merges without manual intervention
- Genuinely pending commits could be masked if stale-anchor detection has a false positive, but `doctor` provides a diagnostic path
- Shifts the responsibility from "fix it now" (hard error) to "surface it for later" (warning + diagnostics), which matches the reality that stale anchors are a known consequence of squash/rebase workflows, not an actionable error

## ADR-3: Union Discovery Over Fallback for Range Entry Matching

**Context:** `--range` flag entry discovery used anchor-commit matching as the primary strategy, with a file-based `git diff --name-only` fallback that only triggered when anchor matching returned zero results. In partial-stale scenarios — where some entries had valid anchors and others had stale ones — the fallback never triggered because the result set wasn't empty. Entries with stale anchors were silently dropped.

**Decision:** Run both anchor-based and diff-based discovery unconditionally, union the results by entry ID, and deduplicate. Neither strategy is treated as primary or fallback.

**Consequences:**
- Partial-stale scenarios now return complete results — no silent entry drops regardless of merge strategy
- Slightly more work per query (two discovery passes instead of one), but entry sets are small enough that the cost is negligible
- Eliminates an entire class of "works in testing, fails in production" bugs where the fallback path was only exercised in the zero-result case

## ADR-4: File-Based Fallback for Squash-Merged Entry Discovery

**Context:** `query --range` used anchor-commit ancestry to find entries within a commit range. After squash-merge, the original feature-branch SHAs that entries reference no longer exist in `main`'s history, so ancestry-based matching found nothing — even though the entry files themselves were clearly present in the diff between the range endpoints.

**Decision:** Added `DiffNameOnly` to the `GitOps` interface. When anchor matching returns zero results, a fallback discovers entries via `git diff --name-only A..B -- .timbers/`, finding entry files that were introduced in the range regardless of whether their internal anchor SHAs are reachable.

**Consequences:**
- Entries are discoverable after squash-merge, making `--range` work regardless of merge strategy
- File-based discovery can't distinguish entries *about* the range from entries that *happen to be in* the range (e.g., an entry committed alongside unrelated code) — anchor matching is more semantically precise when it works
- The `GitOps` interface gained another method (`DiffNameOnly`), increasing the surface area for test mocking, but it's a clean single-responsibility addition

## ADR-5: Three-Voice Essay Structure Over Carmack .plan Style for Devblog Template

**Context:** The devblog template originally used a Carmack `.plan` style — stream-of-consciousness narration of what happened. Generated posts read as flat recaps without clear takeaways or narrative arc.

**Decision:** Replaced with a structured three-voice approach: Storyteller (narrative arc), Conceptualist (abstractions and patterns), and Practitioner (concrete details and code). Posts follow a Hook → Work → Insight → Landing scaffolding. Added a no-headers constraint after test generation showed that section headers broke the essay feel.

**Consequences:**
- Generated essays have clearer narrative structure with identifiable takeaways, rather than chronological brain-dumps
- The three-voice structure is more opinionated — it constrains the tone in ways that may not suit all types of development work (e.g., pure bugfix sessions may not have enough "Conceptualist" material)
- The template is more complex to maintain and tune, but the quality improvement in output justified the investment
- Existing site posts were regenerated to maintain consistency, creating a one-time churn cost
