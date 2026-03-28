+++
title = 'Release Notes'
date = '2026-03-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now use `--range` with `timbers query` to filter entries by commit range, just like `export` and `draft`

## Improvements

- Hooks automatically pause during rebase, merge, cherry-pick, and revert — no more getting blocked mid-operation
- Stale anchors (from squash merges or rebases) no longer block hooks or flood `timbers pending` with false positives — you get a clear explanation instead of a wall of phantom commits
- `timbers doctor` now detects merge strategy misconfigurations and stale anchors before they cause problems
- The devblog template now uses a richer three-voice essay structure for more engaging output from `timbers draft devblog`
- Updated dependencies to address security vulnerabilities in JSON parsing and the Go standard library

## Bug Fixes

- `doctor --fix` now correctly removes stale hooks from global settings (previously it only cleaned project-local settings, leaving the real problem untouched)
- `--range` queries now reliably find entries after squash merges instead of returning empty results
- `--range` no longer silently drops entries when only some anchors are stale — both discovery methods now run unconditionally
