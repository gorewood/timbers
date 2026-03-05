+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Pre-commit Hook Over Claude Code Protocol for Enforcement

**Context:** Timbers needed to prevent agents from committing without documenting work. Two mechanisms were available: Claude Code's `PreToolUse` JSON protocol (which can deny specific tool calls) and standard git pre-commit hooks (which block `git commit` at the git level). The initial implementation used both — `PreToolUse` to deny commits and `Stop` as a session-end backstop.

**Decision:** Drop `PreToolUse` and rely on pre-commit hook blocking plus `Stop` backstop. Pre-commit hooks are universal across all git clients, not just Claude Code. The `PreToolUse` handler added complexity (JSON protocol negotiation, Claude Code-specific dispatch) for marginal benefit — it only prevented "stacking" commits, which pre-commit already handles.

**Consequences:**
- Positive: Enforcement works for any git client or agent, not just Claude Code
- Positive: Significant complexity reduction (removed handler, dispatch case, tests)
- Positive: Single enforcement point is easier to reason about
- Negative: Loses the ability to intercept commit *intent* before the user even runs `git commit`
- Constraint: `Stop` hook must remain as backstop for agents that somehow bypass pre-commit

## ADR-2: Separate Commits for Entries Over Co-committing

**Context:** Entry IDs use the format `tb_<timestamp>_<anchor-short-sha>` where the SHA comes from the commit being documented. Co-committing entries with code would be cleaner (one commit instead of two), but creates a circular dependency: entry content changes the tree hash, which changes the commit SHA, which changes the entry ID. ~12 approaches were explored including two-pass commits, predictable IDs, post-commit amendment, and deferred IDs.

**Decision:** Keep separate commits for entries. The SHA reference is a feature — entries point to exact commits — not a limitation. All co-committing approaches added complexity or broke the clean SHA-addressable property.

**Consequences:**
- Positive: Entry IDs are stable, deterministic, and point to real commits
- Positive: No complex workarounds (two-pass, amendment, deferred resolution)
- Negative: Every documented piece of work produces two commits (code + entry)
- Negative: `HasPendingCommits` must filter out ledger-only commits to avoid false positives
- Constraint: Any future optimization must preserve the separate-commit model

## ADR-3: Structured JSON Hooks Over Plain-text Echo

**Context:** The original hook implementation used shell pipelines (`grep` + `echo`) that printed reminders to stdout. Claude Code silently consumed `PostToolUse` stdout and never surfaced it to agents. Agents never saw reminders and commits went undocumented every session.

**Decision:** Use Claude Code's structured JSON protocol — `permissionDecision`/`deny` for `PreToolUse` (later removed per ADR-1), `decision`/`reason` JSON for `Stop` hooks. Structured JSON with `permissionDecision`/`deny` is the only mechanism Claude Code actually respects for blocking.

**Consequences:**
- Positive: Enforcement actually works — agents see and respond to blocking
- Positive: Machine-parseable output enables programmatic handling
- Negative: Tightly coupled to Claude Code's JSON protocol format
- Negative: Protocol changes in Claude Code could silently break enforcement

## ADR-4: Display-layer Filtering for Pre-timbers History

**Context:** When timbers is installed in an existing repo, all prior commits appear as "pending" (undocumented). The initial fix changed `GetPendingCommits` in the storage layer to return empty when no entries exist. This broke `timbers log`, `batch log`, and MCP log — they all need commits to create the first entry.

**Decision:** Keep storage behavior unchanged. Move the no-entries check to the display layer — `pending.go` and `doctor_checks.go` intercept `latest==nil` before rendering. Callers that need commits (log, batch, MCP) continue to receive them.

**Consequences:**
- Positive: All consumers of `GetPendingCommits` keep working without special-casing
- Positive: New users get a friendly message instead of a wall of "pending" commits
- Negative: Display-layer callers must each remember to check `latest==nil`
- Constraint: Any new display of pending state must also handle the no-entries case

## ADR-5: Batch Subprocess Over Per-commit Filtering

