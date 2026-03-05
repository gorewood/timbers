+++
title = 'Release Notes'
date = '2026-03-04'
tags = ['example', 'release-notes']
+++

Generated with `timbers draft release-notes --last 20 | claude -p --model opus`

---



## New Features

- **Pipe your drafts to more LLM providers.** You can now use `timbers draft` with Codex and Gemini CLIs in addition to Claude, and the new `--models` flag shows available providers and API key setup.
- **Session health checks at startup.** `timbers prime` now runs a quick health check and tells you if hooks or integrations are missing—before you start working.
- **Post-commit reminders keep you on track.** A new git post-commit hook nudges you to run `timbers log` after each commit. Install it with `timbers init --hooks` or let `timbers doctor --fix` set it up automatically.
- **`timbers doctor` checks your generation setup.** Doctor now verifies that you have a CLI or API key configured for draft generation, so you're not surprised when piping fails.

## Improvements

- **Faster pending-commit detection in large repos.** Repos with thousands of commits no longer wait on slow per-commit lookups—batch processing makes `timbers doctor` and pending checks dramatically faster.
- **Friendlier onboarding experience.** If you set up timbers in a repo with existing history, it no longer treats every past commit as "pending"—just the ones after you started.
- **Better terminal colors on dark backgrounds.** Output is now legible on dark terminal themes like Solarized Dark, with colors that adapt to your background.
- **Renamed `exec-summary` template to `standup`.** The name better describes what it generates—use `timbers draft standup --since 1d` for daily standups.
- **Improved PR description template.** The `pr-description` template now focuses on intent and decisions rather than restating diffs.
- **Simpler hook enforcement.** Timbers now relies on a universal pre-commit git hook instead of editor-specific mechanisms, so it works the same way regardless of your git client.
- **`timbers uninstall` cleans up retired hooks.** Upgrading from older versions no longer leaves stale hook configurations behind.
- **macOS compatibility fix for example recipes.** Justfile examples now work correctly with macOS's default bash 3.2.

## Bug Fixes

- **Fixed false "pending commits" warning after every `timbers log`.** Ledger-only commits (from timbers itself) are no longer incorrectly flagged as undocumented work.
- **`timbers log` no longer fails after a squash merge.** If a branch was squash-merged and the anchor commit disappears, timbers now warns and continues instead of blocking you.
- **Fixed a security dependency issue** (CVE-2026-27896) in the MCP SDK.
- **Cleaned up leaked LLM commentary from the published changelog.** Meta-text from the generation process no longer appears in release documentation.
- **Stop hook now shows the full `timbers log` command syntax.** Previously, the session-end reminder was too terse to be actionable.
