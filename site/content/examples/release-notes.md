+++
title = 'Release Notes'
date = '2026-05-20'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now mark commits as intentionally skipped with `timbers ack`, recording an honest skip-with-reason instead of leaving them in pending.
- The new `.timbersignore` file at your repo root lets you configure per-repo skip rules, including `author:` globs for filtering bot or automation commits.
- Set `TIMBERS_DEBUG=1` to trace why specific commits are being skipped or surfaced as pending.
- `timbers prime --full` (or `--guide`) shows the complete onboarding guide; the default session-start output is now a compact summary.
- The default `devblog` template now coaches an operator-and-collaborator voice with explicit anti-fabrication guardrails for emotion, theme, and consequence.

## Improvements

- `timbers pending` no longer surfaces empty-file merge commits, cutting noise from auto-rebases and clean merges.
- In parallel-agent workflows, the pending gate now follows the first-parent line so one agent's undocumented commits no longer block another's. Set `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` to bypass remaining edge cases.
- `timbers log` now warns when the commit you're documenting has already been pushed upstream, catching push-before-log mistakes that strand entries locally.
- The post-commit hook now stays quiet for commits that touch only files like `.beads/issues.jsonl` — no more misleading "log this" nudges for non-actionable changes.
- Lockfiles (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, `Gemfile.lock`) are skipped by default when they appear alone in a commit; pair them with a manifest change and they stay pending.
- The session-start protocol now spells out commit → log → push ordering with an explicit "never push between" callout.
- Compact `timbers prime` output keeps full entry IDs (resolvable by `timbers show`) and hints when a custom `PRIME.md` is active.
- `timbers prime --json` now reports the requested mode honestly instead of always reporting `full`.
- `timbers prime` now coaches agents to draft PR descriptions from your ledger entries via the `pr-description` template when entries exist.
- Provider model aliases for Anthropic, OpenAI, and Gemini have been refreshed to current official model IDs; Gemini Flash-Lite now resolves to the stable 3.1 model.
- The `changelog`, `decision-log`, `pr-description`, `release-notes`, `sprint-report`, and `standup` templates have been tuned to better match each artifact's audience and reduce fabrication risk. ADRs now include `Status`, `Date`, and supersession fields.

## Bug Fixes

- Hooks no longer install a `timbers prime` invocation that breaks when the binary isn't on PATH — they now degrade gracefully with an install hint.

## Breaking Changes

- **`.timbersignore` now lives at your repo root**, not inside `.timbers/`. If you created one under the old location, move it to the repo root (e.g. `mv .timbers/.timbersignore .timbersignore`).
