+++
title = 'Release Notes'
date = '2026-06-10'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## Timbers v0.23.0

### New Features

- A new readable panel renders multi-line **what / why / how / notes** in aligned, wrapped columns when you run `timbers show` and `timbers log --dry-run` — long fields no longer collapse into one bleeding line.
- You can now tune the staleness window per repo with a `session-window:` directive in `.timbersignore`, so long-running sessions (multi-hour refactors, agent fan-outs) aren't treated as stale. `timbers doctor` warns if the value is malformed.

### Improvements

- The commit gate now recognizes work that isn't yours to document — commits from a different author (resolved through your `.mailmap`) and commits older than the session window are skipped automatically instead of nagging you to log them.
- `timbers prime` now counts only in-session commits as pending, so the number you're asked to drive to zero reflects work you can actually document.
- When the gate aborts a commit, `timbers` now tells you your staged changes are still in the index and points you at `git diff --cached` to inspect them.
- `.timbersignore` is now discoverable: `timbers pending` hints at it when rules exist, `timbers pending --explain` classifies every commit, and `timbers help timbersignore` documents the rule types.
- `timbers doctor` now flags a common footgun — an `author:dependabot[bot]`-style glob that silently matches nothing — and warns when `user.email` is unset.
- `timbers log --anchor <sha>` now documents that single commit even when nothing is detected as pending, and refusals point you at `--range` as the explicit escape hatch.
- `timbers pending` now distinguishes a genuinely clean state from one computed off an off-first-parent anchor, instead of b-showing a bare "No pending commits."
- When a long session skips its own stale work, a post-commit note now surfaces that something was auto-skipped so it doesn't vanish silently.

### Bug Fixes

- `timbers log --dry-run` no longer drops the **Notes** field from its preview.
- The diffstat now renders consistently between `timbers show` and `timbers log --dry-run`.
- `timbers status` now counts commits skipped by a `msg:` rule correctly — previously it could under-report whether a newly added rule was filtering anything.

### Breaking Changes

- **`timbers log` now refuses to run on a dirty working tree** instead of warning and proceeding — this closes a hole where an aborted commit could produce a phantom ledger entry. Commit your work first, or use `timbers log --dry-run` to inspect an entry without writing it. There is intentionally no `--allow-dirty` flag.