**Context:** `filterLedgerOnlyCommits` spawned one `git` subprocess per commit to check if it only touched `.timbers/` files. In repos with ~2000 commits, `doctor` took 16 seconds due to O(N) process spawns.

**Decision:** Batch all commit file lookups into a single `git diff-tree --stdin` call via `CommitFilesMulti`, reducing subprocess spawns from O(N) to O(1).

**Consequences:**
- Positive: `doctor` performance drops from 16s to sub-second in large repos
- Positive: Pattern is reusable — `HasPendingCommits` also uses `CommitFilesMulti`
- Negative: `GitOps` interface grows — all mocks must implement `CommitFilesMulti`
- Negative: `diff-tree --stdin` parsing is more complex than per-commit calls

## ADR-6: Delegate to GetPendingCommits Over Fast HasPendingCommits

**Context:** `HasPendingCommits` was added as a ~15ms fast path (simple HEAD-vs-anchor comparison) to avoid the full `GetPendingCommits` call on the `Stop` hook hot path. However, it false-positived after every `timbers log` because auto-committed entries change HEAD, making it always appear that new commits exist.

**Decision:** Remove the fast path. `HasPendingCommits` now delegates to `GetPendingCommits` which filters ledger-only commits via `CommitFilesMulti`. The extra subprocess runs once per session — acceptable cost.

**Consequences:**
- Positive: `Stop` hook no longer blocks every session end with false positives
- Positive: Single source of truth for "what's pending" — no divergent logic
- Negative: Lost the ~85ms performance advantage of the simple comparison
- Constraint: Performance is acceptable only because this runs once per session, not on every commit

## ADR-7: Install Hooks to core.hooksPath Over Hardcoded .git/hooks

**Context:** Beads sets `core.hooksPath=.beads/hooks/` to manage its own pre-commit hook. Timbers was hardcoding `.git/hooks/` for hook installation, so hooks were installed where git never reads them.

**Decision:** Read `git config core.hooksPath` and install there, falling back to `.git/hooks/` when unset. This is acknowledged as a band-aid — both beads and timbers want to own the pre-commit hook, and `core.hooksPath` is winner-take-all.

**Consequences:**
- Positive: Timbers hooks actually work in repos using `core.hooksPath`
- Positive: `--chain` correctly backs up whatever hook was at the resolved path
- Negative: Still fragile — two tools competing for hook ownership via the same mechanism
- Constraint: Real fix requires beads to have a plugin/chain mechanism so timbers can register without owning the hook file

## ADR-8: Git Post-commit Hook Over Claude Code PostToolUse for Reminders

**Context:** Timbers needed a way to remind agents to run `timbers log` after committing. Claude Code's `PostToolUse` hooks write to stdout, but that output is silently consumed — agents never see it.

**Decision:** Use a standard git `post-commit` hook. Git hook stdout is visible to agents, making it the reliable nudge mechanism. Installed via `timbers init --hooks`, checked and auto-fixed by `doctor --fix`.

**Consequences:**
- Positive: Works across all git clients, not just Claude Code
- Positive: Agents actually see the reminder text
- Positive: `doctor` can verify and repair the hook installation
- Negative: Adds another hook file to manage alongside pre-commit
- Negative: Reminder is passive (stdout) not active (blocking) — agents can still ignore it

## ADR-9: Actionable Stop Hook Reason Over Terse Message

**Context:** The `Stop` hook blocked session end with a message saying to run `timbers log`, but agents receiving just those two words would fail — they didn't know the required flags or syntax.

**Decision:** Expand the `Stop` hook reason string to include the full `timbers log` command syntax with `--why`/`--how` placeholder arguments, plus a `timbers pending` command to see what's undocumented.

**Consequences:**
- Positive: Agents can act on the blocking message without prior knowledge of timbers CLI
- Positive: Reduces failed recovery attempts and wasted tokens
- Negative: Longer reason strings consume more agent context
- Constraint: Reason format must stay current as CLI flags evolve
