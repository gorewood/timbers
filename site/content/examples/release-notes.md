+++
title = 'Release Notes'
date = '2026-05-02'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- A new `.timbersignore` file at your repo root lets you extend the built-in skip rules with custom patterns (vendor directories, lockfiles, etc.) without code changes.
- `timbers status --verbose` now reports how many infrastructure-only commits have been skipped since the latest entry, so you can spot when your skip rules are over-filtering. The count is always present in `--json` output as `infra_skipped_since_entry`.
- Reverts of already-documented commits are now automatically excluded from pending — the original entry serves as the audit trail. Multi-revert commits with any undocumented SHA stay pending to avoid silently dropping context.
- `timbers draft` accepts a repeatable `--var key=value` flag for caller-supplied template variables, accessible inside templates as `{{vars.key}}`.
- The `decision-log` template now supports durable ADR numbering via `{{vars.starting_number}}` so ADR-12 always means the same decision across regenerations. The `just decision-log` recipe finds the next number from your existing file and passes it through.
- Draft templates can declare default values for variables in their frontmatter under `vars:`.

## Improvements

- Lockfiles (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `go.sum`, `Cargo.lock`, `Gemfile.lock`) are now skipped from pending detection by default when committed alone. Lockfile changes paired with manifest changes still surface normally.
- Default skip rules now also cover housekeeping files like `.gitignore`, `.editorconfig`, and narrowly-scoped `.github/` metadata that carry no design intent.
- A latent bug where `.gitignore` would also match files like `.gitignores` is fixed — exact-path skip rules now match exactly.
- Built-in draft templates (`changelog`, `decision-log`, `devblog`, `pr-description`, `release-notes`, `sprint-report`, `standup`) have been substantially refined — clearer structure, anti-fabrication guardrails, and artifact-appropriate signal. ADR entries now include `Status` and `Date` fields with supersession support.
- `timbers prime` now coaches agents to draft PR descriptions from your ledger entries by default (when entries exist for the branch range), and includes operator-voice guidance for devblog drafting.
- `timbers prime --verbose` continues to surface the why/how of recent entries.

## Bug Fixes

- **`go install github.com/.../timbers/cmd/timbers@latest` now works on tagged releases.** Previously, colons in ledger filenames broke Go's module proxy. Existing entries remain readable; run `timbers doctor --fix` to migrate your ledger to the colon-free filenames.
- Fixed a "timbers-on-timbers" loop where repos with the beads auto-stage hook would have `.beads/issues.jsonl` appear in timbers entry commits, breaking pending detection. `.beads/` is now treated as infrastructure alongside `.timbers/`.

## Breaking Changes

- **`.timbersignore` moved from `.timbers/.timbersignore` to the repo root.** If you created one in the inner location during pre-release, move it to your repo root. This aligns with `.gitignore`, `.dockerignore`, and other ecosystem conventions.
