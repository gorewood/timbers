+++
title = 'Release Notes'
date = '2026-03-31'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## Bug Fixes

- Rebase, merge, cherry-pick, and revert operations no longer deadlock — hooks now detect these in-progress git operations and stay out of the way
- `timbers pending` no longer shows phantom commits after rebasing; stale anchors from rewritten history are caught immediately instead of lingering
- `--range` queries no longer silently drop entries when some anchors are stale — both discovery methods now run unconditionally and results are merged
- `timbers doctor --fix` now correctly cleans up stale hooks from global settings, not just project-local ones
- Infrastructure files (like `.beads/` tracking data) no longer appear as undocumented pending work

## Improvements

- Stale anchors degrade gracefully — `timbers pending` reports zero actionable commits with clear guidance instead of flooding you with hundreds of false positives
- `timbers doctor` now checks your merge strategy (`pull.rebase`, `merge.ff`) and warns about configurations that tend to cause stale anchors
- New "How It Works" section in the README explains the commit design, how to filter ledger commits from `git log`, and how batch mode covers gaps when hooks are bypassed
