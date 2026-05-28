+++
title = 'Release Notes'
date = '2026-05-27'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- Record an honest skip-with-reason for a commit you're intentionally not documenting with the new `timbers ack` command — instead of leaving it pending or bypassing the hook.
- Exempt commits by author with `author:` rules in `.timbersignore`, so bot commits (like `dependabot`) stop showing up as work to document.
- Skip housekeeping commits by subject with `msg:` rules in `.timbersignore` — for example, release and changelog commits — without hiding real edits to the same files.
- Set `TIMBERS_DEBUG=1` to trace why each commit was counted as pending, documented, or skipped.
- Run `timbers help timbersignore` to see the supported rule types and a ready-to-use recipe for skipping bot commits.

## Improvements

- When `timbers pending` reports nothing pending but the result was computed from an off-main-line (side-branch) anchor, it now says so and points you at `--explain` and `--range` instead of a bare "No pending commits".
- `timbers pending` now reminds you that a `.timbersignore` is in effect when one is present.
- Empty merge commits no longer clutter your `timbers pending` list.
- `timbers pending --explain` now classifies every commit, showing why each one is pending or skipped.
- `timbers doctor` now warns when an older `timbers` binary earlier on your `PATH` could silently block your commits, and flags `.timbersignore` globs that look like literal `[...]` character classes (the `author:dependabot[bot]` footgun).
- `timbers log` now warns if you've already pushed the commit you're documenting before logging it.
- `timbers log --anchor` now documents that single commit even when nothing is pending, and its refusal message points you at `--range`.
- When `timbers ack` can't auto-commit, the error now explains the staged-but-uncommitted state and how to recover, and ack hints are now copy-pasteable for relinking entries after a rebase.

## Bug Fixes

- `timbers log --batch` no longer anchors an entry to a side-branch commit in merged histories — it prefers a commit on your main line.
- `timbers pending` now correctly recognizes already-documented commits in merge topologies, so they no longer reappear as pending.
