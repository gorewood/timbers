+++
title = 'Decision Log'
date = '2026-02-27'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Grep Over jq for Hook Input Parsing

**Context:** Claude Code hooks receive JSON on stdin. The `PostToolUse` hook needed to detect `git commit` commands to remind users about `timbers pending`. Options were full JSON parsing with `jq` to extract `tool_input.command` specifically, or grepping the raw JSON blob.

**Decision:** Grep the full JSON blob for `git commit` instead of parsing with `jq`. Simpler, no external dependency, and false positive risk is negligible — the string `git commit` appearing in unrelated JSON fields is unlikely enough to not warrant structured parsing.

**Consequences:**
- Positive: Zero dependencies, single-line implementation, works on any system with `grep`
- Positive: Easier to debug — no JSON schema coupling to break
- Negative: Theoretically matches false positives if `git commit` appears in other JSON fields
- Negative: Can't extract specific fields if future hooks need richer input inspection

---

## ADR-2: `--color` Flag Over Full Theme Config

**Context:** Users on Solarized Dark reported Color 8 (bright black) was invisible. Terminals don't report their color scheme, so the tool can't auto-detect. Options ranged from a full theme config system (env vars, config files, multiple palettes) to a simple `--color` flag paired with `AdaptiveColor`.

**Decision:** Ship `--color never/auto/always` as a persistent global flag combined with lipgloss `AdaptiveColor`, deferring full theme config. This covers 95% of cases without maintenance burden. Lipgloss v1.1.0's `HasDarkBackground()` was noted for future use but not adopted yet.

**Consequences:**
- Positive: Immediate fix for dark terminal users with minimal API surface
- Positive: `AdaptiveColor` handles light/dark switching automatically for most terminals
- Negative: Users with unusual color schemes still can't fine-tune individual colors
- Negative: Deferred `HasDarkBackground()` means some auto-detection capability is left on the table

---

## ADR-3: Auto-Commit Entry Files with Pathspec Scoping

**Context:** `timbers log` created entry files in `.timbers/` but left them staged-and-uncommitted, causing user confusion. The gap likely originated from the git-notes era where entries didn't create commits on the working branch. Options: (1) auto-commit the entry file, (2) store entries on a separate branch (like beads or Entire.io), (3) leave as manual step.

**Decision:** Auto-commit using `git commit -m ... -- <path>` with pathspec scoping. Separate branch was rejected because agent DX depends on filesystem visibility — `timbers prime` and `timbers draft` need to read entries without worktree indirection. The `--` pathspec prevents sweeping other staged files into the timbers commit.

**Consequences:**
- Positive: Eliminates the staged-but-uncommitted gap entirely
- Positive: Entries are immediately available to all timbers commands without extra steps
- Positive: Pathspec scoping makes the auto-commit surgically safe
- Negative: Creates additional commits on the working branch (one per `timbers log`)
- Negative: Entries must live on the working branch, ruling out branch-based isolation patterns

---

## ADR-4: Remove PostToolUse Hook Over Platform Workaround

**Context:** Diagnostic confirmed the `PostToolUse` hook fired correctly and received valid JSON on stdin, but Claude Code didn't surface stdout — the hook's output went nowhere visible. Options: work around the platform behavior (write to a file, use notifications, etc.) or remove the hook and rely on the existing `Stop` hook.

**Decision:** Remove `PostToolUse` entirely. The `Stop` hook already runs `timbers pending` at session end, covers the same use case (reminding about undocumented commits), and actually displays its output. Working around Claude Code behavior adds fragile complexity.

**Consequences:**
- Positive: Eliminates a silently broken hook that was a no-op since creation
- Positive: Simpler hook surface — fewer events to maintain and test
- Negative: No per-commit reminder; feedback only at session end via `Stop` hook
- Negative: If a user makes many commits without documenting, they only find out at the end

---

## ADR-5: Warnings and Coaching Over Reset Command for Stale Anchors

