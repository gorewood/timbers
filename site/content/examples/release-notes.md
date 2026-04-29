+++
title = 'Release Notes'
date = '2026-04-29'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now pass custom variables to `timbers draft` templates with the `--var key=value` flag, enabling caller-supplied values without baking them into templates.
- The `decision-log` template now produces durable ADR numbers — append new decisions to an existing log without renumbering, and references like "ADR-12" stay stable forever.

## Improvements

- `timbers draft` templates can declare default variables in their frontmatter, so common cases work without extra flags.
- `timbers pending` no longer flags commits that touch only `.timbers/` or `.beads/` infrastructure files, keeping the ledger-on-ledger loop quiet.
- Beads sync now uses simple git-tracked `.beads/issues.jsonl` auto-flush — no more separate Dolt push/pull steps.
- The README now explains how timbers works upfront, including the separate-commit design and the optional pre-commit hook that enforces 1:1 commit cadence.
- `timbers log --help` and `timbers prime` now point you to the design rationale behind separate commits for ledger entries.
- Duplicate `--var` keys are now caught with a clear error instead of silently keeping the last value.

## Bug Fixes

- `go install github.com/...timbers@latest` now works — ledger filenames no longer contain colons that broke Go's module zip format. Existing repos can clean up via `timbers doctor --fix`, and older entries remain readable.
- `timbers pending` no longer shows phantom commits after a `git pull --rebase` — stale anchors are now detected via reachability, not just object existence.
- Hooks no longer fire during `git rebase`, `merge`, `cherry-pick`, or `revert`, fixing a deadlock where mid-rebase pending checks blocked progress.
- `timbers doctor --fix` now correctly cleans up stale hooks in global settings, not just project-local ones.
