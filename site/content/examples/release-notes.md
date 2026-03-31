+++
title = 'Release Notes'
date = '2026-03-31'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## Bug Fixes

- `timbers pending` no longer shows false-positive commits after a rebase — stale anchors are now caught immediately instead of lingering for weeks
- `timbers query --range` no longer returns empty results after a squash merge — entries are discoverable regardless of your merge strategy
- `--range` no longer silently drops entries when only some anchors are stale; all matching entries are now included
- `doctor --fix` now correctly removes stale hooks from global settings, not just project-local ones

## Improvements

- Hooks automatically pause during rebase, merge, cherry-pick, and revert — no more deadlocks that block you mid-operation
- Stale anchors degrade gracefully: `timbers pending` reports zero actionable items with clear guidance instead of dumping hundreds of false positives
- `timbers doctor` now checks your merge strategy configuration and detects stale anchors before they cause problems
- Security dependency update addressing a JSON parsing vulnerability in the underlying SDK
