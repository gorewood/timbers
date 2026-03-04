+++
title = 'Decision Log'
date = '2026-03-04'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---



## ADR-1: Pre-commit Hook Over PreToolUse for Commit Enforcement

**Context:** Timbers needed to prevent agents from committing code without documenting their work via `timbers log`. Two enforcement mechanisms existed: Claude Code's `PreToolUse` JSON protocol (which could deny `git commit` tool calls) and standard git `pre-commit` hooks (which block commits at the git level). PreToolUse was Claude Code-specific and required JSON protocol negotiation; pre-commit hooks work universally across all git clients.

Additionally, co-committing entries with code was explored (~12 approaches including two-pass commits, predictable IDs, and post-commit amendment), but entry IDs contain the anchor commit SHA — creating a circular dependency where the entry changes the tree hash, which changes the commit SHA, which changes the entry ID.

**Decision:** Drop `PreToolUse` enforcement entirely and rely on `pre-commit` hook blocking, with `Stop` hook as a session-end backstop. Separate commits for entries is the correct design — the SHA reference is a feature (entries point to exact commits), not a limitation.

**Consequences:**
- Enforcement works across all git clients, not just Claude Code
- Simpler codebase — removed PreToolUse handler, tests, and dispatch logic (~213 lines deleted)
- `PreToolUse` moved to `retiredEvents` requiring cleanup logic in uninstall
- Agents that bypass pre-commit (e.g., `--no-verify`) have only the `Stop` backstop
- Entry commits are always separate from code commits, adding one extra commit per documentation event

---

## ADR-2: Structured JSON Hooks Over Plain-Text Echo for Claude Code Enforcement

**Context:** Early hook implementations used shell pipelines (`grep` + `echo`) that printed reminders to stdout. Claude Code silently consumed `PostToolUse` stdout and never enforced plain-text `Stop` output. Agents consistently ignored reminders and commits went undocumented every session.

**Decision:** Use Claude Code's structured JSON protocol (`permissionDecision`/`deny` for PreToolUse, `decision`/`reason` for Stop) as the only mechanism that Claude Code actually respects for blocking. Added `HasPendingCommits()` as a fast ~15ms check (HEAD vs anchor) to avoid full `GetPendingCommits()` on the hot path.

**Consequences:**
- Enforcement actually works — agents can no longer silently ignore documentation requirements
- `HasPendingCommits()` may false-positive on ledger-only commits, but this only triggers a logging prompt (acceptable trade-off)
- All hook logic now lives in the timbers binary rather than shell scripts
- Shell script hooks in `.claude/hooks/*.sh` are an anti-pattern — Claude Code reads hooks from JSON settings only

---

## ADR-3: Batch Git Subprocess Calls via `diff-tree --stdin` for Performance

**Context:** `filterLedgerOnlyCommits` spawned one `git` subprocess per commit to check file lists. In repositories with ~2k commits, `doctor` took 16 seconds due to O(N) process spawns.

**Decision:** Batch all commit file lookups into a single `git diff-tree --stdin` call, reducing subprocess spawns from O(N) to O(1). Also added early-return in `doctor` when no entries exist.

**Consequences:**
- `doctor` performance drops from 16s to sub-second in large repos
- `CommitFilesMulti` added to `GitOps` interface, requiring mock updates across all test files
- Single-commit `CommitFiles` still exists for callers that need it, minor API surface increase

---

## ADR-4: Display-Layer Filtering for Pre-Timbers History Over Storage-Layer Change

**Context:** New users installing timbers saw their entire git history reported as "pending" undocumented commits. The fix options were: (A) change `GetPendingCommits` in storage to return empty when no entries exist, or (B) have display callers check for no-entries state and show a friendly message.

**Decision:** Fix at the display layer — `pending.go` and `doctor_checks.go` intercept the no-entries state. Storage behavior unchanged.

**Consequences:**
- `timbers log`, `batch log`, and MCP log continue to receive commits for creating the first entry (they need commits to work)
- Friendly onboarding experience — no wall of "pending" commits on first use
- Two display callers carry the guard logic instead of one storage function, slight duplication
- JSON output includes a `status` field distinguishing "no entries yet" from "all caught up"

---

## ADR-5: Actionable Warnings Over Anchor-Reset Command for Stale Anchors

