+++
title = 'Decision Log'
date = '2026-03-02'
tags = ['example', 'decision-log']
+++

Generated with `timbers draft decision-log --last 20 | claude -p --model opus`

---

# Decision Log

## ADR-1: Self-Healing Warnings Over Anchor Reset Command

**Context:** After squash merges or rebases, the timbers anchor commit disappears from history, leaving `ErrStaleAnchor`. Three options existed: (1) add an explicit `timbers anchor reset` command, (2) surface actionable warnings and let the anchor self-heal on next `timbers log`, or (3) treat it as a fatal error. Commands like `pending`, `prime`, and MCP already handled stale anchors gracefully, but `log` and `batch log` treated it as fatal — breaking the self-healing claim.

**Decision:** Warnings plus coaching, no reset command. `log` and `batch log` now accept fallback commits from `GetPendingCommits` when `ErrStaleAnchor` occurs, warn, and proceed. The anchor self-heals the next time a real entry is created, making an explicit reset unnecessary. Prime output includes a `<stale-anchor>` coaching section explaining the situation to agents.

**Consequences:**
- Agents and users recover from squash merges without manual intervention
- No new command surface area to maintain or document
- Messaging must be clear enough to prevent confusion — if the warning text is poor, users may think something is broken
- Self-healing behavior is harder to reason about than an explicit reset; debugging anchor issues requires understanding the implicit recovery path

## ADR-2: Health Checks Surfaced in Prime Entry Point

**Context:** Agents start sessions with `timbers prime`, which injects workflow context. Missing hooks, integrations, or configuration issues weren't discovered until agents encountered errors mid-session — after potentially significant wasted work.

**Decision:** Run a quick health check during `timbers prime` and surface issues with a `doctor --fix` hint in the output. Since prime is the session entry point, agents see the fix hint before starting any work.

**Consequences:**
- Agents can self-fix configuration issues at session start rather than failing mid-task
- `prime` output grows slightly, consuming context window tokens even when healthy
- Health checks are duplicated between `prime` (quick) and `doctor` (comprehensive) — changes to checks must stay in sync

## ADR-3: Stop Hook Over PostToolUse for Session-End Checks

**Context:** A `PostToolUse` hook was implemented to nudge agents toward `timbers log` after git commits. Diagnostic testing confirmed the hook fired correctly and received valid JSON on stdin. However, Claude Code does not surface `PostToolUse` hook stdout to the agent — the output goes nowhere visible.

**Decision:** Remove `PostToolUse` hook entirely. Rely on the existing `Stop` hook, which runs `timbers pending` at session end and whose output is displayed. Rather than work around Claude Code's behavior, lean on the mechanism that already works.

**Consequences:**
- Nudging happens once at session end rather than after each commit — agents may batch more commits before logging
- Simpler hook configuration with fewer events to manage
- Added `retiredEvents` cleanup list so upgrades automatically remove the defunct hook
- Dependent on Claude Code continuing to surface `Stop` hook output

## ADR-4: Git Post-Commit Hook Over Claude Code Hooks for Agent Nudging

**Context:** The `Stop` hook (ADR-3) only fires at session end, missing mid-session nudges. Claude Code's `PostToolUse` hooks fire but don't surface stdout. Git hooks, by contrast, produce stdout that is visible in the agent's bash output after every commit.

**Decision:** Use a native git `post-commit` hook (`timbers hook run post-commit`) that prints a one-line reminder. `timbers init --hooks` installs it, and `timbers doctor` checks for it with `--fix` auto-installation.

**Consequences:**
- Agents see a nudge after every commit, not just at session end
- Works with any tool that shells out to git, not just Claude Code
- Git hooks are per-repo and can be overwritten by other tools or user configuration
- Adds a new `hook` command namespace and hook installation/detection logic to maintain

## ADR-5: No-Entries Handling at Display Layer, Not Storage

**Context:** New repositories showed all pre-timbers git history as "pending" commits, overwhelming users on first run. The initial plan modified `GetPendingCommits` in the storage layer to return empty when no entries existed. This broke `timbers log`, `batch log`, and MCP log — all of which need commits to create the first entry.

**Decision:** Handle the no-entries state at the display layer. `pending.go` and `doctor_checks.go` check `latest==nil` and intercept before rendering. Storage behavior is unchanged, keeping all write-path callers working.

**Consequences:**
- `timbers log` works correctly for first-ever entry creation — the write path gets real commits
- Display callers (`pending`, `doctor`) must independently check for the no-entries case
- The storage API's contract is preserved — no special-case return values that downstream callers must handle
- New display-layer callers must remember to add the `latest==nil` guard

## ADR-6: Input Validation Over LLM Refusal for Empty Entries

**Context:** The devblog CI workflow invoked LLM generation even when no timbers entries existed for the period. The LLM generated "apology posts" explaining there was nothing to write about. Two options: teach the LLM to refuse gracefully via prompt engineering, or check for empty input before invoking the LLM at all.

**Decision:** Check-before-generate. Added an entry count step in the CI workflow, gating generation/commit/push on count > 0. Deleted 10 previously generated blank posts.

**Consequences:**
- Zero LLM cost for empty periods — no API calls made at all
- Simpler than prompt engineering a reliable refusal; no risk of the LLM deciding to generate anyway
- The guard is in CI workflow YAML, not in the tool itself — other callers of `timbers draft` must implement their own empty-input checks
- Immediate cleanup debt: 10 garbage posts had to be manually deleted

## ADR-7: Static Examples Over Per-Release LLM Regeneration

**Context:** Site examples (changelog, decision-log, etc.) were regenerated on every release via LLM. Dense date ranges with many entries produce good examples, but sparse ranges produce thin ones. Each regeneration costs LLM tokens and produces non-deterministic output.

**Decision:** Split the justfile into `examples` (dynamic, per-release) and `examples-static` (fixed to the dense Feb 10–14 date range). Static examples use a known-good entry range that consistently produces rich output.

**Consequences:**
- Eliminates redundant LLM calls for examples that don't benefit from fresh data
- Static examples are deterministic and reviewable — no surprise output changes on release
- Static examples become stale relative to schema or template changes — requires periodic manual refresh
- Dynamic examples still exist for content that should reflect the latest release

## ADR-8: Clean Template Rename Without Backward-Compat Alias

**Context:** The `exec-summary` draft template was renamed to `standup` for better discoverability. The question was whether to keep `exec-summary` as an alias for backward compatibility.

**Decision:** Clean break, no alias. The project is pre-GA with low usage. Backward compatibility adds code complexity — alias resolution, documentation for both names, test coverage for both paths — for effectively zero real-world benefit.

**Consequences:**
- Simpler template resolution with one canonical name per template
- Any existing scripts or documentation referencing `exec-summary` break without warning
- Sets a precedent that pre-GA naming changes are clean breaks — reduces hesitancy to fix naming mistakes early
- Must be revisited post-GA when breaking changes carry real user cost
