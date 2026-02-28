+++
title = 'Decision Log'
date = '2026-02-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Auto-Commit Entries on Working Branch Over Separate Branch

**Context:** `timbers log` created entry files in `.timbers/` but left them staged-and-uncommitted, causing user confusion. The original design likely assumed git-notes-style storage where entries don't land on the working branch. Two alternatives existed: auto-commit on the working branch (scoped to the entry file via pathspec), or store entries on a separate branch (similar to beads/entire.io).

**Decision:** Auto-commit on the working branch using `git commit -m ... -- <path>` to scope the commit to the entry file only. A separate branch was rejected because agent DX depends on filesystem visibility — `timbers prime` and `timbers draft` need to read entries without worktree indirection.

**Consequences:**
- Eliminates the staged-but-uncommitted gap that confused users
- Entries are immediately visible to all commands without branch-switching
- Pathspec (`--`) prevents accidentally sweeping other staged files into the timbers commit
- Entry commits interleave with work commits on the same branch, adding noise to `git log`
- No isolation between ledger state and code state — a force-push loses both

## ADR-2: Minimal Color Flag Over Full Theme Configuration

**Context:** Users on dark terminals (Solarized Dark) reported invisible text — `lipgloss.Color(8)` (bright black) doesn't render on dark backgrounds. Options ranged from a simple `--color` flag with `AdaptiveColor`, to full theme configuration via env vars and config files. Lipgloss v1.1.0 also offers `HasDarkBackground()` for runtime detection.

**Decision:** Ship a global `--color` persistent flag (`never`/`auto`/`always`) paired with `AdaptiveColor` for all color values. Full theme configuration deferred — `AdaptiveColor` + `--color` covers 95% of cases without maintenance burden.

**Consequences:**
- Immediate fix for dark terminal users with minimal surface area
- No config file format to design or maintain
- Users who need fine-grained control (specific palette, per-element colors) are not served
- `HasDarkBackground()` remains unused — could enable fully automatic detection later without breaking the flag

## ADR-3: Stop Hook Over PostToolUse for Pending Commit Checks

**Context:** A `PostToolUse` hook was intended to remind users about undocumented commits after tool executions. Diagnostic testing confirmed the hook fires correctly and receives valid JSON on stdin, but Claude Code does not surface `stdout` from `PostToolUse` hooks — output goes nowhere visible.

**Decision:** Remove `PostToolUse` hook entirely and rely on the existing `Stop` hook, which runs `timbers pending` at session end. Rather than working around Claude Code's behavior, lean on the hook that actually displays output and checks real state.

**Consequences:**
- Users see pending-commit reminders at session end rather than after each tool call
- Simpler hook configuration with fewer moving parts
- Added `retiredEvents` cleanup list so upgrades remove stale hooks automatically
- Mid-session reminders are lost — if a user works a long session without stopping, they won't be reminded until the end
- Couples the design to Claude Code's current hook behavior, which may change

## ADR-4: Clean Break Rename Over Backward-Compatible Alias

**Context:** The `exec-summary` template was renamed to `standup` for better discoverability. The question was whether to keep `exec-summary` as an alias during the transition.

**Decision:** Clean break — no alias. Pre-GA project with low usage; backward compatibility adds complexity for no real benefit.

**Consequences:**
- Zero alias-resolution code to maintain
- Any existing scripts or muscle memory referencing `exec-summary` break immediately
- Sets precedent that pre-GA naming changes are clean breaks, reducing future alias debt

## ADR-5: PR Template Focused on Intent Over Diff Review

**Context:** The `pr-description` template (v4) needed a rewrite. The previous version attempted to summarize code diffs, but agents reviewing PRs already have full diff access and don't need the template to duplicate that work.

**Decision:** Shift the PR template to focus on intent and design decisions rather than diff summarization. Agents already review diffs — the template's value is in the "why" context that diffs don't convey.

**Consequences:**
- PR descriptions capture information that's genuinely additive to the diff
- Template output is shorter and more focused
- Reviewers who rely solely on the PR description (without reading diffs) get less mechanical detail
- Aligns with timbers' core philosophy: capture decisions, not mechanics

## ADR-6: Check-Before-Generate Over Teaching LLM to Refuse

**Context:** The devblog CI workflow invoked the LLM even when no new entries existed, producing "apology posts" — content about having nothing to write about. Two fixes: add an entry-count check before generation, or improve the prompt to teach the LLM to output nothing when given empty input.

**Decision:** Check-before-generate. Gate the LLM call on entry count > 0. Cheaper and more reliable than prompt engineering a refusal behavior.

**Consequences:**
- Zero LLM cost on days with no entries
- Deterministic behavior — no risk of the LLM deciding to generate anyway
- Required deleting 10 existing blank posts from the repository
- If the check logic has bugs (e.g., timezone edge cases), entries could be silently skipped

## ADR-7: Govulncheck Separate from Main Quality Gate

**Context:** `govulncheck` (Go vulnerability scanning) is not included in `golangci-lint` and needs a separate invocation. The question was whether to add it to `just check` (the required pre-commit gate) or keep it as a standalone recipe.

**Decision:** Separate `just vulncheck` recipe, not part of `just check`. Vulnerability reports against stdlib patches would block development on issues developers can't immediately fix.

**Consequences:**
- `just check` remains fast and actionable — every failure is fixable before commit
- Vulnerability scanning requires explicit invocation, which means it can be forgotten
- Stdlib false positives don't block development workflow
- CI could run `vulncheck` separately with advisory-only status

## ADR-8: Actionable Warnings Over Explicit Reset Command for Stale Anchors

**Context:** After squash merges, timbers' anchor commit disappears from history, producing confusing `pending` output. Two approaches: add a `timbers anchor-reset` command for explicit recovery, or improve warning messages and add coaching to `prime` output so users understand the self-healing behavior.

**Decision:** Warnings plus coaching, no reset command. The anchor self-heals on the next `timbers log` — an explicit reset command is unnecessary machinery when messaging alone is sufficient.

**Consequences:**
- No new command to document, test, or maintain
- Users must understand the self-healing model to trust the warnings
- Warning messages needed improvement across both CLI and MCP surfaces to be genuinely actionable
- If self-healing assumptions break (e.g., orphaned repos), there's no manual escape hatch

## ADR-9: Static Examples from Fixed Date Ranges Over Per-Release Regeneration

**Context:** Site examples were regenerated on every release using `timbers draft`, invoking the LLM each time. Examples drawn from dense date ranges (Feb 10-14) with rich entries produced consistently good output. Regenerating the same high-quality range on every release wasted LLM calls without improving results.

**Decision:** Split example generation into `examples` (dynamic, per-release) and `examples-static` (fixed Feb 10-14 range). Static examples are generated once and committed; only new templates or format changes trigger regeneration.

**Consequences:**
- Eliminates redundant LLM calls for content that doesn't change
- Best-quality examples are preserved rather than risked on each regeneration
- Static examples may drift from current template behavior if templates evolve significantly
- Two generation paths to maintain instead of one
