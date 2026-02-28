+++
title = 'Release Notes'
date = '2026-02-27'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now control terminal colors with `--color never|auto|always` — useful if your terminal theme makes text hard to read
- `timbers log` now auto-commits the entry file, so you no longer need a separate `git add` and `git commit` step
- `timbers doctor` checks whether you have a generation tool configured (Claude CLI or API key) and tells you exactly what to set up
- The `exec-summary` template is now called `standup` — same content, easier to find when you need it
- `timbers prime` now includes content safety reminders, helping agents avoid accidentally logging secrets or personal data in entries

## Improvements

- Colors automatically adapt to dark terminal backgrounds — dim text that was invisible on themes like Solarized Dark now renders clearly
- The `pr-description` template focuses on intent and design decisions instead of restating diffs
- `timbers prime` coaching now defaults to pipe-first generation (`timbers draft ... | claude -p`), which uses your subscription instead of API tokens
- Stale anchor warnings after squash merges are now actionable — you get clear guidance instead of confusing errors, and the anchor self-heals on your next `timbers log`
- Reinstalling timbers automatically cleans up retired hook events, so you don't accumulate stale hooks over time

## Bug Fixes

- Fixed the post-commit reminder hook, which had been silently broken since it was created — it was reading an empty environment variable instead of stdin
- Fixed a crash in `timbers prime` when encountering a stale anchor commit after a squash merge

## Breaking Changes

- The `exec-summary` template has been renamed to `standup` — update any scripts or aliases that reference the old name
