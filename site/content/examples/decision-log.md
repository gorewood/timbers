+++
title = 'Decision Log'
date = '2026-03-31'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Timbers Architectural Decision Log

## ADR-1: Separate Git Commits for Ledger Entries

**Context:** Each `timbers log` invocation creates its own git commit for the ledger entry file, doubling the visible commit count in `git log`. This was flagged as a potential UX concern — users may perceive the extra commits as noise. Three alternatives were evaluated across two council rounds: side-branch storage, amending the previous commit, and stage-and-flush on next real commit.

**Decision:** Keep separate commits. Side-branch was rejected due to clone gaps and push regression. Amend was rejected due to <20% success rate in practice and because it breaks `filterLedgerOnlyCommits`, which depends on ledger entries being isolated in their own commits. Stage-and-flush was viable but added unnecessary complexity for a cosmetic problem already mitigated by filtering.

**Consequences:**
- Positive: `filterLedgerOnlyCommits` works reliably — one simple check per commit
- Positive: Agents can filter ledger commits with `git log --invert-grep --grep="^timbers: document"`; humans get the same trivial filter
- Positive: Pending detection stays simple — each entry commit is self-contained
- Negative: ~2x commit count in raw `git log` output, which requires explanation for new users
- Negative: Pre-commit hook enforcement needed to maintain the 1:1 cadence

## ADR-2: Graceful Degradation Over Hard Errors for Stale Anchors

**Context:** After squash-merge or rebase, timbers' anchor commit (the SHA linking the last entry to git history) becomes unreachable. `HasPendingCommits` returned an error, blocking hooks. `pending` dumped all reachable commits as false positives. Agents interpreted the false positives as undocumented work and attempted to re-document, creating duplicates.

**Decision:** Treat stale anchors as a recoverable state, not an error. `HasPendingCommits` returns `false` (not error) on `ErrStaleAnchor`. `pending` reports 0 actionable commits with a clear warning and guidance. The anchor self-heals on the next real `timbers log` after a commit.

**Consequences:**
- Positive: Hooks no longer deadlock agents mid-workflow after squash-merge
- Positive: Agents stop creating duplicate entries for already-documented work
- Positive: Self-healing means no manual intervention needed — normal workflow resumes naturally
- Negative: Genuinely pending commits during a stale-anchor window are invisible until the anchor heals
- Negative: Users must understand the warning message to know why pending shows zero during the gap

## ADR-3: Suppress Hooks During Interactive Git Operations

**Context:** Timbers hooks (pre-commit pending check, session-start prime) ran during `git rebase`, `git merge`, `git cherry-pick`, and `git revert`. This created a deadlock: the agent couldn't continue the rebase (blocked by pending check) and couldn't log (can't commit mid-rebase).

**Decision:** Detect interactive git operations by checking `.git` state files (`rebase-merge/`, `rebase-apply/`, `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`) via `git.IsInteractiveGitOp()`. All hooks early-return during these operations.

**Consequences:**
- Positive: Agents can rebase, merge, cherry-pick, and revert without hook interference
- Positive: Detection is filesystem-based (checking `.git` state files), so it's fast and doesn't shell out
- Negative: Pending check is genuinely skipped during multi-step git operations — a commit during rebase won't be flagged
- Negative: Any new hook added in the future must remember to call `IsInteractiveGitOp()` or risk the same deadlock

## ADR-4: Reachability Check Over Object Existence for Anchor Validation

**Context:** After `git pull --rebase`, old commit SHAs linger in the object store for ~2 weeks (git's GC grace period). Stale anchor detection only fired when the SHA was completely absent. This meant `git log` succeeded with phantom results — commits reachable from the old SHA but not from HEAD — causing `pending` to show already-documented commits as undocumented. Reported by a user after rebase.

**Decision:** Added `git merge-base --is-ancestor` check before `git log` in `GetPendingCommits`. The anchor must be reachable from HEAD, not merely present in the object store. Added `IsAncestorOf` to the `GitOps` interface.

**Consequences:**
- Positive: Catches stale anchors immediately after rebase, not after 2-week GC delay
- Positive: Eliminates phantom pending commits entirely — no false positives from lingering objects
- Negative: Extra git call on every `pending` check (though `merge-base --is-ancestor` is fast)
- Negative: Expands the `GitOps` interface, adding a method that's only used for this one validation

## ADR-5: Union Discovery Over Fallback for `--range` Entry Resolution

**Context:** `--range` flag resolved entries by matching anchor commits to the given commit range. When anchors became stale (e.g., after rebase on a feature branch), a fallback used diff-based discovery. But the fallback only triggered when *zero* anchor matches were found. In partial-stale cases — some entries with valid anchors, some with stale feature-branch anchors — the fallback was short-circuited, silently dropping the stale entries.

**Decision:** Run both anchor-based and diff-based discovery unconditionally, then union and deduplicate results by entry ID. No fallback logic — both paths always execute.

**Consequences:**
- Positive: Partial-stale scenarios no longer silently drop entries
- Positive: Simpler control flow — no conditional fallback branching
- Negative: Both discovery paths run every time, even when anchors are fully valid (minor performance cost)
- Negative: Deduplication logic required to prevent double-counting entries found by both paths

## ADR-6: Broadening Ledger-Only Filter to Infrastructure-Only with Configurable Prefixes

**Context:** `isLedgerOnlyCommit` filtered out commits that only touched `.timbers/` files, preventing timbers from showing its own entry commits as pending work. When beads added an auto-stage hook for `.beads/issues.jsonl`, timbers entry commits started containing beads files too, breaking the filter and creating a timbers-on-timbers loop: each entry commit appeared as pending, triggering another entry.

**Decision:** Renamed `isLedgerOnlyCommit` to `isInfrastructureOnlyCommit` with a configurable prefix list (`.timbers/`, `.beads/`). A commit is "infrastructure-only" if all changed files fall under one of the configured prefixes.

**Consequences:**
- Positive: Breaks the infinite loop caused by beads auto-staging into timbers commits
- Positive: Configurable prefix list means future infrastructure tools (beyond timbers and beads) can be added without code changes
- Negative: The filter is now a denylist — any new tool that auto-stages files into commits must be explicitly added to the prefix list or it will trigger false pending results
- Negative: Slightly looser semantics — "infrastructure" is a broader concept than "ledger," so the filter may suppress commits that a user considers meaningful if they happen to only touch files under a configured prefix

## ADR-7: Git-Native Auto-Flush Sync Over Dolt Push/Pull for Beads

**Context:** Beads issue tracking previously synced via Dolt's own push/pull mechanism against a Dolt remote. This required a running Dolt server, was blocked by Claude Code's sandbox (TCP to localhost), and added a separate sync workflow orthogonal to git.

**Decision:** Replaced Dolt sync with auto-flush to `.beads/issues.jsonl` — a git-tracked JSONL file that ships with every commit via pre-commit hook auto-staging. Import happens automatically after `git pull`.

**Consequences:**
- Positive: Sync piggybacks on existing `git push`/`git pull` — no separate sync commands or Dolt remote needed
- Positive: Eliminates sandbox TCP issues — no localhost connections required
- Positive: Beads state is visible in git history and diffs, improving auditability
- Negative: Merge conflicts in `issues.jsonl` are possible when multiple collaborators modify issues concurrently
- Negative: Dolt's native conflict resolution (cell-level merge) is lost — conflicts fall to git's line-level merge