**Context:** After squash merges, the anchor commit disappears from history, producing confusing `timbers pending` output. Options: (1) add an explicit `timbers anchor-reset` command, (2) use actionable warnings plus coaching to guide users, relying on the anchor's self-healing behavior (it auto-corrects on the next `timbers log`).

**Decision:** Warnings plus coaching, no reset command. The anchor self-heals on the next `timbers log`, making an explicit reset unnecessary. Messaging alone is sufficient — tell users what happened and that normal workflow will fix it.

**Consequences:**
- Positive: No new command to document, test, and maintain
- Positive: Users learn the mental model (anchors self-heal) instead of reaching for a command
- Negative: First encounter with stale anchor is still confusing until the user reads the warning
- Negative: No escape hatch if self-healing doesn't cover an edge case

---

## ADR-6: Pre-Generate Check Over LLM Refusal for Empty Entries

**Context:** The devblog workflow invoked the LLM even when no timbers entries existed for the period, causing the LLM to generate apologetic "nothing to report" posts. Options: teach the LLM to refuse gracefully when given empty input, or check entry count before invoking generation.

**Decision:** Check entry count before generation and skip the entire pipeline when count is zero. Cheaper and cleaner than prompt-engineering the LLM to produce a useful refusal.

**Consequences:**
- Positive: Eliminates wasted LLM calls and their associated cost/latency
- Positive: No "apology posts" to detect and clean up after the fact
- Positive: Simple boolean gate — easier to reason about than LLM behavior
- Negative: Slightly more workflow complexity (conditional step)
- Negative: If the check has bugs, generation silently skips when it shouldn't

---

## ADR-7: Govulncheck Separate from Lint Check

**Context:** `govulncheck` isn't included in `golangci-lint` and needs to be run as a separate tool. The question was whether to add it to the `just check` gate (which must pass before every commit) or keep it as a separate recipe.

**Decision:** Separate `just vulncheck` recipe, not part of `just check`. Stdlib vulnerability patches can flag findings that aren't actionable yet, and blocking every commit on them would be disruptive.

**Consequences:**
- Positive: `just check` stays fast and actionable — developers aren't blocked by upstream stdlib patches
- Positive: Vulnerability scanning is still available on demand
- Negative: Vulncheck isn't enforced automatically — requires discipline to run periodically
- Negative: Vulnerabilities could ship if nobody remembers to check

---

## ADR-8: Clean Rename Over Backward-Compat Alias Pre-GA

**Context:** The `exec-summary` template was renamed to `standup` for better discoverability. Options: clean rename (breaking change) or keep `exec-summary` as an alias alongside the new name.

**Decision:** Clean break, no alias. Pre-GA project with low usage means backward compatibility adds complexity for no real benefit. Aliases accumulate maintenance cost and confuse documentation.

**Consequences:**
- Positive: Single canonical name — no confusion about which to use
- Positive: No alias resolution code to maintain
- Negative: Anyone using `exec-summary` in scripts breaks (mitigated by low pre-GA adoption)
- Negative: Sets precedent that names can change before GA — users may hesitate to automate

---

## ADR-9: PR Template Focused on Intent/Decisions Over Diff Summary

**Context:** The `pr-description` template was at v3, producing summaries that largely restated the diff. Agents (and human reviewers) already read diffs directly, making diff summaries redundant. The question was what a PR description template should actually contain.

**Decision:** Rewrote to v4, shifting focus to intent and design decisions. Since agents already review diffs, the template's value is in surfacing *why* changes were made and what trade-offs were considered — information not visible in the code itself.

**Consequences:**
- Positive: PR descriptions carry information that complements rather than duplicates the diff
- Positive: Leverages timbers' unique strength — the why/notes fields that diffs don't capture
- Negative: Requires richer ledger entries to produce good output (garbage in, garbage out)
- Negative: Reviewers accustomed to "what changed" summaries may initially find the format unfamiliar
