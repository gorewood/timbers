+++
title = 'Release Notes'
date = '2026-05-27'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes (v0.20.0 – v0.22.3)

## New Features
- You can now run `timbers ack` to record an honest reason for intentionally not documenting a commit, instead of bypassing the hook.
- You can now skip commits by author with `author:<glob>` lines in `.timbersignore` — handy for filtering out bot commits.
- You can now skip commits by their subject line with `msg:<glob>` lines in `.timbersignore`.
- Set `TIMBERS_DEBUG=1` to see a trace of how `timbers pending` classifies each commit.
- Set `TIMBERS_SKIP_CROSS_AGENT_DEBT` to bypass the commit gate when it's blocking on another agent's undocumented work.

## Improvements
- In parallel-agent workflows, the commit gate now only considers commits on your branch's first-parent line, so you're no longer blocked by another agent's undocumented commits.
- `timbers pending` no longer lists empty merge commits that add no work to your branch, reducing noise.
- `timbers pending` and `timbers doctor` now flag when your latest entry's anchor sits on a side branch and point you at the next step.
- `timbers log` now warns when you've already pushed the commit you're documenting, catching entries that would otherwise be stranded locally.
- `timbers doctor` now detects when a stale `timbers` binary earlier on your `PATH` is shadowing your installed version.
- When `timbers ack` can't commit its record, the error now explains the staged-but-uncommitted state and points you at recovery.
- `timbers prime` now emits a more compact session-context payload.

## Bug Fixes
- The post-commit hook no longer nudges you to document commits that only touch skipped files (such as `.beads/`-only commits).
- `timbers pending` now correctly recognizes already-documented commits across merge and branch topologies, instead of re-listing documented work.
- `timbers log --batch` now anchors entries to a commit on your branch's first-parent line rather than occasionally picking a side-branch commit.
