+++
title = 'Release Notes'
date = '2026-02-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now control terminal colors with `--color never|auto|always` — helpful if dim text was hard to read on your terminal theme
- `timbers log` now auto-commits the entry file, so you no longer need a separate `git add` and `git commit` step
- `timbers draft --models` shows which AI providers are available and whether their API keys are configured
- `timbers doctor` now checks your draft generation setup — it tells you which CLI tools and API keys are ready for `timbers draft` piping
- You can now pipe `timbers draft` output to Gemini CLI and Codex CLI in addition to Claude

## Improvements

- The `exec-summary` template is now called `standup` — easier to remember (`timbers draft standup`)
- The `pr-description` template focuses on intent and design decisions instead of rehashing diffs
- Agents are now coached to keep secrets, API keys, and personal data out of ledger entries
- Stale hooks from older versions are automatically cleaned up on upgrade

## Bug Fixes

- Colors now adapt to dark terminal backgrounds — dim text is no longer invisible on themes like Solarized Dark
- `timbers prime` now handles stale anchors gracefully after squash merges, instead of showing confusing errors
- Fixed the post-commit reminder hook, which had been silently broken since it was first created

## Breaking Changes

- The `exec-summary` template has been renamed to `standup` — update any scripts that reference the old name
