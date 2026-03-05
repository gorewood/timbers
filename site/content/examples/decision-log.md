+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Pre-commit Hook Over PreToolUse for Commit Blocking

**Context:** Timbers needed to prevent commits when work was undocumented. Two mechanisms existed: Claude Code's `PreToolUse` JSON protocol (which could deny the `git commit` tool call) and standard git `pre-commit` hooks (which block at the git layer). Both could prevent commits, but they operated at different levels — one was Claude Code-specific, the other universal across all git clients.

**Decision:** Drop `PreToolUse` and rely on `pre-commit` hooks as the primary blocking mechanism, with `Stop` as a session-end backstop. Pre-commit hooks are universal across all git clients, not just Claude Code. The `PreToolUse` handler added complexity (Claude Code JSON protocol negotiation) for marginal benefit — it only prevented "stacking" commits, which pre-commit already handles.

**Consequences:**
- Positive: Simpler codebase (~213 lines deleted), enforcement works in any git client (VS Code, CLI, other agents), single mechanism to reason about
- Positive: No dependency on Claude Code's hook protocol for core enforcement — works even if the agent framework changes
- Negative: Loses the ability to block at the tool-decision level (agent sees the denial before attempting), though in practice pre-commit failure achieves the same outcome
- Constraint: `Stop` hook must remain as a backstop for agents that somehow bypass pre-commit (e.g., `--no-verify`)

## ADR-2: Separate Commits for Ledger Entries Over Co-committing

**Context:** Timbers entry IDs use the format `tb_<timestamp>_<anchor-short-sha>`, where the SHA comes from the commit being documented. Co-committing entries alongside code would be more efficient (one commit instead of two), but creates a circular dependency: the entry content changes the tree hash, which changes the commit SHA, which changes the entry ID. Approximately 12 approaches were explored including two-pass commits, predictable IDs, post-commit amendment, and deferred IDs.

**Decision:** Keep entries in separate commits. The SHA reference is a feature — entries point to exact commits — not a limitation. All co-committing approaches added complexity or broke the clean SHA-addressable property.

**Consequences:**
- Positive: Entry IDs are stable, deterministic, and directly reference the work they document
- Positive: Simple mental model — commit code, then document it
- Negative: Every documented commit produces a second ledger commit, doubling commit count
- Negative: Requires filtering ledger-only commits in `GetPendingCommits` to avoid false positives (see ADR-3)

## ADR-3: Batch Subprocess for Ledger Filtering Over Per-commit Spawning

**Context:** `HasPendingCommits` needed to distinguish real code commits from timbers' own ledger auto-commits. The initial fast path (~15ms) used a naive HEAD-vs-anchor check, but this always triggered after `timbers log` because the auto-committed entry changed HEAD. An alternative `GetPendingCommits` function filtered properly but spawned one `git` subprocess per commit — O(N) processes made `doctor` take 16 seconds in 2000-commit repos.

**Decision:** Batch filtering via `git diff-tree --stdin` in a single subprocess (`CommitFilesMulti`). This is O(1) process spawns regardless of commit count. `HasPendingCommits` delegates to the full `GetPendingCommits` rather than maintaining a separate fast path.

**Consequences:**
- Positive: `doctor` dropped from ~16s to sub-second in large repos
- Positive: Single code path for pending detection — no divergence between fast-check and full-check
- Negative: Adds one git subprocess per pending check (~85ms slower than the abandoned fast path), but runs once per session — acceptable
- Negative: Required updating `GitOps` interface and all mocks

## ADR-4: Display-layer Filtering for Pre-timbers History Over Storage-layer

**Context:** When timbers is first installed in a repo with existing history, all prior commits appeared as "pending" — overwhelming and useless. The original plan changed `GetPendingCommits` to return empty when no entries existed, but this broke `timbers log`, `batch log`, and MCP log, which all need commits to create the first entry.

**Decision:** Handle the no-entries case at the display layer (`pending.go`, `doctor_checks.go`) by checking `latest==nil`, while leaving storage behavior unchanged. Callers that need commits for entry creation continue to receive them.

