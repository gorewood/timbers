+++
title = 'Release Notes'
date = '2026-05-17'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now exclude housekeeping files from pending detection per-repo by adding a `.timbersignore` file at the repo root, using the same prefix/suffix grammar as built-in skip rules.
- Reverts of already-documented commits are now auto-skipped from pending detection — the original entry serves as the audit trail, no fresh entry required.
- Set `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` to bypass the post-commit gate when working alongside other agents whose undocumented commits arrived via merge.
- `timbers status --verbose` now reports how many infrastructure-skipped commits have landed since the latest entry, so you can spot over-eager `.timbersignore` rules; `--json` always emits the field as `infra_skipped_since_entry`.
- `timbers prime` now coaches agents to draft PR descriptions from your ledger entries using the `pr-description` template when opening pull requests without a dictated body.

## Improvements

- `timbers prime` now produces a compact session-start payload by default, preserving operational ledger safeguards while spending less context on repeated coaching. Use `--full` (or `guide`) for the long-form version.
- Compact `prime` output keeps full, resolvable `tb_<timestamp>_<sha>` entry IDs so you can paste them straight into `timbers show`.
- Compact `prime` now hints when `PRIME.md` overrides the default workflow content, and `--json` correctly reports `mode: full` when requested.
- The post-commit hook no longer nudges you to log when there's no actionable pending work — `.beads/`-only commits, ignored files, and skipped vendor changes stop triggering false reminders.
- Multi-agent friendly: the pending gate now follows your branch's first-parent line, so another agent's undocumented commits arriving via merge don't block your work.
- Lockfile-only commits are now skipped by default across major ecosystems (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, `Gemfile.lock`). Manifest changes (`package.json`, `go.mod`, `Cargo.toml`) still stay pending.
- Provider model aliases for Anthropic, OpenAI, and Gemini now resolve to current official model IDs, so `draft` and `generate` commands use supported defaults.
- The built-in `devblog` template now surfaces operator intent and human-agent collaboration, with sharper tone calibration and tighter length budget.
- The built-in `pr-description`, `changelog`, `decision-log`, `release-notes`, `sprint-report`, and `standup` templates have been tuned for their specific audiences, with anti-fabrication guards against soft invention of emotions, themes, metrics, and benefits.
- ADR drafts now include `Status` and `Date` fields with explicit supersession support.
- `release-notes` drafts honestly flag breaking changes whose migration steps aren't in the entries, rather than silently dropping them.

## Breaking Changes

- **BREAKING:** `.timbersignore` moved from `.timbers/.timbersignore` to the repo root (`.timbersignore`) to match the convention of `.gitignore`, `.dockerignore`, and `.npmignore`. If you created one against a pre-release build, move it to your repo root.
