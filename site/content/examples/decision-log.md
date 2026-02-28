+++
title = 'Decision Log'
date = '2026-02-28'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Architectural Decision Log — Timbers (2026-02-14 to 2026-02-28)

## ADR-1: Auto-Commit Entry Files with Pathspec Scoping

**Context:** After `timbers log` created an entry file in `.timbers/`, users had to manually `git add` and `git commit` it. This staged-but-uncommitted gap caused confusion — entries existed on disk but weren't part of the commit history. Three approaches were considered: auto-commit scoped to the entry file, writing entries to a separate branch (like beads and entire.io), or leaving the manual step.

**Decision:** Auto-commit the entry file using `git commit -m ... -- <path>` with pathspec scoping. The separate-branch approach was rejected because agent DX depends on filesystem visibility — `timbers prime` and `timbers draft` need to read entries without worktree indirection. The pathspec `--` delimiter ensures only the entry file is committed, never sweeping other staged files into the timbers commit.

**Consequences:**
- Eliminates the manual commit step, reducing workflow friction
- Entries are immediately part of commit history, visible to `git log` and push
- Pathspec scoping prevents accidentally committing unrelated staged changes
- Dependency injection via `GitCommitFunc` in `FileStorage` keeps the commit operation testable
- Locks entries to the working branch — cannot use branch-based isolation patterns later without migration

## ADR-2: Stop Hook Over PostToolUse for Pending Reminders

**Context:** A `PostToolUse` hook was implemented to remind agents about undocumented commits after tool use. Diagnostics confirmed the hook fired correctly and received proper JSON on stdin. However, Claude Code does not surface `stdout` from `PostToolUse` hooks — the output goes nowhere visible.

**Decision:** Remove the `PostToolUse` hook entirely and rely on the existing `Stop` hook, which runs `timbers pending` at session end. Rather than work around Claude Code's behavior (e.g., writing to files, using stderr), lean on the `Stop` hook which does display output and checks actual state.

**Consequences:**
- Simpler hook configuration — one fewer event to manage
- Pending check happens once at session end rather than after every tool use (less noise)
- Agents won't get mid-session reminders about undocumented commits
- Added `retiredEvents` cleanup list so upgrades remove stale hook registrations
- Dependency on Claude Code's `Stop` hook remaining reliable for output display

## ADR-3: Actionable Warnings Over Anchor-Reset Command for Stale Anchors

**Context:** After squash merges or rebases, the anchor commit referenced by `timbers pending` could go missing from history, producing confusing output. Two approaches: add an explicit `timbers anchor reset` command, or improve warning messages and add coaching to the prime workflow.

**Decision:** Warnings plus coaching, no reset command. The anchor self-heals the next time `timbers log` runs after a real commit, making an explicit reset unnecessary. Messaging alone is sufficient.

**Consequences:**
- No new command surface area to maintain or document
- Users understand what happened and why via actionable warning text
- Prime workflow coaching teaches agents to not re-document already-covered commits
- Self-healing behavior means no manual intervention required in the common case
- If a pathological case arises where self-healing isn't sufficient, there's no escape hatch — would need to add the command later

## ADR-4: Check-Before-Generate Over Teaching LLM to Refuse

**Context:** The devblog CI workflow invoked the LLM even when no timbers entries existed for the period. The LLM would generate "apology posts" explaining there was nothing to write about. Two approaches: add a precondition check for entry count > 0 before invoking the LLM, or engineer the prompt to handle empty input gracefully.

**Decision:** Check-before-generate. Gate the generate/commit/push steps on entry count > 0. This is cheaper (no LLM invocation at all) and more reliable than depending on prompt engineering to produce correct refusal behavior.

**Consequences:**
- Zero LLM cost on days with no entries
- Deterministic behavior — no risk of prompt drift producing unwanted posts
- Required deleting 10 already-published blank/apology posts
- Slightly more CI workflow complexity (extra step), but trivially simple logic
- Pattern generalizes: validate inputs before LLM calls rather than relying on LLM judgment for control flow