**Context:** After squash merges, the anchor commit disappears from history, causing `ErrStaleAnchor`. Options were: (A) add an explicit `timbers anchor reset` command, or (B) improve warning messages and rely on self-healing behavior (anchor updates automatically on next `timbers log`).

**Decision:** Warnings plus coaching — no anchor-reset command. The anchor self-heals on the next `timbers log`, so explicit reset is unnecessary. Added coaching section to prime workflow output explaining the situation.

**Consequences:**
- Simpler CLI surface — no command to maintain, document, or test
- Users who hit the issue get an explanation and know it resolves itself
- `log` and `batch log` now accept fallback commits when `ErrStaleAnchor` occurs instead of failing fatally
- If self-healing assumptions change in the future, a reset command would need to be added retroactively

---

## ADR-6: Check-Before-Generate Over LLM Refusal for Empty Entry Sets

**Context:** The devblog CI workflow invoked the LLM even when no timbers entries existed for the period, producing "apology posts" where the model explained it had nothing to write about. Options: teach the LLM to refuse gracefully, or check entry count before invoking.

**Decision:** Gate generation on entry count > 0 in the CI workflow. Don't invoke the LLM at all when there's nothing to generate.

**Consequences:**
- Zero LLM cost on days with no entries (cheaper and faster)
- No risk of apology posts or hallucinated content leaking to the site
- Required deleting 10 already-published blank posts
- Any future generation workflows need the same guard pattern

---

## ADR-7: Clean Break for Template Rename Over Backward-Compatible Alias

**Context:** The `exec-summary` draft template was renamed to `standup` for better discoverability. Options were: (A) rename and add `exec-summary` as an alias, or (B) clean break with no alias.

**Decision:** Clean break — rename to `standup` with no backward compatibility alias. Pre-GA project with low usage makes this the right time for breaking changes.

**Consequences:**
- Simpler template resolution code — no alias mapping to maintain
- Better discoverability — `standup` is immediately understood
- Any existing scripts or habits referencing `exec-summary` break silently
- Sets precedent that pre-GA naming changes don't carry aliases

---

## ADR-8: PR Template Focused on Intent/Decisions Over Diff Summary

**Context:** The `pr-description` draft template was generating diff summaries — listing what files changed and what code was modified. But agents reviewing PRs already read the diff; duplicating it in the description wastes space.

**Decision:** Rewrote `pr-description` (v4) to focus on intent and design decisions rather than code changes. The template pulls from `why` fields and `notes` to surface trade-offs and reasoning.

**Consequences:**
- PR descriptions provide context that can't be derived from the diff alone
- Reviewers get the "why" up front, reducing back-and-forth questions
- Template is more dependent on entry quality — thin `why` fields produce thin PR descriptions
- Agents and humans both benefit from decision-focused summaries

---

## ADR-9: Git Hook stdout for Agent Nudges Over Claude Code PostToolUse

**Context:** Needed a way to remind agents to run `timbers log` after committing. `PostToolUse` hooks in Claude Code had stdout silently consumed — agents never saw the reminders. Git `post-commit` hook stdout, however, is visible to agents in their tool output.

**Decision:** Use git `post-commit` hook to print a one-line `timbers log` reminder. `timbers init --hooks` installs it; `doctor` checks for it and `--fix` auto-installs.

**Consequences:**
- Reminders are visible to all agents, not just Claude Code
- Non-blocking — post-commit hooks don't prevent the commit, just nudge
- Requires git hook installation as part of setup (another moving part)
- Combined with pre-commit blocking (ADR-1), creates a two-layer system: block before commit if pending, remind after commit to document

---

## ADR-10: `govulncheck` Separate from `just check` Quality Gate

**Context:** Go's `govulncheck` is not included in `golangci-lint` and needs to run as a separate tool. Options: add it to `just check` (blocking on every commit) or keep it as a separate `just vulncheck` recipe.

**Decision:** Separate `just vulncheck` recipe, not part of the `just check` quality gate.

**Consequences:**
- `just check` remains fast and doesn't block on stdlib vulnerability patches outside the developer's control
- Vulnerability scanning is opt-in per session rather than mandatory per commit
- Risk of shipping known vulnerabilities if developers forget to run it
- Can be added to CI independently with different failure thresholds
