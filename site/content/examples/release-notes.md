+++
title = 'Release Notes'
date = '2026-03-02'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---

## New Features

- You can now get a quick health check every time you start a session — `timbers prime` surfaces missing hooks and integrations with a hint to run `timbers doctor --fix`
- After each commit, a post-commit hook gently reminds you to run `timbers log` — set it up with `timbers init --hooks` or let `timbers doctor --fix` install it for you
- You can now check which LLM providers are available with `timbers draft --models`, including API key status
- You can now pipe `timbers draft` output to Gemini CLI and Codex CLI in addition to Claude

## Improvements

- Fresh installs no longer show your entire pre-timbers git history as "pending" — `timbers pending` starts clean from your first logged entry
- `timbers doctor` now checks whether content generation is set up (CLI tools and API keys) and tells you exactly what to configure
- The `exec-summary` template is now called `standup` — easier to discover and matches how people actually use it (`timbers draft standup`)
- The `pr-description` template now focuses on intent and decisions rather than rehashing diffs
- Colors now adapt to your terminal background — no more invisible text on dark themes like Solarized Dark

## Bug Fixes

- Fixed confusing warnings after squash merges — `timbers prime` now explains what happened and self-heals on your next `timbers log`
- Fixed example recipes failing on macOS default bash (3.x compatibility)
- Fixed a security vulnerability in a dependency (CVE-2026-27896)
- Fixed stray LLM commentary leaking into the published changelog

## Breaking Changes

- The `exec-summary` draft template has been renamed to `standup` — update any scripts that reference the old name
- The PostToolUse hook has been removed — session-end pending checks via the Stop hook now cover the same case more reliably
