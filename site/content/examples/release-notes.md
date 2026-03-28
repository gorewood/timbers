+++
title = 'Release Notes'
date = '2026-03-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now use `--range` with `timbers query` to filter entries by commit range, matching what `export` and `draft` already support

## Improvements

- Hooks now stay out of your way during rebase, merge, cherry-pick, and revert — no more getting blocked mid-operation
- Squash merges no longer confuse `timbers pending` with hundreds of false-positive "undocumented" commits — you get a clear message instead
- `timbers doctor` now checks your merge strategy and warns about stale anchors before they cause problems

## Bug Fixes

- Fixed `--range` silently dropping entries when only some anchor commits were stale (e.g., after a partial squash merge)
- Fixed `timbers query --range` returning empty results after squash merges — entries are now found regardless of merge strategy
- Updated Go runtime and dependencies to patch security vulnerabilities in URL parsing and directory traversal
