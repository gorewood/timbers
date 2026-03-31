+++
title = 'Decision Log'
date = '2026-03-31'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log — Timbers 2026-03-23 to 2026-03-31

## ADR-1: File-Based Fallback for Entry Discovery After Squash Merge

**Context:** Entry `anchor_commit` fields reference feature-branch SHAs. After squash-merge into main, those SHAs no longer appear in main's commit history, so `query --range` returned zero results even though the entry files were present in the diff between the two range endpoints.

**Decision:** Added a `git diff --name-only A..B -- .timbers/` fallback path in `entry_filter.go`. When anchor-based commit-ancestry matching returns zero results, the system discovers entries by checking which `.timbers/` files were added or modified in the commit range. Exposed `EntryPathsInRange` in `storage.go` and `DiffNameOnly` in the `GitOps` interface.

**Consequences:**
- Entries are now discoverable regardless of merge strategy (squash, rebase, fast-forward)
- Introduces a secondary discovery mechanism that must be maintained alongside anchor-based matching
- The fallback only triggers on zero anchor matches, which created a gap addressed in ADR-2

## ADR-2: Union Discovery Over Fallback for `--range` Partial-Stale Anchors

**Context:** ADR-1's fallback only triggered when anchor-based matching returned zero results. In partial-stale scenarios — where some entries have valid anchors and others have stale feature-branch anchors — the fallback never fired. Entries with stale anchors were silently dropped from range queries.

**Decision:** Run both anchor-based and `git diff`-based discovery unconditionally, then union the results by entry ID. This replaces the zero-results-only fallback with an always-on dual-path approach.

**Consequences:**
- Eliminates the silent data loss from partial-stale anchor sets
- Every range query now does two git operations instead of one, with a deduplication step
- Simpler mental model: both paths always run, no conditional fallback logic to reason about

## ADR-3: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** After squash-merge, anchor tracking breaks — the last documented anchor SHA is no longer in HEAD's history. `pending` was dumping all reachable commits (hundreds) as false positives, `HasPendingCommits` was returning errors that blocked hooks, and agents were attempting to re-document already-covered commits, creating duplicates.

**Decision:** Chose graceful degradation: `HasPendingCommits` returns `false` (not an error) on `ErrStaleAnchor`, `pending` shows 0 actionable commits with a clear warning and guidance, and `doctor` gained merge-strategy checks (`pull.rebase`, `merge.ff`) and stale-anchor detection.

**Consequences:**
- Hooks no longer block during stale-anchor conditions — agents can continue working
- Agents stop creating duplicate entries from false-positive pending lists
- Trades correctness for availability: genuinely pending commits during a stale-anchor window won't be flagged until the anchor self-heals on the next `timbers log`
- `doctor` now proactively surfaces merge strategy misconfigurations before they cause stale anchors

## ADR-4: Reachability Check Over Existence Check for Stale Anchor Detection

**Context:** After `git pull --rebase`, old commit SHAs linger in the object store for ~2 weeks before garbage collection. The existing stale-anchor detection only fired when a SHA was completely absent from the object store. This meant `git log` succeeded with phantom results — commits that existed as objects but weren't in HEAD's ancestry — causing `timbers pending` to show already-documented commits as pending.

**Decision:** Added a `git merge-base --is-ancestor` check before `git log` in `GetPendingCommits`. If the anchor commit exists but isn't an ancestor of HEAD, it's treated as stale. Added `IsAncestorOf` to the `GitOps` interface.

**Consequences:**
- Stale anchors are detected immediately after rebase, not after a 2-week GC delay
- Eliminates the phantom-commit class of false positives entirely
- Adds one extra git operation per `pending` check (the ancestry test), but it's fast and avoids the far more expensive phantom enumeration
- Depends on git's `merge-base --is-ancestor` semantics, which are well-defined and stable

## ADR-5: Suppress Hooks During Interactive Git Operations

**Context:** Timbers hooks (pre-commit, post-commit) created a deadlock during rebase, merge, cherry-pick, and revert. The agent couldn't continue the rebase (blocked by the pending-commits check) and couldn't run `timbers log` to clear the check (can't commit mid-rebase). Code review caught two additional issues: relative `gitDir` paths broke in non-root CWD, and `pending.go` was checking mid-operation after already running `GetPendingCommits`.

**Decision:** Added `git.IsInteractiveGitOp()` which checks for `.git` state files (`rebase-merge/`, `rebase-apply/`, `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`). All hooks early-return during interactive operations.

**Consequences:**
- Eliminates the deadlock — agents can complete rebases and merges without manual intervention
- Commits made during interactive operations are temporarily invisible to timbers until the operation completes
- Relies on git's internal state file conventions, which are stable but technically implementation details
- The suppression is broad (all hooks, all interactive ops) rather than targeted, which is simpler but means no timbers instrumentation during these windows

## ADR-6: Separate Commits for Ledger Entries By Design

**Context:** Each `timbers log` creates a separate commit for the ledger entry file, roughly doubling the commit count visible in `git log`. Three alternatives were evaluated across two council rounds: (1) side-branch storage — rejected due to clone gaps and push regression; (2) amending the previous commit — rejected due to <20% success rate and breaking `filterLedgerOnlyCommits`; (3) staging entries and flushing in a batch — viable but unnecessary complexity.

**Decision:** Kept separate commits as the intentional design. The `filterLedgerOnlyCommits` function depends on the separation to distinguish ledger-only commits from work commits. The git log noise is cosmetic — already mitigated for agents via filtering, and trivially filterable for humans (`git log --invert-grep --grep="timbers: document"`). Documented as a deliberate design choice rather than leaving it as an apparent wart.

**Consequences:**
- `filterLedgerOnlyCommits` works reliably because ledger commits are structurally distinct
- `timbers pending` can precisely identify which work commits lack documentation
- Git log shows ~2x the expected commit count, which may surprise new users
- Batch mode (`timbers log --batch`) provides graceful degradation when hooks are bypassed, grouping undocumented commits after the fact
