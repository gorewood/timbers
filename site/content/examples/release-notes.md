+++
title = 'Release Notes'
date = '2026-03-04'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- You can now pipe `timbers draft` output to additional AI CLIs — Codex and Gemini are now documented alongside Claude, and `timbers draft --models` shows available providers and API keys on your system
- `timbers prime` now runs a quick health check at session start, surfacing missing hooks or integrations with a `timbers doctor --fix` hint so you can resolve setup issues immediately
- After each commit, a post-commit hook reminds you to run `timbers log` — works with any git client, not just Claude Code
- `timbers doctor` now checks whether you have an AI CLI or API key configured for draft generation, so you know if `timbers draft` will work before you need it

## Improvements

- `timbers doctor` is significantly faster in large repositories — what previously took 16 seconds in a 2,000-commit repo now completes in under a second
- Terminal output is now readable on dark backgrounds (Solarized Dark, etc.) — colors adapt automatically to your terminal theme
- New repos no longer show every historical commit as "pending" — `timbers pending` recognizes a fresh setup and suggests `timbers log --batch` for your first entry instead
- `timbers log` now works smoothly after squash merges instead of failing with a stale anchor error — it warns and proceeds, self-healing on the next entry
- The `exec-summary` template has been renamed to `standup` (`timbers draft standup --since 1d`) for easier discovery
- The `pr-description` template now focuses on intent and decisions rather than restating the diff
- Example recipes in the documentation now work on macOS default bash (3.2)
- `timbers uninstall` now properly removes hooks from older versions, so upgrades leave a clean slate
- Hook enforcement is simpler and more reliable — `timbers init --hooks` sets up a pre-commit hook that works universally across all git clients

## Bug Fixes

- Fixed `timbers prime` crashing on stale anchor instead of showing a helpful warning
- Fixed a security dependency (CVE-2026-27896) in the MCP SDK
- Fixed leaked AI-generated commentary that had slipped into the published changelog
- Fixed the automated blog skipping generation when there are no new entries, instead of publishing empty posts
