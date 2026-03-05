+++
title = 'Release Notes'
date = '2026-03-04'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- **`timbers draft --models`** shows you which AI providers are available and how to pipe to them—whether you use Claude, Codex, or Gemini
- **Post-commit reminders** nudge you to run `timbers log` after each commit, so nothing slips through the cracks
- **`timbers prime` now runs a quick health check** at session start—missing hooks or integrations surface immediately with a hint to fix them
- **Pre-commit hook enforcement** blocks commits when you have pending undocumented work, with a `Stop` backstop at session end

## Improvements

- **Friendlier onboarding**: repositories with no timbers history no longer show every past commit as "pending"—you start clean
- **`Stop` hook reason now includes the full `timbers log` syntax** so agents can act on it immediately instead of guessing
- **`timbers doctor`** is faster in large repositories—pending checks that previously stalled now complete promptly
- **`timbers log` works after squash merges** instead of erroring on a stale anchor—it warns and proceeds gracefully
- **macOS bash 3.x compatibility** for example justfile recipes
- **Multi-CLI piping documentation** verified for Codex and Gemini alongside Claude

## Bug Fixes

- Chained pre-commit hooks now correctly propagate the timbers exit code—previously a backup hook could silently override a block
- `HasPendingCommits` no longer false-positives after `timbers log`—ledger-only commits are filtered out instead of re-triggering the reminder
- `timbers uninstall` now removes retired hooks from older versions (v0.12/v0.13) that were previously left behind
- Fixed a security dependency (CVE-2026-27896) in the MCP SDK
- Cleaned up leaked LLM commentary from the published changelog
