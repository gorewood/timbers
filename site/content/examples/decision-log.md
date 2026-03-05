+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Append-Section Over Backup-and-Chain for Git Hook Installation

**Context:** Timbers needed to install git hooks in environments where other tools (notably beads) also manage hooks. The original strategy was backup-and-chain: rename the existing hook to `.original` and generate a new hook that calls the backup. This took ownership of hook files and conflicted with tools like beads that set `core.hooksPath`. Symlinks and binary hooks added further edge cases.

**Decision:** Replace backup-and-chain with append-section, where timbers appends a clearly delimited section to existing hook scripts. Refuse symlinks and binaries with guidance instead of maintaining a fallback code path. The `.original`/`.backup` naming debate became moot — append-section eliminates the need for renaming entirely.

**Consequences:**
- Timbers no longer takes ownership of hook files, reducing conflicts in multi-tool repos
- Simpler codebase — no second code path for rename-and-chain fallback
- Symlink and binary hooks are not supported; users must convert them manually
- Uninstall is a section removal rather than a file restore, which is more predictable

## ADR-2: Pre-Commit Hook Blocking Over Claude Code PreToolUse Protocol

**Context:** Timbers enforced documentation compliance through multiple mechanisms: a `PreToolUse` hook that denied `git commit` via Claude Code's JSON protocol, a `Stop` hook as a session-end backstop, and a git `pre-commit` hook. The PreToolUse handler required understanding Claude Code's structured JSON protocol and only worked in Claude Code, while the pre-commit hook works with all git clients.

**Decision:** Drop PreToolUse entirely. Rely on pre-commit hook as the primary enforcement (universal across all git clients) with Stop as a backstop for agents that bypass pre-commit. PreToolUse added complexity for marginal stacking prevention that pre-commit already handles.

**Consequences:**
- Enforcement works in any git client, not just Claude Code
- Simpler hook codebase — no JSON protocol negotiation for PreToolUse
- Lost the ability to block before the commit attempt (pre-commit blocks during, not before)
- Stop hook remains as a second line of defense for session-end compliance
- Co-committing entries with code was explored (~12 approaches) and rejected — entry IDs embed the anchor commit SHA, creating a circular dependency if entries are committed alongside code. Separate commits for entries is the correct design.

## ADR-3: Structured JSON Hooks Over Plain-Text Echo for Claude Code Enforcement

**Context:** Early hook implementations used grep+echo pipelines that printed reminders to stdout. Claude Code silently consumed stdout from PostToolUse hooks and never enforced plain-text Stop hook output. Agents never saw reminders and commits went undocumented every session.

**Decision:** Use Claude Code's structured JSON protocol with `permissionDecision`/`deny` for the Stop hook. This is the only mechanism Claude Code actually respects for blocking behavior.

**Consequences:**
- Agents are now genuinely blocked at session end when pending commits exist
- Hooks are coupled to Claude Code's JSON protocol, which is an undocumented interface
- Required implementing all hook logic in the `timbers` binary rather than shell scripts
- Added `HasPendingCommits()` as a fast ~15ms check to avoid full `GetPendingCommits()` on the hot path

## ADR-4: Accurate Pending Detection Over Fast-Path Approximation

**Context:** `HasPendingCommits()` was introduced as a ~15ms fast path that simply compared HEAD against the anchor commit. The original design noted "may false-positive on ledger-only commits; acceptable trade-off for ~15ms." Testing in a real repo immediately revealed it blocks every session end, because `timbers log` auto-commits the entry file, advancing HEAD past the anchor.

**Decision:** Delegate to `GetPendingCommits()` which filters ledger-only commits via `CommitFilesMulti` batch lookup. Accept the additional git subprocess cost (~85ms) since it runs once per session.

**Consequences:**
- Stop hook no longer fires false positives after every `timbers log`
- One extra git subprocess per session — negligible for a once-per-session check
- The "fast path" optimization was abandoned after a single real-world test proved the trade-off unacceptable
- Reinforces that performance optimizations in enforcement paths must not compromise correctness

## ADR-5: Batch Git Subprocess via `diff-tree --stdin` Over Per-Commit Spawning

**Context:** `filterLedgerOnlyCommits` spawned one `git` subprocess per commit to check which files each commit touched. In repos with ~2000 commits, `timbers doctor` took 16 seconds due to O(N) process spawns.

**Decision:** Batch all commit file lookups into a single `git diff-tree --stdin` invocation via a new `CommitFilesMulti` function, reducing subprocess count from O(N) to O(1). Also added an early return in `doctor` when no entries exist.

**Consequences:**
- `doctor` performance improved from 16s to sub-second in large repos
- `GitOps` interface expanded with `CommitFilesMulti`, requiring mock updates across all test files
- The batch approach uses `--stdin` which is less commonly used — slightly harder to understand at a glance
- Pattern is reusable for any future bulk commit inspection

## ADR-6: Install Hooks to `core.hooksPath` Rather Than Hardcoded `.git/hooks/`

**Context:** Beads sets `core.hooksPath=.beads/hooks/` to manage its own hooks. Timbers was hardcoding `.git/hooks/` for hook installation, meaning hooks were installed where git never reads them in beads-managed repos.

**Decision:** Read `git config core.hooksPath` and install there, falling back to `.git/hooks/` when unset. This was acknowledged as a band-aid — both tools want to own the pre-commit hook, and `core.hooksPath` is winner-take-all.

**Consequences:**
- Hooks now work in beads-managed repos
- Timbers is subordinate to whatever tool set `core.hooksPath` — it installs into their directory
- The real fix (a plugin/chain mechanism in beads) is deferred
- `--chain` backup path must be dynamic based on `hooksPath`, adding a parameter to `GeneratePreCommitHook`

## ADR-7: Git Hook Stdout for Agent Nudges Over Claude Code PostToolUse Hooks

**Context:** Timbers needed a reliable way to remind agents to run `timbers log` after commits. Claude Code's PostToolUse hooks write to stdout but it's consumed silently — agents never see the output. Git hook stdout, however, is visible to agents as part of the git command output.

**Decision:** Use a git `post-commit` hook that prints a one-line reminder via `timbers hook run post-commit`. Surface it through `init --hooks`, check it in `doctor`, and auto-install via `doctor --fix`.

**Consequences:**
- Reminder is visible to any tool that captures git output, not just Claude Code
- Adds a git hook dependency — repos must have the hook installed
- `prime` now runs a quick health check and surfaces missing hooks before agents start work
- Doctor gains another check and fix capability, increasing its value as a diagnostic tool

## ADR-8: No Interactive Prompts in Init for Hook Installation

**Context:** During the graceful hook deference work, the team debated whether `timbers init` in Tier 1 (uncontested) environments should prompt the user about installing hooks.

**Decision:** Don't prompt. The Claude Code Stop hook (steering) is the primary enforcement mechanism; git hooks are a bonus. Adding a second interactive prompt to init was a UX regression.

**Consequences:**
- `init` stays fast and non-interactive, better for scripted/automated setups
- Users must run `timbers hooks install` separately or use `doctor --fix`
- Keeps a clean separation: init sets up the ledger, hooks are opt-in enforcement
- Agents running init won't get stuck on an interactive prompt
