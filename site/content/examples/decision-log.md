+++
title = 'Decision Log'
date = '2026-03-02'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

## ADR-1: Auto-Commit Entry Files on Working Branch with Pathspec Scoping

**Context:** `timbers log` created entry files in `.timbers/` but left them staged-and-uncommitted, requiring a manual `git commit` step. The original design likely assumed git-notes storage where entries didn't create commits on the working branch. Three approaches were considered: (1) keep manual commit, (2) auto-commit to a separate branch like beads/entire.io, (3) auto-commit on the working branch.

**Decision:** Auto-commit on the working branch using `git commit -m ... -- <path>` pathspec scoping. A separate branch was rejected because agent DX depends on filesystem visibility — `timbers prime` and `timbers draft` need to read entries without worktree indirection. The pathspec `--` operator scopes the commit to the entry file only, preventing other staged files from being swept in.

**Consequences:**
- Eliminates the staged-but-uncommitted gap that confused users
- Entries are immediately visible to all commands without cross-branch reads
- `DefaultGitCommit` is injected via `GitCommitFunc` for testability
- Creates additional commits on the working branch (one per `timbers log` call)
- Constrains future storage migration — moving off working-branch files would break the visibility contract

## ADR-2: Handle No-Entries State at Display Layer, Not Storage Layer

**Context:** Newly onboarded repos showed all pre-timbers history as "pending commits," which was confusing. The initial plan changed `GetPendingCommits` to return empty when no entries existed, but this broke `timbers log`, `timbers log --batch`, and MCP log — all of which need commits returned to create the first entry.

**Decision:** Move the fix to the display layer. `pending.go` and `doctor_checks.go` check `latest == nil` and intercept the no-entries state before rendering. Storage behavior is unchanged — `GetPendingCommits` continues to return commits even when no entries exist.

**Consequences:**
- All write callers (`log`, `batch`, MCP) continue working for first-entry creation
- Display callers must each handle the `latest == nil` case independently
- Storage API remains honest about repository state rather than lying based on caller context
- New display commands that show pending state need to add the same nil check

## ADR-3: Coaching and Warnings Over Reset Command for Stale Anchors

**Context:** After squash merges, the anchor commit disappears from history, causing `timbers pending` to show confusing results. Two approaches were considered: (1) add a `timbers anchor-reset` command to explicitly fix the state, or (2) surface actionable warnings and add coaching to `timbers prime` output.

**Decision:** Actionable warnings plus coaching, no reset command. The anchor self-heals the next time `timbers log` runs after a real commit, so an explicit reset is unnecessary machinery.

**Consequences:**
- No new command surface area to maintain or document
- Users get clear messaging about what happened and that it will self-heal
- Agents receive coaching in `prime` output explaining the pattern
- If a user's anchor is stale and they haven't committed new work yet, they must wait for the next commit cycle — no manual escape hatch exists

## ADR-4: Drop PostToolUse Hook in Favor of Existing Stop Hook

**Context:** A `PostToolUse` hook was configured to remind agents about undocumented commits after tool calls. Diagnostics confirmed the hook fired correctly and received proper JSON on stdin, but Claude Code did not surface the hook's stdout to the agent. Two options: (1) work around Claude Code's behavior, or (2) rely on the existing `Stop` hook which runs `timbers pending` at session end.

**Decision:** Remove `PostToolUse` entirely and lean on the `Stop` hook. Rather than building workarounds for platform behavior, use the hook point that actually works and checks real state at session end.

**Consequences:**
- Simpler hook configuration — one fewer event to maintain
- Agents don't get mid-session reminders (only end-of-session check)
- Added `retiredEvents` cleanup list so upgrades remove the stale hook
- If Claude Code later fixes PostToolUse stdout surfacing, the hook could be re-added

## ADR-5: Pre-Check Entry Count Over Teaching LLM to Refuse

**Context:** The devblog CI workflow invoked the LLM even when no timbers entries existed for the period. The LLM responded by generating "apology posts" explaining there was nothing to write about. Two fixes: (1) add prompt engineering to teach the LLM to output nothing when given empty input, or (2) check entry count before invoking the LLM.

**Decision:** Check-before-generate. Gate the generate/commit/push steps on entry count > 0 in the GitHub Actions workflow.

**Consequences:**
- Zero LLM cost on days with no entries (cheaper and deterministic)
- No reliance on LLM following "output nothing" instructions (which is notoriously unreliable)
- Required deleting 10 blank/apology posts already committed to the site
- If the count check has edge cases (e.g., entries exist but are all empty), the LLM never gets a chance to handle them gracefully

## ADR-6: Clean Break Rename Over Backward-Compatible Alias

**Context:** The `exec-summary` template was being renamed to `standup` for better discoverability. The question was whether to keep `exec-summary` as an alias during the transition.

**Decision:** Clean break — rename without alias. The project is pre-GA with low usage, so backward compatibility adds complexity for no real benefit.

**Consequences:**
- Simpler template resolution code — no alias mapping layer
- Existing scripts or muscle memory using `exec-summary` break immediately
- Sets precedent that pre-GA naming changes can be breaking
- Post-GA, the same decision would likely go the other way

## ADR-7: PR Template Focused on Intent and Decisions Over Diff Summary

**Context:** The `pr-description` draft template was being rewritten (v4). The previous version summarized code diffs, but agents using the template already have access to `git diff` output.

**Decision:** Shift the PR template to focus on intent and decisions. Agents already review diffs — the template's value is synthesizing *why* changes were made and what trade-offs were involved, not restating *what* changed.

**Consequences:**
- PR descriptions carry architectural context that diffs alone don't convey
- Better leverages timbers' `why` and `notes` fields as primary source material
- Reviewers get decision rationale without reading every line of diff
- Template is less useful in workflows where `why`/`notes` fields are thin or absent

## ADR-8: Static Examples from Dense Date Ranges Over Per-Release Regeneration

**Context:** Site examples were being regenerated on every release, but the LLM calls were expensive and the output quality depended on having a dense cluster of interesting entries. Some date ranges produced much better examples than others.

**Decision:** Split the justfile into `examples` (dynamic, regenerated per release) and `examples-static` (fixed to a known-good Feb 10-14 date range that produced high-quality output from a dense work period).

**Consequences:**
- Eliminates redundant LLM calls for content that doesn't improve with more entries
- Best examples are preserved regardless of future entry quality or density
- Static examples may grow stale if schema or features change significantly
- Two generation paths to maintain instead of one
