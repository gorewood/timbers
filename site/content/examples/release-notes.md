+++
title = 'Release Notes'
date = '2026-05-07'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- A `.timbersignore` file at your repo root now extends the built-in skip rules — list path prefixes, exact paths, or suffix patterns to keep pending detection quiet on housekeeping commits.
- Reverts of already-documented commits are now skipped from `timbers pending` automatically. The original entry remains the audit trail.
- `timbers status --verbose` now reports how many commits since your last entry were skipped by infrastructure rules; the same count is available as `infra_skipped_since_entry` in `--json`.
- `timbers doctor --fix` migrates legacy colon-encoded ledger filenames to the new dashed format. Run it once on existing repos to update entries written by older binaries.

## Improvements

- `timbers prime` now produces compact session-start output by default, preserving operational ledger safeguards without re-injecting the full guide every session. Use `timbers prime --full` (or `guide`) to print the complete workflow.
- Compact `timbers prime` output prints full entry IDs (so you can paste them straight into `timbers show`), hints when a custom `PRIME.md` is in use, exposes `custom_workflow` in JSON output, and correctly reports `mode: full` when `--full` is passed.
- Default skip rules now cover lockfile-only commits across major ecosystems: `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, and `Gemfile.lock`. Lockfile changes paired with manifest updates still appear as pending.
- Default skip rules also cover additional housekeeping files (`.gitignore`, `.editorconfig`, and similar), reducing pending-check noise.
- The seven built-in `timbers draft` templates (changelog, decision-log, devblog, pr-description, release-notes, sprint-report, standup) have been retuned for clearer output and stricter anti-fabrication guards. Drafts are less likely to invent emotions, themes, metrics, or test plans the entries don't actually support.
- The decision-log (ADR) template now produces entries with explicit `Status` and `Date` fields and supports supersession.
- The release-notes template now flags incomplete migration steps in breaking-change bullets rather than silently dropping them.
- `timbers prime` now coaches agents to draft PR descriptions from your ledger entries when you ask them to open a PR without a dictated body.
- Provider shorthand model aliases for Anthropic, OpenAI, and Gemini have been refreshed to current official model IDs, including a fix for the Gemini Flash-Lite alias to point at the stable 3.1 model. `timbers draft` and `timbers generate` now resolve to supported defaults.

## Bug Fixes

- `go install github.com/.../timbers@latest` now works on tagged releases. Colons in ledger filenames previously broke Go's module zip format and blocked installation for v0.16.x and v0.17.x. Older entries written with colon filenames are still read transparently.
- Skip-rule matching no longer treats `.gitignore` as a prefix that would match files like `.gitignores`.
- The infrastructure-skipped count surfaced by `timbers status` now uses the same filter as pending detection, so the number you see matches what actually gets skipped.
- Revert detection tightened its SHA match to 12+ characters to avoid short-SHA collisions when auto-skipping documented reverts.
