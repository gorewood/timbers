+++
title = 'Decision Log'
date = '2026-03-02'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Stop Hook Over PostToolUse for Pending Reminders

**Context:** Timbers used a Claude Code `PostToolUse` hook to remind agents about undocumented commits after each tool call. Diagnostics confirmed the hook fired correctly and received valid JSON on stdin, but Claude Code did not surface the hook's stdout to the agent — the reminder went nowhere visible.

**Decision:** Remove the `PostToolUse` hook entirely and rely on the existing `Stop` hook, which runs `timbers pending` at session end. The `Stop` hook's output *is* displayed, and checking actual pending state once at session end covers the same compliance goal.

**Consequences:**
- Agents no longer receive per-commit nudges — only a single end-of-session reminder
- Eliminates a hook that executed on every tool call with zero visible effect (wasted cycles)
- Simpler hook configuration with one fewer event to maintain
- Depends on Claude Code continuing to surface `Stop` hook output — same platform coupling, just on a hook that works
- Added `retiredEvents` cleanup list so upgrades automatically remove the dead hook

---

## ADR-2: Pre-Generate Entry Count Check Over Teaching the LLM to Handle Empty Input

**Context:** The devblog CI workflow invoked an LLM to generate blog posts from timbers entries. When no entries existed for the period, the LLM received empty input and generated "apology posts" — content about having nothing to write about.

**Decision:** Add an entry count check step *before* LLM invocation, gating the entire generate/commit/push pipeline on count > 0. Checking preconditions is cheaper and more reliable than prompt-engineering the LLM to gracefully refuse.

**Consequences:**
- Zero wasted LLM API calls on empty input — cost savings on every no-op run
- Deterministic behavior: empty input always produces no output, no prompt sensitivity
- Required deleting 10 previously generated blank posts
- Pattern generalizes: validate inputs before LLM calls rather than relying on the model to detect degenerate cases

---

## ADR-3: Actionable Warnings Over Anchor-Reset Command for Stale Anchors

**Context:** After squash merges, timbers' anchor commit (the last-documented commit SHA) disappears from history, causing confusing pending output. Two approaches were considered: (1) add a `timbers anchor reset` command for explicit repair, or (2) improve warning messages and add coaching documentation.

**Decision:** Warnings plus coaching, no new command. The anchor self-heals the next time `timbers log` runs after a real commit, making explicit reset unnecessary. Messaging alone is sufficient to bridge the gap.

**Consequences:**
- No new command to document, test, or maintain
- Users seeing the warning get actionable guidance without needing to learn a repair workflow
- Coaching section added to `prime` workflow output so agents handle it automatically
- If self-healing assumptions break (e.g., orphan branches with no new commits), there's no manual escape hatch — would need to revisit

---

## ADR-4: Clean Rename Over Backward-Compatible Alias for exec-summary Template

**Context:** The `exec-summary` draft template was being renamed to `standup` for better discoverability. The question was whether to keep `exec-summary` as an alias for backward compatibility.

**Decision:** Clean break — rename without alias. Pre-GA project with low usage means backward compatibility adds complexity for no real benefit.

**Consequences:**
- Simpler template registry with no alias resolution logic
- Anyone using `exec-summary` in scripts gets a clear error rather than silent redirect
- Sets precedent: pre-GA naming changes are clean breaks, not aliased migrations
- Would not apply post-GA where users have automation depending on template names

---

## ADR-5: PR Template Focused on Intent and Decisions Over Diff Summary

**Context:** The `pr-description` draft template (v3) summarized code diffs. Since agents reviewing PRs already have full diff access, the generated description duplicated information the reviewer could see directly.

**Decision:** Rewrite the PR template (v4) to focus on *intent* and *design decisions* — the things not visible in the diff. Agents already review diffs; the template should add context they can't infer from code alone.

**Consequences:**
- PR descriptions complement rather than duplicate the diff view
- Leverages timbers' unique data (why/how fields, notes) that no diff tool captures
- Requires entries with substantive why fields — thin entries produce thin PR descriptions
- Template is more opinionated about what a good PR description contains

---

## ADR-6: Display-Layer Handling of Pre-Timbers History Over Storage-Layer Filtering

**Context:** New users installing timbers on existing repos saw every prior commit listed as "pending" — potentially hundreds of commits they never intended to document. The initial fix changed `GetPendingCommits` in the storage layer to return empty when no entries existed, but this broke `timbers log`, `batch log`, and MCP log, which all need those commits to create the first entry.

**Decision:** Handle the no-entries state at the display layer. `pending.go` and `doctor_checks.go` check `latest==nil` and show a friendly message instead of a commit list. Storage behavior remains unchanged, preserving all callers that need commit data.

**Consequences:**
- `timbers log` continues to work for creating the first entry from pre-existing commits
- New users see a welcome message with a `--catchup` tip instead of an overwhelming pending list
- Fix is localized to two display callsites rather than a storage behavior change affecting all consumers
- Any new command that displays pending state needs to remember the `latest==nil` guard

---

## ADR-7: Git Hook Over Claude Code Hook for Agent Log Compliance

**Context:** After removing the `PostToolUse` hook (ADR-1), agents had no mid-session reminder to run `timbers log`. Two mechanisms were considered: another Claude Code hook event, or a native git `post-commit` hook. Claude Code hooks had proven unreliable for surfacing stdout to agents.

**Decision:** Use a git `post-commit` hook. Git hook stdout is visible to agents (it appears in the `git commit` tool output), making it the reliable nudge mechanism. Implemented as `timbers hook run post-commit` printing a one-line reminder.

**Consequences:**
- Works with any agent framework that shells out to git, not just Claude Code
- `timbers init --hooks` installs the hook; `doctor` checks and `--fix` auto-installs — full lifecycle management
- Occupies the `post-commit` hook slot — users with existing post-commit hooks need to chain them
- Couples reminder visibility to git's hook stdout forwarding behavior, which is well-established and unlikely to change

---

## ADR-8: Static Examples from Dense Date Ranges Over Per-Release Regeneration

**Context:** Site examples (changelog, decision-log, etc.) were regenerated on every release via LLM calls. This was expensive and produced inconsistent output across runs, since the LLM might rephrase the same entries differently each time.

**Decision:** Split example generation into static (fixed date range with dense, high-quality entries) and dynamic (per-release). Static examples use a curated Feb 10-14 range that showcases rich entries; only release-specific content regenerates.

**Consequences:**
- Eliminates redundant LLM calls for showcase content that doesn't need to change
- Curated date range guarantees examples always demonstrate the tool's best output
- Static examples may drift from current template behavior over time — requires occasional manual refresh
- Dynamic examples still regenerate per-release, keeping release-specific content current
