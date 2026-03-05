+++
title = 'Release Notes'
date = '2026-03-04'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- **Hook enforcement keeps you honest.** Timbers now blocks commits when you have undocumented work, and reminds you at session end. You'll see a clear `timbers log` command with the exact syntax to run—no guessing.
- **Post-commit reminders.** After each commit, you'll get a gentle nudge to document your work. Visible in any git client, not just Claude Code.
- **`timbers hooks status`** lets you see which hooks are installed and their current state.
- **Health check in `timbers prime`.** Session start now surfaces missing hooks or integrations upfront, with a `doctor --fix` hint so you can resolve issues before they bite.

## Improvements

- **Plays nicely with other tools.** Timbers now detects your environment—if another tool (like beads) manages git hooks, timbers adapts instead of overwriting. No more hook conflicts in multi-tool setups.
- **`timbers doctor`** is dramatically faster in large repositories. What previously took many seconds now completes near-instantly.
- **`timbers init`** makes smarter defaults based on your existing setup, so you're less likely to need manual configuration.
- **Old hook formats are automatically migrated** when you upgrade—no manual cleanup needed.

## Bug Fixes

- **`timbers log` no longer fails after a squash merge.** Previously, a squash merge could leave a stale reference that blocked logging entirely. Now it warns and continues.
- **Pending commit detection no longer false-positives on ledger entries.** Previously, timbers' own auto-committed entries would trigger "you have undocumented work" warnings—including at every session end.
- **Chained pre-commit hooks now properly propagate failures.** If timbers blocked a commit, a chained hook from another tool could silently override the block. Fixed.
- **Hooks now install to the correct directory** when `core.hooksPath` is configured. Previously, hooks were always written to `.git/hooks/`, which git ignores when `core.hooksPath` points elsewhere.
- **Uninstalling timbers now removes all hooks**, including ones from older versions that used different hook events.
