+++
title = 'Release Notes'
date = '2026-04-20'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now pass custom variables to draft templates with `--var key=value` for durable, caller-controlled output.
- The `decision-log` template now supports stable ADR numbering across appends, so `ADR-12` always refers to the same decision.
- A new `just decision-log` recipe automatically continues ADR numbering from your existing file.

## Improvements

- `timbers log` no longer gets confused when the `beads` auto-stage hook includes `.beads/` files in entry commits.
- Pending-commit detection now catches phantom commits that linger after a `git pull --rebase`, so you won't see work that's already documented.
- Hooks now stay out of your way during `git rebase`, `merge`, `cherry-pick`, and `revert` — no more deadlocks mid-operation.
- `timbers doctor --fix` now correctly cleans up retired hooks in global settings, not just project-local ones.
- The README now explains how Timbers works upfront, including why ledger entries get their own commits and how to filter them out of `git log`.
- Duplicate `--var` keys now raise a clear error instead of silently picking the last one.

## Bug Fixes

- Fixed `--range` silently dropping entries when some anchor commits were stale and others were valid.
- Fixed `timbers pending` showing already-documented commits after a rebase.
- Fixed `doctor --fix` leaving stale `PreToolUse` hooks in global Claude settings.
