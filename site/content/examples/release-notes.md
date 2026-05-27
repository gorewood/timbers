+++
title = 'Release Notes'
date = '2026-05-27'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

# Release Notes

## New Features

- You can now run `timbers ack <commit>` to record an honest skip-with-reason for a commit you don't want to log — no more fabricated entries or bypassing the hook with `--no-verify`.
- `timbers pending --explain` classifies every commit, showing why each one is pending, documented, or skipped.
- `.timbersignore` now supports `author:<glob>` rules, so you can skip commits from specific authors such as bot accounts.
- `.timbersignore` now supports `msg:<glob>` rules, so you can skip commits whose subject matches a pattern (for example, `msg:chore: changelog for v*`).
- Run `timbers help timbersignore` for a built-in guide to the `.timbersignore` rule types and the canonical bot-skip recipe.
- Set `TIMBERS_DEBUG=1` to see a trace of how each commit was classified during a pending check.
- Set `TIMBERS_SKIP_CROSS_AGENT_DEBT=1` to bypass the commit gate when the undocumented commits came from another agent rather than your own work.

## Improvements

- `timbers doctor` now warns when an out-of-date `timbers` earlier on your `PATH` is shadowing your current install — a common, hard-to-spot cause of blocked commits.
- `timbers doctor` now flags `.timbersignore` `author:`/`msg:` globs containing literal-looking `[...]` brackets, which silently match nothing.
- `timbers pending` now points you at `.timbersignore` when skip rules apply, and surfaces clearer diagnostics when your latest entry sits on a side branch.
- `timbers log` now warns when you've already pushed the commit you're documenting but haven't recorded its entry yet, so entries don't get stranded locally.
- After a rebase, `timbers pending` and `timbers doctor` suggest a ready-to-paste `ack` reason for re-linking moved commits.
- Empty merge commits no longer clutter `timbers pending`.
- When an `ack`'s auto-commit is blocked by a stale hook, the error now explains the staged-but-uncommitted state and how to recover.

## Bug Fixes

- `timbers` no longer blocks your commit over undocumented commits that came from a parallel agent or a merge — the commit gate now only considers work on your branch's own line of history.
- `timbers log --batch` no longer anchors an entry to a side-branch commit, which previously produced entries that downstream commands mis-read.
- Commits you've already documented are now reliably recognized as documented, even when your latest entry is on a side branch.
- Clean and empty merge commits no longer block your commit at the gate.
