+++
title = 'Release Notes'
date = '2026-03-02'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now get a gentle reminder to document your work after every commit — `timbers init --hooks` installs a post-commit hook, and `timbers doctor --fix` can set it up automatically
- `timbers draft --models` shows which LLM providers and API keys are available on your system, so you don't have to guess what's configured
- `timbers doctor` now checks whether you have LLM generation set up and tells you exactly which environment variables to set if something's missing
- Piping `timbers draft` output to Codex and Gemini CLIs is now documented with verified syntax — Claude is no longer the only supported option

## Improvements

- New repos no longer flood `timbers pending` with your entire pre-timbers commit history — you start clean
- The `exec-summary` template has been renamed to `standup` for easier discovery (`timbers draft standup --since 1d`)
- The `pr-description` template now focuses on intent and design decisions rather than repeating diffs your reviewer can already see
- Terminal colors adapt to your background — no more invisible text on dark themes like Solarized Dark
- After a squash merge, you now get a clear, actionable warning instead of confusing stale-anchor errors — and the anchor self-heals on your next `timbers log`
- Retired hooks are automatically cleaned up when you upgrade

## Bug Fixes

- Fixed example recipes failing on macOS default bash (3.x compatibility)
- Fixed LLM commentary leaking into generated changelogs
- Fixed empty blog posts being generated when no ledger entries exist
- Patched a security dependency (CVE-2026-27896)

## Breaking Changes

- The `exec-summary` draft template is now `standup` — update any scripts that reference the old name