**Consequences:**
- Positive: `timbers pending` and `doctor` show a friendly onboarding message instead of listing hundreds of "undocumented" commits
- Positive: `timbers log` still works for the first-ever entry — storage provides the commits it needs
- Negative: Display-layer filtering is less discoverable than a storage-layer contract — new callers must know to check `latest==nil` themselves
- Constraint: Any new command that displays pending state needs the same guard

## ADR-5: Structured JSON Hooks Over Plain-text Echo for Agent Enforcement

**Context:** Timbers initially used plain-text `echo` hooks and grep pipelines to remind agents about undocumented commits. Claude Code silently consumed stdout from `PostToolUse` hooks, and `Stop` hook output was printed but not enforced. Agents never saw the reminders, and commits went undocumented every session.

**Decision:** Use Claude Code's structured JSON protocol — `permissionDecision: "deny"` for blocking, `decision`/`reason` JSON for Stop hooks. All logic lives in the `timbers` binary rather than shell scripts.

**Consequences:**
- Positive: Actually enforces compliance — Claude Code respects JSON `deny` responses, unlike plain-text output
- Positive: Logic in Go binary means testable, cross-platform, and version-controlled behavior
- Negative: Tightly coupled to Claude Code's JSON hook protocol — other agent frameworks would need different integrations
- Negative: Shell script hooks are silently ignored by Claude Code, which is a surprising behavior for users expecting POSIX conventions

## ADR-6: Actionable Command Syntax in Stop Hook Reason Over Generic Message

**Context:** The Stop hook's reason string told agents that `timbers log` was needed, but agents receiving just "run `timbers log`" would fail because they didn't know the required `--why` and `--how` flags.

**Decision:** Include the full `timbers log` syntax with placeholder arguments and a `timbers pending` precursor in the reason string, making it directly actionable without the agent needing to discover the CLI interface.

**Consequences:**
- Positive: Agents can copy the command template and fill in values — zero discovery overhead
- Positive: Reduces failed attempts and retry loops at session end
- Negative: Reason string is longer and more verbose than a simple message
- Negative: If the CLI syntax changes, the reason string must be updated in lockstep

## ADR-7: Git Hook Stdout for Logging Reminders Over Claude Code PostToolUse

**Context:** Agents needed a nudge to run `timbers log` after committing. Two delivery mechanisms existed: Claude Code's `PostToolUse` hooks (which run after tool calls) and git's native `post-commit` hook (which prints to stdout after every commit). Claude Code silently consumed `PostToolUse` stdout, making it invisible to agents.

**Decision:** Use git `post-commit` hook stdout, which is visible to agents in their terminal output. `timbers hook run post-commit` prints a one-line reminder. `init --hooks` installs it, and `doctor --fix` auto-installs it.

**Consequences:**
- Positive: Reliable delivery — git hook stdout appears in the agent's terminal regardless of agent framework
- Positive: Works for all git clients, not just Claude Code
- Negative: Post-commit hooks are advisory (agents can ignore the message), unlike pre-commit which can block
- Design: Combined with pre-commit blocking (ADR-1), this creates a two-tier system — gentle reminder after commit, hard block at session end

## ADR-8: Static Date Ranges for Site Examples Over Per-release Regeneration

**Context:** The project site included generated examples (changelogs, decision logs) produced by piping `timbers draft` through an LLM. Regenerating all examples on every release meant redundant LLM calls for historical content that wouldn't change.

**Decision:** Split justfile recipes into `examples` (dynamic, regenerated per-release) and `examples-static` (fixed date range, generated once from a dense Feb 10-14 window).

**Consequences:**
- Positive: Avoids redundant LLM API calls and associated cost/latency on every release
- Positive: Static examples remain stable — no risk of LLM output drift changing published content
- Negative: Two categories of examples to maintain with different update workflows
- Negative: Static examples may become stale if the output format evolves significantly