## ADR-5: Clean Break Rename Over Backward-Compatible Alias

**Context:** The `exec-summary` template was being renamed to `standup` for better discoverability. The question was whether to keep `exec-summary` as an alias during a transition period.

**Decision:** Clean break, no alias. The project is pre-GA with low usage — backward compatibility adds complexity for no real benefit at this stage.

**Consequences:**
- Simpler codebase — no alias resolution logic or deprecation warnings
- Users of `exec-summary` get a clear error, not silent redirection
- Sets precedent that pre-GA naming can change freely
- If adoption were higher, this decision would need revisiting — it's stage-appropriate, not universally correct

## ADR-6: Intent-Focused PR Template Over Diff-Centric Format

**Context:** The `pr-description` draft template (v3) focused on summarizing the diff. But agents already review diffs natively — a PR description that restates the diff adds no information. The template needed a different angle.

**Decision:** Rewrote `pr-description` (v4) to focus on intent and decisions: why these changes were made, what trade-offs were chosen, what the reviewer should pay attention to. Agents review diffs; humans review intent.

**Consequences:**
- PR descriptions complement rather than duplicate what reviewers can already see
- Leverages timbers' unique data (why/notes fields) that isn't in the diff
- Requires entries with good why/notes fields to produce useful output — garbage in, garbage out
- Template is opinionated about PR review culture (intent over mechanics)

## ADR-7: AdaptiveColor Plus --color Flag Over Full Theme Configuration

**Context:** User feedback reported that `lipgloss.Color(8)` (bright black) was invisible on Solarized Dark terminals. Terminals don't reliably report their color scheme. Options ranged from a full theme system (env vars, config files, multiple palettes) to a simpler flag-based approach.

**Decision:** `AdaptiveColor` for automatic dark/light switching plus a `--color` persistent flag (`never`/`auto`/`always`) for explicit user control. Full theme configuration deferred — `AdaptiveColor` + `--color` covers 95% of cases without maintenance burden.

**Consequences:**
- Most users get correct colors automatically via `AdaptiveColor`
- Power users can force behavior with `--color=never` for piping or `--color=always` for forced output
- Global persistent flag plumbed through `ResolveColorMode` to all `NewPrinter` call sites — moderate touch surface
- Lipgloss v1.1.0's `HasDarkBackground()` available but not yet used (future improvement path)
- If users need per-element color customization, the flag approach won't scale — but that's speculative

## ADR-8: Static Examples Split from Dynamic Per-Release Generation

**Context:** Site examples were regenerated via LLM on every release, even when the underlying entries hadn't changed. Dense date ranges (like Feb 10-14 with many entries) produced good examples that didn't benefit from re-generation.

**Decision:** Split the justfile into `examples` (dynamic, regenerated per-release) and `examples-static` (fixed date range, generated once). Static examples from known-good dense periods are cached; only dynamic examples incur LLM cost.

**Consequences:**
- Eliminates redundant LLM calls for stable content
- Static examples serve as a reliable baseline — known-good output that doesn't regress
- Two generation paths to maintain instead of one
- Static examples will eventually feel dated if the schema or template quality evolves significantly
- Trade-off is appropriate for a small project; at scale, a proper caching layer would replace this split

## ADR-9: govulncheck Separate from Lint Pipeline

**Context:** `govulncheck` is not included in `golangci-lint` and needed to be added as a separate tool. The question was whether to include it in `just check` (the mandatory pre-commit gate) or keep it as a separate recipe.

**Decision:** Separate `just vulncheck` recipe, not part of `just check`. Vulnerability reports against stdlib patches would block all commits even when the fix requires a Go version upgrade outside the developer's control.

**Consequences:**
- `just check` remains fast and actionable — every failure is something the developer can fix immediately
- Vulnerability scanning is opt-in, run on-demand or in CI
- Risk of forgetting to run it; relies on CI or developer discipline
- Clean separation between "code quality" gates and "supply chain" checks
