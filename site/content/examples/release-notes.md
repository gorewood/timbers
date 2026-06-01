+++
title = 'Release Notes'
date = '2026-06-01'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now skip commits from the ledger by subject line with `msg:` rules in `.timbersignore` — handy for excluding release and changelog commits.
- Run `timbers pending --explain` to see exactly why each commit is, or isn't, counted as pending.
- `timbers doctor` now flags malformed `author:` and `msg:` globs (such as the `author:dependabot[bot]` pattern that silently matches nothing).
- `timbers doctor` now warns when an older `timbers` binary earlier on your `PATH` is shadowing your current install — a common cause of commits being unexpectedly blocked.
- A new `help timbersignore` topic documents the available rule types and the recommended recipe for ignoring bot commits.

## Improvements

- `timbers log --anchor` now documents the commit you name even when no pending commits are detected, instead of refusing.
- When your tree is clean but the anchor sits off the first-parent line, `timbers pending` now explains the situation rather than just reporting "No pending commits".
- `timbers pending` and `timbers doctor` now surface side-branch anchor topology, so merge-related pending behavior is no longer opaque.
- After an aborted commit, the pre-commit gate now tells you your staged changes are still in the index and points you at `git diff --cached` to inspect them.
- The `ack` commit-blocked message now explains the staged-but-uncommitted state and how to recover.
- `.timbersignore` rules are easier to discover, with hints surfaced in `timbers pending` and during onboarding.

## Bug Fixes

- `timbers status` now correctly counts commits skipped by `msg:` rules; it previously undercounted, making it look like a new rule wasn't filtering anything.
- `timbers log --batch` now anchors entries to a commit on the main line instead of sometimes selecting a side-branch commit.
- Pending detection now correctly recognizes already-documented commits in merge and side-branch histories, clearing spurious "pending" reports.

## Breaking Changes

- **`timbers log` now refuses to run on a dirty working tree** (it previously warned and proceeded, which could create phantom entries after an aborted commit). Commit your work first, or pass `--dry-run` to preview an entry without writing it; use `git diff --cached` to see what's staged.
