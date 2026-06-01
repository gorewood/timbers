+++
title = 'Release Notes'
date = '2026-06-01'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now record an `ack` to mark a commit as documented elsewhere — useful after a rebase, when the reasoning already lives in another ledger entry.
- You can now skip commits by author with `author:` rules in `.timbersignore` (for example, to exclude bot commits like `dependabot[bot]`).
- You can now skip commits by subject line with `msg:` rules in `.timbersignore` — handy for release and housekeeping commits that touch otherwise-tracked files.
- Run `timbers pending --explain` to see every commit classified, with the reason each one is or isn't pending.
- A new `help timbersignore` topic documents the three rule types and shows the canonical recipe for excluding bot commits.
- `timbers doctor` now flags a stale `timbers` binary earlier in your `PATH` that could silently block commits, and warns when an `author:`/`msg:` glob uses a literal `[..]` that won't match anything.
- Set `TIMBERS_DEBUG` to trace how each commit is classified when you're tuning your skip rules.

## Improvements

- `timbers log --anchor <commit>` now documents that single commit even when nothing is pending, instead of refusing; the refusal message points you to `--range` for explicit control.
- When your anchor sits off the first-parent line, `timbers pending` now explains the situation and points at `--explain`/`--range` rather than reporting a bare "No pending commits".
- `timbers pending` shows a `.timbersignore` hint when skip rules are active, and surfaces when merge commits are being skipped.
- When a stale hook binary blocks an `ack` commit, the error now explains the staged-but-uncommitted state and points you at the upgrade, `timbers doctor`, and `git commit` recovery steps.

## Bug Fixes

- `timbers log` now refuses to run on a dirty working tree instead of warning and proceeding, which previously produced phantom entries that rode the wrong commit. Commit your work first, or use `--dry-run` to inspect an entry while debugging.
- `timbers status` now counts commits skipped by a `msg:` rule correctly — it previously undercounted, so a newly added rule looked like it wasn't filtering anything.
- `timbers log --batch` now anchors entries to a commit on your first-parent line instead of sometimes landing on a side-branch commit.
- Pending detection now correctly recognizes already-documented commits and handles merged side-branch topologies, so fewer commits are wrongly reported as pending.
