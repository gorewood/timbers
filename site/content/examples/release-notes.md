+++
title = 'Release Notes'
date = '2026-05-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now exempt commits from pending detection with a `.timbersignore` file — match by path, commit author (`author:dependabot[bot]`), or commit subject (`msg:chore: changelog for v*`).
- `timbers pending --explain` classifies every commit in range, showing exactly why each one is pending or skipped.
- `timbers help timbersignore` documents the three rule types and the canonical bot-exemption recipe.
- `timbers log --anchor <sha>` now documents a single commit even when nothing is pending, instead of refusing.
- `TIMBERS_DEBUG` environment variable surfaces a per-commit classification trace for diagnosing what the gate is doing.
- You can record an `ack` after a rebase to relink documented work — see the new `RebaseRelinkGuidance` section in `timbers prime` and the MCP helper output.

## Improvements

- `timbers status` now correctly counts commits skipped by `msg:` rules — previously the tally silently undercounted.
- `timbers doctor` now flags two stale-environment footguns: literal `[..]` character classes in `author:`/`msg:` globs (e.g. `author:dependabot[bot]` is a no-op), and shadowed `timbers` binaries on `PATH` reporting different versions.
- `timbers pending` prints a `.timbersignore` hint when matches exist, and at `count:0` notes when the anchor sits off the first-parent line (so "clean" isn't confused with "computed from an unusual anchor").
- When `timbers ack` can't commit because a stale hook binary blocked it, the error now explains the staged-but-uncommitted state and points at the upgrade, `timbers doctor`, and `git commit` recovery steps.
- `timbers log --batch` now picks an anchor on the first-parent line instead of naively using the first commit in the group — fixes side-branch anchors on multi-agent / multi-branch merges.
- Pending detection now correctly classifies a commit as "documented" via direct ledger membership (not only via revert detection), and falls back to a reachable-commits walk when the latest anchor sits off the first-parent line.
- `just release` now polls for the published GitHub release and installs it locally as a normal consumer would — sanity-checking the full release pipeline and keeping your installed binary current.
- `just release` no longer aborts if the ancillary site-example regeneration step flakes; it warns and proceeds to tag and push.

## Bug Fixes

- Fixed `timbers status` undercounting auto-skipped commits when a `msg:` rule matched.
- Fixed `timbers log --batch` occasionally anchoring entries to a side-branch commit instead of one on the first-parent line.
- Fixed pending detection returning weird ranges when the latest anchor was on a side branch.
- Fixed `timbers pending` treating documented commits as pending in cases where the documentation link came from direct ledger membership rather than revert detection.
