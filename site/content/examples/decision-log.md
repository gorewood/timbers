+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Pre-commit Git Hook Over Claude Code PreToolUse for Enforcement

**Context:** Timbers needed to enforce documentation compliance — blocking commits until pending work is logged. Two mechanisms existed: Claude Code's `PreToolUse` JSON protocol (which intercepts tool calls within Claude Code sessions) and standard git `pre-commit` hooks (which block `git commit` regardless of client). The team had already implemented `PreToolUse` denial via structured JSON.

**Decision:** Drop `PreToolUse` enforcement entirely and rely on `pre-commit` git hooks as the primary blocking mechanism, with a `Stop` hook as a session-end backstop. Pre-commit hooks are universal across all git clients (CLI, IDE, CI, any agent), while `PreToolUse` only works inside Claude Code. The added complexity of Claude Code JSON protocol negotiation wasn't justified when the simpler mechanism had broader coverage.

**Consequences:**
- Positive: Enforcement works for any git client, not just Claude Code — future-proofs against agent platform changes
- Positive: Removes ~1000 lines of `PreToolUse` handler code and tests; simpler hook surface area
- Positive: No dependency on Claude Code's internal JSON protocol, which has changed between versions
- Negative: Pre-commit hooks can be bypassed with `--no-verify` — but so could any client-side enforcement
- Negative: Loses the ability to block *before* the user writes a commit message (pre-commit fires after message entry in some workflows)
- Constraint: Entry IDs contain anchor commit SHAs, making co-committing entries with code impossible (circular dependency) — separate commits for entries is a deliberate design choice, not a limitation

## ADR-2: Structured JSON Hooks Over Plain-Text Echo for Agent Communication

**Context:** Timbers used shell pipeline hooks (`grep` + `echo`) to remind agents about pending documentation. These printed to stdout but Claude Code silently consumed the output — agents never saw the reminders, and commits went undocumented every session.

**Decision:** Switch to structured JSON responses using Claude Code's `permissionDecision`/`deny` protocol. This is the only mechanism Claude Code actually respects for blocking actions. Plain-text stdout from hooks is silently discarded by the runtime.

**Consequences:**
- Positive: Agents actually receive and act on enforcement signals — documentation compliance went from 0% to effective
- Positive: `Stop` hook can block session end with a reason string, ensuring agents can't silently finish without logging
- Negative: Ties hook format to Claude Code's JSON protocol, which is underdocumented and may change
- Negative: Hook logic must live in the `timbers` binary (not simple shell scripts), increasing binary scope

## ADR-3: Display-Layer Filtering for Pre-Timbers History Over Storage-Layer Changes

**Context:** In fresh repos, `GetPendingCommits` returned all historical commits as "pending" since no timbers entries existed yet. The initial fix changed `GetPendingCommits` to return empty when no entries existed, but this broke `timbers log`, `batch log`, and MCP log — they all need commits to create the first entry.

**Decision:** Keep storage behavior unchanged and intercept the no-entries state at the display layer. `pending.go` and `doctor_checks.go` check `latest==nil` before rendering, while `log` and `batch` commands still receive commits from storage to create the initial entry.

**Consequences:**
- Positive: Storage API stays honest — callers that need commits (log, batch, MCP) continue working
- Positive: Display callers show a friendly "no entries yet" message instead of listing hundreds of irrelevant commits
- Negative: Every new display caller must remember to check `latest==nil` — the guard isn't enforced by the type system
- Negative: Slight violation of DRY — the nil check is repeated across display sites

## ADR-4: Batch Subprocess via `git diff-tree --stdin` Over Per-Commit Spawning

**Context:** `filterLedgerOnlyCommits` spawned one `git` subprocess per commit to check if a commit only touched `.timbers/` files. In repos with ~2000 commits, `timbers doctor` took 16 seconds due to O(N) process spawns.

**Decision:** Batch all commit lookups into a single `git diff-tree --stdin` call via `CommitFilesMulti`, reducing subprocess count from O(N) to O(1). Doctor also early-returns when no entries exist, avoiding the expensive call entirely.

