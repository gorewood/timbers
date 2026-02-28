+++
title = 'Release Notes'
date = '2026-02-28'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now control terminal colors with `--color` (`never`, `auto`, or `always`) — helpful if your color scheme makes text hard to read
- `timbers log` now automatically commits the entry file, so you no longer need a separate `git add` and `git commit` step
- `timbers doctor` checks whether you have generation tools configured (LLM API keys or a CLI like `claude`) and tells you exactly what to set
- `timbers prime` now includes content safety reminders to help keep secrets and personal data out of your ledger entries
- New `timbers draft standup` template for generating daily standups from your recent entries

## Improvements

- Colors now adapt automatically to your terminal background — dark and light themes both work without configuration
- `timbers draft pr-description` has been rewritten to focus on intent and design decisions rather than restating diffs
- `timbers prime` now includes guidance for piping drafts through CLI tools
- Stale anchor warnings (after squash merges or rebases) are now actionable, with clear explanation and next steps
- Running `timbers init` on an existing installation detects and replaces outdated hooks automatically

## Bug Fixes

- Fixed dim and hint text being invisible on dark terminals like Solarized Dark
- Fixed `timbers prime` crashing on stale anchors instead of showing a helpful warning
- Fixed the post-commit reminder hook that was silently broken — it never actually detected commits
- Fixed draft generation producing empty posts when no ledger entries exist for the time range

## Breaking Changes

- `timbers draft exec-summary` has been renamed to `timbers draft standup`
- The PostToolUse hook has been removed — the Stop hook now handles pending-entry reminders at session end instead
