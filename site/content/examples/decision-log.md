+++
title = 'Decision Log'
date = '2026-02-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Auto-Commit Entry Files on Working Branch

**Context:** When `timbers log` creates an entry file in `.timbers/`, the file was staged but not committed, leaving a gap where users had to manually `git commit`. The original design likely assumed git-notes storage where entries didn't create commits on the working branch. An alternative was committing entries on a separate branch (like beads/entire.io use), which would keep the working branch clean.

**Decision:** Auto-commit the entry file directly on the working branch using `git commit -m ... -- <path>` (pathspec-scoped). A separate branch was rejected because agent DX depends on filesystem visibility — `timbers prime` and `timbers draft` need to read entries without worktree indirection.

**Consequences:**
- Eliminates the staged-but-uncommitted confusion that caused user friction
- Pathspec (`--`) prevents sweeping other staged files into the timbers commit — safe by construction
- Entry files are immediately visible to all tools without branch switching
- Adds commits to the working branch that are tooling artifacts, not code changes
- Requires `GitCommitFunc` injection in `FileStorage` for testability

---

## ADR-2: `--color` Flag Over Full Theme Configuration

**Context:** Color 8 (bright black) was invisible on Solarized Dark terminals. Users reported dim/hint text was unreadable. Options ranged from a full theme system (env vars, config files, multiple palettes) to lipgloss `AdaptiveColor` with a simple override flag. Lipgloss v1.1.0 has `HasDarkBackground()` but it wasn't yet integrated.

**Decision:** Ship a global `--color` persistent flag (`never`/`auto`/`always`) plumbed through `ResolveColorMode`, combined with `AdaptiveColor` for automatic light/dark switching. Deferred full theme configuration.

**Consequences:**
- Covers ~95% of terminal color compatibility cases without maintenance burden
- `AdaptiveColor` provides sensible defaults without user intervention for most terminals
- Users with non-standard schemes have an escape hatch via `--color`
- Defers `HasDarkBackground()` integration and per-element theme control to a future iteration
- Every `NewPrinter` call site must thread the color mode through

---

## ADR-3: Warnings and Coaching Over Reset Command for Stale Anchors

**Context:** After squash merges or rebases, the anchor commit in a timbers entry can go missing from history. This caused confusing behavior for users. Two approaches: build an explicit `timbers anchor-reset` command, or rely on actionable warnings plus coaching documentation to guide users through self-healing.

**Decision:** Warnings plus coaching, no reset command. The anchor self-heals the next time `timbers log` runs after a real commit, making an explicit reset unnecessary.

**Consequences:**
- No new command to maintain, test, or document
- Users get clear guidance in the moment (warning messages) and in context (coaching section in `prime` output)
- Self-healing means no permanent state corruption from squash merges
- `timbers pending` may show already-documented commits during the stale window, which could confuse users who don't read the warning
- If the self-healing assumption breaks in edge cases, there's no manual override available

---

## ADR-4: Remove PostToolUse Hook in Favor of Stop Hook

**Context:** A `PostToolUse` hook was intended to remind agents to run `timbers log` after `git commit`. Diagnostic testing confirmed the hook fired correctly and received valid JSON on stdin. However, Claude Code doesn't surface hook stdout to the user or agent — the output went nowhere visible, making the post-commit reminder a silent no-op since it was created.

**Decision:** Remove `PostToolUse` entirely. The existing `Stop` hook already runs `timbers pending` at session end, which covers the same case by checking actual state rather than intercepting tool calls.

**Consequences:**
- Eliminates dead code that appeared to work but had no observable effect
- `Stop` hook checks real state (`timbers pending`) rather than inferring intent from tool calls — more reliable
- Reminder shifts from per-commit to session-end, which means agents may accumulate undocumented commits during a session
- Added `retiredEvents` cleanup list so upgrades remove stale hooks automatically

---

## ADR-5: Pre-Check Over LLM Refusal for Empty Inputs

**Context:** The devblog generation workflow invoked the LLM even when no timbers entries existed for the period. The LLM responded by generating apologetic "nothing happened" posts. Two approaches: teach the LLM to refuse gracefully when given empty input, or check entry count before invoking the LLM at all.

**Decision:** Check entry count before generation. Gate the generate/commit/push steps on count > 0.

**Consequences:**
- Cheaper — avoids an LLM API call entirely when there's nothing to generate
- Deterministic — no risk of the LLM deciding to generate something anyway
- Simpler — a count check is a shell conditional, not prompt engineering
- Doesn't generalize to cases where the LLM should make nuanced judgments about input sufficiency
- Required deleting 10 blank posts that had already been published

---

## ADR-6: PR Template Focused on Intent/Decisions, Not Diff Summaries

**Context:** The `pr-description` template (v3) summarized code diffs. But agents reviewing PRs already read the diff themselves — a template that restates the diff adds no value. The template needed to provide what agents *can't* infer from code alone.

**Decision:** Rewrote `pr-description` to v4, shifting focus to intent and design decisions. The template now extracts *why* choices were made and what trade-offs were considered, using the ledger's `why` and `notes` fields as primary sources.

**Consequences:**
- PR descriptions complement rather than duplicate what reviewers already see in the diff
- Leverages timbers' unique data (design decisions) rather than competing with tools that summarize code changes
- Requires entries to have substantive `why` fields — thin entries produce thin PR descriptions
- Agents reviewing PRs get the context they actually need to evaluate whether changes are appropriate

---

## ADR-7: Grep Over jq for Hook Stdin JSON Parsing

**Context:** Claude Code hooks receive JSON on stdin containing tool call details. The `PostToolUse` hook (before removal) needed to detect `git commit` invocations. Options: parse the JSON properly with `jq` to extract `tool_input.command`, or grep the raw JSON blob for the string `git commit`.

**Decision:** Grep the full JSON blob directly. No `jq` dependency, simpler implementation, and the false positive risk of matching `git commit` anywhere in the JSON is negligible in practice.

**Consequences:**
- Zero external dependencies — works on any system with grep
- Simpler hook script (one pipeline vs JSON extraction)
- Theoretically could match `git commit` in unrelated JSON fields, but practically this never happens in Claude Code's hook payload
- If JSON structure changes or payloads grow more complex, grep becomes less reliable than structured parsing

---

## ADR-8: Intentional Documentation Over Automatic Session Capture

**Context:** Entire.io launched with automatic session capture — recording everything an agent does. Timbers needed to define its positioning in this emerging space. The fork: capture everything automatically, or require intentional documentation of design decisions.

**Decision:** Positioned Timbers as intentional documentation — structured `what/why/how` records written deliberately, not automatic session transcripts. The landing page and messaging emphasize this contrast.

**Consequences:**
- Entries contain curated design decisions rather than raw session noise
- Requires user/agent effort to write entries — adoption depends on the value being worth the friction
- Produces higher signal-to-noise for downstream consumers (changelogs, ADRs, standups)
- Cannot capture decisions that users forget to document — no safety net of automatic recording
- Competitive differentiation is clear but requires ongoing messaging to maintain