**Consequences:**
- Positive: Doctor runtime dropped from ~16s to sub-second in large repos
- Positive: `CommitFilesMulti` is reusable — also adopted by `HasPendingCommits` fix
- Negative: `--stdin` mode requires parsing interleaved multi-commit output, adding parsing complexity
- Negative: `GitOps` interface grew by one method, requiring mock updates across all test files

## ADR-5: Delegate `HasPendingCommits` to Full `GetPendingCommits` Over Fast HEAD-vs-Anchor Check

**Context:** `HasPendingCommits` was a ~15ms fast path that compared HEAD against the anchor commit. This was ~85ms faster than the full `GetPendingCommits` call. However, the naive check false-positived after every `timbers log` because auto-committing the entry file changed HEAD — making the Stop hook fire on every session end, blocking agents from completing.

**Decision:** Replace the fast path with a delegation to `GetPendingCommits`, which filters out ledger-only commits via `CommitFilesMulti`. Accept the ~85ms penalty since this runs once per session at most.

**Consequences:**
- Positive: Stop hook no longer blocks every session end with false-positive pending warnings
- Positive: Single source of truth for "what's pending" — no divergence between fast check and full check
- Negative: Lost the ~15ms fast path; every pending check now spawns git subprocesses
- Lesson: The original design note said "may false-positive on ledger-only commits; acceptable trade-off" but real-world testing immediately proved it unacceptable

## ADR-6: Rename `exec-summary` to `standup` as Clean Break Pre-GA

**Context:** The `exec-summary` template name was less discoverable than `standup` for its primary use case (daily standup summaries). The PR description template also needed rewriting — it was summarizing diffs, but agents already review diffs; the template should focus on intent and decisions instead.

**Decision:** Rename `exec-summary` to `standup` with no backward-compatibility alias. Pre-GA project with low usage makes this a clean break without migration cost. Rewrote `pr-description` to focus on intent/decisions rather than diff summaries.

**Consequences:**
- Positive: `timbers draft standup` is immediately understandable; reduces documentation burden
- Positive: No alias maintenance or deprecation warnings cluttering the codebase
- Negative: Any existing scripts or muscle memory referencing `exec-summary` will break silently
- Positive: PR descriptions now complement rather than duplicate what reviewers already see in the diff

## ADR-7: Git Hook Stdout for Agent Nudges Over Claude Code PostToolUse Hooks

**Context:** Timbers needed a way to remind agents to run `timbers log` after committing. Claude Code's `PostToolUse` hooks write to stdout, but that output is silently consumed by the runtime — agents never see it. Git's `post-commit` hook stdout, however, is visible to agents as part of the git command output.

**Decision:** Use git `post-commit` hooks to print one-line reminders. `timbers hook run post-commit` outputs the nudge; `init --hooks` installs it; `doctor` checks for it and `--fix` auto-installs.

**Consequences:**
- Positive: Reminders are visible to any agent or human using git, not just Claude Code
- Positive: `doctor --fix` provides self-healing — missing hooks are auto-installed
- Negative: Git hooks are per-clone, not per-repo — each fresh clone needs `timbers init --hooks`
- Negative: Post-commit hooks are advisory only; they can't block (unlike pre-commit)

## ADR-8: Warn and Proceed on Stale Anchor Over Fatal Error

**Context:** After squash merges or rebases, the anchor commit (last documented commit SHA) disappears from history. `timbers log` and `batch log` treated `ErrStaleAnchor` as fatal, even though `pending`, `prime`, and MCP already handled it gracefully by accepting fallback commits.

**Decision:** Accept fallback commits from `GetPendingCommits` when `ErrStaleAnchor` occurs, warn the user, and proceed. This makes the "self-healing anchor" claim actually true across all commands, not just some.

**Consequences:**
- Positive: `timbers log` works after squash merges without manual intervention
- Positive: Consistent behavior across all commands — no surprise fatals in `log` when `pending` works fine
- Negative: Fallback commits may include already-documented work from the squash-merged branch — users might document something twice
- Negative: Warning message may confuse users who don't understand anchor mechanics
