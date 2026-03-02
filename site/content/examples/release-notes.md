+++
title = 'Release Notes'
date = '2026-03-02'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- `timbers log` now automatically commits the entry file — you no longer need a separate `git add` and `git commit` after logging your work
- You can now run `timbers draft --models` to see which AI providers and API keys are available on your system
- `timbers doctor` now checks whether you have an AI CLI or API key configured for draft generation, with setup hints if not

## Improvements

- New repos no longer show a wall of "pending" commits when you first set up timbers — `timbers pending` and `timbers doctor` recognize a fresh start and show a helpful tip instead
- The `exec-summary` draft template is now called `standup` — a more intuitive name for daily use (`timbers draft standup --since 1d`)
- The `pr-description` template now focuses on intent and decisions rather than repeating what's already visible in the diff
- `timbers prime` now coaches agents to keep secrets, PII, and credentials out of entries — a safety guardrail since entries are committed to git
- Piping `timbers draft` output to Gemini CLI and Codex is now documented with verified syntax
- Terminal colors now adapt to your background — no more invisible text on dark themes like Solarized Dark

## Bug Fixes

- Fixed stale anchor warnings after squash merges — you now get clear, actionable guidance instead of confusing errors
- Example recipes in the docs now work on macOS default bash (3.2) without requiring bash 4+
- Removed a session hook that fired silently with no visible output, eliminating a source of confusion
- Fixed a security vulnerability in a dependency (CVE-2026-27896)

## Breaking Changes

- The `exec-summary` draft template has been renamed to `standup` — update any scripts or aliases that reference the old name
