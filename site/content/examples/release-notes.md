+++
title = 'Release Notes'
date = '2026-05-10'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now place a `.timbersignore` at your repo root to extend the built-in skip rules with per-repo patterns (newline-delimited, supports prefix, suffix, and exact-path matching, `#` for comments).
- `timbers status --verbose` now reports an `infra_skipped_since_entry` count so you can see how many commits were filtered out by skip rules — also surfaced unconditionally in `--json` output.
- Reverts of already-documented commits are now auto-skipped from pending detection — the original entry serves as the audit trail. Multi-revert commits with any undocumented original still show up as pending.
- `timbers prime` now coaches agents to draft PR descriptions from your ledger entries when entries exist for the branch, falling back to ad-hoc summaries only when entries are missing or the operator dictates the body.

## Improvements

- `timbers prime` now emits a compact session-start summary by default, keeping the operational ledger safeguards while moving the full coaching guide behind `--full` / `guide`. Custom `PRIME.md` workflows surface as a hint instead of being inlined.
- Compact prime output now prints full resolvable entry IDs (`tb_<timestamp>_<sha>`) so you can paste them directly into `timbers show`.
- The default skip-rule set has been expanded to cover lockfiles across ecosystems (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, `Gemfile.lock`) and common housekeeping files (`.gitignore`, `.editorconfig`, and narrowly-scoped `.github/` metadata). Manifest changes (`package.json`, `go.mod`, `Cargo.toml`) still stay pending.
- The post-commit hook no longer nudges you to run `timbers log` when there's nothing actionable to document — it now agrees with `timbers pending` and `timbers log` on what counts as undocumented work.
- Provider model aliases (Anthropic, OpenAI, Gemini) have been refreshed to resolve to current official model IDs, with Gemini Flash-Lite pointing to the stable 3.1 model.
- The built-in `devblog` draft template now produces narratives that surface operator intent and human-agent collaboration instead of solo-developer hero voice, with a tightened length budget.
- The remaining six built-in draft templates (`changelog`, `decision-log`, `pr-description`, `release-notes`, `sprint-report`, `standup`) have been tuned for artifact-appropriate signal — including ADR Status/Date fields with supersession support, PR test-plan honesty, and stricter user-observable filtering on release notes.
- `timbers prime --json` now honestly reports the active mode in the output payload.

## Bug Fixes

- Short-SHA revert matching now requires at least 12 characters to avoid collisions between different commits.
- The exact-path skip matcher no longer mismatches files like `.gitignores` against the `.gitignore` rule.
- The compact health line in `timbers prime` now wraps consistently at 96 characters to match entry truncation.

## Breaking Changes

- **`.timbersignore` now lives at your repo root instead of inside `.timbers/`.** If you created one under `.timbers/.timbersignore` on a pre-release build, move it to `<repo>/.timbersignore`. No released version shipped the old location, so most users will not be affected.
