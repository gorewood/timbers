+++
title = 'Decision Log'
date = '2026-03-02'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Git Hooks Over PostToolUse for Agent Compliance Nudging

**Context:** Timbers needed a mechanism to remind agents to run `timbers log` after commits. Two options existed: Claude Code's `PostToolUse` hook system (fires after tool invocations) or a standard git `post-commit` hook. Diagnostic testing confirmed `PostToolUse` hooks fire correctly and receive valid JSON on stdin, but Claude Code does not surface their stdout to the agent — output goes nowhere visible.

**Decision:** Use git `post-commit` hooks instead of `PostToolUse` hooks. Git hook stdout is visible to agents, making it the reliable nudge mechanism. The existing `Stop` hook (which runs `timbers pending` at session end) was retained as a fallback. `PostToolUse` was removed entirely rather than worked around, and a `retiredEvents` cleanup mechanism was added so upgrades remove stale hook registrations.

**Consequences:**
- Agents see the reminder immediately after each commit, at the point of highest relevance
- No dependency on Claude Code's hook output behavior, which could change without notice
- Requires git hook installation (`timbers init --hooks`), adding a setup step users might skip
- The `Stop` hook still catches missed logging at session end, providing defense in depth
- Future Claude Code fixes to `PostToolUse` stdout won't retroactively change the approach — git hooks are the primary mechanism now

## ADR-2: Display-Layer Pending Fix Over Storage-Layer Change

**Context:** Fresh repositories with no timbers entries showed all historical commits as "pending," which was confusing for new users. The initial plan changed `GetPendingCommits` in the storage layer to return empty when no entries existed. This broke `timbers log`, `timbers log --batch`, and the MCP log handler — all three need commits from `GetPendingCommits` to create the very first entry.

**Decision:** Fix at the display layer, not the storage layer. `pending.go` and `doctor_checks.go` check `latest == nil` and show a friendly onboarding message instead of a wall of "pending" commits. Storage behavior is unchanged — `GetPendingCommits` still returns all commits when no entries exist.

**Consequences:**
- All callers that need commits for entry creation (`log`, `batch`, MCP) continue working unmodified
- The fix is localized to two display-layer files rather than requiring every storage consumer to handle a new edge case
- New JSON `status` field distinguishes "no entries yet" from "all caught up" for machine consumers
- Storage layer remains a faithful representation of git state, with interpretation pushed to the edges

## ADR-3: Actionable Warnings Over Anchor-Reset Command for Stale Anchors

**Context:** After squash merges or rebases, a timbers entry's anchor commit can disappear from history, causing confusing behavior. Two approaches were considered: add a `timbers anchor reset` command for explicit recovery, or improve warning messages with coaching and let the anchor self-heal on the next `timbers log`.

**Decision:** Actionable warnings plus coaching, no reset command. The anchor self-heals naturally when the user runs `timbers log` after their next real commit. Warning messages were improved across CLI and MCP to explain the situation and what to do (or not do). A `stale-anchor` coaching section was added to the prime workflow output.

**Consequences:**
- Zero new commands to maintain, document, or teach
- Users don't need to understand anchor internals — they just keep working normally
- Self-healing behavior means the problem resolves without intervention in every case
- If self-healing ever proves insufficient for some edge case, a reset command can still be added later
- Relies on users reading warnings, which agents do reliably but humans may not

## ADR-4: Check-Before-Generate Over LLM Refusal for Empty Entry Sets

**Context:** The automated devblog generation workflow invoked the LLM even when no timbers entries existed in the date range. The LLM, having no entries to summarize, generated apologetic placeholder posts ("Sorry, no development activity to report"). Two fixes were possible: teach the LLM prompt to output nothing when given empty input, or check entry count before invoking the LLM at all.

**Decision:** Add an entry count check in the CI workflow before the LLM generation step. Gate the generate/commit/push steps on count > 0.

**Consequences:**
- Saves LLM API costs on days with no development activity
- Deterministic behavior — empty input always means no output, no prompt engineering required
- Eliminated 10 existing blank/apology posts that had been published
- The pattern generalizes: any draft template pipeline should check inputs before invoking LLM generation

## ADR-5: Health Check in Prime Over Doctor-Only Diagnostics

**Context:** `timbers doctor` performs comprehensive health checks, but agents only run it when something is already broken. Missing hooks or integrations could go undetected for entire sessions because `doctor` isn't part of the standard workflow. `timbers prime` is the session entry point — every agent session starts with it.

**Decision:** Add a quick health check to `timbers prime` output that surfaces missing post-commit hooks and agent environment issues. When problems are found, a Health section appears with a `timbers doctor --fix` hint. The check is lightweight — a subset of what `doctor` does, focused on the most impactful issues.

**Consequences:**
- Agents see configuration problems before starting work, not after wasting a session
- Quick check adds minimal latency to `prime` (two fast filesystem checks vs. `doctor`'s full suite)
- Creates a layered diagnostic approach: `prime` catches common issues proactively, `doctor` handles deep investigation
- `prime` output grows slightly, though the Health section only appears when issues exist

## ADR-6: Rename exec-summary to standup, Clean Break Without Aliases

**Context:** The `exec-summary` template name was not discoverable — users didn't think to ask for an "executive summary" of their daily work. `standup` better matches the use case (daily status updates). The question was whether to keep `exec-summary` as a backward-compatible alias.

**Decision:** Clean rename to `standup` with no alias. Pre-GA project with low usage means backward compatibility adds complexity for no real benefit.

**Consequences:**
- Template name now matches the mental model users have for the task ("what did I do today")
- No alias maintenance, no dual-name documentation, no "which name do I use" confusion
- Any existing scripts using `exec-summary` break — acceptable given pre-GA status
- Sets precedent: pre-GA is the time to make breaking naming changes freely

## ADR-7: PR Template Focused on Intent and Decisions Over Diff Summary

**Context:** The `pr-description` template (v3) summarized code changes from the diff. But agents reviewing PRs already have full diff access — a template that restates the diff adds no information. Timbers entries capture intent (`why`) and deliberation (`notes`) that diffs don't show.

**Decision:** Rewrote `pr-description` (v4) to focus on intent, design decisions, and trade-offs extracted from timbers entries rather than summarizing what changed in the code.

**Consequences:**
- PR descriptions now contain information reviewers can't get from the diff alone
- Reviewers understand *why* changes were made, enabling better review feedback
- Depends on entry quality — thin `why` fields produce thin PR descriptions
- Template is more opinionated about what a good PR description contains

## ADR-8: Self-Service Provider Discovery via `--models` Flag

**Context:** Users on non-Claude CLIs (Codex, Gemini) had no documentation for piping `timbers draft` output through their preferred LLM. Agents needed to discover which providers and API keys were available without external documentation.

**Decision:** Added `draft --models` flag backed by a `ProviderInfos()` export in `llm.go`. Agents can query available providers and their required environment variables at runtime. Documentation was updated with verified piping syntax for each CLI.

**Consequences:**
- Agents can self-serve provider discovery without hardcoded knowledge of API key names
- `ProviderInfos()` returns a copy via `maps.Copy` to prevent callers from mutating the provider registry (caught in review)
- Each CLI's piping syntax was locally verified — Gemini auto-detects piped stdin (no `-p` needed), Codex `-m` flag belongs on the `exec` subcommand
- Provider list requires manual updates when new CLIs emerge, but the discovery mechanism itself is stable
