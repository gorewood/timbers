+++
title = 'Release Notes'
date = '2026-05-21'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes — v0.22.0

## New Features

- **`timbers ack`** — You can now acknowledge commits as intentionally skipped (with a reason) instead of being nudged to log them. This gives you an honest skip path for work that legitimately doesn't need a ledger entry.
- **Author globs in `.timbersignore`** — You can now skip commits by author using `author:<glob>` lines in `.timbersignore` (e.g. `author:dependabot[bot]*`). Useful for filtering bot-authored commits out of pending detection.
- **`TIMBERS_DEBUG=1`** — Set this environment variable to get a trace of how pending detection classifies each commit, so you can see why something was (or wasn't) flagged.
- **PR-description authoring guidance** — `timbers prime` now coaches agents to draft PR bodies from your ledger entries by default when entries exist for the branch.

## Improvements

- Merge commits with no file changes are no longer shown in `timbers pending`, removing noise from the most common false positives.
- `timbers log` now warns when you've already pushed the commit you're documenting — surfacing the push-before-log race that previously stranded entries locally.
- The session-start protocol now spells out the commit → log → push ordering explicitly, with a "never push between commit and log" callout.
- `docs/agent-dx-guide.md` covers the new surfaces (`ack`, author globs, `TIMBERS_DEBUG`, merge-skip display) alongside existing gates.

## Bug Fixes

- **Post-commit hook no longer nudges on commits with no actionable pending work** — fixes the false nudge when commits touch only `.beads/` or other ignored paths (v0.20.1).
- **First-parent gate fix for parallel agents** — the timbers gate now scopes to the first-parent line of history, so agent A is no longer blocked by undocumented commits from agent B on a merged side branch (v0.21.0).
- **`TIMBERS_SKIP_CROSS_AGENT_DEBT`** escape hatch added for the residual case where a merge commit itself touched source files during conflict resolution (v0.21.0).
- Provider model aliases refreshed — `draft` and `generate` shortcuts now resolve to current official model IDs (including the stable Gemini 3.1 Flash-Lite).
